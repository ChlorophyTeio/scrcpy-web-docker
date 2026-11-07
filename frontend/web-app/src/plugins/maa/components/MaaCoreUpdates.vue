<template>
 <section class="maa-core-updates">
  <h3>MaaCore 版本更新</h3><p class="maa-section-hint">检查官方稳定版。升级时同时安装配套基础资源，独立热更新资源继续保留。</p>
  <dl class="maa-core-versions"><div><dt>当前版本</dt><dd>{{state.current||'等待检查'}}</dd></div><div><dt>最新稳定版</dt><dd>{{state.latest||'尚未检查'}}</dd></div><div><dt>上次检查</dt><dd>{{formatTime(state.checked_at)}}</dd></div><div v-if="state.backup_version"><dt>可回退版本</dt><dd>{{state.backup_version}}</dd></div></dl>
  <div class="maa-config-actions"><button @click="check" :disabled="blocked">{{busy==='check'||state.checking?'正在检查…':'手动检查更新'}}</button><button @click="start('start','core_update')" :disabled="blocked||!state.available">立即升级稳定版</button><button v-if="state.backup_version&&state.backup_version!==state.current" @click="start('rollback','core_rollback')" :disabled="blocked">回退到 {{state.backup_version}}</button><button v-if="status.running&&['core_update','core_rollback'].includes(status.current)" @click="$emit('stop')" :disabled="status.stopping">停止维护</button></div>
  <p role="status" class="maa-section-note" :class="{'maa-import-error':error}">{{error||notice||state.message||'点击手动检查更新，只检查版本，不安装。'}}</p>
  <h3>自动升级</h3><label class="maa-check-field"><input type="checkbox" v-model="enabled" :disabled="!!busy||!loaded"/>每天检查并自动升级稳定版</label><label class="maa-value-field"><span>北京时间（UTC+8）</span><input type="time" v-model="updateTime" :disabled="!!busy||!loaded"/></label><p class="maa-section-hint">默认 04:00。仅在有新版时升级；当天遇到任务运行则跳过。升级前备份核心和基础资源，安装或验证失败自动恢复。</p><button @click="save" :disabled="!!busy||!loaded">保存自动更新设置</button>
 </section>
</template>
<script setup>
import {ref,computed,onMounted,onUnmounted} from 'vue'
const props=defineProps({disabled:Boolean,status:{type:Object,default:()=>({})}})
const emit=defineEmits(['started','stop'])
const state=ref({}),enabled=ref(false),updateTime=ref('04:00'),loaded=ref(false),busy=ref(''),error=ref(''),notice=ref('')
const blocked=computed(()=>props.disabled||!!busy.value||!!state.value.checking||!loaded.value)
let timer,disposed=false
async function request(path='',body){const r=await fetch('/api/plugins/maa/proxy/api/gui/core-update'+path,{method:body===undefined?'GET':'POST',headers:{'Content-Type':'application/json'},body:body===undefined?undefined:JSON.stringify(body),cache:'no-store'});const d=await r.json();if(!r.ok)throw new Error(d.error||'版本更新请求失败');return d}
async function refresh(){try{const d=await request();if(disposed)return;if(state.value.message!==d.message)notice.value='';state.value=d;if(!loaded.value){enabled.value=d.enabled;updateTime.value=d.time;loaded.value=true}}catch(e){if(!disposed)error.value=e.message}}
async function check(){if(blocked.value)return;busy.value='check';error.value='';notice.value='';try{state.value=await request('/check',{});notice.value=state.value.message}catch(e){error.value=e.message}finally{busy.value=''}}
async function save(){busy.value='save';error.value='';notice.value='';try{state.value=await request('/settings',{enabled:enabled.value,time:updateTime.value});notice.value='自动更新设置已保存'}catch(e){error.value=e.message}finally{busy.value=''}}
async function start(path,current){if(blocked.value)return;busy.value=path;error.value='';notice.value='';try{const d=await request('/'+path,{});emit('started',{...d,current});await refresh()}catch(e){error.value=e.message}finally{busy.value=''}}
function formatTime(value){return value?new Date(value).toLocaleString('zh-CN',{timeZone:'Asia/Shanghai',hour12:false}):'尚未检查'}
onMounted(()=>{refresh();timer=setInterval(refresh,5000)})
onUnmounted(()=>{disposed=true;clearInterval(timer)})
</script>
