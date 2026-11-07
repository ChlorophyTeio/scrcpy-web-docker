const descriptions={ '电源':'模拟手机电源键，会锁屏或唤醒。仅关闭实体屏幕请使用熄屏功能。','首页':'回到手机桌面。','返回':'返回手机的上一个页面。','音量+':'增加手机音量。','音量-':'降低手机音量。','截图':'保存当前手机画面。','送剪贴板':'将浏览器剪贴板中的文字发送到手机。','收剪贴板':'将手机剪贴板文字复制到浏览器。','映射':'设置键盘按键与手机触摸操作的对应关系。','终端':'打开该手机的命令终端。','粘贴到手机':'打开文字面板，可从本机剪贴板读取、编辑后粘贴到手机输入框。','复制手机文字':'读取手机中已复制的文字，再复制到本机剪贴板。','最近任务':'打开手机最近使用的应用。','页面全屏':'在当前网页中放大设备窗口，标题栏和快捷操作仍可使用。按 Esc 退出。','系统全屏':'使用浏览器全屏模式，不改变手机或浏览器方向。','画中画':'把当前画面放到独立小窗，主网页继续保持连接。','设置':'调整画质、帧率、音频和网络连接参数。' }
let timer,start,pressed,longPressed=false,blockedUntil=0,blockedTarget,tip
const find=target=>target instanceof Element?target.closest('.window-action,.item-btn,.sidebar-size-toggle,.sidebar-btn,.fab-item,.nav-key-btn,.fullscreen-fab,.webfullscreen-fab,.control-btn,.screen-power-button,.camera-action,.cam-btn,.quick-screen-action,.unlock-device-button'):null
function hide(){tip?.remove();tip=null}
function cancel(){clearTimeout(timer);timer=null;pressed=null}
function show(button){hide();const title=button.title||button.textContent.trim();tip=document.createElement('div');tip.className='control-longpress-help';tip.setAttribute('role','tooltip');tip.textContent=descriptions[title]||title||'手机控制';(document.fullscreenElement||document.body).append(tip);const r=button.getBoundingClientRect(),w=Math.min(300,innerWidth-24);tip.style.width=w+'px';tip.style.left=Math.max(12,Math.min(innerWidth-w-12,r.x+r.width/2-w/2))+'px';tip.style.top=Math.max(12,r.y-tip.offsetHeight-12)+'px';longPressed=true;blockedTarget=button;blockedUntil=Date.now()+1200;setTimeout(hide,3500)}
document.addEventListener('pointerdown',e=>{const button=find(e.target);if(!button)return;cancel();hide();longPressed=false;pressed=button;start=[e.clientX,e.clientY];timer=setTimeout(()=>{if(pressed===button)show(button)},550)},true)
document.addEventListener('pointermove',e=>{if(pressed&&(Math.abs(e.clientX-start[0])>12||Math.abs(e.clientY-start[1])>12)){cancel();hide()}},true)
document.addEventListener('pointerup',()=>{if(longPressed)blockedUntil=Date.now()+900;cancel()},true)
document.addEventListener('pointercancel',cancel,true)
document.addEventListener('click',e=>{if(Date.now()<blockedUntil&&find(e.target)===blockedTarget){e.preventDefault();e.stopImmediatePropagation()}else hide()},true)
document.addEventListener('contextmenu',e=>{const button=find(e.target);if(button){e.preventDefault();show(button)}},true)
