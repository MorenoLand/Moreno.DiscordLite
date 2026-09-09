package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const (
	vencordJSURL  = "https://github.com/Vendicated/Vencord/releases/download/devbuild/browser.js"
	vencordCSSURL = "https://github.com/Vendicated/Vencord/releases/download/devbuild/browser.css"
	userAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed discord.ico
var icon []byte

//go:embed vencord/browser.js
var embeddedVencordJS string

//go:embed vencord/browser.css
var embeddedVencordCSS string

type bridgeRequest struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Args   json.RawMessage `json:"args"`
}

type bridgeDownloadArgs struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
}

type bridgeResizeArgs struct {
	Edge string `json:"edge"`
}

type bridgeURLArgs struct {
	URL string `json:"url"`
}

type discordApp struct {
	app            *application.App
	window         *application.WebviewWindow
	recentMu       sync.Mutex
	recentDownload map[string]time.Time
}

func showMainWindow(window application.Window) {
	window.UnMinimise()
	window.Show()
	window.SetAlwaysOnTop(true)
	window.Focus()
	window.SetAlwaysOnTop(false)
}

func (d *discordApp) handleMessage(window application.Window, message string, _ *application.OriginInfo) {
	var request bridgeRequest
	if json.Unmarshal([]byte(message), &request) != nil || request.Method == "" {
		return
	}
	switch request.Method {
	case "vc_minimize":
		window.Minimise()
		d.respond(window, request.ID, nil, nil)
	case "vc_toggle_maximize":
		if window.IsMaximised() {
			window.UnMaximise()
		} else {
			window.Maximise()
		}
		d.respond(window, request.ID, nil, nil)
	case "vc_hide":
		window.Hide()
		d.respond(window, request.ID, nil, nil)
	case "vc_open_devtools":
		window.OpenDevTools()
		d.respond(window, request.ID, nil, nil)
	case "vc_start_drag":
		window.HandleMessage("wails:drag")
		d.respond(window, request.ID, nil, nil)
	case "vc_start_resize":
		var args bridgeResizeArgs
		if err := json.Unmarshal(request.Args, &args); err != nil || !validResizeEdge(args.Edge) {
			d.respond(window, request.ID, nil, fmt.Errorf("invalid resize edge"))
			return
		}
		window.HandleMessage("wails:resize:" + args.Edge)
		d.respond(window, request.ID, nil, nil)
	case "vc_open_url":
		var args bridgeURLArgs
		if err := json.Unmarshal(request.Args, &args); err != nil || args.URL == "" {
			d.respond(window, request.ID, nil, fmt.Errorf("invalid URL"))
			return
		}
		d.respond(window, request.ID, nil, openURL(args.URL))
	case "vc_reload_vencord":
		css, _ := readVencordFile("browser.css", embeddedVencordCSS)
		encoded, _ := json.Marshal(css)
		window.ExecJS(fmt.Sprintf("var s=document.getElementById('vencord-css');if(s)s.textContent=%s;else{s=document.createElement('style');s.id='vencord-css';s.textContent=%s;document.head.appendChild(s)}location.reload();", encoded, encoded))
		d.respond(window, request.ID, nil, nil)
	case "vc_update_vencord":
		go func() {
			err := updateVencord()
			if err == nil {
				d.respond(window, request.ID, "Vencord updated. Restart to apply.", nil)
			} else {
				d.respond(window, request.ID, nil, err)
			}
		}()
	case "vc_download":
		var args bridgeDownloadArgs
		if err := json.Unmarshal(request.Args, &args); err != nil || args.URL == "" {
			d.respond(window, request.ID, nil, fmt.Errorf("invalid download URL"))
			return
		}
		if !d.allowDownload(args.URL) {
			d.respond(window, request.ID, nil, nil)
			return
		}
		go func() {
			err := d.download(window, args.URL, args.Filename)
			d.respond(window, request.ID, nil, err)
		}()
	}
}

func validResizeEdge(edge string) bool {
	switch edge {
	case "nw-resize", "n-resize", "ne-resize", "w-resize", "e-resize", "sw-resize", "s-resize", "se-resize":
		return true
	default:
		return false
	}
}

