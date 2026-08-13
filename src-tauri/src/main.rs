#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::{
    collections::HashMap,
    io,
    path::Path,
    sync::{Arc, Mutex},
    time::{Duration, Instant},
};
use tauri::{
    menu::{Menu, MenuItem},
    tray::TrayIconBuilder,
    Manager, WebviewWindowBuilder,
};
#[cfg(target_os = "windows")]
use windows::{
    core::HSTRING,
    Win32::{
        Foundation::{CloseHandle, GetLastError, HANDLE, INVALID_HANDLE_VALUE, ERROR_PIPE_CONNECTED},
        Storage::FileSystem::{ReadFile, WriteFile, PIPE_ACCESS_DUPLEX},
        System::Pipes::{ConnectNamedPipe, CreateNamedPipeW, DisconnectNamedPipe, PIPE_READMODE_BYTE, PIPE_TYPE_BYTE, PIPE_UNLIMITED_INSTANCES, PIPE_WAIT},
    },
};

#[tauri::command]
async fn vc_minimize(window: tauri::WebviewWindow) { window.minimize().ok(); }
#[tauri::command]
async fn vc_toggle_maximize(window: tauri::WebviewWindow) {
    if window.is_maximized().unwrap_or(false) { window.unmaximize().ok(); } else { window.maximize().ok(); }
}
#[tauri::command]
async fn vc_hide(window: tauri::WebviewWindow) { window.hide().ok(); }
fn show_main_window(window: &tauri::WebviewWindow) {
    window.unminimize().ok();
    window.show().ok();
    window.set_always_on_top(true).ok();
    window.set_focus().ok();
    window.set_always_on_top(false).ok();
}
#[tauri::command]
async fn vc_start_drag(window: tauri::WebviewWindow) { window.start_dragging().ok(); }
#[tauri::command]
async fn vc_open_url(url: String) {
    #[cfg(target_os = "windows")]
    std::process::Command::new("cmd").args(["/C","start","", &url]).spawn().ok();
    #[cfg(target_os = "macos")]
    std::process::Command::new("open").arg(&url).spawn().ok();
    #[cfg(target_os = "linux")]
    std::process::Command::new("xdg-open").arg(&url).spawn().ok();
}

#[tauri::command]
async fn vc_reload_vencord(window: tauri::WebviewWindow) {
    let vencord_dir = std::path::PathBuf::from("vencord");
    let vencord_css = std::fs::read_to_string(vencord_dir.join("browser.css")).unwrap_or_else(|_| include_str!("../vencord/browser.css").to_string());
    let css_json = serde_json::to_string(&vencord_css).unwrap();
    window.eval(&format!(
        "var s=document.getElementById('vencord-css');if(s)s.textContent={};else{{var s=document.createElement('style');s.id='vencord-css';s.textContent={};document.head.appendChild(s);}}location.reload();",
        css_json, css_json
    )).ok();
}

#[tauri::command]
async fn vc_update_vencord(_window: tauri::WebviewWindow) -> Result<String, String> {
    let vencord_dir = std::path::PathBuf::from("vencord");
    std::fs::create_dir_all(&vencord_dir).map_err(|e| e.to_string())?;
    let client = reqwest::Client::new();
    let js_resp = client.get("https://github.com/Vendicated/Vencord/releases/download/devbuild/browser.js").send().await.map_err(|e| e.to_string())?;
    if !js_resp.status().is_success() { return Err(format!("Failed to download browser.js: {}", js_resp.status())); }
    let js_bytes = js_resp.bytes().await.map_err(|e| e.to_string())?;
    std::fs::write(vencord_dir.join("browser.js"), js_bytes.as_ref()).map_err(|e| e.to_string())?;
    let css_resp = client.get("https://github.com/Vendicated/Vencord/releases/download/devbuild/browser.css").send().await.map_err(|e| e.to_string())?;
    if !css_resp.status().is_success() { return Err(format!("Failed to download browser.css: {}", css_resp.status())); }
    let css_bytes = css_resp.bytes().await.map_err(|e| e.to_string())?;
    std::fs::write(vencord_dir.join("browser.css"), css_bytes.as_ref()).map_err(|e| e.to_string())?;
    if let Ok(ts) = std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH) {
        std::fs::write(vencord_dir.join(".last_update"), ts.as_secs().to_string()).ok();
    }
    Ok("Vencord updated. Restart to apply.".to_string())
}

