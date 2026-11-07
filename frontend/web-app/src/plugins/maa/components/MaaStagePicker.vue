<template><section class="maa-stage-picker" aria-label="关卡选择">
 <div class="maa-stage-current"><div><strong>关卡选择</strong><span>{{selected?.label||modelValue||'当前 / 上次关卡'}}</span></div><button type="button" @click="load(true)" :disabled="busy||disabled">{{busy?'更新中…':'刷新关卡'}}</button></div>
 <p v-if="selected" class="maa-section-note">{{selected.detail}}<span v-if="selected.cost"> · {{selected.cost}} 理智</span><span v-if="!selected.open"> · 今日未开放</span></p>
 <div class="maa-stage-filters"><input v-model="search" placeholder="搜索关卡、材料或活动名称" aria-label="搜索关卡" :disabled="disabled"/><select v-if="group==='main'" v-model="chapter" aria-label="主线章节" :disabled="disabled"><option value="">全部章节</option><option v-for="n in chapters" :key="n" :value="String(n)">第 {{n}} 章</option></select></div>
 <nav class="maa-stage-groups" aria-label="关卡分类"><button v-for="g in groups" :key="g.id" @click="group=g.id;chapter=''" :class="{active:group===g.id}" type="button">{{g.label}}</button></nav>
 <p v-if="notice" role="status" :class="{error}">{{notice}}</p>
 <div class="maa-stage-list" role="group" aria-label="可选关卡">
  <div v-for="item in filtered.slice(0,100)" :key="item.code" class="maa-stage-option" :class="{selected:modelValue===item.code,closed:!item.open}"><button type="button" class="maa-stage-select" :disabled="disabled||needsCore(item)" @click="choose(item)"><strong>{{item.label}}</strong><small>{{item.detail||item.note||'常驻关卡'}}</small><span v-if="!item.open">今日关闭 · {{item.note}}</span><span v-else-if="needsCore(item)">需要核心 {{item.minimum}}</span><span v-else-if="item.cost">{{item.cost}} 理智</span></button><button type="button" class="maa-stage-favorite" :aria-label="(favorites.includes(item.code)?'取消收藏 ':'收藏 ')+item.label" @click="favorite(item.code)">{{favorites.includes(item.code)?'★':'☆'}}</button></div>
  <p v-if="!busy&&!filtered.length" class="maa-section-note">没有匹配关卡，可切换分类或在下方填写自定义关卡。</p>
  <p v-if="filtered.length>100" class="maa-section-note">还有 {{filtered.length-100}} 项，请搜索或按章节筛选。</p>
 </div>
 <label class="maa-stage-custom">自定义关卡<input :value="modelValue" @change="manual($event.target.value)" placeholder="例如 1-7、TL-9；留空为当前关卡" :disabled="disabled" aria-label="自定义关卡"/></label>
 <p v-if="modelValue&&!selected" class="maa-section-note">此关卡不在当前列表中；请确认 MAA 支持自动导航，且已在游戏中解锁并配置代理。</p>
 <p class="maa-stage-note">{{note}}<span v-if="updated"> · 活动数据 {{new Date(updated).toLocaleDateString('zh-CN')}}</span></p>
</section></template>
<script setup>
import {ref,computed,watch,onMounted} from 'vue'
import {normalizeStageCode,stageVersionOlder} from '@/plugins/maa/utils/maaStages'
const props=defineProps({modelValue:{type:String,default:''},client:{type:String,default:'Official'},coreVersion:{type:String,default:''},disabled:Boolean})
const emit=defineEmits(['update:modelValue'])
const groups=[{id:'common',label:'常用 / 收藏'},{id:'activity',label:'当前活动'},{id:'main',label:'主线'},{id:'supplies',label:'资源本'},{id:'chips',label:'芯片'},{id:'annihilation',label:'剿灭'},{id:'side',label:'别传'}]
const items=ref([]),group=ref('common'),chapter=ref(''),search=ref(''),busy=ref(false),notice=ref(''),error=ref(false),note=ref(''),updated=ref(''),favorites=ref([])
try{const saved=JSON.parse(localStorage.getItem('maa-stage-favorites')||'[]');if(Array.isArray(saved))favorites.value=saved.filter(v=>typeof v==='string').slice(0,40)}catch{}
const selected=computed(()=>items.value.find(s=>s.code===props.modelValue)),chapters=computed(()=>[...new Set(items.value.filter(s=>s.category==='main').map(s=>s.chapter))].sort((a,b)=>a-b))
const popular=['','1-7','CE-6','LS-6','AP-5','CA-5','SK-5','Annihilation']
const filtered=computed(()=>{const q=search.value.trim().toLowerCase();return items.value.filter(s=>(q||group.value==='common'?[...popular,...favorites.value].includes(s.code)||!!q:s.category===group.value)&&(!chapter.value||q||String(s.chapter)===chapter.value)&&(!q||([s.code,s.label,s.detail,s.note].some(v=>v?.toLowerCase().includes(q))||s.code.toLowerCase().replace(/[-\s]/g,'')===q.replace(/[-\s]/g,''))))})
function needsCore(item){return item.minimum&&stageVersionOlder(props.coreVersion,item.minimum)}
function choose(item){emit('update:modelValue',item.code);notice.value=!item.open?'已选择 '+item.label+'；按常规开放日今日关闭，可用于后续计划，执行前请确认游戏实际开放':item.code?'已选择 '+item.label+'，请保存设置':'将使用游戏当前 / 上次关卡，请保存设置';error.value=false}
function manual(value){emit('update:modelValue',normalizeStageCode(value))}
function favorite(code){favorites.value=favorites.value.includes(code)?favorites.value.filter(v=>v!==code):[...favorites.value,code].slice(-40);try{localStorage.setItem('maa-stage-favorites',JSON.stringify(favorites.value))}catch{}}
let generation=0
async function load(refresh=false){const g=++generation;busy.value=true;error.value=false;try{const r=await fetch('/api/plugins/maa/proxy/api/gui/stages'+(refresh?'/refresh':'')+'?client='+encodeURIComponent(props.client),{method:refresh?'POST':'GET',cache:'no-store'});const d=await r.json();if(g!==generation)return;if(!r.ok)throw Error(d.error||'读取关卡失败');items.value=d.items||[];note.value=d.note;updated.value=d.updated_at;if(refresh)notice.value='关卡数据已刷新，当前选择保持不变'}catch(e){if(g===generation){notice.value=e.message;error.value=true}}finally{if(g===generation)busy.value=false}}
watch(()=>props.client,()=>load());onMounted(()=>load())
</script>
