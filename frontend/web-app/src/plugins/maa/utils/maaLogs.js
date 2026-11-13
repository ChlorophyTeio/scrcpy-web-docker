// Modified for the optional-plugins distribution (2026). See NOTICE.
export const MAA_LOG_LIMIT = 1500
const taskNames = ['启动游戏','开始唤醒','理智作战','基建换班','自动公招','信用收支','领取奖励','自动肉鸽','生息演算','更新数据','自动战斗','保全派驻作业','悖论模拟作业','公招计算','仓库识别','干员识别','更新干员数据','更新仓库数据','核心版本更新','核心版本回退','资源热更新','退出游戏']
export const logLevelLabels = {debug:'调试',info:'信息',success:'成功',warning:'警告',error:'错误'}
export function stripLogAnsi(value) { return String(value??'').replace(/\x1b\[[0-?]*[ -/]*[@-~]/g,'').replace(/\r\n/g,'\n').replace(/\r/g,'') }
function unwrapLogrus(raw) {
  const match=raw.match(/(?:^|\s)msg=("(?:\\.|[^"\\])*"|[^\r\n]*)/)
  if(!match)return raw
  if(!match[1].startsWith('"'))return match[1]
  try{return JSON.parse(match[1])}catch{try{return JSON.parse(match[1].replace(/\\(?:x([\da-f]{2})|([abv])|(.))/gi,(all,hex,control)=>hex?'\\u00'+hex:control?({a:'\\u0007',b:'\\b',v:'\\u000b'}[control]):all))}catch{return match[1].slice(1,-1)}}
}
export function parseMaaLog(raw,metadata={},contextTask='') {
  const original=String(raw??metadata.message??'').slice(0,16384).trimEnd()
  let text=stripLogAnsi(metadata.message??unwrapLogrus(original)).replace(/^\[GUI\]\s*/,'').trimEnd()
  const timestamp=metadata.time||original.match(/(?:^|\s)time="([^"]+)"/)?.[1]||text.match(/^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})[^\]]*\]/)?.[1]||''
  const marker=text.match(/^\[\d{4}-\d{2}-\d{2}[^\]]*\s(DEBUG|TRACE|INFO|WARN(?:ING)?|ERROR|FATAL)\]/i)
  if(marker)text=text.slice(marker[0].length).trimStart()
  let level=String(metadata.level||original.match(/(?:^|\s)level=(\w+)/)?.[1]||marker?.[1]||'info').toLowerCase()
  if(['warn','warning'].includes(level))level='warning'
  else if(['fatal','panic','error'].includes(level))level='error'
  else if(['trace','debug'].includes(level))level='debug'
  else level='info'
  const severityText=text.replace(/\b(?:no|0) errors?\b/gi,'').replace(/(?:没有|未发现|无)(?:任何)?(?:错误|异常)/g,'')
  if(marker&&/ERROR|FATAL/i.test(marker[1])||/(?:失败|出错|异常)|\b(?:error|fatal|panic)\b/i.test(severityText))level='error'
  else if(level!=='error'&&(marker&&/WARN/i.test(marker[1])||/(?:警告|注意：|未确认|未能)|\bwarn(?:ing)?\b/i.test(severityText)))level='warning'
  else if(level==='info'&&/(?:全部任务已完成|已确认完成|更新完成|检查通过)|\b(?:Completed|TaskChainCompleted)\b/i.test(text))level='success'
  let task=contextTask
  const start=text.match(/^开始任务[：:]\s*(.+)/)
  const bracket=text.match(/^\[([^\]]+)\]/)?.[1]
  const prefix=taskNames.find(name=>text.startsWith(name+'：')||text.startsWith(name+':'))
  if(start)task=start[1].trim()
  else if(taskNames.includes(bracket))task=bracket
  else if(prefix)task=prefix
  else if(/^\[MAA 更新\]/.test(text))task='核心版本更新'
  else if(/^(?:全部任务|任务执行结束|任务失败|任务已停止|MAA plugin listening)/.test(text))task=''
  return {raw:original,text,time:timestamp,clock:timestamp.match(/(?:T| )(\d{2}:\d{2}:\d{2})/)?.[1]||'',level,task,taskStart:!!start,truncated:!!metadata.truncated||String(raw??'').length>16384,search:[text,original,task,logLevelLabels[level]].join(' ').toLocaleLowerCase()}
}
// KMP over complete log lines: overlap removes replayed history, not repeated live events.
export function historyOverlap(cursor,incoming) {
  if(!cursor.length||!incoming.length)return 0
  const tokens=[...incoming,Symbol('separator'),...cursor],prefix=new Array(tokens.length).fill(0)
  for(let i=1;i<tokens.length;i++){let j=prefix[i-1];while(j&&tokens[i]!==tokens[j])j=prefix[j-1];if(tokens[i]===tokens[j])j++;prefix[i]=j}
  return Math.min(prefix.at(-1),incoming.length)
}
export function createMaaLogStore(limit=MAA_LOG_LIMIT) {
  let next=0,task='',cursor=[],stream='',seen=0,removed=0
  const entries=[]
  const append=(raw,meta={})=>{const row={...parseMaaLog(raw,meta,task),id:++next};task=row.task;entries.push(row);cursor.push(row.raw);if(cursor.length>1500)cursor.splice(0,cursor.length-1500);if(entries.length>limit){removed+=entries.length-limit;entries.splice(0,entries.length-limit)}return row}
  return {entries,get removed(){return removed},clear(){entries.splice(0);removed=0},receive(frame){
    const added=[]
    if(frame.stream_id&&(Array.isArray(frame.entries)||frame.entry)){
      if(stream!==frame.stream_id){stream=frame.stream_id;seen=0;task='';cursor=[]}
      const records=Array.isArray(frame.entries)?frame.entries:[frame.entry]
      records.forEach((record,i)=>{const sequence=Number(record?.id);if(!Number.isSafeInteger(sequence)||sequence<=seen)return;seen=sequence;added.push(append(Array.isArray(frame.history)?frame.history[i]??record.message:frame.msg??record.message,record))})
    }else if(Array.isArray(frame.history)){
      const history=frame.history.map(line=>String(line).slice(0,16384).trimEnd()),overlap=historyOverlap(cursor,history)
      for(const line of history.slice(overlap))added.push(append(line))
    }else if(typeof frame.msg==='string')added.push(append(frame.msg))
    return added
  }}
}
export function filterMaaLogs(entries,{query='',level='all',task='all'}={}) {
  const terms=String(query).trim().toLocaleLowerCase().split(/\s+/).filter(Boolean)
  return entries.filter(row=>(level==='all'||level==='issues'&&['warning','error'].includes(row.level)||row.level===level)&&(task==='all'||row.task===task)&&terms.every(term=>row.search.includes(term)))
}
export function formatMaaLogs(entries,raw=false) {return entries.map(row=>raw?row.raw:[row.time&&`[${row.time}]`,`[${logLevelLabels[row.level]}]`,row.task&&`[${row.task}]`,row.text].filter(Boolean).join(' ')).join('\n')}
