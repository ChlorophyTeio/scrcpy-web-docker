<template>
 <div class="maa-content-page maa-schedules">
  <header class="schedule-header"><div><h2>定时任务</h2><p class="maa-section-hint">北京时间 · 关闭浏览器后仍会执行</p></div><div class="maa-config-actions"><button @click="load" :disabled="busy">刷新</button><button class="primary" @click="create">新建计划</button></div></header>
  <p class="schedule-notice">每个计划保存独立的任务配置。到点时手机需已连接、解锁并可进入游戏；若正在执行其他任务，本次跳过。离线期间错过的任务不会补跑。</p>
  <p role="status" :class="{error:failed}" class="schedule-feedback">{{notice}}</p>
  <form v-if="editing" class="schedule-editor" @submit.prevent="save">
   <h3>{{editing.id?'编辑计划':'新建计划'}}</h3>
   <div class="schedule-fields"><label>计划名称<input v-model.trim="editing.name" required maxlength="40" placeholder="例如：早间基建"/></label><label>重复方式<select v-model="editing.frequency"><option value="daily">每天</option><option value="weekly">每周</option><option value="once">仅一次</option></select></label><label>执行时间<input type="time" v-model="editing.time" required/></label><label v-if="editing.frequency==='weekly'">星期<select v-model.number="editing.weekday"><option v-for="(day,i) in days" :key="i" :value="i">{{day}}</option></select></label><label v-if="editing.frequency==='once'">执行日期<input type="date" v-model="editing.date" required/></label></div>
   <label class="schedule-check"><input type="checkbox" v-model="editing.enabled"/>启用此计划</label>
   <label class="schedule-check" v-if="editing.id"><input type="checkbox" v-model="replacePlan"/>用当前“一键长草”的任务与参数替换计划配置</label>
   <p class="maa-section-note">{{editing.id&&!replacePlan?'保留计划已保存的任务与参数':'将保存当前一键长草中的任务与参数，之后修改一键长草不会影响此计划'}}。</p>
   <div class="schedule-task-chips"><span v-for="id in previewPlan.enabled" :key="id">{{taskName(id)}}</span></div>
   <div class="maa-config-actions"><button type="button" @click="editing=null">取消</button><button class="primary" :disabled="busy">{{busy?'保存中…':'保存计划'}}</button></div>
  </form>
  <div v-if="loading" class="schedule-empty" role="status">正在读取计划…</div>
  <div v-else-if="!items.length" class="schedule-empty"><strong>还没有定时任务</strong><p>先在一键长草中选好任务与参数，再创建一个计划。</p><button @click="create">创建第一个计划</button></div>
  <div class="schedule-grid"><article v-for="item in items" :key="item.id" class="schedule-card" :class="{paused:!item.enabled}">
   <div class="schedule-card-heading"><h3>{{item.name}}</h3><span class="schedule-state" :class="{enabled:item.enabled}">{{item.enabled?'已启用':'已停用'}}</span></div>
   <div class="schedule-time">{{item.time}}<span>{{frequencyLabel(item)}}</span></div>
   <div class="schedule-task-chips"><span v-for="id in item.plan.enabled" :key="id">{{taskName(id)}}</span></div>
   <p class="schedule-result">{{item.last_run?item.last_run+' · '+item.result:'尚未触发'}}</p>
   <div class="maa-config-actions"><button :disabled="busy" @click="toggle(item)">{{item.enabled?'停用':'启用'}}</button><button @click="edit(item)">编辑</button><button class="danger" @click="remove(item)" :disabled="busy">删除</button></div>
  </article></div>
  <section v-if="legacy.length" class="schedule-legacy"><h3>已有计划</h3><p class="maa-section-note">原计划与任务参数均保留，可在这里修改时间、启停或删除。新建计划使用上方的新版配置。</p><article v-for="item in legacy" :key="item.hash" class="schedule-card">
   <div class="schedule-card-heading"><h3>{{item.alias}}</h3><span class="schedule-state" :class="{enabled:item.isEnable}">{{item.isEnable?'已启用':'已停用'}}</span></div>
   <div class="schedule-fields"><label>重复<select v-model="item.type"><option value="day">每天</option><option value="week">每周</option><option value="month">每月</option><option value="custom">单次</option></select></label><label>下次执行（北京时间）<input type="datetime-local" v-model="item.localTime" required/></label></div>
   <p class="maa-section-note">{{item.tasks.map(taskName).join(' · ')}}</p><div class="maa-config-actions"><button @click="saveLegacy(item)" :disabled="busy">保存时间</button><button @click="saveLegacy(item,!item.isEnable)" :disabled="busy">{{item.isEnable?'停用':'启用'}}</button><button class="danger" @click="deleteLegacy(item)" :disabled="busy">删除</button></div>
  </article></section>
 </div>