func (d *discordApp) respond(window application.Window, id string, value any, err error) {
	if id == "" {
		return
	}
	response := map[string]any{"id": id, "ok": err == nil}
	if err != nil {
		response["error"] = err.Error()
	} else {
		response["result"] = value
	}
	encoded, _ := json.Marshal(response)
	window.ExecJS("window.__vcWailsResponse(" + string(encoded) + ")")
}

func (d *discordApp) allowDownload(downloadURL string) bool {
	now := time.Now()
	d.recentMu.Lock()
	defer d.recentMu.Unlock()
	for key, seen := range d.recentDownload {
		if now.Sub(seen) >= 5*time.Second {
			delete(d.recentDownload, key)
		}
	}
	if seen, ok := d.recentDownload[downloadURL]; ok && now.Sub(seen) < 1500*time.Millisecond {
		return false
	}
	d.recentDownload[downloadURL] = now
	return true
}

func (d *discordApp) download(window application.Window, downloadURL, suggestedName string) error {
	filename := sanitizeFilename(suggestedName)
	if suggestedName == "" {
		filename = bestDownloadFilename(downloadURL)
	}
	pathName, err := d.app.Dialog.SaveFile().SetFilename(filename).AttachToWindow(window).PromptForSingleSelection()
	if err != nil {
		return err
	}
	if pathName == "" {
		return nil
	}
	client := &http.Client{Timeout: 15 * time.Minute}
	response, err := client.Get(downloadURL)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", response.Status)
	}
	file, err := os.Create(pathName)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func openURL(rawURL string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("cmd", "/C", "start", "", rawURL)
	case "darwin":
		command = exec.Command("open", rawURL)
	default:
		command = exec.Command("xdg-open", rawURL)
	}
	hideCommandWindow(command)
	return command.Start()
}

func readVencordFile(name, fallback string) (string, error) {
	data, err := os.ReadFile(path.Join("vencord", name))
	if err != nil {
		return fallback, err
	}
	return string(data), nil
}

func updateVencord() error {
	client := &http.Client{Timeout: 15 * time.Second}
	js, err := fetchBytes(client, vencordJSURL)
	if err != nil {
		return fmt.Errorf("browser.js: %w", err)
	}
	css, err := fetchBytes(client, vencordCSSURL)
	if err != nil {
		return fmt.Errorf("browser.css: %w", err)
	}
	if err := os.MkdirAll("vencord", 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path.Join("vencord", "browser.js"), js, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(path.Join("vencord", "browser.css"), css, 0644); err != nil {
		return err
	}
	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	_ = os.WriteFile(path.Join("vencord", ".last_update"), []byte(timestamp), 0644)
	return nil
}

func fetchBytes(client *http.Client, rawURL string) ([]byte, error) {
	response, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %s", response.Status)
	}
	return io.ReadAll(response.Body)
}

func checkAndUpdateVencord() {
	data, err := os.ReadFile(path.Join("vencord", ".last_update"))
	if err == nil {
		if timestamp, parseErr := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); parseErr == nil {
			if time.Since(time.Unix(timestamp, 0)) <= 24*time.Hour {
				return
			}
		}
	}
	_ = updateVencord()
}

func sanitizeFilename(filename string) string {
	filename = strings.Trim(strings.TrimSpace(filename), "\"")
	var builder strings.Builder
	for _, char := range filename {
		if char < 32 || strings.ContainsRune("<>:\"/\\|?*", char) {
			builder.WriteByte('_')
		} else {
			builder.WriteRune(char)
		}
	}
	filename = strings.Trim(builder.String(), " .")
	if filename == "" {
		return "download"
	}
	return filename
}