fn sanitize_filename(filename: &str) -> String {
    let cleaned = filename
        .trim()
        .trim_matches('"')
        .chars()
        .map(|c| match c {
            '<' | '>' | ':' | '"' | '/' | '\\' | '|' | '?' | '*' | '\0'..='\u{1f}' => '_',
            _ => c,
        })
        .collect::<String>();
    let cleaned = cleaned.trim_matches([' ', '.']);
    if cleaned.is_empty() { "download".to_string() } else { cleaned.to_string() }
}

fn percent_decode(input: &str) -> String {
    let bytes = input.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' && i + 2 < bytes.len() {
            if let Ok(hex) = u8::from_str_radix(&input[i + 1..i + 3], 16) {
                out.push(hex);
                i += 3;
                continue;
            }
        }
        out.push(bytes[i]);
        i += 1;
    }
    String::from_utf8_lossy(&out).into_owned()
}

fn filename_from_content_disposition(value: &str) -> Option<String> {
    for part in value.split(';').map(str::trim) {
        if let Some(encoded) = part.strip_prefix("filename*=") {
            let encoded = encoded.trim_matches('"');
            let encoded = encoded.strip_prefix("UTF-8''").unwrap_or(encoded);
            return Some(sanitize_filename(&percent_decode(encoded)));
        }
        if let Some(filename) = part.strip_prefix("filename=") {
            return Some(sanitize_filename(&percent_decode(filename)));
        }
    }
    None
}

fn extension_from_content_type(value: &str) -> Option<&'static str> {
    match value.split(';').next()?.trim().to_ascii_lowercase().as_str() {
        "image/avif" => Some("avif"),
        "image/bmp" => Some("bmp"),
        "image/gif" => Some("gif"),
        "image/jpeg" => Some("jpg"),
        "image/png" => Some("png"),
        "image/svg+xml" => Some("svg"),
        "image/webp" => Some("webp"),
        "video/mp4" => Some("mp4"),
        "video/quicktime" => Some("mov"),
        "video/webm" => Some("webm"),
        "audio/mpeg" => Some("mp3"),
        "audio/ogg" => Some("ogg"),
        "audio/wav" | "audio/wave" | "audio/x-wav" => Some("wav"),
        "application/pdf" => Some("pdf"),
        "application/zip" => Some("zip"),
        "application/x-7z-compressed" => Some("7z"),
        "text/plain" => Some("txt"),
        _ => None,
    }
}

fn filename_from_url(url: &str) -> String {
    let path = url.split('?').next().unwrap_or(url);
    sanitize_filename(&percent_decode(path.rsplit('/').next().unwrap_or("download")))
}

fn filename_from_destination(path: &Path) -> Option<String> {
    path.file_name().and_then(|name| name.to_str()).map(sanitize_filename).filter(|name| !name.is_empty())
}

fn best_download_filename(url: &str) -> String {
    let fallback = filename_from_url(url);
    let has_extension = Path::new(&fallback).extension().is_some();
    let response = reqwest::blocking::Client::builder()
        .timeout(Duration::from_secs(2))
        .redirect(reqwest::redirect::Policy::limited(5))
        .build()
        .and_then(|client| client.head(url).send());

    if let Ok(response) = response {
        if let Some(filename) = response
            .headers()
            .get(reqwest::header::CONTENT_DISPOSITION)
            .and_then(|value| value.to_str().ok())
            .and_then(filename_from_content_disposition)
        {
            return filename;
        }

        if !has_extension {
            if let Some(ext) = response
                .headers()
                .get(reqwest::header::CONTENT_TYPE)
                .and_then(|value| value.to_str().ok())
                .and_then(extension_from_content_type)
            {
                return format!("{fallback}.{ext}");
            }
        }
    }

    fallback
}

