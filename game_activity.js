(function(){
if(window.__vcGameActivityInit)return;
window.__vcGameActivityInit=1;
var navId='vc-game-activity-nav',panelId='vc-game-activity-panel',active=false,loading=false,lastRenderedSignature='',pausePollUntil=0;
function invoke(method,args){return window.__vcWails.invoke(method,args||{});}
function addStyles(){
if(document.getElementById('vc-game-activity-css'))return;
var style=document.createElement('style');
style.id='vc-game-activity-css';
style.textContent=
'#vc-game-activity-li{list-style:none!important;}' +
'#vc-game-activity-nav{cursor:pointer!important;}' +
'#vc-game-activity-nav:hover{background-color:var(--background-modifier-hover,rgba(255,255,255,0.07))!important;color:var(--interactive-hover,var(--text-normal,#fff))!important;}' +
'#vc-game-activity-nav.vc-selected,#vc-game-activity-nav[class*="selected_"]{color:var(--interactive-active,var(--text-normal,#fff))!important;background-color:var(--background-modifier-selected,#4e5058)!important;border-radius:var(--radius-sm,4px)!important;}' +
'#vc-game-activity-nav.vc-selected *,#vc-game-activity-nav[class*="selected_"] *{color:var(--interactive-active,var(--text-normal,#fff))!important;}' +
'#vc-game-activity-nav .vc-game-activity-icon{width:20px!important;height:20px!important;flex-shrink:0!important;}' +
'#vc-game-activity-panel{box-sizing:border-box!important;position:fixed!important;z-index:2147483647!important;background:var(--background-primary,#1e1f22)!important;color:var(--text-normal,#dbdee1)!important;font-family:var(--font-primary,Whitney,"Helvetica Neue",Helvetica,Arial,sans-serif)!important;display:flex!important;flex-direction:column!important;overflow:hidden!important;padding:0!important;margin:0!important;border-top-right-radius:8px!important;border-bottom-right-radius:8px!important;}' +
'#vc-game-activity-panel *{box-sizing:border-box!important;}' +
'#vc-game-activity-panel .vc-header{height:56px!important;min-height:56px!important;display:flex!important;align-items:center!important;justify-content:space-between!important;padding:0 40px!important;border-bottom:1px solid var(--background-modifier-accent,rgba(255,255,255,0.08))!important;background:var(--background-primary,#1e1f22)!important;flex-shrink:0!important;border-top-right-radius:8px!important;}' +
'#vc-game-activity-panel .vc-breadcrumb{font-size:15px!important;font-weight:600!important;color:var(--header-primary,#f2f3f5)!important;}' +
'#vc-game-activity-panel .vc-close-btn{display:inline-flex!important;align-items:center!important;justify-content:center!important;width:32px!important;height:32px!important;padding:0!important;border-radius:4px!important;background:transparent!important;color:var(--interactive-normal,#b5bac1)!important;border:none!important;cursor:pointer!important;transition:background-color .15s ease,color .15s ease!important;}' +
'#vc-game-activity-panel .vc-close-btn:hover{background:var(--background-modifier-hover,rgba(255,255,255,0.07))!important;color:var(--interactive-hover,#fff)!important;}' +
'#vc-game-activity-panel .vc-close-btn svg{width:16px!important;height:16px!important;}' +
'#vc-game-activity-panel .vc-body{flex:1!important;overflow-y:scroll!important;overflow-x:hidden!important;padding:32px 40px 60px!important;border-bottom-right-radius:8px!important;scrollbar-width:thin!important;scrollbar-color:var(--scrollbar-auto-thumb,#1a1b1e) transparent!important;}' +
'#vc-game-activity-panel .vc-body::-webkit-scrollbar{width:8px!important;background:transparent!important;}' +
'#vc-game-activity-panel .vc-body::-webkit-scrollbar-track{background:transparent!important;border:none!important;}' +
'#vc-game-activity-panel .vc-body::-webkit-scrollbar-thumb{background-color:var(--scrollbar-auto-thumb,#1a1b1e)!important;min-height:40px!important;border-radius:4px!important;border:2px solid transparent!important;background-clip:padding-box!important;}' +
'#vc-game-activity-panel .vc-body::-webkit-scrollbar-thumb:hover{background-color:var(--scrollbar-auto-thumb-hover,#111214)!important;}' +
'#vc-game-activity-panel .vc-body::-webkit-scrollbar-button{display:none!important;width:0!important;height:0!important;}' +
'#vc-game-activity-panel .vc-body::-webkit-scrollbar-corner{background:transparent!important;}' +
'#vc-game-activity-panel .vc-inner{max-width:740px!important;width:100%!important;margin:0!important;}' +
'#vc-game-activity-panel h1{font-size:20px!important;line-height:24px!important;font-weight:600!important;margin:0 0 8px!important;color:var(--header-primary,#f2f3f5)!important;}' +
'#vc-game-activity-panel h2{font-size:13px!important;font-weight:700!important;line-height:18px!important;text-transform:uppercase!important;letter-spacing:0.02em!important;margin:28px 0 12px!important;color:var(--header-secondary,#b5bac1)!important;}' +
'#vc-game-activity-panel p{font-size:14px!important;line-height:20px!important;color:var(--text-muted,#b5bac1)!important;margin:0 0 16px!important;}' +
'#vc-game-activity-panel .vc-game-actions{display:flex!important;gap:8px!important;align-items:center!important;margin:16px 0 20px!important;}' +
'#vc-game-activity-panel .vc-game-actions input{flex:1!important;min-width:0!important;height:36px!important;padding:0 12px!important;border-radius:4px!important;border:1px solid var(--background-modifier-accent,#4e5058)!important;background:var(--input-background,#1e1f22)!important;color:var(--text-normal,#dbdee1)!important;font-size:14px!important;outline:none!important;}' +
'#vc-game-activity-panel .vc-game-actions input:focus{border-color:var(--brand-experiment,#5865f2)!important;}' +
'#vc-game-activity-panel button{border:0!important;border-radius:4px!important;background:var(--button-secondary-background,#4e5058)!important;color:var(--button-secondary-text,#fff)!important;font-size:14px!important;font-weight:500!important;height:36px!important;padding:0 16px!important;cursor:pointer!important;display:inline-flex!important;align-items:center!important;justify-content:center!important;transition:background-color .15s ease!important;}' +
'#vc-game-activity-panel button:hover{background:var(--button-secondary-background-hover,#686d75)!important;}' +
'#vc-game-activity-panel button.vc-primary{background:var(--button-brand-background,#5865f2)!important;}' +
'#vc-game-activity-panel button.vc-primary:hover{background:var(--button-brand-background-hover,#4752c4)!important;}' +
'#vc-game-activity-panel button.vc-danger{background:transparent!important;color:var(--text-danger,#f23f42)!important;height:auto!important;padding:6px 8px!important;}' +
'#vc-game-activity-panel button.vc-danger:hover{background:var(--background-modifier-hover,rgba(255,255,255,0.07))!important;}' +
'#vc-game-activity-panel .vc-game-toggle{display:flex!important;align-items:center!important;gap:12px!important;font-size:14px!important;color:var(--text-normal,#dbdee1)!important;cursor:pointer!important;padding-bottom:20px!important;border-bottom:1px solid var(--background-modifier-accent,rgba(255,255,255,0.08))!important;}' +
'#vc-game-activity-panel .vc-game-row{display:flex!important;align-items:center!important;gap:16px!important;min-height:56px!important;padding:12px 0!important;border-bottom:1px solid var(--background-modifier-accent,rgba(255,255,255,0.08))!important;}' +
'#vc-game-activity-panel .vc-game-row.vc-disabled .vc-game-name{color:var(--text-muted,#80848e)!important;}' +
'#vc-game-activity-panel .vc-game-row-main{min-width:0!important;flex:1!important;}' +
'#vc-game-activity-panel .vc-game-name{font-size:15px!important;font-weight:600!important;color:var(--header-primary,#f2f3f5)!important;white-space:nowrap!important;overflow:hidden!important;text-overflow:ellipsis!important;}' +
'#vc-game-activity-panel .vc-game-detail{font-size:13px!important;line-height:18px!important;color:var(--text-muted,#b5bac1)!important;white-space:nowrap!important;overflow:hidden!important;text-overflow:ellipsis!important;}' +
'#vc-game-activity-panel .vc-switch{position:relative!important;display:inline-block!important;width:36px!important;height:20px!important;flex-shrink:0!important;cursor:pointer!important;}' +
'#vc-game-activity-panel .vc-switch input{position:absolute!important;top:0!important;left:0!important;width:100%!important;height:100%!important;margin:0!important;opacity:0!important;z-index:2!important;cursor:pointer!important;}' +
'#vc-game-activity-panel .vc-slider{position:absolute!important;pointer-events:none!important;top:0!important;left:0!important;right:0!important;bottom:0!important;background-color:var(--background-modifier-accent,#4e5058)!important;transition:transform .15s ease-in-out,background-color .15s ease-in-out!important;border-radius:10px!important;z-index:1!important;}' +
'#vc-game-activity-panel .vc-slider:before{position:absolute!important;content:""!important;height:14px!important;width:14px!important;left:3px!important;bottom:3px!important;background-color:#fff!important;transition:transform .15s ease-in-out!important;border-radius:50%!important;}' +
'#vc-game-activity-panel .vc-switch input:checked+.vc-slider{background-color:var(--brand-experiment,#5865f2)!important;}' +
'#vc-game-activity-panel .vc-switch input:checked+.vc-slider:before{transform:translateX(16px)!important;}' +
'#vc-game-activity-panel .vc-game-empty{padding:16px 0!important;color:var(--text-muted,#b5bac1)!important;font-size:14px!important;}';
document.head.appendChild(style);
}
function findExact(root,text){var nodes=root.querySelectorAll('*');for(var i=0;i<nodes.length;i++){var node=nodes[i];if(node.children.length===0&&node.textContent.trim()===text)return node;}return null;}
function findSidebarActivityPrivacy(){
var li=document.querySelector('li[data-settings-sidebar-item="activity_privacy_panel"]');
if(li)return li;
var btn=document.querySelector('[data-list-item-id="settings-sidebar___activity_privacy_panel"]');
if(btn)return btn.closest('li')||btn;
var sidebar=document.querySelector('nav[class*="sidebar"],[class*="sidebarRegion"],[class*="sidebar"],ul[class*="sectionList"]');
if(!sidebar)return null;
var label=findExact(sidebar,'Activity Privacy');
if(label){var item=label.closest('[role="link"],[role="button"],[role="tab"],a,button,[class*="item_"]')||label.parentElement;return item.closest('li')||item;}
return null;
}
function findSidebarConnectedApps(){
var li=document.querySelector('li[data-settings-sidebar-item="connected_apps_panel"]');
if(li)return li;
var btn=document.querySelector('[data-list-item-id="settings-sidebar___connected_apps_panel"]');
if(btn)return btn.closest('li')||btn;
var sidebar=document.querySelector('nav[class*="sidebar"],[class*="sidebarRegion"],[class*="sidebar"],ul[class*="sectionList"]');
if(!sidebar)return null;
var label=findExact(sidebar,'Connected Apps');
if(label){var item=label.closest('[role="link"],[role="button"],[role="tab"],a,button,[class*="item_"]')||label.parentElement;return item.closest('li')||item;}
return null;
}
function isSettingsOpen(){
return !!(document.querySelector('[data-settings-sidebar-item],[data-list-item-id^="settings-sidebar"],[class*="standardSidebarView"],[class*="sidebarRegion"],[class*="container_abd9a8"]')||findSidebarActivityPrivacy());
}
function findContentRegion(){
return document.querySelector(
'[class*="content_e9e3ed"], ' +
'[class*="container_abd9a8"] > [class*="content_"], ' +
'aside[class*="sidebar"] + div[class*="content_"], ' +
'[class*="contentRegion"]'
);
}
function getContentBounds(){
var cr=findContentRegion();
if(cr&&cr.isConnected){
var r=cr.getBoundingClientRect();
if(r.height>100&&r.width>100){
return {left:Math.round(r.left),top:Math.round(r.top),width:Math.round(r.width),height:Math.round(r.height)};
}
}
var refLi=findSidebarActivityPrivacy();
var sidebar=(refLi&&refLi.closest('[class*="sidebarRegion"],nav[class*="sidebar"],[class*="sidebar"]'))||document.querySelector('[class*="sidebarRegion"],nav[class*="sidebar"],[class*="sidebar__409aa"]');
if(sidebar&&sidebar.isConnected){
var sb=sidebar.getBoundingClientRect();
if(sb.width>50){
return {left:Math.round(sb.right),top:Math.round(sb.top),width:Math.max(300,Math.round(window.innerWidth-sb.right)),height:Math.max(300,Math.round(window.innerHeight-sb.top))};
}
}
return {left:240,top:0,width:Math.max(300,window.innerWidth-240),height:window.innerHeight};
}
function position(panel){
if(!panel)return;
var b=getContentBounds();
panel.style.setProperty('position','fixed','important');
panel.style.setProperty('z-index','2147483647','important');
panel.style.setProperty('left',b.left+'px','important');
panel.style.setProperty('top',b.top+'px','important');
panel.style.setProperty('width',b.width+'px','important');
panel.style.setProperty('height',b.height+'px','important');
panel.style.setProperty('right','auto','important');
panel.style.setProperty('bottom','auto','important');
var cr=findContentRegion();
if(cr&&cr.isConnected){
try{
var st=window.getComputedStyle(cr);
if(st.borderTopRightRadius&&st.borderTopRightRadius!=='0px'){
panel.style.setProperty('border-top-right-radius',st.borderTopRightRadius,'important');
panel.style.setProperty('border-bottom-right-radius',st.borderBottomRightRadius,'important');
}
}catch(e){}
}
}
function findDiscordSettingsCloseButton(){
var buttons=document.querySelectorAll(
'[class*="contentHeader"] button[aria-label="Close"], ' +
'button[data-mana-component="button"][aria-label="Close"], ' +
'[class*="container_abd9a8"] button[aria-label="Close"], ' +
'aside + div button[aria-label="Close"], ' +
'[aria-label="Close"][class*="button_"], ' +
'[class*="standardSidebarView"] [class*="toolsContainer"] [class*="closeButton"], ' +
'[class*="toolsContainer"] [role="button"], ' +
'button[aria-label="Close"]'
);
var panel=document.getElementById(panelId);
for(var i=0;i<buttons.length;i++){
var b=buttons[i];
if(panel&&panel.contains(b))continue;
if(b.isConnected&&b.offsetParent!==null)return b;
}
for(var j=0;j<buttons.length;j++){
var b2=buttons[j];
if(panel&&panel.contains(b2))continue;
if(b2.isConnected)return b2;
}
return null;
}
function closeSettings(){
var discordClose=findDiscordSettingsCloseButton();
if(discordClose&&discordClose.isConnected){
discordClose.click();
dispose();
return;
}
var d=getFluxDispatcher();
if(d){
try{d.dispatch({type:'USER_SETTINGS_MODAL_CLOSE'});}catch(e){}
}
dispose();
}
function normId(p){return (p||'').trim().replace(/^["']+|["']+$/g,'').replace(/[\\\/]+/g,'\\').toLowerCase();}
function isGameDisabled(disabledMap,p){
if(!disabledMap||!p)return false;
var norm=normId(p);
return disabledMap[norm]===true||disabledMap[(p||'').toLowerCase()]===true;
}
function text(value){return document.createTextNode(value||'');}
function button(label,action,extra){
var node=document.createElement('button');
node.type='button';
node.textContent=label;
node.dataset.vcAction=action;
if(extra)node.className=extra;
return node;
}
function row(name,detail,path,isEnabled,remove){
var r=document.createElement('div');
r.className='vc-game-row'+(isEnabled?'':' vc-disabled');
var main=document.createElement('div');
main.className='vc-game-row-main';
var title=document.createElement('div');
title.className='vc-game-name';
title.textContent=name;
var sub=document.createElement('div');
sub.className='vc-game-detail';
sub.textContent=detail||path;
sub.title=path;
main.append(title,sub);
r.append(main);
var switchSpan=document.createElement('span');
switchSpan.className='vc-switch';
switchSpan.title=isEnabled?'Disable activity detection for this game':'Enable activity detection for this game';
var toggle=document.createElement('input');
toggle.type='checkbox';
toggle.checked=!!isEnabled;
toggle.dataset.vcGameToggle='1';
toggle.dataset.vcPath=path;
var slider=document.createElement('span');
slider.className='vc-slider';
switchSpan.append(toggle,slider);
r.append(switchSpan);
if(remove){
var action=button('Remove','remove','vc-danger');
action.dataset.vcPath=path;
r.append(action);
}
return r;
}
function getStateSignature(data){
if(!data)return '';
var running=(data.runningGames||[]).map(function(g){return (g.pid||'')+'|'+normId(g.path)+'|'+(g.windowTitle||'')+'|'+(g.name||'');}).join(';');
var seen=(data.gamesSeen||[]).map(function(g){return normId(g.path)+'|'+(g.name||'')+'|'+(g.source||'');}).join(';');
var disabled=Object.keys(data.disabledGames||{}).filter(function(k){return data.disabledGames[k]===true;}).sort().join(';');
var overrides=Object.keys(data.overrides||{}).sort().map(function(k){return k+':'+data.overrides[k];}).join(';');
return (data.detectionEnabled?'1':'0')+'##'+running+'##'+seen+'##'+disabled+'##'+overrides;
}
function render(data){
var panel=document.getElementById(panelId);
if(!panel||!data)return;
var newSig=getStateSignature(data);
if(newSig&&newSig===lastRenderedSignature&&panel.children.length>0)return;
var prevScroll=0,prevInput=panel.querySelector('[data-vc-game-path]'),prevText=prevInput?prevInput.value:'';
var bodyContainer=panel.querySelector('.vc-body');
if(bodyContainer)prevScroll=bodyContainer.scrollTop;
panel.innerHTML='';

var header=document.createElement('div');
header.className='vc-header';
var breadcrumb=document.createElement('div');
breadcrumb.className='vc-breadcrumb';
breadcrumb.textContent='Registered Games';
var closeBtn=document.createElement('button');
closeBtn.type='button';
closeBtn.className='vc-close-btn';
closeBtn.title='Close (ESC)';
closeBtn.setAttribute('aria-label','Close');
closeBtn.innerHTML='<svg width="16" height="16" viewBox="0 0 24 24"><path fill="currentColor" d="M19.3 20.7a1 1 0 0 0 1.4-1.4L13.42 12l7.3-7.3a1 1 0 0 0-1.42-1.4L12 10.58l-7.3-7.3a1 1 0 0 0-1.4 1.42L10.58 12l-7.3 7.3a1 1 0 1 0 1.42 1.4L12 13.42l7.3 7.3Z"/></svg>';
closeBtn.addEventListener('click',function(e){
e.preventDefault();
e.stopPropagation();
closeSettings();
});
header.append(breadcrumb,closeBtn);
panel.append(header);

var body=document.createElement('div');
body.className='vc-body scroller__6131a auto_d125d2 scrollerBase_d125d2';
body.setAttribute('dir','ltr');
var inner=document.createElement('div');
inner.className='vc-inner';

var heading=document.createElement('h1');
heading.textContent='Registered Games';
var intro=document.createElement('p');
intro.textContent='DiscordLite watches visible Windows applications and can show them as your activity. Add an executable manually when detection does not find it.';

var actions=document.createElement('div');
actions.className='vc-game-actions';
var pathInput=document.createElement('input');
pathInput.type='text';
pathInput.placeholder='Executable path (optional)';
pathInput.dataset.vcGamePath='1';
if(prevText)pathInput.value=prevText;
actions.append(pathInput,button('Browse','add','vc-primary'),button('Add','add-path'),button('Refresh','refresh'));

var toggleLabel=document.createElement('label');
toggleLabel.className='vc-game-toggle';
var switchSpan=document.createElement('span');
switchSpan.className='vc-switch';
var toggle=document.createElement('input');
toggle.type='checkbox';
toggle.checked=!!data.detectionEnabled;
toggle.dataset.vcDetectionToggle='1';
var slider=document.createElement('span');
slider.className='vc-slider';
switchSpan.append(toggle,slider);
toggleLabel.append(switchSpan,text('Enable game detection'));

inner.append(heading,intro,actions,toggleLabel);

var currentHeading=document.createElement('h2');
currentHeading.textContent='Running now';
inner.append(currentHeading);
var running=data.runningGames||[];
if(!running.length){
var empty=document.createElement('div');
empty.className='vc-game-empty';
empty.textContent=data.detectionEnabled?'No visible games detected.':'Game detection is disabled.';
inner.append(empty);
}else{
running.forEach(function(game){
var isEnabled=!isGameDisabled(data.disabledGames,game.path);
inner.append(row(resolveDisplayName(game.name,game.path,data.overrides),game.windowTitle,game.path,isEnabled,false));
});
}

var seenHeading=document.createElement('h2');
seenHeading.textContent='Games seen';
inner.append(seenHeading);
var seen=data.gamesSeen||[];
if(!seen.length){
var emptySeen=document.createElement('div');
emptySeen.className='vc-game-empty';
emptySeen.textContent='No games have been registered yet.';
inner.append(emptySeen);
}else{
seen.forEach(function(game){
var isEnabled=!isGameDisabled(data.disabledGames,game.path);
inner.append(row(resolveDisplayName(game.name,game.path,data.overrides),game.source==='manual'?'Added manually':'Detected',game.path,isEnabled,true));
});
}

body.append(inner);
panel.append(body);
if(prevScroll)body.scrollTop=prevScroll;
lastRenderedSignature=newSig;
lastAppliedData=data;
}
function getFluxDispatcher(){
try{
if(window.Vencord&&window.Vencord.Webpack){
if(window.Vencord.Webpack.Common&&window.Vencord.Webpack.Common.FluxDispatcher)return window.Vencord.Webpack.Common.FluxDispatcher;
if(typeof window.Vencord.Webpack.findByProps==='function'){
var d=window.Vencord.Webpack.findByProps('dispatch','subscribe');
if(d&&typeof d.dispatch==='function')return d;
}
}
}catch(e){}
return null;
}
var KNOWN_GAMES={
'wow.exe':{id:'356875762940379136',name:'World of Warcraft'},
'wowclassic.exe':{id:'356875762940379136',name:'World of Warcraft'},
'wowt.exe':{id:'356875762940379136',name:'World of Warcraft'},
'wowb.exe':{id:'356875762940379136',name:'World of Warcraft'},
'league of legends.exe':{id:'401518687463948290',name:'League of Legends'},
'valorant.exe':{id:'700144211132645406',name:'VALORANT'},
'overwatch.exe':{id:'356867200780468224',name:'Overwatch'},
'csgo.exe':{id:'738864303494791248',name:'Counter-Strike 2'},
'cs2.exe':{id:'738864303494791248',name:'Counter-Strike 2'},
'dota2.exe':{id:'738864293411684352',name:'Dota 2'},
'gta5.exe':{id:'436993026818867200',name:'Grand Theft Auto V'},
'minecraft.exe':{id:'356875127150903296',name:'Minecraft'},
'rocketleague.exe':{id:'356877028164632576',name:'Rocket League'},
'fortniteclient-win64-shipping.exe':{id:'432980957394370572',name:'Fortnite'},
'genshinimpact.exe':{id:'762434991303950386',name:'Genshin Impact'},
'starrail.exe':{id:'1100344445853245480',name:'Honkai: Star Rail'},
'ffxiv_dx11.exe':{id:'468936993781252096',name:'FINAL FANTASY XIV'},
'r5apex.exe':{id:'542385150820417537',name:'Apex Legends'}
};
function resolveDisplayName(name,path,overrides){
var id=path?path.toLowerCase():'';
var raw=(overrides&&overrides[id])||name||'';
if(raw&&!/[\\\/]/.test(raw))return raw;
var exe=(path||raw).replace(/^.*[\\\/]/,'').toLowerCase();
if(exe==='javaw.exe'||exe==='java.exe'){
if(raw&&/minecraft/i.test(raw))return 'Minecraft';
return raw&&!/[\\\/]/.test(raw)?raw:exe.replace(/\.[^.]+$/,'');
}
var known=KNOWN_GAMES[exe];
if(known&&known.name)return known.name;
var app=detectableExes?detectableExes.get(exe):null;
if(app&&app.name)return app.name;
return exe.replace(/\.[^.]+$/,'')||raw;
}
var dispatchQueue=[];
var isProcessingQueue=false;
function queueDispatch(dispatcher,payload){
dispatchQueue.push({dispatcher:dispatcher,payload:payload});
processDispatchQueue();
}
function processDispatchQueue(){
if(isProcessingQueue)return;
if(dispatchQueue.length===0)return;
var next=dispatchQueue[0];
var dispatcher=next.dispatcher||getFluxDispatcher();
if(!dispatcher){
setTimeout(processDispatchQueue,100);
return;
}
if(typeof dispatcher.isDispatching==='function'&&dispatcher.isDispatching()){
setTimeout(processDispatchQueue,25);
return;
}
isProcessingQueue=true;
dispatchQueue.shift();
try{
dispatcher.dispatch(next.payload);
}catch(err){
console.warn('[DiscordLite] Dispatch failed:',err);
}finally{
isProcessingQueue=false;
}
if(dispatchQueue.length>0){
setTimeout(processDispatchQueue,25);
}
}
var detectableExes=null,detectablePromise=null;
function loadDetectableApps(){
if(detectableExes)return Promise.resolve(detectableExes);
if(detectablePromise)return detectablePromise;
detectablePromise=fetch('/api/v9/applications/detectable')
.then(function(r){return r.ok?r.json():[];})
.then(function(apps){
var map=new Map();
if(Array.isArray(apps)){
for(var i=0;i<apps.length;i++){
var app=apps[i];
if(app&&app.executables){
for(var j=0;j<app.executables.length;j++){
var item=app.executables[j];
if(item&&item.name){
var parts=item.name.split('/');
var exe=parts[parts.length-1].toLowerCase();
if(!map.has(exe))map.set(exe,app);
}
}
}
}
}
detectableExes=map;
return map;
})
.catch(function(err){
console.warn('[DiscordLite] Failed to load detectable applications:',err);
detectablePromise=null;
return null;
});
return detectablePromise;
}
var activePresenceKey=null,pendingPresenceData=null,lastAppliedData=null;
var pendingRPCActivities={};
function hookDispatcher(d){
if(!d||d.__vcPresenceHooked)return;
d.__vcPresenceHooked=true;
function onOpen(){
activePresenceKey=null;
if(lastAppliedData)updateDetectedGamePresence(lastAppliedData);
}
try{
if(typeof d.subscribe==='function'){
d.subscribe('CONNECTION_OPEN',onOpen);
d.subscribe('POST_CONNECTION_OPEN',onOpen);
}
}catch(e){}
}
function updateDetectedGamePresence(data){
if(!data)return;
var dispatcher=getFluxDispatcher();
if(!dispatcher){
pendingPresenceData=data;
return;
}
hookDispatcher(dispatcher);
var running=(data.detectionEnabled&&data.runningGames)||[];
var activeGames=running.filter(function(g){return !isGameDisabled(data.disabledGames,g.path);});
if(!activeGames.length){
if(activePresenceKey!==null){
activePresenceKey=null;
queueDispatch(dispatcher,{type:'LOCAL_ACTIVITY_UPDATE',socketId:'GameActivity',activity:null});
}
return;
}
var primary=activeGames[0];
var exeName=primary.path?primary.path.replace(/^.*[\\\/]/,'').toLowerCase():'';
var known=KNOWN_GAMES[exeName];
var app=detectableExes?detectableExes.get(exeName):null;
var appId=primary.applicationId||(app&&app.id)||(known&&known.id)||'';
var gameName=resolveDisplayName(primary.name,primary.path,data.overrides);
var presenceKey=primary.path+'|'+gameName+'|'+appId;
if(activePresenceKey===presenceKey)return;
activePresenceKey=presenceKey;
var activity={
name:gameName,
type:0,
flags:1,
platform:'desktop',
timestamps:{
start:(typeof primary.start==='number'&&primary.start>0)?primary.start:Date.now()
}
};
if(appId&&appId!=='0')activity.application_id=appId;
var payload={
type:'LOCAL_ACTIVITY_UPDATE',
socketId:'GameActivity',
activity:activity
};
if(appId&&appId!=='0')payload.applicationId=appId;
if(typeof primary.pid==='number'&&primary.pid>0)payload.pid=primary.pid;
queueDispatch(dispatcher,payload);
}
function applyRPCActivity(dispatcher,socketId,activity,pid){
if(activity){
var parts=socketId.split(':');
var appId=activity.application_id||parts[1]||'0';
if(!activity.application_id)activity.application_id=appId;
if(typeof activity.type!=='number')activity.type=0;
if(typeof activity.flags!=='number')activity.flags=1;
if(!activity.platform)activity.platform='desktop';
queueDispatch(dispatcher,{
type:'LOCAL_ACTIVITY_UPDATE',
socketId:socketId,
applicationId:appId,
pid:typeof pid==='number'?pid:0,
activity:activity
});
}else{
queueDispatch(dispatcher,{
type:'LOCAL_ACTIVITY_UPDATE',
socketId:socketId,
activity:null
});
}
}
function flushPendingPresence(){
var d=getFluxDispatcher();
if(!d)return;
hookDispatcher(d);
if(pendingPresenceData){
var pd=pendingPresenceData;
pendingPresenceData=null;
updateDetectedGamePresence(pd);
}
for(var socketId in pendingRPCActivities){
var item=pendingRPCActivities[socketId];
delete pendingRPCActivities[socketId];
if(item)applyRPCActivity(d,socketId,item.activity,item.pid);
}
}
window.__vcGameActivityApply=function(data){
loading=false;
if(!data)return;
lastAppliedData=data;
if(Date.now()>=pausePollUntil)render(data);
loadDetectableApps().finally(function(){
updateDetectedGamePresence(data);
});
};
window.__vcSetRPCActivity=function(payload){
if(!payload||!payload.socketId)return;
var dispatcher=getFluxDispatcher();
if(!dispatcher){
pendingRPCActivities[payload.socketId]={activity:payload.activity,pid:payload.pid};
return;
}
applyRPCActivity(dispatcher,payload.socketId,payload.activity,payload.pid);
};
function setGameToggleOptimistic(path,enabled){
pausePollUntil=Date.now()+3000;
var norm=normId(path);
if(lastAppliedData){
lastAppliedData.disabledGames=lastAppliedData.disabledGames||{};
if(enabled){
delete lastAppliedData.disabledGames[norm];
delete lastAppliedData.disabledGames[(path||'').toLowerCase()];
}else{
lastAppliedData.disabledGames[norm]=true;
}
lastRenderedSignature=getStateSignature(lastAppliedData);
updateDetectedGamePresence(lastAppliedData);
}
var panel=document.getElementById(panelId);
if(panel){
var toggles=panel.querySelectorAll('[data-vc-game-toggle]');
toggles.forEach(function(t){
if(normId(t.dataset.vcPath)===norm){
t.checked=enabled;
var r=t.closest('.vc-game-row');if(r)r.classList.toggle('vc-disabled',!enabled);
var s=t.closest('.vc-switch');if(s)s.title=enabled?'Disable activity detection for this game':'Enable activity detection for this game';
}
});
}
}
function setDetectionToggleOptimistic(enabled){
pausePollUntil=Date.now()+3000;
if(lastAppliedData){
lastAppliedData.detectionEnabled=enabled;
lastRenderedSignature=getStateSignature(lastAppliedData);
updateDetectedGamePresence(lastAppliedData);
}
var panel=document.getElementById(panelId);
if(panel){
var dt=panel.querySelector('[data-vc-detection-toggle]');
if(dt)dt.checked=enabled;
}
}
function refresh(){
if(loading||Date.now()<pausePollUntil)return;
loading=true;
var panel=document.getElementById(panelId),initial=lastAppliedData||window.__vcGameActivityInitial||{detectionEnabled:true,runningGames:[],gamesSeen:[],overrides:{}};
if(panel&&!panel.children.length)render(initial);
var timeout=setTimeout(function(){loading=false;},3000);
invoke('vc_game_activity').then(function(data){
if(data&&Date.now()>=pausePollUntil){
lastAppliedData=data;
render(data);
updateDetectedGamePresence(data);
}else if(!lastAppliedData){
render(initial);
}
}).catch(function(error){
console.error('Game activity refresh failed',error);
}).finally(function(){
clearTimeout(timeout);
loading=false;
});
}
function close(){
active=false;
var item=document.getElementById(navId),panel=document.getElementById(panelId);
if(item){
item.classList.remove('vc-selected');
for(var i=item.classList.length-1;i>=0;i--){
if(item.classList[i].indexOf('selected')>=0)item.classList.remove(item.classList[i]);
}
}
if(panel)panel.style.setProperty('display','none','important');
}
function open(){
active=true;
var item=document.getElementById(navId),panel=document.getElementById(panelId);
if(!item){
mount();
item=document.getElementById(navId);
panel=document.getElementById(panelId);
}
if(item){
var otherSelected=document.querySelector('[class*="item_"][class*="selected_"]'),selClass=otherSelected?otherSelected.className.match(/\bselected_\w+\b/)?.[0]:null;
if(otherSelected&&selClass)otherSelected.classList.remove(selClass);
item.classList.add('vc-selected');
if(selClass)item.classList.add(selClass);
}
if(!panel)panel=installPanel();
if(panel){
panel.style.setProperty('display','flex','important');
panel.style.setProperty('visibility','visible','important');
panel.style.setProperty('opacity','1','important');
panel.style.setProperty('pointer-events','auto','important');
position(panel);
var currentData=lastAppliedData||window.__vcGameActivityInitial;
if(currentData)render(currentData);
refresh();
}
}
function installPanel(){
var panel=document.getElementById(panelId);
if(!panel){
panel=document.createElement('div');
panel.id=panelId;
panel.style.setProperty('display','none','important');
document.body.appendChild(panel);
}
position(panel);
return panel;
}
function getBaseItemClass(el){
if(!el||!el.className)return '';
return el.className.split(/\s+/).filter(function(c){return c&&c.indexOf('selected')===-1&&c!=='vc-selected';}).join(' ');
}
function mount(){
addStyles();
var refLi=findSidebarActivityPrivacy();
if(!refLi)return;
var refItem=refLi.querySelector('[role="link"],[role="button"],[role="tab"],a,button,[class*="item_"]')||refLi;
var refContent=refItem.querySelector('[class*="itemContent"]')||refItem;
var label=findExact(refContent,'Activity Privacy')||refContent.querySelector('[class*="text-"]')||refContent;
var container=refLi.parentElement;
if(!container)return;
var connectedLi=findSidebarConnectedApps();
var before=(connectedLi&&connectedLi.parentElement===container)?connectedLi:(refLi.nextElementSibling===document.getElementById('vc-game-activity-li')?refLi.nextElementSibling.nextElementSibling:refLi.nextElementSibling);
var panel=installPanel();
var li=document.getElementById('vc-game-activity-li');
var item=document.getElementById(navId);
var baseItemClass=getBaseItemClass(refItem);
var baseLiClass=refLi.className||'';
if(!li||!item||li.parentElement!==container){
if(li)li.remove();
if(item&&!li)item.remove();
li=document.createElement(refLi.tagName.toLowerCase());
li.id='vc-game-activity-li';
if(baseLiClass)li.className=baseLiClass;
li.setAttribute('data-settings-sidebar-item','registered_games_panel');
item=document.createElement('div');
item.id=navId;
if(baseItemClass)item.className=baseItemClass;
item.setAttribute('role',refItem.getAttribute('role')||'link');
item.tabIndex=0;
item.setAttribute('data-list-item-id','settings-sidebar___registered_games_panel');
var content=document.createElement('div');
if(refContent&&refContent.className)content.className=refContent.className;
var refIcon=(refContent&&refContent.querySelector('svg,[class*="icon"]'))||refItem.querySelector('svg,[class*="icon"]');
if(refIcon){
var icon=refIcon.cloneNode(true);
icon.classList.add('vc-game-activity-icon');
content.appendChild(icon);
}
var itemLabel=document.createElement('div');
if(label&&label.className)itemLabel.className=label.className;
if(label&&label.dataset&&label.dataset.textVariant)itemLabel.dataset.textVariant=label.dataset.textVariant;
itemLabel.style.color='currentColor';
itemLabel.textContent='Registered Games';
content.appendChild(itemLabel);
item.appendChild(content);
li.appendChild(item);
item.addEventListener('click',function(event){
event.preventDefault();
event.stopPropagation();
event.stopImmediatePropagation();
open();
});
item.addEventListener('keydown',function(event){
if(event.key==='Enter'||event.key===' '){
event.preventDefault();
event.stopPropagation();
open();
}
});
if(before)container.insertBefore(li,before);
else container.appendChild(li);
}else{
if(baseLiClass&&li.className!==baseLiClass)li.className=baseLiClass;
if(li.previousElementSibling!==refLi){
if(before&&before!==li)container.insertBefore(li,before);
else if(!before)container.appendChild(li);
}
if(!active&&baseItemClass&&item.className!==baseItemClass){
item.className=baseItemClass;
}
}
if(active){
var selClass=document.querySelector('[class*="item_"][class*="selected_"]')?.className.match(/\bselected_\w+\b/)?.[0];
item.classList.add('vc-selected');
if(selClass)item.classList.add(selClass);
panel.style.setProperty('display','flex','important');
position(panel);
}
}
window.addEventListener('click',function(event){
var closeButton=event.target.closest&&event.target.closest('button[aria-label="Close"]');
if(closeButton){
setTimeout(function(){if(!isSettingsOpen())dispose();},50);
}
var nav=event.target.closest&&event.target.closest('#'+navId);
if(!nav)return;
event.preventDefault();
event.stopPropagation();
event.stopImmediatePropagation();
open();
},true);
function showActionError(error){
var panel=document.getElementById(panelId);
if(!panel)return;
var message=document.createElement('div');
message.className='vc-game-empty';
message.textContent='Unable to update games: '+(error&&error.message||String(error));
panel.append(message);
}
function handlePanelAction(action){
var kind=action.dataset.vcAction;
if(kind==='add'){
invoke('vc_game_activity_add',{path:''}).then(function(d){
if(d){
lastAppliedData=d;
render(d);
updateDetectedGamePresence(d);
}
}).catch(showActionError);
}else if(kind==='add-path'){
var panel=document.getElementById(panelId),input=panel&&panel.querySelector('[data-vc-game-path]'),path=input&&input.value.trim();
if(!path){
showActionError(new Error('Enter an executable path or use Browse'));
return;
}
invoke('vc_game_activity_add',{path:path}).then(function(d){
if(d){
lastAppliedData=d;
render(d);
updateDetectedGamePresence(d);
}
}).catch(showActionError);
}else if(kind==='refresh'){
lastRenderedSignature='';
refresh();
}else if(kind==='remove'){
invoke('vc_game_activity_remove',{path:action.dataset.vcPath}).then(function(d){
if(d){
lastAppliedData=d;
render(d);
updateDetectedGamePresence(d);
}
}).catch(showActionError);
}
}
window.addEventListener('click',function(event){
var panel=document.getElementById(panelId);
if(!panel||!panel.contains(event.target))return;
var action=event.target.closest&&event.target.closest('[data-vc-action]');
if(!action)return;
event.preventDefault();
event.stopPropagation();
event.stopImmediatePropagation();
handlePanelAction(action);
},true);
document.addEventListener('click',function(event){
var nav=event.target.closest&&event.target.closest('#'+navId);
if(nav){
event.preventDefault();
event.stopPropagation();
event.stopImmediatePropagation();
open();
return;
}
var panel=document.getElementById(panelId),item=document.getElementById(navId);
if(panel&&panel.style.display!=='none'){
var sidebar=event.target.closest('[class*="sidebar"],[class*="Sidebar"],[role="tablist"]');
if(sidebar&&(!item||!item.contains(event.target)))close();
}
if(!panel||!panel.contains(event.target))return;
var action=event.target.closest('[data-vc-action]');
if(!action)return;
handlePanelAction(action);
},true);
document.addEventListener('change',function(event){
var action=event.target.closest&&event.target.closest('[data-vc-detection-toggle]');
if(action){
var enabled=!!action.checked;
setDetectionToggleOptimistic(enabled);
invoke('vc_game_activity_set_detection',{enabled:enabled}).then(function(data){
if(data){
lastAppliedData=data;
lastRenderedSignature=getStateSignature(data);
updateDetectedGamePresence(data);
}
}).catch(function(error){
setDetectionToggleOptimistic(!enabled);
console.error(error);
});
return;
}
var gameToggle=event.target.closest&&event.target.closest('[data-vc-game-toggle]');
if(gameToggle){
var path=gameToggle.dataset.vcPath,enabled=!!gameToggle.checked;
if(path){
setGameToggleOptimistic(path,enabled);
invoke('vc_game_activity_toggle_game',{path:path,enabled:enabled}).then(function(data){
if(data){
lastAppliedData=data;
lastRenderedSignature=getStateSignature(data);
updateDetectedGamePresence(data);
}
}).catch(function(error){
setGameToggleOptimistic(path,!enabled);
showActionError(error);
});
}
}
},true);
window.addEventListener('keydown',function(event){
if(event.key==='Escape'&&active&&isSettingsOpen()){
event.preventDefault();
event.stopPropagation();
event.stopImmediatePropagation();
closeSettings();
}
},true);
window.addEventListener('resize',function(){
var panel=document.getElementById(panelId);
if(panel&&panel.style.display!=='none')position(panel);
});
function dispose(){
active=false;
var panel=document.getElementById(panelId),item=document.getElementById(navId),li=document.getElementById('vc-game-activity-li');
if(panel)panel.remove();
if(li)li.remove();
else if(item)item.remove();
}
var observer=new MutationObserver(function(){
if(!isSettingsOpen()){
if(document.getElementById(panelId)||document.getElementById(navId)||document.getElementById('vc-game-activity-li'))dispose();
return;
}
mount();
});
setInterval(function(){
if(active&&!loading&&isSettingsOpen()){
var panel=document.getElementById(panelId);
if(panel&&panel.style.display!=='none')position(panel);
refresh();
}
flushPendingPresence();
},1000);
function start(){
mount();
observer.observe(document.documentElement,{childList:true,subtree:true});
if(window.__vcGameActivityInitial){
lastAppliedData=window.__vcGameActivityInitial;
render(window.__vcGameActivityInitial);
updateDetectedGamePresence(window.__vcGameActivityInitial);
}
loadDetectableApps().finally(function(){
var data=lastAppliedData||window.__vcGameActivityInitial;
if(data)updateDetectedGamePresence(data);
});
}
if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',start,{once:true});
else start();
})();