func filenameFromContentDisposition(value string) string {
	for _, part := range strings.Split(value, ";") {
		part = strings.TrimSpace(part)
		lower := strings.ToLower(part)
		if strings.HasPrefix(lower, "filename*=") {
			filename := strings.Trim(strings.TrimSpace(part[len("filename*="):]), "\"")
			if strings.HasPrefix(strings.ToLower(filename), "utf-8''") {
				filename = filename[7:]
			}
			if decoded, err := url.PathUnescape(filename); err == nil {
				filename = decoded
			}
			return sanitizeFilename(filename)
		}
		if strings.HasPrefix(lower, "filename=") {
			filename := strings.Trim(strings.TrimSpace(part[len("filename="):]), "\"")
			if decoded, err := url.PathUnescape(filename); err == nil {
				filename = decoded
			}
			return sanitizeFilename(filename)
		}
	}
	return ""
}

func extensionFromContentType(value string) string {
	switch strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0])) {
	case "image/avif":
		return "avif"
	case "image/bmp":
		return "bmp"
	case "image/gif":
		return "gif"
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/svg+xml":
		return "svg"
	case "image/webp":
		return "webp"
	case "video/mp4":
		return "mp4"
	case "video/quicktime":
		return "mov"
	case "video/webm":
		return "webm"
	case "audio/mpeg":
		return "mp3"
	case "audio/ogg":
		return "ogg"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return "wav"
	case "application/pdf":
		return "pdf"
	case "application/zip":
		return "zip"
	case "application/x-7z-compressed":
		return "7z"
	case "text/plain":
		return "txt"
	default:
		return ""
	}
}

func filenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "download"
	}
	filename := path.Base(parsed.Path)
	if decoded, err := url.PathUnescape(filename); err == nil {
		filename = decoded
	}
	return sanitizeFilename(filename)
}

func bestDownloadFilename(rawURL string) string {
	fallback := filenameFromURL(rawURL)
	client := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	request, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return fallback
	}
	response, err := client.Do(request)
	if err != nil {
		return fallback
	}
	defer response.Body.Close()
	if filename := filenameFromContentDisposition(response.Header.Get("Content-Disposition")); filename != "" {
		return filename
	}
	if path.Ext(fallback) == "" {
		if extension := extensionFromContentType(response.Header.Get("Content-Type")); extension != "" {
			return fallback + "." + extension
		}
	}
	return fallback
}

const imageSaveJS = `(function(){
if(window.__vcImageSaveInit)return;window.__vcImageSaveInit=1;
function originalImageURL(raw){var u;try{u=new URL(raw,location.href);}catch(ex){return null;}var host=u.hostname.toLowerCase();if(u.protocol!=='https:'||(host!=='cdn.discordapp.com'&&host!=='media.discordapp.net')||!/^\/(?:attachments|ephemeral-attachments|avatars|icons|banners|emojis|app-assets)\//i.test(u.pathname))return null;['format','quality','width','height','size'].forEach(function(key){u.searchParams.delete(key);});if(host==='media.discordapp.net'&&/^\/(?:attachments|ephemeral-attachments)\//i.test(u.pathname))u.hostname='cdn.discordapp.com';return u.href===raw?null:u.href;}
document.addEventListener('contextmenu',function(e){var img=e.target&&e.target.closest?e.target.closest('img'):null;if(!img)return;var raw=img.currentSrc||img.src;if(!raw)return;var original=originalImageURL(raw);if(!original)return;img.removeAttribute('srcset');img.src=original;},true);
})();`

func (d *discordApp) injectPage(window *application.WebviewWindow) {
	vencordJS, _ := readVencordFile("browser.js", embeddedVencordJS)
	vencordCSS, _ := readVencordFile("browser.css", embeddedVencordCSS)
	css := vencordCSSInjectionScript(vencordCSS)
	for _, script := range []string{wailsBridgeJS, resizeJS, vencordJS, spoofJS, titlebarJS, downloadJS, imageSaveJS, css} {
		window.ExecJS(script)
	}
}

func vencordCSSInjectionScript(css string) string {
	encodedCSS, _ := json.Marshal(css)
	return fmt.Sprintf(`(function(){
var css=%s;
function apply(){
if(!document.head){setTimeout(apply,50);return;}
var s=document.getElementById('vencord-css');
if(!s){s=document.createElement('style');s.id='vencord-css';document.head.appendChild(s);}
if(s.textContent!==css)s.textContent=css;
}
apply();
})();`, encodedCSS)
}