fn check_and_update_vencord() {
    let vencord_dir = std::path::PathBuf::from("vencord");
    let last_update_file = vencord_dir.join(".last_update");
    let should_update = match std::fs::read_to_string(&last_update_file) {
        Ok(content) => match content.parse::<u64>() {
            Ok(timestamp) => {
                let now = std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).unwrap().as_secs();
                now.saturating_sub(timestamp) > 86400
            }
            Err(_) => true,
        }
        Err(_) => true,
    };
    if !should_update { return; }
    let rt = tokio::runtime::Runtime::new().unwrap();
    rt.block_on(async {
        let _ = std::fs::create_dir_all(&vencord_dir);
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(15))
            .build().unwrap();
        let js = match client.get("https://github.com/Vendicated/Vencord/releases/download/devbuild/browser.js").send().await {
            Ok(resp) if resp.status().is_success() => resp.bytes().await.ok(),
            _ => None,
        };
        let css = match client.get("https://github.com/Vendicated/Vencord/releases/download/devbuild/browser.css").send().await {
            Ok(resp) if resp.status().is_success() => resp.bytes().await.ok(),
            _ => None,
        };
        if let (Some(js), Some(css)) = (js, css) {
            let _ = std::fs::write(vencord_dir.join("browser.js"), js.as_ref());
            let _ = std::fs::write(vencord_dir.join("browser.css"), css.as_ref());
            if let Ok(ts) = std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH) {
                let _ = std::fs::write(vencord_dir.join(".last_update"), ts.as_secs().to_string());
            }
        }
    });
}

#[cfg(target_os = "windows")]
fn start_discord_rpc_bridge() {
    for index in 0..1 {
        std::thread::spawn(move || loop {
            if let Ok(pipe) = create_rpc_pipe(index) {
                let connected = unsafe { ConnectNamedPipe(pipe, None).is_ok() || GetLastError() == ERROR_PIPE_CONNECTED };
                if connected { serve_rpc_client(pipe).ok(); }
                unsafe {
                    DisconnectNamedPipe(pipe).ok();
                    CloseHandle(pipe).ok();
                }
            }
            std::thread::sleep(Duration::from_millis(100));
        });
    }
}

#[cfg(not(target_os = "windows"))]
fn start_discord_rpc_bridge() {}

#[cfg(target_os = "windows")]
fn create_rpc_pipe(index: u8) -> io::Result<HANDLE> {
    let name = HSTRING::from(format!(r"\\.\pipe\discord-ipc-{index}"));
    let pipe = unsafe {
        CreateNamedPipeW(
            &name,
            PIPE_ACCESS_DUPLEX,
            PIPE_TYPE_BYTE | PIPE_READMODE_BYTE | PIPE_WAIT,
            PIPE_UNLIMITED_INSTANCES,
            65536,
            65536,
            0,
            None,
        )
    };
    if pipe == INVALID_HANDLE_VALUE { Err(io::Error::last_os_error()) } else { Ok(pipe) }
}

#[cfg(target_os = "windows")]
fn serve_rpc_client(pipe: HANDLE) -> io::Result<()> {
    loop {
        let Some((op, payload)) = read_rpc_frame(pipe)? else { return Ok(()); };
        let value = serde_json::from_slice::<serde_json::Value>(&payload).unwrap_or_else(|_| serde_json::json!({}));
        match op {
            0 => {
                let client_id = value.get("client_id").and_then(|v| v.as_str()).unwrap_or("0");
                write_rpc_frame(pipe, 1, &serde_json::json!({"cmd":"DISPATCH","evt":"READY","data":{"v":1,"config":{"cdn_host":"cdn.discordapp.com","api_endpoint":"//discord.com/api","environment":"production"},"user":{"id":"0","username":"Discord","discriminator":"0000","avatar":null,"bot":false},"client_id":client_id}}))?;
            }
            1 => handle_rpc_command(pipe, value)?,
            3 => write_rpc_frame(pipe, 4, &value)?,
            _ => {}
        }
    }
}

