<template>
  <div class="clipboard-overlay" @click.self="$emit('close')">
    <section ref="dialog" class="clipboard-dialog dialog" role="dialog" aria-modal="true" :aria-labelledby="titleId" @keydown="onKeydown">
      <header><div><h3 :id="titleId">{{ mode === 'send' ? '粘贴到手机' : '复制手机文字' }}</h3><p>{{ deviceId }}</p></div><button class="clipboard-close" aria-label="关闭剪贴板" @click="$emit('close')">×</button></header>
      <div class="clipboard-tabs" aria-label="剪贴板方向"><button :class="{active:mode==='send'}" @click="$emit('change-mode','send')">本机 → 手机</button><button :class="{active:mode==='receive'}" @click="$emit('change-mode','receive')">手机 → 本机</button></div>
      <label class="clipboard-label" :for="textId">{{ mode === 'send' ? '要发送的文字' : '手机剪贴板内容' }}<span>{{ text.length.toLocaleString() }} 字符</span></label>
      <textarea :id="textId" ref="textarea" :value="text" :readonly="mode==='receive'" :placeholder="mode==='send'?'粘贴或输入文字，支持中文、多行和表情':'读取后显示手机已复制的文字'" spellcheck="false" @input="$emit('update:text',$event.target.value)" />
      <p class="clipboard-status" :class="{error}" role="status" aria-live="polite">{{ busy ? '正在读取手机剪贴板…' : message || (mode==='send'?'发送前请先在手机中选中输入框。':'先在手机应用中选中文字并复制，再点击读取。') }}</p>
      <div class="clipboard-actions" v-if="mode==='send'"><button @click="$emit('read-local')">读取本机剪贴板</button><button :disabled="!text.length" @click="$emit('send',false)">仅写入剪贴板</button><button class="primary" :disabled="!text.length" @click="$emit('send',true)">粘贴到手机</button></div>
      <div class="clipboard-actions" v-else><button :disabled="busy" @click="$emit('read-device')">{{ busy?'读取中…':'重新读取手机' }}</button><button class="primary" :disabled="busy||!text.length" @click="$emit('copy',textarea)">复制到本机</button></div>
      <label class="clipboard-sync"><input type="checkbox" :checked="autoSync" @change="$emit('update:autoSync',$event.target.checked)" />双向剪贴板联动</label>
      <p class="clipboard-note">{{autoSync?'手机复制后同步到本机；本机复制或切回此页时同步到当前手机。Ctrl+C / Ctrl+V 可直接复制粘贴。':'联动已关闭，可用上方按钮手动传输。'}} 文字仅在当前会话中保留。</p>
      <div class="clipboard-link-footer"><span role="status">{{syncStatus}}</span><button type="button" @click="$emit('sync-now')">授权 / 同步一次</button></div>
    </section>
  </div>
</template>
<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
const props=defineProps({deviceId:String,mode:String,text:{type:String,default:''},message:String,error:Boolean,busy:Boolean,autoSync:Boolean,syncStatus:String})
const emit=defineEmits(['close','change-mode','update:text','read-local','read-device','send','copy','update:autoSync','sync-now'])
const dialog=ref(null),textarea=ref(null),previousFocus=document.activeElement
const titleId='clipboard-title-'+props.deviceId,textId='clipboard-text-'+props.deviceId
function onKeydown(e){
  e.stopPropagation()
  if(e.key==='Escape'){e.preventDefault();emit('close')}
  if((e.ctrlKey||e.metaKey)&&e.key==='Enter'&&props.mode==='send'&&props.text.length){e.preventDefault();emit('send',true)}
  if(e.key==='Tab'){
    const items=[...dialog.value.querySelectorAll('button:not(:disabled),textarea,input')],first=items[0],last=items.at(-1)
    if(e.shiftKey&&document.activeElement===first){e.preventDefault();last?.focus()}
    else if(!e.shiftKey&&document.activeElement===last){e.preventDefault();first?.focus()}
  }
}
onMounted(()=>nextTick(()=>textarea.value?.focus()))
onUnmounted(()=>previousFocus?.isConnected&&previousFocus.focus?.({preventScroll:true}))
</script>
<style scoped>
.clipboard-overlay{position:fixed;inset:0;z-index:10050;display:grid;place-items:center;padding:16px;background:#020811b8;backdrop-filter:blur(6px);box-sizing:border-box}
.clipboard-dialog{width:min(600px,100%);max-height:calc(100dvh - 32px);overflow:auto;box-sizing:border-box;padding:22px;background:#101c2b;color:#dce8f3;border:1px solid #344b63;border-radius:16px;box-shadow:0 20px 80px #0009;font-size:13px}
header{display:flex;align-items:flex-start;justify-content:space-between;gap:12px}h3{margin:0;font-size:18px}header p{margin:6px 0 0;color:#91a8be;font-size:11px;overflow-wrap:anywhere}button{min-height:40px;padding:8px 12px;border:1px solid #3b536a;border-radius:8px;background:#1c2d40;color:inherit;font:inherit;cursor:pointer;touch-action:manipulation}button:hover:not(:disabled){background:#29425b}button:disabled{opacity:.4;cursor:not-allowed}button:focus-visible,textarea:focus-visible,input:focus-visible{outline:2px solid #62c7fa;outline-offset:2px}.clipboard-close{min-width:40px;font-size:22px;padding:2px;background:transparent}.clipboard-tabs{display:flex;gap:6px;padding:16px 0}.clipboard-tabs button{flex:1}.clipboard-tabs .active{background:#183b52;color:#7cd8ff;border-color:#3684aa}.clipboard-label{display:flex;justify-content:space-between;gap:12px;margin:0 0 8px;font-weight:500}.clipboard-label span{color:#8da6bb;font-weight:400;font-size:11px}textarea{display:block;width:100%;min-height:180px;max-height:40dvh;resize:vertical;box-sizing:border-box;border:1px solid #344d63;border-radius:10px;padding:12px;color:inherit;background:#0a1420;font:inherit;line-height:1.6}.clipboard-status{min-height:36px;margin:10px 0;color:#a4c5de;font-size:12px;line-height:1.5;overflow-wrap:anywhere}.clipboard-status.error{color:#ffb4aa}.clipboard-actions{display:flex;flex-wrap:wrap;gap:8px}.primary{background:#16709a;border-color:#3198c0;color:#fff}.clipboard-sync{display:flex;align-items:center;gap:8px;margin-top:20px;line-height:1.5}.clipboard-sync input{width:17px;height:17px;accent-color:#38bdf8}.clipboard-note{font-size:11px;line-height:1.6;color:#8aa3b9;margin:8px 0 0}
.clipboard-link-footer{display:flex;align-items:center;justify-content:space-between;gap:10px;margin-top:10px;color:#a8c8df;font-size:11px;line-height:1.5}.clipboard-link-footer span{flex:1}.clipboard-link-footer button{flex:none;font-size:11px}
@media(max-width:480px){.clipboard-overlay{padding:10px}.clipboard-dialog{padding:16px;max-height:calc(100dvh - 20px)}textarea{min-height:150px}.clipboard-actions button{flex:1;min-width:100px;min-height:44px}}
</style>