func main() {
	checkAndUpdateVencord()
	startDiscordRPCBridge()
	discord := &discordApp{recentDownload: make(map[string]time.Time)}
	app := application.New(application.Options{
		Name:        "Discord",
		Description: "Unofficial Discord desktop wrapper",
		Icon:        icon,
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
			AdditionalBrowserArgs: []string{
				"--user-agent=" + userAgent,
				"--disable-features=msWebOOUI,msPdfOOUI,msSmartScreenProtection",
				"--disable-extensions",
				"--disable-component-update",
				"--disable-background-networking",
				"--no-first-run",
				"--disable-default-apps",
				"--disable-sync",
				"--disable-translate",
				"--process-per-site",
				"--js-flags=--max-old-space-size=512",
				"--enable-low-res-tiling",
				"--num-raster-threads=2",
				"--use-fake-ui-for-media-stream",
			},
		},
		Linux: application.LinuxOptions{DisableQuitOnLastWindowClosed: true},
		RawMessageHandler: func(window application.Window, message string, originInfo *application.OriginInfo) {
			discord.handleMessage(window, message, originInfo)
		},
	})
	discord.app = app
	initialVencordJS, _ := readVencordFile("browser.js", embeddedVencordJS)
	initialVencordCSS, _ := readVencordFile("browser.css", embeddedVencordCSS)
	initializationJS := strings.Join([]string{wailsBridgeJS, resizeJS, titlebarJS, initialVencordJS, spoofJS, downloadJS, imageSaveJS, vencordCSSInjectionScript(initialVencordCSS)}, "\n")
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:          "main",
		Title:         "Discord",
		HTML:          "<!doctype html><html><head><meta charset=\"utf-8\"></head><body><script>location.replace(\"https://discord.com/app\")</script></body></html>",
		JS:            initializationJS,
		Width:         1280,
		Height:        860,
		MinWidth:      480,
		MinHeight:     400,
		Frameless:     true,
		DisableResize: false,
		Permissions: map[application.PermissionType]application.Permission{
			application.PermissionCamera:        application.PermissionAllow,
			application.PermissionMicrophone:    application.PermissionAllow,
			application.PermissionClipboardRead: application.PermissionAllow,
		},
		EnableFileDrop:  false,
		DevToolsEnabled: true,
		Windows:         application.WindowsWindow{DisableIcon: true, NonClientRegionSupport: true},
		StartState:      application.WindowStateNormal,
		InitialPosition: application.WindowCentered,
	})
	discord.window = window
	window.OnWindowEvent(events.Windows.WebViewNavigationCompleted, func(_ *application.WindowEvent) {
		discord.injectPage(window)
	})

	tray := app.SystemTray.New()
	tray.SetIcon(icon)
	tray.SetTooltip("Discord")
	menu := application.NewMenu()
	menu.Add("Show").OnClick(func(*application.Context) { showMainWindow(window) })
	menu.Add("Hide").OnClick(func(*application.Context) { window.Hide() })
	menu.Add("Quit").OnClick(func(*application.Context) { app.Quit() })
	tray.SetMenu(menu)
	tray.OnClick(func() {
		if window.IsVisible() {
			window.Hide()
		} else {
			showMainWindow(window)
		}
	})
	tray.OnRightClick(func() { tray.OpenMenu() })
	tray.Run()
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

const wailsBridgeJS = `(function(){
if(window.__vcWails)return;
var pending=Object.create(null),nextId=0;
window.__vcWailsResponse=function(message){try{var response=typeof message==='string'?JSON.parse(message):message;var item=pending[response.id];if(!item)return;delete pending[response.id];if(response.ok)item.resolve(response.result);else item.reject(new Error(response.error||'Wails request failed'));}catch(e){}};
function invoke(method,args){return new Promise(function(resolve,reject){var id=String(++nextId);pending[id]={resolve:resolve,reject:reject};try{window._wails.invoke(JSON.stringify({id:id,method:method,args:args||{}}));}catch(e){delete pending[id];reject(e);}});}
window.__vcWails={invoke:invoke,shell:{open:function(url){return invoke('vc_open_url',{url:url});}}};
document.addEventListener('keydown',function(e){if(e.key!=='F12'&&e.code!=='F12')return;e.preventDefault();e.stopPropagation();e.stopImmediatePropagation();invoke('vc_open_devtools').catch(function(){});},true);
})();`