</template>
<script setup>
import {ref,computed,onMounted} from 'vue'
import {tasks,clonePlan} from '@/plugins/maa/utils/maaTasks'
const props=defineProps({config:{type:Object,required:true}})
const items=ref([]),legacy=ref([]),loading=ref(true),busy=ref(false),notice=ref(''),failed=ref(false),editing=ref(null),replacePlan=ref(false)
const days=['星期日','星期一','星期二','星期三','星期四','星期五','星期六']
const previewPlan=computed(()=>editing.value?.id&&!replacePlan.value?editing.value.plan:props.config)
function taskName(id){return tasks.find(t=>t.id===id)?.name||id}
function frequencyLabel(s){return s.frequency==='daily'?'每天':s.frequency==='weekly'?'每周'+days[s.weekday].slice(2):s.date+' · 单次'}
function localTime(v){return new Date(new Date(v).getTime()+8*3600000).toISOString().slice(0,16)}
async function request(path,body){const r=await fetch('/api/plugins/maa/proxy/api/'+path,{method:body===undefined?'GET':'POST',headers:{'Content-Type':'application/json'},body:body===undefined?undefined:JSON.stringify(body)});const data=await r.json();if(!r.ok||data.code)throw Error(data.error||data.msg||'保存失败');return data}
async function load(){loading.value=true;try{const [a,b]=await Promise.all([request('gui/schedules'),request('GetClusters')]);items.value=a.items||[];legacy.value=Object.values(b.data).flat().map(v=>({...v,localTime:localTime(v.time)}))}catch(e){notice.value=e.message;failed.value=true}finally{loading.value=false}}
function create(){editing.value={name:'',enabled:false,frequency:'daily',time:'08:00',weekday:1,date:localTime(Date.now()).slice(0,10)};replacePlan.value=true}
function edit(v){editing.value=JSON.parse(JSON.stringify(v));replacePlan.value=false}
async function action(fn,message){if(busy.value)return;busy.value=true;failed.value=false;try{await fn();notice.value=message;await load()}catch(e){notice.value=e.message;failed.value=true}finally{busy.value=false}}
async function save(){const value={...editing.value,plan:clonePlan(previewPlan.value)};await action(async()=>{await request('gui/schedules',value);editing.value=null},'计划已保存')}
function toggle(item){return action(()=>request('gui/schedules',{...item,enabled:!item.enabled}),item.enabled?'计划已停用':'计划已启用')}
function remove(item){if(confirm(`删除计划“${item.name}”？`))action(()=>request('gui/schedules/delete',{id:item.id}),'计划已删除')}
function saveLegacy(item,enabled=item.isEnable){action(()=>{const date=new Date(item.localTime+':00+08:00');if(!Number.isFinite(date.getTime())||enabled&&date.getTime()<=Date.now())throw Error('请将下次执行时间设为未来时间');return request('ChangeCluster',{ApiType:'ModifyCluster',NewTaskCluster:{...item,time:date.toISOString(),isEnable:enabled}})},'原计划已保存')}
function deleteLegacy(item){if(confirm(`删除原计划“${item.alias}”及其任务配置？`))action(()=>request('ChangeCluster',{ApiType:'DeleteCluster',NewTaskCluster:item}),'原计划已删除')}
onMounted(load)
</script>
