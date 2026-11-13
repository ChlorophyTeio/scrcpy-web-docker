<template>
 <div class="maa-content-page maa-settings-page">
  <h2>设置</h2>
  <nav class="maa-settings-nav" aria-label="设置分组"><button v-for="item in sections" :key="item[0]" @click="section=item[0]" :class="{active:section===item[0]}">{{item[1]}}</button></nav>
  <div v-if="section==='connection'" class="maa-settings-grid">
   <section><h3>手机与游戏</h3><div class="maa-value-field"><span>ADB 连接</span><span>{{connection.adb||'共享 USB ADB'}}</span></div><div class="maa-value-field"><span>设备标识</span><code>{{connection.device||'未检测到'}}</code></div><label class="maa-value-field"><span>游戏客户端</span><select v-model="config.client" :disabled="disabled" @change="$emit('check')"><option value="Official">官服</option><option value="Bilibili">B服</option></select></label><p class="maa-section-hint">{{connection.connected?'USB 已连接':'手机尚未连接'}} · {{connection.game_installed?'已安装所选游戏':'未检测到所选游戏客户端'}}</p><div class="maa-config-actions"><button @click="$emit('check')">检测连接</button><button @click="$emit('save')" :disabled="disabled">保存设置</button></div></section>
   <section><h3>MAA 版本</h3><pre class="maa-version-details">{{connection.version||'正在读取…'}}</pre><p class="maa-section-note">使用当前 Unraid 共享 USB ADB。Windows 模拟器路径和本机 ADB 地址不能用于此服务器。</p><h3>配置参考</h3><a href="https://github.com/MaaAssistantArknights/MaaAssistantArknights/blob/v6.20.1/docs/zh-cn/protocol/integration.md" target="_blank" rel="noopener">官方 MAA 6.20.1 任务参数 ↗</a></section>
  </div>
  <section v-else-if="section==='updates'" class="maa-settings-section"><MaaCoreUpdates :disabled="disabled" :status="status" @started="$emit('maintenance-started',$event)" @stop="$emit('stop')"/><h3>独立资源热更新</h3><p class="maa-section-hint">更新 MAA 的热更新资源，不升级 MaaCore 或 maa-cli，不启动游戏任务。无需手机在线；运行其他任务时不可更新。</p><pre class="maa-version-details">{{connection.version||'正在读取版本…'}}</pre><div class="maa-config-actions"><button @click="$emit('resource-update')" :disabled="disabled">{{status.running&&status.current==='resource_update'?'正在更新资源…':'立即热更新资源'}}</button><button v-if="status.running&&status.current==='resource_update'" @click="$emit('stop')" :disabled="status.stopping">停止热更新</button></div><p class="maa-section-note" role="status">{{status.current==='resource_update'?status.message:'更新结果和执行过程会显示在下方日志中。'}}</p><MaaLogPanel class="embedded-log-panel" :entries="logEntries" :connection="logConnection" @clear="$emit('clear-logs')"/></section>
  <section v-else-if="section==='tasks'" class="maa-settings-section"><h3>运行设置</h3><label class="maa-value-field"><span>运行时长上限（分钟，0 不限制）</span><input type="number" :value="config.timeout_minutes||0" @input="config.timeout_minutes=Number($event.target.value)" :min="0" :max="1440" :disabled="disabled"/></label><p class="maa-section-hint">达到上限会停止当前任务并恢复手机显示设置。不会关闭或重启 Unraid。</p><h3>执行顺序</h3><p class="maa-section-hint">上下移动日常任务调整顺序；启动游戏固定在最前，更新数据固定在最后。未勾选的任务不会执行。</p><ol class="maa-order-list"><li v-for="(task,index) in ordered" :key="task.id"><span>{{task.name}}</span><button @click="move(index,-1)" :disabled="disabled||index<=1||index===ordered.length-1" :aria-label="'上移'+task.name">↑</button><button @click="move(index,1)" :disabled="disabled||index===0||index>=ordered.length-2" :aria-label="'下移'+task.name">↓</button></li></ol><p class="maa-section-note">各任务提供常规设置、高级设置和数据汇报。需要完整协议参数时，可展开“核心参数 JSON”编辑。</p><button @click="$emit('save')" :disabled="disabled">保存设置</button></section>
  <section v-else-if="section==='migration'" class="maa-settings-section">
   <h3>导入 Windows / 网页配置</h3><p class="maa-section-hint">支持官方新版 gui.json（Configurations / TaskQueue）和此网页导出的配置。读取文件只生成预览；应用后仍需点击保存，不会自动开始任务。</p>
   <label class="maa-file-picker">选择配置 JSON<input type="file" accept=".json,application/json" @change="readFile" :disabled="disabled"/></label>
   <label v-if="profiles.length" class="maa-value-field"><span>Windows 配置档案</span><select v-model="profile" @change="previewImport" :disabled="disabled"><option v-for="name in profiles" :key="name">{{name}}</option></select></label>
   <p v-if="error" class="maa-import-error" role="alert">{{error}}</p>
   <div v-if="preview" class="maa-import-preview"><h3>迁移预览</h3><p>{{preview.summary}}</p><p>客户端：{{preview.config.client==='Bilibili'?'B服':'官服'}} · 药剂 {{preview.config.params.fight?.medicine||0}} · 源石 {{preview.config.params.fight?.stone||0}}</p><ul><li v-for="warning in preview.warnings" :key="warning">{{warning}}</li></ul><button @click="applyImport" :disabled="disabled">应用到当前草稿</button></div>
   <h3>配置备份</h3><p class="maa-section-hint">导出当前页面的任务、顺序和参数，包含尚未保存的修改。备份可能包含游戏账号名和数据汇报 ID。</p><div class="maa-config-actions"><button @click="exportConfig" :disabled="disabled">导出网页配置</button><button @click="$emit('save')" :disabled="disabled">保存当前设置</button></div><p role="status" class="maa-section-note">{{notice}}</p>
  </section>
  <MaaPerformance v-else-if="section==='performance'" :disabled="disabled"/>
  <section v-else class="maa-settings-section"><h3>平台适配</h3><p class="maa-section-note">已接入任务参数、任务顺序、配置迁移、基建排班文件与数据汇报设置。Windows 的模拟器启动、关机/休眠、托盘热键与桌面通知属于本机功能；服务器完成后操作提供“无操作”和“退出明日方舟”。Windows 定时计划不会在导入时自动启用。</p></section>
 </div>