const resizeJS = `(function(){
if(window.__vcResizeInit)return;window.__vcResizeInit=1;
var active='',pending='',hover='',savedCursor='',startX=0,startY=0;
function edgeAt(e){var x=e.clientX,y=e.clientY,w=window.outerWidth,h=window.outerHeight,d=8,l=x<=d,r=x>=w-d,t=y<=d,b=y>=h-d;if(t&&l)return'nw-resize';if(t&&r)return'ne-resize';if(b&&l)return'sw-resize';if(b&&r)return'se-resize';if(t)return'n-resize';if(b)return's-resize';if(l)return'w-resize';if(r)return'e-resize';return'';}
function setHover(edge){if(edge===hover)return;if(!hover&&edge)savedCursor=document.documentElement.style.cursor;hover=edge;document.documentElement.style.cursor=edge||savedCursor;if(!edge)savedCursor='';}
function begin(edge){active=edge;pending='';window.__vcWails.invoke('vc_start_resize',{edge:edge}).catch(clear);}
function clear(){active='';pending='';setHover('');}
document.addEventListener('mousemove',function(e){if(active)return;var edge=edgeAt(e);if(pending){if(!(e.buttons&1)||edge!==pending){clear();return;}if(Math.abs(e.clientX-startX)+Math.abs(e.clientY-startY)>=2){e.preventDefault();e.stopPropagation();e.stopImmediatePropagation();begin(pending);return;}}setHover(edge);},true);
document.addEventListener('mousedown',function(e){if(e.button!==0||active||pending)return;var edge=edgeAt(e);if(!edge)return;pending=edge;startX=e.clientX;startY=e.clientY;setHover(edge);e.preventDefault();e.stopPropagation();e.stopImmediatePropagation();},true);
document.addEventListener('mouseup',clear,true);window.addEventListener('blur',clear,true);
})();`

const spoofJS = `(function(){
function patchSp(sp){try{var o=JSON.parse(atob(sp));if(o.browser==='chrome'){o.browser='discord';o.browser_version='';}return btoa(JSON.stringify(o));}catch(e){return sp;}}
var origFetch=window.fetch;
window.fetch=function(input,init){if(init&&init.headers){var h=init.headers;if(typeof h==='object'&&!(h instanceof Headers)){if(h['X-Super-Properties'])h['X-Super-Properties']=patchSp(h['X-Super-Properties']);if(h['x-super-properties'])h['x-super-properties']=patchSp(h['x-super-properties']);}else if(h instanceof Headers&&h.has('X-Super-Properties'))h.set('X-Super-Properties',patchSp(h.get('X-Super-Properties')));}return origFetch.apply(this,arguments);};
var origOpen=XMLHttpRequest.prototype.open;
var origSend=XMLHttpRequest.prototype.send;
XMLHttpRequest.prototype.send=function(body){if(this._vcSpHeader)this.setRequestHeader('X-Super-Properties',this._vcSpHeader);return origSend.apply(this,arguments);};
XMLHttpRequest.prototype.open=function(method,url){this._vcUrl=url;return origOpen.apply(this,arguments);};
var origSetReqHeader=XMLHttpRequest.prototype.setRequestHeader;
XMLHttpRequest.prototype.setRequestHeader=function(name,value){if(name.toLowerCase()==='x-super-properties')value=patchSp(value);return origSetReqHeader.apply(this,arguments);};
})();`