#[cfg(target_os = "windows")]
fn handle_rpc_command(pipe: HANDLE, value: serde_json::Value) -> io::Result<()> {
    let cmd = value.get("cmd").and_then(|v| v.as_str()).unwrap_or("");
    let nonce = value.get("nonce").cloned().unwrap_or(serde_json::Value::Null);
    match cmd {
        "SET_ACTIVITY" => write_rpc_frame(pipe, 1, &serde_json::json!({"cmd":"SET_ACTIVITY","data":null,"evt":null,"nonce":nonce})),
        "SUBSCRIBE" | "UNSUBSCRIBE" => write_rpc_frame(pipe, 1, &serde_json::json!({"cmd":cmd,"data":null,"evt":null,"nonce":nonce})),
        "AUTHORIZE" => write_rpc_frame(pipe, 1, &serde_json::json!({"cmd":"AUTHORIZE","data":{"code":"discord-app-local"},"evt":null,"nonce":nonce})),
        "AUTHENTICATE" => write_rpc_frame(pipe, 1, &serde_json::json!({"cmd":"AUTHENTICATE","evt":"ERROR","data":{"code":4000,"message":"authentication unavailable"},"nonce":nonce})),
        _ => write_rpc_frame(pipe, 1, &serde_json::json!({"cmd":cmd,"evt":"ERROR","data":{"code":4000,"message":"unknown command"},"nonce":nonce})),
    }
}

#[cfg(target_os = "windows")]
fn read_rpc_frame(pipe: HANDLE) -> io::Result<Option<(u32, Vec<u8>)>> {
    let mut header = [0u8; 8];
    if !read_exact_pipe(pipe, &mut header)? { return Ok(None); }
    let op = u32::from_le_bytes(header[0..4].try_into().unwrap());
    let len = u32::from_le_bytes(header[4..8].try_into().unwrap()) as usize;
    if len > 16 * 1024 * 1024 { return Err(io::Error::new(io::ErrorKind::InvalidData, "rpc frame too large")); }
    let mut payload = vec![0u8; len];
    if !read_exact_pipe(pipe, &mut payload)? { return Ok(None); }
    Ok(Some((op, payload)))
}

#[cfg(target_os = "windows")]
fn read_exact_pipe(pipe: HANDLE, buf: &mut [u8]) -> io::Result<bool> {
    let mut offset = 0;
    while offset < buf.len() {
        let mut read = 0u32;
        if unsafe { ReadFile(pipe, Some(&mut buf[offset..]), Some(&mut read), None).is_err() } { return Ok(false); }
        if read == 0 { return Ok(false); }
        offset += read as usize;
    }
    Ok(true)
}

#[cfg(target_os = "windows")]
fn write_rpc_frame(pipe: HANDLE, op: u32, value: &serde_json::Value) -> io::Result<()> {
    let payload = serde_json::to_vec(value).map_err(|e| io::Error::new(io::ErrorKind::InvalidData, e))?;
    let mut frame = Vec::with_capacity(8 + payload.len());
    frame.extend_from_slice(&op.to_le_bytes());
    frame.extend_from_slice(&(payload.len() as u32).to_le_bytes());
    frame.extend_from_slice(&payload);
    let mut written = 0u32;
    unsafe { WriteFile(pipe, Some(&frame), Some(&mut written), None).map_err(|_| io::Error::last_os_error())?; }
    if written as usize == frame.len() { Ok(()) } else { Err(io::Error::new(io::ErrorKind::WriteZero, "short rpc write")) }
}

