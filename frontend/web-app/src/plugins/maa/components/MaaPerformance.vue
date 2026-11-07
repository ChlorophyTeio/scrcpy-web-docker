<template><section class="maa-settings-section maa-performance"><h3>识别性能</h3><p class="maa-section-hint">针对当前服务器 CPU 优化 MAA 推理进程，下次任务生效。</p><p role="status" :class="{error}">{{notice}}</p>
 <template v-if="info"><dl class="maa-performance-info"><dt>处理器</dt><dd>{{info.cpu_model||'未识别'}}</dd><dt>可用逻辑核心</dt><dd>{{info.allowed_cpus?.join(', ')}}</dd><dt>高性能核心</dt><dd>{{info.p_cores?.join(', ')||'未区分，使用默认调度'}}</dd><dt>当前模式</dt><dd>{{info.mode==='performance'?'高性能核心优先':'系统默认调度'}}</dd><dt>MAA 推理可用核心</dt><dd>{{info.effective_cpus?.join(', ')}}</dd></dl>
 <div class="maa-config-actions"><button @click="apply('performance')" :class="{primary:info.mode==='performance'}" :disabled="disabled||busy||!info.p_cores?.length">高性能核心优先</button><button @click="apply('balanced')" :class="{primary:info.mode==='balanced'}" :disabled="disabled||busy">恢复系统默认</button><button @click="load" :disabled="busy">刷新状态</button></div>
 <p class="maa-section-note">仅调整 MAA 任务进程的 CPU 亲和性，不独占核心。保持官方 CPU OCR、原始模型与识别精度；未可靠区分核心时使用默认调度。</p>
 <section v-if="info.benchmark" class="maa-benchmark"><h4>本机模型推理实测</h4><p class="maa-section-note">{{info.benchmark.cpu_model}} · 核心 {{info.benchmark.core}} · ONNX Runtime {{info.benchmark.runtime}} · {{info.benchmark.threads}} 线程 · 每组 3 轮</p><div class="maa-table-scroll"><table><thead><tr><th>模型</th><th>默认调度</th><th>高性能核心</th><th>耗时降低</th></tr></thead><tbody><tr v-for="item in info.benchmark.models" :key="item.name"><td>{{item.name}}</td><td>{{item.before_ms.toFixed(2)}} ms</td><td>{{item.after_ms.toFixed(2)}} ms</td><td>{{((1-item.after_ms/item.before_ms)*100).toFixed(1)}}%</td></tr></tbody></table></div><p class="maa-section-note">采用固定合成输入、同一 MAA 模型和运行库，预热后取中位耗时，输出校验一致。此处是模型推理，不等于完整任务提速；截图传输、游戏动画和服务器负载仍会影响速度。</p></section>
 </template></section></template>
<script setup>
import {ref,onMounted} from 'vue'
defineProps({disabled:Boolean})
const info=ref(null),busy=ref(false),notice=ref(''),error=ref(false)
async function request(body){const r=await fetch('/api/plugins/maa/proxy/api/gui/performance',{method:body?'POST':'GET',headers:{'Content-Type':'application/json'},body:body?JSON.stringify(body):undefined,cache:'no-store'});const d=await r.json();if(!r.ok)throw Error(d.error||'识别配置请求失败');return d}
async function load(){try{info.value=await request()}catch(e){notice.value=e.message;error.value=true}}
async function apply(mode){busy.value=true;error.value=false;try{info.value=await request({mode});notice.value='设置已保存，下次 MAA 任务生效'}catch(e){notice.value=e.message;error.value=true}finally{busy.value=false}}
onMounted(load)
</script>
