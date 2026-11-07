<template>
 <button type="button" class="screen-power-button" :class="{'is-screen-off':on===false}" :disabled="disabled||busy||checking&&on===null" :title="title" :aria-label="label" :aria-busy="busy||checking" @click.stop="toggle">
  <svg v-if="showIcon" class="quick-control-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
   <template v-if="on===false"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M4.9 4.9l1.4 1.4m11.4 11.4 1.4 1.4M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/></template>
   <path v-else d="M20.9 13a9 9 0 0 1-9.9-9.9A9 9 0 1 0 20.9 13Z"/>
  </svg><span>{{label}}</span>
 </button>
</template>
<script setup>
import {ref,computed,onMounted,onUnmounted,watch} from 'vue'
import {useDeviceStore} from '@/stores/devices'
const props=defineProps({deviceId:{type:String,required:true},disabled:Boolean,showIcon:Boolean})
const store=useDeviceStore(),on=ref(null),busy=ref(false),checking=ref(false),error=ref('')
const label=computed(()=>busy.value?'确认中…':on.value===null?(checking.value?'检测中…':'重新检测'):on.value?'熄屏':'亮屏')
const title=computed(()=>error.value||(on.value===null?'重新读取手机实体屏幕状态':on.value?'当前实体屏幕已亮，点击熄屏；远控与后台应用保持运行':'当前实体屏幕已熄，点击亮屏'))
let timer,controller,disposed=false
async function refresh(){
 if(disposed||props.disabled)return null
 controller?.abort();const request=new AbortController();controller=request;checking.value=true
 const timeout=setTimeout(()=>request.abort(),6500)
 try{
  const r=await fetch('/api/devices/'+encodeURIComponent(props.deviceId)+'/screen-state',{signal:request.signal,cache:'no-store'});const data=await r.json()
  if(!r.ok)throw Error(data.error||'读取屏幕状态失败')
  if(controller!==request||disposed)return null
  on.value=data.known?!!data.on:null;error.value=data.known?'':'无法确认实体屏幕状态，点击重新检测'
  return on.value
 }catch(e){if(controller===request&&!disposed){on.value=null;error.value='屏幕状态检测失败，点击重试'}return null}
 finally{clearTimeout(timeout);if(controller===request)checking.value=false}
}
const delay=ms=>new Promise(resolve=>setTimeout(resolve,ms))
async function toggle(){
 if(busy.value||props.disabled)return
 busy.value=true
 try{
  // Always recheck immediately before acting: the physical power key may have changed it.
  const id=props.deviceId,expected=on.value,wasUnknown=on.value===null
  const current=await refresh();if(current===null||wasUnknown||props.deviceId!==id)return
  if(current!==expected){error.value='屏幕状态已变化，请确认后再次操作';return}
  if(!store.sendGroupControlEvent([props.deviceId],{type:'set_display_power',on:!current})){error.value='控制连接未就绪，请稍后再试';return}
  for(let i=0;i<5;i++){await delay(400);if(disposed)return;if(await refresh()===!current){window.dispatchEvent(new CustomEvent('screen-power-changed',{detail:props.deviceId}));return}}
  error.value='未确认屏幕切换成功，请检查手机或重试'
 }finally{busy.value=false}
}
function resume(){if(!document.hidden&&!busy.value)refresh()}
function changed(e){if(e.detail===props.deviceId&&!busy.value)refresh()}
watch(()=>[props.deviceId,props.disabled],()=>{controller?.abort();on.value=null;error.value='';if(!props.disabled)refresh()})
onMounted(()=>{refresh();timer=setInterval(resume,5000);window.addEventListener('focus',resume);document.addEventListener('visibilitychange',resume);window.addEventListener('screen-power-changed',changed)})
onUnmounted(()=>{disposed=true;clearInterval(timer);controller?.abort();window.removeEventListener('focus',resume);document.removeEventListener('visibilitychange',resume);window.removeEventListener('screen-power-changed',changed)})
</script>
<style scoped>.screen-power-button{font:inherit;min-height:36px;min-width:68px;padding:7px 10px;border:1px solid #62e3c44d;border-radius:8px;color:#62e3c4;background:#62e3c409;cursor:pointer;white-space:nowrap}.screen-power-button:disabled{opacity:.4;cursor:not-allowed}</style>