const SPOOF_JS: &str = r#"
(function(){
  function patchSp(sp){try{var o=JSON.parse(atob(sp));if(o.browser==='chrome'){o.browser='discord';o.browser_version='';}return btoa(JSON.stringify(o));}catch(e){return sp;}}
  var origFetch=window.fetch;
  window.fetch=function(input,init){
    if(init&&init.headers){
      var h=init.headers;
      if(typeof h==='object'&&!(h instanceof Headers)){if(h['X-Super-Properties'])h['X-Super-Properties']=patchSp(h['X-Super-Properties']);if(h['x-super-properties'])h['x-super-properties']=patchSp(h['x-super-properties']);}
      else if(h instanceof Headers){if(h.has('X-Super-Properties'))h.set('X-Super-Properties',patchSp(h.get('X-Super-Properties')));}
    }
    return origFetch.apply(this,arguments);
  };
  var origOpen=XMLHttpRequest.prototype.open;
  var origSend=XMLHttpRequest.prototype.send;
  XMLReqPatched:XMLHttpRequest.prototype.send=function(body){
    if(this._vcSpHeader)this.setRequestHeader('X-Super-Properties',this._vcSpHeader);
    return origSend.apply(this,arguments);
  };
  XMLHttpRequest.prototype.open=function(method,url){
    this._vcUrl=url;
    return origOpen.apply(this,arguments);
  };
  var origSetReqHeader=XMLHttpRequest.prototype.setRequestHeader;
  XMLHttpRequest.prototype.setRequestHeader=function(name,value){
    if(name.toLowerCase()==='x-super-properties')value=patchSp(value);
    return origSetReqHeader.apply(this,arguments);
  };
})();
"#;

const TITLEBAR_JS: &str = r#"
(function(){
  if(window.__vcTbInit)return;window.__vcTbInit=1;
  var s=document.createElement('style');s.id='vc-tb-css';
  s.textContent='.vc-win-btn{width:46px!important;height:32px!important;border:none!important;background:none!important;color:var(--text-normal,#dbdee1)!important;display:flex!important;align-items:center!important;justify-content:center!important;cursor:pointer!important;transition:background .15s!important;}.vc-win-btn:hover{background:var(--background-modifier-hover,#35373c)!important;}.vc-win-btn.vc-close:hover{background:#ed4245!important;color:#fff!important;}[class*="winButton"]{display:none!important;}#vc-tb-btns{display:flex!important;height:100%!important;align-items:center!important;}[data-list-item-id="guildsnav___app-download-button"]{display:none!important;}';
  document.head.appendChild(s);
  function iv(cmd,cb){
    try{
      var t=window.__vcTauri;
      if(t&&t.invoke){t.invoke(cmd).then(function(r){if(cb)cb(r);}).catch(function(e){});return true;}
    }catch(e){}
    return false;
  }
  function inject(){
    var trailing=document.querySelector('[class*="trailing_"][class*="c3"]');
    if(!trailing||document.getElementById('vc-tb-btns'))return;
    var w=document.createElement('div');w.id='vc-tb-btns';
    w.innerHTML='';
    trailing.appendChild(w);
    function mkBtn(id,cls,svg){
      var b=document.createElement('button');b.className='vc-win-btn'+(cls?' '+cls:'');b.id=id;b.innerHTML=svg;return b;
    }
    w.appendChild(mkBtn('vc-min','',  '<svg width="10" height="1"><rect width="10" height="1" fill="currentColor"/></svg>'));
    w.appendChild(mkBtn('vc-max','',  '<svg width="10" height="10"><rect x=".5" y=".5" width="9" height="9" fill="none" stroke="currentColor" stroke-width="1"/></svg>'));
    w.appendChild(mkBtn('vc-close','vc-close','<svg width="10" height="10"><line x1="1" y1="1" x2="9" y2="9" stroke="currentColor" stroke-width="1.2"/><line x1="9" y1="1" x2="1" y2="9" stroke="currentColor" stroke-width="1.2"/></svg>'));
    console.log('vc-tb-btns injected, child count:', w.children.length);
    document.getElementById('vc-min').addEventListener('click',function(e){e.preventDefault();e.stopPropagation();e.stopImmediatePropagation();console.log('vc-min clicked, __TAURI__:',!!window.__TAURI__);iv('vc_minimize');});
    document.getElementById('vc-max').addEventListener('click',function(e){e.preventDefault();e.stopPropagation();e.stopImmediatePropagation();console.log('vc-max clicked');iv('vc_toggle_maximize');});
    document.getElementById('vc-close').addEventListener('click',function(e){e.preventDefault();e.stopPropagation();e.stopImmediatePropagation();console.log('vc-close clicked');iv('vc_hide');});
  }
  inject();
  var ti=0,mo=new MutationObserver(function(){if(ti)return;ti=setTimeout(function(){ti=0;inject();},500);});
  function om(){if(document.body)mo.observe(document.body,{childList:true,subtree:true});else setTimeout(om,100);}
  om();
  document.addEventListener('click',function(e){
    var a=e.target.closest('a[href]');
    if(a&&a.target==='_blank'){
      try{var u=new URL(a.href,location.origin);if(u.hostname!=='discord.com'&&u.hostname.indexOf('.discord.com')<0&&u.hostname.indexOf('.discordapp.com')<0){e.preventDefault();e.stopPropagation();window.__vcTauri.shell.open(u.href);}}catch(ex){}
    }
  },true);
  document.addEventListener('mousedown',function(e){
    if(e.button!==0)return;
    if(e.target.closest('.vc-win-btn,.clickable,[role="button"],a,button,input,select,textarea,[contenteditable]'))return;
    if(e.target.closest('[class*="bar_"][class*="c3"]')){e.preventDefault();iv('vc_start_drag');}
  },true);
})();
"#;

