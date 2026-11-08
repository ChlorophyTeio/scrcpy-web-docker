<!-- Modified for the optional-plugins distribution (2026). See NOTICE. -->
<template>
  <div class="mobile-remote-controls" @pointerdown="emit('focus')">
    <nav class="mobile-remote-dock" aria-label="手机快捷操作">
      <button v-for="key in navigation" :key="key.id" type="button" :disabled="!connected" :title="key.hint" @click="act(key.id)">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="key.path"/></svg><span>{{key.label}}</span>
      </button>
      <button type="button" @click="act('paste')" title="编辑文字并粘贴到远端手机">
        <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="5" y="5" width="14" height="16" rx="2"/><rect x="9" y="2" width="6" height="5" rx="1"/></svg><span>剪贴板</span>
      </button>
      <button type="button" aria-haspopup="dialog" :aria-expanded="open" :class="{active:open}" @click="emit('update:open',!open)">
        <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/></svg><span>更多</span>
      </button>
    </nav>
    <dialog ref="sheet" class="mobile-controls-sheet" :class="{dialog:open}" aria-label="远控功能" @cancel.prevent="close" @close="onClose" @click="onBackdrop">
      <header><div><h2>远控功能</h2><p>{{connected?'控制已连接':statusText}}</p></div><button type="button" class="sheet-close" aria-label="关闭远控功能" @click="close">×</button></header>
      <div class="mobile-sheet-scroll">
        <section aria-label="屏幕与输入"><h3>屏幕与输入</h3><div class="mobile-action-grid">
          <ScreenPowerButton v-if="open" :device-id="deviceId" :disabled="!connected" show-icon />
          <div v-if="canShell" class="sheet-unlock" @click.capture="close"><UnlockDeviceButton :device-id="deviceId" /></div>
          <button type="button" :disabled="!connected" @click="act('power')"><strong>锁屏 / 唤醒</strong><small>操作手机电源键</small></button>
          <button type="button" :disabled="!connected" @click="act('keyboard')"><strong>手机输入</strong><small>打开本机键盘</small></button>
          <button type="button" @click="act('fullscreen')"><strong>{{fullscreen?'退出全屏':'全屏显示'}}</strong><small>保持当前屏幕方向</small></button>
          <button type="button" @click="act('settings')"><strong>画质设置</strong><small>清晰度、帧率与连接</small></button>
        </div></section>
        <section aria-label="复制与粘贴"><h3>复制与粘贴</h3><div class="mobile-action-grid">
          <button type="button" @click="act('paste')"><strong>粘贴到手机</strong><small>先编辑，再确认发送</small></button>
          <button type="button" :disabled="!connected" @click="act('copy')"><strong>复制手机文字</strong><small>读取远端剪贴板</small></button>
          <button type="button" role="switch" :aria-checked="clipboardAutoSync" @click="emit('clipboard-toggle',!clipboardAutoSync)"><strong>双向联动：{{clipboardAutoSync?'开':'关'}}</strong><small>{{clipboardStatus}}</small></button>
          <button type="button" :disabled="!connected" @click="act('sync')"><strong>立即同步</strong><small>同步两端剪贴板</small></button>
        </div></section>
        <section aria-label="声音与画面"><h3>声音与画面</h3><div class="mobile-action-grid">
          <button type="button" :disabled="!connected" @click="act('volume-up')"><strong>音量 ＋</strong><small>调高手机音量</small></button>
          <button type="button" :disabled="!connected" @click="act('volume-down')"><strong>音量 －</strong><small>调低手机音量</small></button>
          <button type="button" :disabled="!!audioUnavailableReason" :title="audioUnavailableReason" @click="act('audio')"><strong>{{audioAction}}</strong><small>{{audioUnavailableReason||'控制本机播放的声音'}}</small></button>
          <button type="button" :disabled="!connected" @click="act('screenshot')"><strong>截图</strong><small>保存当前手机画面</small></button>
          <button v-if="pictureInPictureSupported" type="button" @click="act('pip')"><strong>画中画</strong><small>小窗显示手机画面</small></button>
          <button type="button" @click="act('page-fullscreen')"><strong>页面全屏</strong><small>在浏览器内铺满</small></button>
        </div></section>
        <details><summary>更多功能</summary><div class="mobile-action-grid">
          <button type="button" :disabled="streamModeDisabled" :title="streamModeTitle" @click="act('stream-mode')"><strong>{{websocketMode?'切回 WebRTC':'切换 WebSocket'}}</strong><small>切换投屏连接方式</small></button>
          <button v-if="canShell" type="button" @click="act('terminal')"><strong>终端</strong><small>打开设备控制台</small></button>
          <button v-if="admin" type="button" @click="act('group')"><strong>{{groupActive?'取消群控':'群控主控'}}</strong></button>
          <button type="button" @click="act('keymap')"><strong>按键映射</strong></button>
          <button type="button" @click="act('key-hints')"><strong>切换按键提示</strong></button>
          <template v-for="(item,index) in customButtons" :key="index"><button type="button" :disabled="!connected" @click="act('custom',index)"><strong>{{item.name}}</strong></button><button type="button" @click="act('delete-custom',index)">删除 {{item.name}}</button></template>
          <button type="button" @click="act('add-custom')"><strong>添加自定义按键</strong></button>
          <button type="button" class="danger" @click="act('disconnect')"><strong>断开远控</strong></button>
          <button v-if="admin" type="button" class="danger" @click="act('quit-agent')"><strong>退出 Agent</strong><small>停止手机端服务</small></button>
        </div></details>
      </div>
    </dialog>
  </div>
