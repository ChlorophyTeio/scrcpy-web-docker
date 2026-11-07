<template><section class="maa-recognition-results"><header class="schedule-header"><div><h3>最近识别结果</h3><p class="maa-section-note">保留最近 10 次公招计算、仓库及干员识别输出，刷新网页后仍可查看。</p></div><div class="maa-config-actions"><button @click="load">刷新结果</button><button @click="clear" :disabled="disabled||!items.length">清空记录</button><button @click="download" :disabled="!items.length">导出记录</button></div></header><p role="status">{{notice}}</p><p v-if="!items.length" class="schedule-empty">暂无记录。识别结束后在这里查看结果。</p><details v-for="item in items" :key="item.id" class="maa-enhancement-card"><summary>{{new Date(item.time).toLocaleString('zh-CN')}} · {{item.tasks.map(name).join('、')}} · {{item.success?'完成':'未完成'}}</summary><pre class="maa-result-output">{{item.output.join('\n')}}</pre></details></section></template>
<script setup>
import {ref,onMounted,watch} from 'vue'
const props=defineProps({disabled:Boolean,running:Boolean})
const items=ref([]),notice=ref('')
function name(t){return({recruitcalc:'公招计算',depot:'仓库识别',operbox:'干员识别',update:'更新数据'})[t]||t}
async function load(){try{const r=await fetch('/api/plugins/maa/proxy/api/gui/results',{cache:'no-store'});const d=await r.json();if(!r.ok)throw Error(d.error||'读取失败');items.value=d.items||[]}catch(e){notice.value=e.message}}
async function clear(){if(!confirm('清空服务器保存的识别结果？'))return;const r=await fetch('/api/plugins/maa/proxy/api/gui/results',{method:'DELETE'});if(r.ok){await load();notice.value='已清空记录'}else{notice.value=(await r.json()).error||'清空失败'}}
function download(){const u=URL.createObjectURL(new Blob([JSON.stringify(items.value,null,2)],{type:'application/json'}));const a=document.createElement('a');a.href=u;a.download='maa-recognition-results.json';a.click();setTimeout(()=>URL.revokeObjectURL(u),1000)}
watch(()=>props.running,(now,before)=>{if(before&&!now)load()});onMounted(load)
</script>