fn main() {
    check_and_update_vencord();
    start_discord_rpc_bridge();

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![vc_minimize, vc_toggle_maximize, vc_hide, vc_start_drag, vc_open_url, vc_reload_vencord, vc_update_vencord])
        .setup(|app| {
            let show = MenuItem::with_id(app, "show", "Show", true, None::<&str>)?;
            let hide = MenuItem::with_id(app, "hide", "Hide", true, None::<&str>)?;
            let quit = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&show, &hide, &quit])?;

            TrayIconBuilder::new()
                .icon(app.default_window_icon().unwrap().clone())
                .menu(&menu)
                .show_menu_on_left_click(false)
                .tooltip("Discord")
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "show" => {
                        if let Some(win) = app.get_webview_window("main") {
                            show_main_window(&win);
                        }
                    }
                    "hide" => {
                        if let Some(win) = app.get_webview_window("main") {
                            win.hide().ok();
                        }
                    }
                    "quit" => app.exit(0),
                    _ => {}
                })
                .on_tray_icon_event(|tray, event| {
                    if let tauri::tray::TrayIconEvent::Click {
                        button: tauri::tray::MouseButton::Left,
                        button_state: tauri::tray::MouseButtonState::Up,
                        ..
                    } = event
                    {
                        let app = tray.app_handle();
                        if let Some(win) = app.get_webview_window("main") {
                            if win.is_visible().unwrap_or(false) {
                                win.hide().ok();
                            } else {
                                show_main_window(&win);
                            }
                        }
                    }
                })
                .build(app)?;

            let vencord_dir = std::path::PathBuf::from("vencord");
            let vencord_js = std::fs::read_to_string(vencord_dir.join("browser.js")).unwrap_or_else(|_| include_str!("../vencord/browser.js").to_string());
            let vencord_css = std::fs::read_to_string(vencord_dir.join("browser.css")).unwrap_or_else(|_| include_str!("../vencord/browser.css").to_string());
            let recent_downloads = Arc::new(Mutex::new(HashMap::<String, Instant>::new()));
            let user_agent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36";

            let _win = WebviewWindowBuilder::new(app, "main", tauri::WebviewUrl::External("https://discord.com/app".parse().unwrap()))
                .title("Discord")
                .decorations(false)
                .inner_size(1280.0, 860.0)
                .min_inner_size(480.0, 400.0)
                .center()
                .user_agent(user_agent)
                .devtools(true)
                .disable_drag_drop_handler()
                .enable_clipboard_access()
                .additional_browser_args("--disable-features=msWebOOUI,msPdfOOUI,msSmartScreenProtection --disable-extensions --disable-component-update --disable-background-networking --no-first-run --disable-default-apps --disable-sync --disable-translate --process-per-site --js-flags=--max-old-space-size=512 --disable-background-timer-throttling --disable-ipc-flooding-protection --disable-renderer-backgrounding --enable-low-res-tiling --num-raster-threads=2 --disable-threaded-animation --disable-backing-store-tiling --use-fake-ui-for-media-stream")
                .initialization_script(&vencord_js)
                .initialization_script("window.__vcTauri={get invoke(){return window.__TAURI_INTERNALS__?.invoke;},get shell(){return window.__TAURI_INTERNALS__?{open:function(u){window.__TAURI_INTERNALS__.invoke('plugin:shell|open',{path:u});}}:null;}};")
                .initialization_script(SPOOF_JS)
                .on_page_load(move |win, payload| {
                    if payload.event() == tauri::webview::PageLoadEvent::Finished {
                        let css_json = serde_json::to_string(&vencord_css).unwrap();
                        win.eval(&format!(
                            "if(!document.getElementById('vencord-css')){{var s=document.createElement('style');s.id='vencord-css';s.textContent={};document.head.appendChild(s);}}",
                            css_json
                        )).ok();
                        win.eval(TITLEBAR_JS).ok();
                    }
                })
                .on_download({
                    let recent_downloads = Arc::clone(&recent_downloads);
                    move |_win, event| {
                    match event {
                        tauri::webview::DownloadEvent::Requested { url, destination } => {
                            let url_str = url.to_string();
                            let now = Instant::now();
                            if let Ok(mut recent) = recent_downloads.lock() {
                                recent.retain(|_, seen| now.duration_since(*seen) < Duration::from_secs(5));
                                if recent
                                    .get(&url_str)
                                    .is_some_and(|seen| now.duration_since(*seen) < Duration::from_millis(1500))
                                {
                                    return false;
                                }
                                recent.insert(url_str.clone(), now);
                            }

                            let filename = filename_from_destination(destination).unwrap_or_else(|| best_download_filename(&url_str));
                            if let Some(path) = rfd::FileDialog::new().set_file_name(filename).save_file() {
                                *destination = path;
                                true
                            } else {
                                false
                            }
                        }
                        tauri::webview::DownloadEvent::Finished { .. } => true,
                        _ => true,
                    }
                    }
                })
                .build()?;

            #[cfg(target_os = "windows")]
            _win.with_webview(|wv| {
                use webview2_com::{
                    Microsoft::Web::WebView2::Win32::{
                        COREWEBVIEW2_PERMISSION_KIND_CAMERA,
                        COREWEBVIEW2_PERMISSION_KIND_MICROPHONE,
                        COREWEBVIEW2_PERMISSION_STATE_ALLOW,
                        ICoreWebView2Controller4,
                    },
                    PermissionRequestedEventHandler,
                };
                use windows::core::Interface;
                unsafe {
                    if let Ok(webview) = wv.controller().CoreWebView2() {
                        let mut permission_token = Default::default();
                        let _ = webview.add_PermissionRequested(
                            &PermissionRequestedEventHandler::create(Box::new(|_, args| {
                                let Some(args) = args else { return Ok(()); };
                                let mut kind = Default::default();
                                args.PermissionKind(&mut kind)?;
                                if matches!(kind, COREWEBVIEW2_PERMISSION_KIND_CAMERA | COREWEBVIEW2_PERMISSION_KIND_MICROPHONE) {
                                    args.SetState(COREWEBVIEW2_PERMISSION_STATE_ALLOW)?;
                                }
                                Ok(())
                            })),
                            &mut permission_token,
                        );
                    }

                    if let Ok(ctrl4) = wv.controller().cast::<ICoreWebView2Controller4>() {
                        let _ = ctrl4.SetAllowExternalDrop(true).ok();
                    }
                }
            }).ok();

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
