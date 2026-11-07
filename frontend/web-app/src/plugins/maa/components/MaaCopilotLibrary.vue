<template>
 <div class="maa-content-page maa-copilot-library">
  <h2>自动战斗作业</h2><p class="maa-section-hint">普通作战、保全派驻、悖论模拟分别保存作业，导入后不会自动执行。</p>
  <nav class="maa-config-actions" aria-label="作业类型"><button v-for="option in modes" :key="option.id" :disabled="busy" @click="kind=option.id" :class="{primary:kind===option.id}">{{option.label}}</button></nav>
  <div class="maa-enhancement-grid"><section class="maa-enhancement-card">
   <h3>{{mode.label}}</h3><p class="maa-section-note">{{mode.hint}}</p>
   <div class="maa-config-actions"><label class="maa-file-picker">导入 JSON<input type="file" accept=".json,application/json" @change="upload" :disabled="disabled||busy"/></label><button @click="exportFile" :disabled="!stored.present">导出作业</button><button @click="remove" class="danger" :disabled="disabled||busy||!stored.present">删除作业</button></div>
   <div v-if="stored.present" class="maa-copilot-summary"><strong>{{stored.stage_name}}</strong><p>{{stored.document?.doc?.title||'已保存作业'}}</p><p>最低核心版本：{{stored.document?.minimum_required||'作业未注明'}}</p><p>{{kind==='ssscopilot'?'分层数：'+(stored.document?.stages?.length||0):'动作数：'+(stored.document?.actions?.length||0)}}</p></div><p v-else class="maa-section-note">尚未导入这一类型的作业。</p>
   <MaaParameterFields :fields="fields" :model="parameters" :disabled="disabled||busy"/>
   <MaaParameterFields v-if="kind==='copilot'" :fields="advancedFields.copilot" :model="parameters" :disabled="disabled||busy"/>
   <div class="maa-config-actions"><button @click="$emit('validate',[kind])" :disabled="disabled||!stored.present">检查配置</button><button @click="$emit('start',[kind])" class="primary" :disabled="disabled||busy||!stored.present">开始{{mode.label}}</button><button v-if="running" @click="$emit('stop')">停止任务</button></div>
   <p role="status" :class="{error}">{{notice}}</p>
  </section><section class="maa-enhancement-card"><h3>运行日志</h3><pre class="maa-result-output">{{logs.join('\n')||'等待执行作业…'}}</pre></section></div>
 </div>
</template>
<script setup>
import {ref,computed,watch} from 'vue'
import MaaParameterFields from '@/plugins/maa/components/MaaParameterFields.vue'
import {copilotTask} from '@/plugins/maa/utils/maaTasks'
import {advancedFields} from '@/plugins/maa/utils/maaSettings'
const props=defineProps({config:Object,disabled:Boolean,running:Boolean,logs:{type:Array,default:()=>[]}})
defineEmits(['start','validate','stop'])
const modes=[{id:'copilot',label:'普通作战',hint:'请按作业要求设置编队；自动编队会修改当前游戏编队。'},{id:'ssscopilot',label:'保全派驻',hint:'导入 type 为 SSS 的分层作业，请先进入对应保全地图。'},{id:'paradoxcopilot',label:'悖论模拟',hint:'导入对应干员的悖论作业，请先确认模拟关卡已经解锁。'}]
const kind=ref('copilot'),stored=ref({}),busy=ref(false),notice=ref(''),error=ref(false)
const mode=computed(()=>modes.find(m=>m.id===kind.value)),parameters=computed(()=>props.config.params[kind.value]||(props.config.params[kind.value]=kind.value==='ssscopilot'?{loop_times:1}:{}))
const fields=computed(()=>kind.value==='copilot'?copilotTask.fields:kind.value==='ssscopilot'?[['loop_times','循环次数','number',1,999]]:[])
let version=0
async function request(method='GET',body){const r=await fetch('/api/plugins/maa/proxy/api/gui/copilot?kind='+kind.value,{method,headers:{'Content-Type':'application/json'},body:body===undefined?undefined:JSON.stringify(body),cache:'no-store'});const d=await r.json();if(!r.ok)throw Error(d.error||'作业请求失败');return d}
async function load(){const v=++version;stored.value={};notice.value='';try{const d=await request();if(v===version)stored.value=d}catch(e){if(v===version){notice.value=e.message;error.value=true}}}
async function upload(e){const f=e.target.files[0];if(!f)return;busy.value=true;error.value=false;try{if(f.size>1900000)throw Error('作业不能超过 1.9 MB');const d=JSON.parse((await f.text()).replace(/^\uFEFF/,''));await request('POST',d);await load();notice.value='作业已保存，尚未运行'}catch(e){error.value=true;notice.value=e.message}finally{busy.value=false;e.target.value=''}}
function exportFile(){const url=URL.createObjectURL(new Blob([JSON.stringify(stored.value.document,null,2)],{type:'application/json'}));const a=document.createElement('a');a.href=url;a.download=kind.value+'-plan.json';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000)}
async function remove(){if(!confirm('删除当前类型的已保存作业？'))return;busy.value=true;try{await request('DELETE');await load();notice.value='作业已删除'}catch(e){notice.value=e.message;error.value=true}finally{busy.value=false}}
watch(kind,load,{immediate:true})
</script>