</template>
<script setup>
import MaaLogPanel from '@/plugins/maa/components/MaaLogPanel.vue'
import {ref,computed} from 'vue'
import MaaPerformance from '@/plugins/maa/components/MaaPerformance.vue'
import MaaCoreUpdates from '@/plugins/maa/components/MaaCoreUpdates.vue'
import {tasks,taskOrder} from '@/plugins/maa/utils/maaTasks'
import {importMAAConfig,windowsProfiles} from '@/plugins/maa/utils/maaWindowsImport'
const props=defineProps({config:{type:Object,required:true},connection:{type:Object,required:true},disabled:Boolean,status:{type:Object,default:()=>({})},logEntries:{type:Array,default:()=>[]},logConnection:{type:String,default:'connecting'}})
const emit=defineEmits(['clear-logs','save','check','import-config','resource-update','stop','maintenance-started'])
const sections=[['connection','连接与版本'],['updates','版本与资源'],['performance','识别性能'],['tasks','任务与顺序'],['migration','配置迁移'],['platform','平台适配']]
const section=ref('connection'),profiles=ref([]),profile=ref(''),preview=ref(null),error=ref(''),notice=ref('')
let imported
const ordered=computed(()=>taskOrder(props.config).map(id=>tasks.find(t=>t.id===id)))
function move(index,delta){const order=taskOrder(props.config);if(index<=0||index>=order.length-1||index+delta<=0||index+delta>=order.length-1)return;[order[index],order[index+delta]]=[order[index+delta],order[index]];props.config.order=order}
function previewImport(){preview.value=null;error.value='';try{preview.value=importMAAConfig(imported,props.config,profile.value)}catch(e){error.value=e.message}}
async function readFile(event){const file=event.target.files[0];if(!file)return;preview.value=null;error.value='';profiles.value=[];try{if(file.size>1024*1024)throw new Error('配置文件不能超过 1 MB');imported=JSON.parse((await file.text()).replace(/^\uFEFF/,''));profiles.value=windowsProfiles(imported);profile.value=profiles.value.includes(imported.Current)?imported.Current:profiles.value[0]||'';previewImport()}catch(e){error.value=e instanceof SyntaxError?'JSON 格式有误，请检查配置文件是否完整':e.message||'无法读取配置'}finally{event.target.value=''}}
function applyImport(){if(!preview.value||props.disabled)return;emit('import-config',preview.value.config);notice.value='已应用到草稿，请核对任务和参数后保存；尚未执行任务';preview.value=null}
function exportConfig(){const body={format:'maa-web',version:1,source:'MAA 6.20.1',config:props.config};const url=URL.createObjectURL(new Blob([JSON.stringify(body,null,2)],{type:'application/json'}));const a=document.createElement('a');a.href=url;a.download='maa-web-config.json';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000)}
</script>
