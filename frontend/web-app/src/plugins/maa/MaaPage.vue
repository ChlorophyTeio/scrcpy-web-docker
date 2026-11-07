<template>
 <section class="maa-desktop" :class="{'maa-is-running':status.running}">
  <header class="maa-window-title"><span class="maa-app-mark">M</span><strong>MAA</strong><span class="maa-version">{{coreVersion}}</span><span class="maa-title-device">{{connection.device||'USB 手机'}} · {{config?.client==='Bilibili'?'B服':'官服'}}</span><span class="maa-connection" :class="{online:connection.connected}">{{connection.connected?'● USB 已连接':'○ 等待手机'}}</span></header>
  <nav class="maa-tabs" aria-label="MAA 功能"><button v-for="tab in tabs" :key="tab.id" :class="{active:activeTab===tab.id}" @click="activeTab=tab.id">{{tab.name}}</button></nav>
  <div v-if="loading" class="maa-loading" role="status">正在加载 MAA…</div>
  <div v-else-if="!config" class="maa-loading"><p>{{message}}</p><button @click="initialize">重新连接</button></div>
  <template v-else>
   <div class="maa-mobile-views" v-if="activeTab==='daily'"><button v-for="view in [['tasks','任务'],['params','参数'],['logs','日志']]" :class="{active:mobileView===view[0]}" @click="mobileView=view[0]" :key="view[0]">{{view[1]}}</button></div>
   <div class="maa-workspace" v-if="activeTab==='daily'" :data-view="mobileView">
    <aside class="maa-task-pane">
     <div class="maa-task-card"><div class="maa-task-list">
      <div v-for="task in orderedTasks" class="maa-task-row" :class="[{selected:selectedTask===task.id},status.states?.[task.id]]" :key="task.id">
       <label :title="status.task_results?.[task.id]||task.hint"><input type="checkbox" :value="task.id" v-model="config.enabled" :disabled="status.running||busy"/><span>{{task.name}}</span></label>
       <button class="maa-gear" @click="selectTask(task.id)" :aria-label="`${task.name}设置`" :title="task.hint">⚙</button>
      </div>
     </div><div class="maa-task-actions"><button @click="config.enabled=tasks.map(t=>t.id)" :disabled="status.running||busy">全选</button><button @click="config.enabled=[]" :disabled="status.running||busy">清空</button></div></div>
     <div class="maa-after"><label for="maa-after-action">完成后</label><select id="maa-after-action" v-model="config.after" :disabled="status.running||busy"><option value="none">无操作</option><option value="close_game">退出明日方舟</option></select></div>
     <button class="maa-link-start" :class="{stop:status.running}" @click="status.running?stop():start(config.enabled)" :disabled="busy||status.stopping||(!status.running&&!config.enabled.length)">{{status.stopping?'正在停止…':status.running?'停止任务':busy?'请稍候…':'Link Start!'}}</button>
     <p class="maa-task-count"><span class="maa-task-scroll-hint">上下滑动查看任务 · </span>已选择 {{config.enabled.length}} 项任务</p>
    </aside>
    <main class="maa-param-pane">
     <h2>{{currentTask.name}}</h2><p class="maa-section-hint">{{currentTask.hint}}</p>
     <div class="maa-param-tabs" v-if="selectedTask!=='launch'&&selectedTask!=='update'"><button @click="parameterTab='basic'" :class="{active:parameterTab==='basic'}">常规设置</button><button @click="parameterTab='advanced'" :class="{active:parameterTab==='advanced'}">高级设置</button></div>
     <div v-if="parameterTab==='basic'||selectedTask==='update'" class="maa-basic-params">
     <fieldset :disabled="status.running||busy" v-if="selectedTask==='fight'" class="maa-fight-fields">
      <label class="maa-fight-limit" v-for="limit in fightLimits" :key="limit.key"><span><input type="checkbox" :checked="limitOn(limit.key)" @change="fightLimit(fight,limit.key,$event.target.checked)"/>{{limit.label}}</span><input type="number" :value="limitValue(limit.key)" @input="setLimit(limit.key,$event.target.value)" :disabled="!limitOn(limit.key)" :min="1" :max="limit.key==='times'?999999:999"/></label>
      <div class="maa-fight-limit"><label><input type="checkbox" :checked="!!Object.keys(fight.drops||{}).length" @change="toggleDrops($event.target.checked)"/>指定材料</label><select :value="dropId" @change="selectDrop($event.target.value)"><option value="">不选择</option><option v-for="item in materials" :key="item.id" :value="item.id">{{item.name}}</option></select></div>
      <label v-if="dropId" class="maa-value-field"><span>材料数量</span><input type="number" :min="1" :max="99999" v-model.number="fight.drops[dropId]"/></label>
      <label class="maa-value-field"><span>代理倍率</span><select v-model.number="fight.series"><option :value="-1">不切换</option><option :value="0">AUTO</option><option v-for="n in 10" :key="n" :value="n">{{n}} 倍</option></select></label>
      <MaaStagePicker v-model="fight.stage" :client="config.client" :core-version="coreVersion" :disabled="status.running||busy"/>
     </fieldset>
     <MaaParameterFields v-else-if="currentTask.fields" :fields="currentTask.fields" :model="parameters(selectedTask)" :disabled="status.running||busy"/>
     <p v-else class="maa-section-note">选择任务参数后保存或检查配置。</p>
     <p v-if="selectedTask==='launch'" class="maa-section-note">默认包名 com.hypergryph.arknights。填写手机已安装应用的包名；此项仅打开应用，“开始唤醒”继续处理游戏首页与登录。</p><p v-if="selectedTask==='update'" class="maa-section-note">使用 MAA 的干员与仓库识别更新数据。每天/每周按北京时间凌晨 04:00 划分游戏日；程序和资源升级在设置 → 版本与资源。</p>
     <div v-if="selectedTask==='infrast'" class="maa-infrast-upload"><label class="maa-file-picker">上传基建排班 JSON<input type="file" accept=".json,application/json" @change="importInfrast" :disabled="status.running||busy"/></label><label v-if="infrastFiles.length" class="maa-value-field"><span>已上传排班</span><select v-model="parameters('infrast').filename" :disabled="status.running||busy"><option value="">请选择</option><option v-for="file in infrastFiles" :key="file">{{file}}</option></select></label></div>
     </div>
     <div v-else class="maa-advanced-params">
      <MaaParameterFields :fields="advancedFields[selectedTask]||[]" :model="parameters(selectedTask)" :disabled="status.running||busy"/>
      <div v-if="selectedTask==='recruit'" class="maa-fields maa-recruit-time"><label class="maa-value-field" v-for="star in ['3','4']" :key="star"><span>{{star}} 星招募时长（分钟）</span><input type="number" v-model.number="parameters('recruit').recruitment_time[star]" :min="60" :max="540" :step="10" :disabled="status.running||busy"/></label></div>
      <details v-if="['fight','recruit'].includes(selectedTask)" class="maa-report-settings"><summary>数据汇报</summary><p class="maa-section-note">开启后在你执行任务时向对应服务汇报数据；默认关闭。</p><MaaParameterFields :fields="reportingFields" :model="parameters(selectedTask)" :disabled="status.running||busy"/></details>
      <MaaRawParameters :key="selectedTask" :model="parameters(selectedTask)" :disabled="status.running||busy" @apply="applyRawParams"/>
     </div>
     <div class="maa-parameter-bottom"><div class="maa-config-actions"><button @click="save" :disabled="status.running||busy">保存设置</button><button @click="validate(config.enabled)" :disabled="status.running||busy">检查配置</button></div><p>设置不会自动启动任务。作战前请保持手机已解锁，并登录所选游戏客户端。</p></div>
    </main>
    <div class="maa-log-pane"><div class="maa-log-status" :class="{error:messageKind==='error',warning:messageKind==='warning'||status.warnings?.length>0}" role="status"><time>{{clock}}</time><span>{{message||status.message||'就绪，等待开始'}}</span></div><div class="maa-log-actions"><span>实时日志</span><button @click="logs=[]">清空</button><button @click="downloadLogs">导出</button></div><pre ref="logPane" class="maa-log-lines" aria-label="MAA 实时日志">{{logs.join('\n')||'等待运行日志…'}}</pre></div>
   </div>
   <MaaCopilotLibrary v-else-if="activeTab==='copilot'" :config="config" :disabled="status.running||busy" :running="status.running" :logs="logs" @start="start" @validate="validate" @stop="stop"/>
   <div v-else-if="activeTab==='tools'" class="maa-content-page"><h2>小工具</h2><div class="maa-tools-grid">
    <button @click="checkConnection"><strong>连接检测</strong><span>检查 USB、ADB 和游戏安装状态</span></button><button @click="openPhone"><strong>手机远控</strong><span>查看手机画面、解锁和手动操作</span></button><button @click="validate(config.enabled)" :disabled="status.running||busy"><strong>配置检查</strong><span>解析配置，不连接或操作游戏</span></button><button @click="downloadLogs"><strong>导出日志</strong><span>下载当前浏览器收到的日志</span></button><button @click="start(['recruitcalc'])" :disabled="status.running||busy"><strong>公招计算</strong><span>请先进入标签界面；只识别与计算，不选择标签或确认招募</span></button><button @click="start(['depot'])" :disabled="status.running||busy"><strong>仓库识别</strong><span>请先手动进入游戏仓库界面</span></button><button @click="start(['operbox'])" :disabled="status.running||busy"><strong>干员识别</strong><span>请先手动进入游戏干员列表</span></button>
   </div><MaaRecognitionResults :disabled="status.running||busy" :running="status.running"/></div>
   <MaaSchedules v-else-if="activeTab==='schedule'" :config="config"/>
   <MaaSettingsPanel v-else :config="config" :connection="connection" :status="status" :logs="logs" @maintenance-started="maintenanceStarted" @resource-update="resourceUpdate" @stop="stop" :disabled="status.running||busy" @save="save" @check="checkConnection" @import-config="applyImportedConfig"/>
   <footer class="maa-message-bar" role="status" :class="{error:messageKind==='error',warning:messageKind==='warning'||status.warnings?.length>0}">{{message||status.message||'就绪'}}</footer>
  </template>
 </section>