</template>
<script setup>
import {ref,watch,nextTick,onBeforeUnmount} from 'vue'
import ScreenPowerButton from '@/components/ScreenPowerButton.vue'
import UnlockDeviceButton from '@/components/UnlockDeviceButton.vue'
const props=defineProps({deviceId:{type:String,required:true},open:Boolean,connected:Boolean,statusText:String,fullscreen:Boolean,canShell:Boolean,admin:Boolean,clipboardAutoSync:Boolean,clipboardStatus:String,audioAction:String,audioUnavailableReason:String,pictureInPictureSupported:Boolean,websocketMode:Boolean,streamModeDisabled:Boolean,streamModeTitle:String,groupActive:Boolean,customButtons:{type:Array,default:()=>[]}})
const emit=defineEmits(['action','update:open','clipboard-toggle','focus'])
const sheet=ref(null)
const navigation=[{id:'back',label:'返回',hint:'返回上一级',path:'M19 12H5m7-7-7 7 7 7'},{id:'home',label:'首页',hint:'回到手机桌面',path:'m3 10 9-7 9 7v10H3zm6 10v-7h6v7'},{id:'recent',label:'最近任务',hint:'查看手机最近任务',path:'M6 3h12v18H6zm4 5h4m-4 5h4'}]
let disposed=false
function close(){if(sheet.value?.open)sheet.value.close();emit('update:open',false)}
function onClose(){if(!disposed&&props.open&&!sheet.value?.open)emit('update:open',false)}
function act(name,index){close();emit('action',name,index)}
function onBackdrop(event){if(event.target!==sheet.value)return;const b=sheet.value.getBoundingClientRect();if(event.clientX<b.left||event.clientX>b.right||event.clientY<b.top||event.clientY>b.bottom)close()}
watch(()=>props.open,async open=>{await nextTick();const el=sheet.value;if(disposed||!el?.isConnected)return;if(open&&!el.open)el.showModal();else if(!open&&el.open)el.close()},{immediate:true})
onBeforeUnmount(()=>{disposed=true;if(sheet.value?.open)sheet.value.close()})
</script>
<style scoped>
.mobile-remote-controls{flex:none;min-width:0;width:100%;background:#101e2d;border-top:1px solid #30465c;z-index:15}
.mobile-remote-dock{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:4px;padding:6px max(6px,env(safe-area-inset-right)) max(6px,env(safe-area-inset-bottom)) max(6px,env(safe-area-inset-left))}
.mobile-remote-dock button{min-width:0;min-height:52px;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:4px;border:1px solid transparent;border-radius:10px;background:transparent;color:#d0e3f5;font:inherit;font-size:12px;touch-action:manipulation}
.mobile-remote-dock svg{width:20px;height:20px;fill:none;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}
.mobile-remote-dock button:active,.mobile-remote-dock .active{background:#244861;border-color:#477b9b;color:#aee5ff}
button:disabled{opacity:.4;cursor:not-allowed}button:focus-visible,summary:focus-visible{outline:2px solid #7cd9ff;outline-offset:-2px}
.mobile-controls-sheet{position:fixed;inset:auto 8px max(8px,env(safe-area-inset-bottom));margin:0 auto;width:calc(100% - 16px);max-width:600px;max-height:calc(100dvh - 24px);padding:0;border:1px solid #405a72;border-radius:18px;background:#102030;color:#d9e9f8;box-shadow:0 12px 50px #0009;overflow:hidden;font-family:inherit;text-align:left}
.mobile-controls-sheet[open]{display:flex;flex-direction:column}
.mobile-controls-sheet::backdrop{background:#030a14a6;backdrop-filter:blur(3px)}
.mobile-controls-sheet header{flex:none;display:flex;justify-content:space-between;align-items:center;gap:12px;padding:12px 14px;border-bottom:1px solid #334b60}
h2{font-size:17px;line-height:1.5;margin:0}header p{font-size:12px;color:#8eb9cb;margin:3px 0 0}.sheet-close{min-width:44px;min-height:44px;border:1px solid #415f78;border-radius:10px;background:#192f43;color:#e2effa;font-size:24px}
.mobile-sheet-scroll{min-height:0;overflow-y:auto;overscroll-behavior:contain;padding:0 14px 16px;scrollbar-width:thin}section{margin:14px 0 18px}h3{font-size:12px;color:#a1b9cf;font-weight:500;margin:0 0 8px}.mobile-action-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px}
.mobile-action-grid>button,.sheet-unlock :deep(button){width:100%;min-width:0;min-height:56px;padding:9px;border:1px solid #365269;border-radius:10px;background:#1a3247;color:#d6eafa;font:inherit;font-size:13px;white-space:normal;line-height:1.4;touch-action:manipulation}
.mobile-action-grid>button{display:flex;flex-direction:column;align-items:flex-start;justify-content:center;gap:3px;text-align:left}.mobile-action-grid strong{font-weight:500;overflow-wrap:anywhere}.mobile-action-grid small{font-size:11px;line-height:1.45;color:#96afc3;overflow-wrap:anywhere}
.mobile-action-grid>.screen-power-button{flex-direction:row;align-items:center;justify-content:flex-start;gap:8px}.screen-power-button :deep(svg){width:20px;height:20px;flex:none}.sheet-unlock :deep(svg){width:20px;height:20px}.sheet-unlock :deep(button){display:flex;align-items:center;justify-content:center;gap:5px}
.mobile-action-grid .danger{color:#ffc3cb;background:#3b2533;border-color:#75505b}details{border-top:1px solid #30485c;padding-top:8px}summary{min-height:44px;line-height:44px;cursor:pointer;color:#b9d4e8;font-size:13px}details[open] summary{margin-bottom:6px}
@media(orientation:landscape) and (max-height:500px){.mobile-remote-dock{padding-top:2px;padding-bottom:max(2px,env(safe-area-inset-bottom))}.mobile-remote-dock button{min-height:44px;flex-direction:row;gap:7px}.mobile-controls-sheet{max-height:calc(100dvh - 16px)}.mobile-action-grid{grid-template-columns:repeat(3,minmax(0,1fr))}}
</style>
