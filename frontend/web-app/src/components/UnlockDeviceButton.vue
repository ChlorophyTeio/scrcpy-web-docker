<template>
  <button ref="opener" type="button" class="unlock-device-button" :disabled="disabled" title="输入已知 PIN，通过 ADB 解锁；无需投屏画面" @click.stop="open">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true"><rect x="5" y="10" width="14" height="11" rx="2"/><path d="M8 10V6a4 4 0 0 1 7.5-2M12 14v3"/></svg><span>PIN 解锁</span>
  </button>
  <Teleport :to="teleportTarget">
    <div v-if="visible" class="unlock-overlay" @click.self="close" @keydown.stop="onKeydown" @keyup.stop @paste.stop>
      <section ref="dialog" class="unlock-dialog" role="dialog" aria-modal="true" :aria-labelledby="titleId">
        <h2 :id="titleId">解锁手机</h2>
        <p class="unlock-device-id">{{ deviceId }}</p>
        <p class="unlock-description">通过 ADB 唤醒并输入一次 PIN，投屏黑屏时也可执行。PIN 不保存，不自动重试。</p>
        <form @submit.prevent="submit" autocomplete="off">
          <label :for="inputId">锁屏 PIN</label>
          <input :id="inputId" ref="input" v-model="pin" type="password" inputmode="numeric" pattern="[0-9]{4,16}" minlength="4" maxlength="16" autocomplete="off" :disabled="pending" placeholder="输入 4–16 位数字 PIN" required />
          <p class="unlock-result" :class="{ success: resultStatus === 'unlocked' }" role="status" aria-live="polite">{{ pending ? '正在执行一次解锁并检查状态…' : message }}</p>
          <div class="unlock-dialog-actions">
            <button type="button" :disabled="pending" @click="close">关闭</button>
            <button type="submit" class="unlock-submit" :disabled="pending || !validPIN">{{ pending ? '执行中…' : '解锁一次' }}</button>
          </div>
        </form>
      </section>
    </div>
  </Teleport>
</template>
<script setup>
import { ref, computed, nextTick, onUnmounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
const props = defineProps({ deviceId: {type:String,required:true}, disabled:Boolean })
const auth=useAuthStore(),visible=ref(false),pin=ref(''),pending=ref(false),message=ref(''),resultStatus=ref(''),opener=ref(null),input=ref(null),dialog=ref(null)
const titleId=`unlock-title-${props.deviceId}-${Math.random().toString(36).slice(2)}`,inputId=`${titleId}-pin`
const teleportTarget=ref('body')
const validPIN=computed(()=>/^[0-9]{4,16}$/.test(pin.value))
async function open(){teleportTarget.value=document.fullscreenElement||'body';pin.value='';message.value='';resultStatus.value='';visible.value=true;await nextTick();input.value?.focus()}
function close(){if(pending.value)return;pin.value='';visible.value=false;opener.value?.focus()}
function onKeydown(e){
 if(e.key==='Escape'){e.preventDefault();e.stopPropagation();close()}
 if(e.key==='Tab'){
  const elements=[...dialog.value.querySelectorAll('button:not(:disabled),input:not(:disabled)')];const first=elements[0],last=elements.at(-1)
  if(!first){e.preventDefault();return}
  if(e.shiftKey&&document.activeElement===first){e.preventDefault();last.focus()}
  else if(!e.shiftKey&&document.activeElement===last){e.preventDefault();first.focus()}
 }
}
async function submit(){
 if(pending.value||!validPIN.value)return
 const body=JSON.stringify({pin:pin.value});pin.value='';pending.value=true;message.value='';resultStatus.value=''
 try{
  const response=await fetch(`/api/devices/${encodeURIComponent(props.deviceId)}/unlock`,{method:'POST',headers:{'Content-Type':'application/json',Authorization:`Bearer ${auth.token}`},body,cache:'no-store'})
  const result=await response.json()
  message.value=result.message||result.error||'执行未完成，请检查手机；未自动重试'
  resultStatus.value=response.ok?result.status:'error'
 }catch{message.value='连接中断，解锁结果未知；请检查手机，未自动重试';resultStatus.value='error'}
 finally{pending.value=false;pin.value='';await nextTick();dialog.value?.querySelector('button[type="button"]')?.focus()}
}
onUnmounted(()=>{pin.value=''})
</script>
<style scoped>
.unlock-device-button{display:flex;align-items:center;justify-content:center;gap:7px;min-height:36px;padding:8px 12px;border:1px solid #39556e;border-radius:8px;background:#172a3d;color:#a8d9ff;font:inherit;font-size:12px;cursor:pointer;touch-action:manipulation}.unlock-device-button svg{width:16px;height:16px;flex:none}.unlock-device-button:disabled{opacity:.4;cursor:not-allowed}.unlock-overlay{position:fixed;inset:0;z-index:2147483600;display:flex;align-items:center;justify-content:center;padding:16px;background:#030913bb;box-sizing:border-box}.unlock-dialog{width:100%;max-width:400px;max-height:calc(100dvh - 32px);overflow:auto;box-sizing:border-box;padding:24px;border:1px solid #385267;border-radius:16px;background:#101e2c;color:#e2edf7;box-shadow:0 20px 70px #0008}.unlock-dialog h2{margin:0;font-size:20px}.unlock-device-id{font-size:12px;color:#8ca6bd;overflow-wrap:anywhere}.unlock-description{font-size:13px;line-height:1.6;color:#aebfd1}.unlock-dialog label{display:block;font-size:13px;margin-bottom:8px}.unlock-dialog input{box-sizing:border-box;width:100%;min-height:46px;border:1px solid #456079;border-radius:9px;background:#091422;color:#fff;font-size:16px;padding:10px 12px;letter-spacing:3px}.unlock-result{min-height:24px;font-size:13px;line-height:1.6;color:#ffcd96;overflow-wrap:anywhere}.unlock-result.success{color:#8fe9ca}.unlock-dialog-actions{display:flex;justify-content:flex-end;gap:10px}.unlock-dialog-actions button{min-height:44px;border:1px solid #3c5368;border-radius:9px;padding:10px 18px;background:#223448;color:#e7f0f8;cursor:pointer}.unlock-dialog-actions .unlock-submit{background:#1b5b4c;border-color:#367c69}.unlock-dialog-actions button:disabled{opacity:.4;cursor:not-allowed}
</style>