</template>
<script setup>
import {ref,computed,onMounted,onUnmounted,nextTick} from 'vue'
import {useAuthStore} from '@/stores/auth'
import {useDeviceStore} from '@/stores/devices'
import MaaStagePicker from '@/plugins/maa/components/MaaStagePicker.vue'
import MaaParameterFields from '@/plugins/maa/components/MaaParameterFields.vue'
import MaaCopilotLibrary from '@/plugins/maa/components/MaaCopilotLibrary.vue'
import MaaRecognitionResults from '@/plugins/maa/components/MaaRecognitionResults.vue'
import MaaSchedules from '@/plugins/maa/components/MaaSchedules.vue'
import MaaSettingsPanel from '@/plugins/maa/components/MaaSettingsPanel.vue'
import MaaRawParameters from '@/plugins/maa/components/MaaRawParameters.vue'
import {advancedFields,reportingFields,normalizePlan} from '@/plugins/maa/utils/maaSettings'
import {tasks,copilotTask,clonePlan,fightLimit,selectedPlan,taskOrder} from '@/plugins/maa/utils/maaTasks'
const auth=useAuthStore(),deviceStore=useDeviceStore(),tabs=[{id:'daily',name:'一键长草'},{id:'copilot',name:'自动战斗'},{id:'schedule',name:'定时任务'},{id:'tools',name:'小工具'},{id:'settings',name:'设置'}]
const activeTab=ref('daily'),mobileView=ref('tasks'),selectedTask=ref('fight'),config=ref(null),loading=ref(true),busy=ref(false),connection=ref({}),status=ref({states:{}}),logs=ref([]),message=ref(''),messageKind=ref('info'),materials=ref([]),copilotName=ref(''),logPane=ref(null),clock=ref('')
const parameterTab=ref('basic'),infrastFiles=ref([])
const orderedTasks=computed(()=>taskOrder(config.value||{}).map(id=>tasks.find(t=>t.id===id)))
let socket,pollTimer,sessionTimer,clockTimer,disposed=false,reconnectTimer
const currentTask=computed(()=>tasks.find(t=>t.id===selectedTask.value)||tasks[1]),fight=computed(()=>parameters('fight')),coreVersion=computed(()=>connection.value.version?.match(/MaaCore[^\n]*|v6\.[^\s]*/i)?.[0]||'')
const fightLimits=[{key:'medicine',label:'使用药剂'},{key:'stone',label:'使用源石'},{key:'times',label:'指定次数'}]
const dropId=computed(()=>Object.keys(fight.value.drops||{})[0]||'')
function parameters(id){if(!config.value)return {};return config.value.params[id]||(config.value.params[id]={})}
function selectTask(id){selectedTask.value=id;mobileView.value='params';parameterTab.value='basic'}
function limitOn(key){return key==='times'?fight.value[key]!==2147483647&&fight.value[key]>0:fight.value[key]>0}
function limitValue(key){return limitOn(key)?fight.value[key]:(key==='times'?999:key==='medicine'?1:0)}
function setLimit(key,value){fight.value[key]=Math.max(1,Math.min(key==='times'?999999:999,Number(value)||1))}
function toggleDrops(on){if(!on)delete fight.value.drops;else if(materials.value.length)fight.value.drops={[materials.value[0].id]:1}}
function selectDrop(id){fight.value.drops=id?{[id]:1}:{}}
async function request(path,body){const response=await fetch('/api/plugins/maa/proxy/api/gui/'+path,{method:body===undefined?'GET':'POST',headers:{'Content-Type':'application/json'},body:body===undefined?undefined:JSON.stringify(body),cache:'no-store'});const data=await response.json();if(!response.ok)throw new Error(data.error||data.message||'MAA 请求失败');return data}
function notify(text,kind='info'){message.value=text;messageKind.value=kind}
async function session(){const response=await fetch('/api/plugins/maa/session',{method:'POST',headers:{Authorization:`Bearer ${auth.token}`},cache:'no-store'});const data=await response.json();if(!response.ok)throw new Error(data.error||'MAA 服务连接失败')}
async function initialize(){loading.value=true;try{await session();const [saved,run,items]=await Promise.all([request('config'),request('status'),request('catalog')]);config.value=normalizePlan(saved);status.value=run;materials.value=items.sort((a,b)=>a.name.localeCompare(b.name,'zh-CN'));connectLogs();await checkConnection(false);await refreshInfrastFiles()}catch(error){notify(error.message,'error')}finally{loading.value=false}}
async function checkConnection(show=true){try{connection.value=await request('connection?client='+encodeURIComponent(config.value?.client||'Official'));if(show)notify(connection.value.connected?(connection.value.game_installed?'手机与游戏已就绪':'USB 已连接，未检测到所选游戏客户端'):'手机未连接，请检查 USB 和 ADB 授权',connection.value.connected?'info':'error')}catch(error){notify(error.message,'error')}}
async function save(){if(busy.value||status.value.running)return;busy.value=true;try{await request('config',clonePlan(config.value));notify('设置已保存，不会自动启动任务')}catch(error){notify(error.message,'error')}finally{busy.value=false}}
async function validate(enabled){if(busy.value||status.value.running)return;busy.value=true;try{const data=await request('validate',selectedPlan(config.value,enabled));notify(data.message)}catch(error){notify(error.message,'error')}finally{busy.value=false}}
async function start(enabled){if(busy.value||status.value.running)return;busy.value=true;try{const data=await request('start',selectedPlan(config.value,enabled));status.value={running:true,id:data.id,states:{}};notify('任务已提交');await poll()}catch(error){notify(error.message,'error')}finally{busy.value=false}}
function maintenanceStarted(data){status.value={running:true,id:data.id,current:data.current,states:{[data.current]:'pending'}};notify('MaaCore 维护已开始，请查看版本更新状态和日志');poll()}
async function resourceUpdate(){if(busy.value||status.value.running)return;busy.value=true;try{const data=await request('resource-update',{});status.value={running:true,id:data.id,current:'resource_update',states:{resource_update:'pending'},message:'正在准备资源热更新'};notify('资源热更新已开始，MaaCore 版本保持不变');await poll()}catch(e){notify(e.message,'error')}finally{busy.value=false}}
async function stop(){if(busy.value||!status.value.running)return;busy.value=true;try{await request('stop',{id:status.value.id});status.value.stopping=true;notify(status.value.current==='resource_update'?'正在停止资源热更新':'正在停止并恢复手机显示设置')}catch(error){notify(error.message,'error')}finally{busy.value=false}}
async function poll(){if(!config.value||disposed)return;try{const prior=status.value.running;status.value=await request('status');if(prior&&!status.value.running){notify(status.value.message,status.value.warnings?.length?'warning':status.value.message?.includes('失败')?'error':'info');if(['core_update','core_rollback'].includes(status.value.current))checkConnection(false)}}catch(error){notify(error.message,'error')}}
async function importCopilot(event){const file=event.target.files[0];if(!file)return;try{if(file.size>2*1024*1024)throw new Error('作业文件不能超过 2 MB');const json=JSON.parse(await file.text());const data=await request('copilot',json);copilotName.value=file.name+' · '+data.stage_name;notify('作业已导入，尚未执行')}catch(error){notify(error.message||'作业导入失败','error')}finally{event.target.value=''}}
async function refreshInfrastFiles(){try{infrastFiles.value=(await request('infrast')).files||[]}catch{}}
async function importInfrast(event){const file=event.target.files[0];if(!file)return;busy.value=true;try{if(file.size>1900000)throw new Error('排班文件不能超过 1.9 MB');const plan=JSON.parse((await file.text()).replace(/^\uFEFF/,''));const data=await request('infrast',{name:file.name,plan});const p=parameters('infrast');p.filename=data.filename;p.mode=10000;p.plan_index=0;await refreshInfrastFiles();notify(`排班已上传，共 ${data.plans} 个方案；请核对编号并保存`)}catch(e){notify(e.message,'error')}finally{busy.value=false;event.target.value=''}}
function applyRawParams(value){const c=clonePlan(config.value);c.params[selectedTask.value]=value;config.value=normalizePlan(c);notify('核心参数已应用到草稿，请检查并保存')}
function applyImportedConfig(value){config.value=normalizePlan(value);notify('Windows 配置已转换到草稿，请核对后保存')}
function openPhone(){const device=deviceStore.devices.find(d=>d.id===connection.value.device);if(!device){notify('未找到在线手机，请刷新连接状态','error');return}deviceStore.openDevice(device.id);if(innerWidth<=1024)window.dispatchEvent(new CustomEvent('cloudphone-navigate',{detail:'/'}))}
function addLogs(lines){for(const line of lines)logs.value.push(String(line).replace(/\x1b\[[0-9;]*m/g,''));if(logs.value.length>1500)logs.value.splice(0,logs.value.length-1500);nextTick(()=>{if(logPane.value)logPane.value.scrollTop=logPane.value.scrollHeight})}
function connectLogs(){if(disposed)return;socket?.close();const url=new URL('/api/plugins/maa/proxy/ws',location.href);url.protocol=location.protocol==='https:'?'wss:':'ws:';socket=new WebSocket(url);socket.onmessage=e=>{try{const data=JSON.parse(e.data);if(data.history)addLogs(data.history);else if(data.msg)addLogs([data.msg])}catch{}};socket.onclose=()=>{if(!disposed)reconnectTimer=setTimeout(connectLogs,3000)};socket.onerror=()=>{}}
function downloadLogs(){const url=URL.createObjectURL(new Blob([logs.value.join('\n')],{type:'text/plain;charset=utf-8'}));const a=document.createElement('a');a.href=url;a.download='maa-log-'+new Date().toISOString().replace(/[:.]/g,'-')+'.txt';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000)}
onMounted(()=>{clock.value=new Date().toLocaleTimeString('zh-CN',{hour12:false});initialize();pollTimer=setInterval(poll,2000);sessionTimer=setInterval(()=>session().catch(e=>notify(e.message,'error')),20*60*1000);clockTimer=setInterval(()=>clock.value=new Date().toLocaleTimeString('zh-CN',{hour12:false}),1000)})
onUnmounted(()=>{disposed=true;clearInterval(pollTimer);clearInterval(sessionTimer);clearInterval(clockTimer);clearTimeout(reconnectTimer);socket?.close()})
</script>


<style src="./style.css"></style>
