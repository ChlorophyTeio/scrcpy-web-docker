import test from 'node:test'
import assert from 'node:assert/strict'
import {parseMaaLog,createMaaLogStore,filterMaaLogs,formatMaaLogs,historyOverlap} from './maaLogs.js'

test('logrus envelope and CLI severity are decoded without losing original text',()=>{
 const raw='time="2026-11-13T17:53:28+08:00" level=info msg="[GUI] [2026-11-13 17:53:28 ERROR] Award Error"'
 const row=parseMaaLog(raw,{},'领取奖励')
 assert.equal(row.level,'error');assert.equal(row.text,'Award Error');assert.equal(row.clock,'17:53:28');assert.equal(row.task,'领取奖励');assert.equal(row.raw,raw)
 assert.equal(parseMaaLog('time="2026-11-13T10:00:00Z" level=warning msg="[GUI] 未确认购买成功"').level,'warning')
 assert.equal(parseMaaLog('无错误，0 errors').level,'info')
})
test('escaped ANSI, newlines and Windows paths are readable',()=>{
 const raw=String.raw`time="2026-11-13T10:00:00Z" level=info msg="[GUI] \x1b[31m失败\x1b[0m\nC:\\x86\\file"`
 const row=parseMaaLog(raw);assert.equal(row.text,'失败\nC:\\x86\\file');assert.equal(row.level,'error')
 assert.equal(parseMaaLog('info', {message:'<script>alert(1)</script>'}).text,'<script>alert(1)</script>')
})
test('task context follows boundaries and explicit task summaries',()=>{
 const store=createMaaLogStore();store.receive({history:['[GUI] 开始任务：信用收支','[GUI] [信用收支] Completed','[GUI] 全部任务已完成']})
 assert.equal(store.entries[0].taskStart,true);assert.equal(store.entries[1].task,'信用收支');assert.equal(store.entries[1].level,'success');assert.equal(store.entries[2].task,'')
})
test('sequenced replay is idempotent while identical messages remain distinct',()=>{
 const store=createMaaLogStore();const entry=id=>({id,message:'同一条内容',level:'info',time:'2026-11-13T12:00:00Z'})
 store.receive({stream_id:'a',entries:[entry(1),entry(2)],history:['same','same']});store.receive({stream_id:'a',entry:entry(3),msg:'same'})
 store.receive({stream_id:'a',entries:[entry(1),entry(2),entry(3)]});assert.equal(store.entries.length,3)
 store.clear();store.receive({stream_id:'a',entries:[entry(1),entry(2),entry(3)]});assert.equal(store.entries.length,0)
 store.receive({stream_id:'a',entry:entry(4)});assert.equal(store.entries.length,1)
 store.receive({stream_id:'b',entries:[entry(1)]});assert.equal(store.entries.length,2)
})
test('legacy overlap handles reconnect, local clear and repeated live lines',()=>{
 assert.equal(historyOverlap(['a','b','c'],['b','c','d']),2);assert.equal(historyOverlap(['x','x'],['x','x','y']),2)
 const store=createMaaLogStore();store.receive({history:['a','b']});store.receive({msg:'b'});store.receive({history:['a','b','b','c']})
 assert.deepEqual(store.entries.map(x=>x.raw),['a','b','b','c']);store.clear();store.receive({history:['a','b','b','c']});assert.equal(store.entries.length,0)
})
test('bounded retention has stable increasing ids and cleared records stay cleared',()=>{
 const store=createMaaLogStore(3);for(let i=0;i<5;i++)store.receive({msg:'line '+i})
 assert.deepEqual(store.entries.map(x=>x.id),[3,4,5]);assert.equal(store.removed,2);store.clear();store.receive({msg:'new'});assert.equal(store.entries[0].id,6)
})
test('search, task and issue filters compose and exports preserve diagnostics',()=>{
 const entries=[parseMaaLog('[GUI] 开始任务：信用收支'),parseMaaLog('[GUI] 购买确认失败',{},'信用收支'),parseMaaLog('[GUI] warning: skipped',{},'领取奖励')]
 assert.equal(filterMaaLogs(entries,{query:'信用收支 失败',level:'issues'}).length,1)
 assert.equal(filterMaaLogs(entries,{task:'领取奖励',level:'warning'}).length,1)
 assert.equal(filterMaaLogs(entries,{query:'不存在'}).length,0)
 assert(formatMaaLogs(entries).includes('[错误]'));assert.equal(formatMaaLogs(entries,true).split('\n')[1],'[GUI] 购买确认失败')
})