const titlebarJS = `(function(){
function start(){
if(window.__vcTbInit)return;if(!document.documentElement||!document.head){setTimeout(start,50);return;}window.__vcTbInit=1;
var s=document.createElement('style');s.id='vc-tb-css';s.textContent='.vc-win-btn{width:46px!important;height:32px!important;border:none!important;background:none!important;color:var(--interactive-normal,var(--text-normal,#dbdee1))!important;display:flex!important;align-items:center!important;justify-content:center!important;cursor:pointer!important;transition:background .15s!important;}.vc-win-btn:hover{background:var(--background-modifier-hover,#35373c)!important;}.vc-win-btn.vc-close:hover{background:#ed4245!important;color:#fff!important;}[class*="winButton"]{display:none!important;}#vc-tb-btns{display:flex!important;height:100%!important;align-items:center!important;flex:0 0 auto!important;margin-left:4px!important;position:relative!important;z-index:2!important;}[data-list-item-id="guildsnav___app-download-button"]{display:none!important;}[class*="bar_"][class*="c3"]{--wails-draggable:drag!important;}';document.head.appendChild(s);
function iv(cmd,args){try{var t=window.__vcWails;if(t&&t.invoke){t.invoke(cmd,args).catch(function(){});return true;}}catch(e){}return false;}
function inject(){var trailing=document.querySelector('[data-window-chrome="true"] > [class*="trailing_"]');if(!trailing||document.getElementById('vc-tb-btns'))return;var w=document.createElement('div');w.id='vc-tb-btns';trailing.appendChild(w);function mkBtn(id,cls,svg){var b=document.createElement('button');b.className='vc-win-btn'+(cls?' '+cls:'');b.id=id;b.innerHTML=svg;return b;}w.appendChild(mkBtn('vc-min','', '<svg width="10" height="1"><rect width="10" height="1" fill="currentColor"/></svg>'));w.appendChild(mkBtn('vc-max','', '<svg width="10" height="10"><rect x=".5" y=".5" width="9" height="9" fill="none" stroke="currentColor" stroke-width="1"/></svg>'));w.appendChild(mkBtn('vc-close','vc-close','<svg width="10" height="10"><line x1="1" y1="1" x2="9" y2="9" stroke="currentColor" stroke-width="1.2"/><line x1="9" y1="1" x2="1" y2="9" stroke="currentColor" stroke-width="1.2"/></svg>'));document.getElementById('vc-min').addEventListener('click',function(e){e.preventDefault();e.stopPropagation();e.stopImmediatePropagation();iv('vc_minimize');});document.getElementById('vc-max').addEventListener('click',function(e){e.preventDefault();e.stopPropagation();e.stopImmediatePropagation();iv('vc_toggle_maximize');});document.getElementById('vc-close').addEventListener('click',function(e){e.preventDefault();e.stopPropagation();e.stopImmediatePropagation();iv('vc_hide');});}
inject();var ti=0,mo=new MutationObserver(function(){if(ti)return;ti=setTimeout(function(){ti=0;inject();},250);});function om(){var root=document.documentElement;if(root)mo.observe(root,{childList:true,subtree:true});else setTimeout(om,100);}om();
document.addEventListener('click',function(e){var a=e.target.closest('a[href]');if(a&&a.target==='_blank'){try{var u=new URL(a.href,location.origin);if(u.hostname!=='discord.com'&&u.hostname.indexOf('.discord.com')<0&&u.hostname.indexOf('.discordapp.com')<0){e.preventDefault();e.stopPropagation();window.__vcWails.shell.open(u.href);}}catch(ex){}}},true);
document.addEventListener('mousedown',function(e){if(e.button!==0)return;if(e.target.closest('.vc-win-btn,.clickable,[role="button"],a,button,input,select,textarea,[contenteditable]'))return;if(e.target.closest('[class*="bar_"][class*="c3"]')){e.preventDefault();iv('vc_start_drag');}},true);
}
start();
})();`

const downloadJS = `(function(){
if(window.__vcDownloadInit)return;window.__vcDownloadInit=1;
document.addEventListener('click',function(e){var a=e.target.closest('a[href]');if(!a)return;var u;try{u=new URL(a.href,location.origin);}catch(ex){return;}var target=e.target.closest?e.target:null;var isDownload=a.hasAttribute('download')||/download/i.test(a.getAttribute('aria-label')||'')||!!(target&&target.closest('[aria-label="Download"],[data-tooltip-content="Download"]'));if(!isDownload)return;e.preventDefault();e.stopPropagation();window.__vcWails.invoke('vc_download',{url:u.href,filename:a.getAttribute('download')||''}).catch(function(){});},true);
})();`
