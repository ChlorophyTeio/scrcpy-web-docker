import test from 'node:test'
import assert from 'node:assert/strict'
import {importMAAConfig,windowsProfiles} from './maaWindowsImport.js'
import {normalizePlan} from './maaSettings.js'
import {selectedPlan} from './maaTasks.js'
const base=()=>({enabled:['startup','fight'],client:'Official',after:'none',params:{fight:{medicine:0,stone:0,times:2147483647,stage:'1-7'},infrast:{mode:0},recruit:{select:[4,5],confirm:[3,4]}}})
test('old web settings gain missing fields without changing user choices',()=>{
 const old=base();old.params.recruit.refresh=false
 const c=normalizePlan(old)
 assert.equal(c.params.recruit.refresh,false);assert.deepEqual(c.params.recruit.recruitment_time,{'3':540,'4':540})
 c.params.fight.stone=5;assert.equal(old.params.fight.stone,0)
})
test('Windows profiles map fields, selected tasks, percent threshold and runtime client',()=>{
 const raw={Current:'phone',Configurations:{phone:{Gui:{RuntimeSettings:{ClientType:'Bilibili',StartGame:false,EnableRunDurationLimit:true,RunDurationLimitMinutes:90}},TaskQueue:[
  {$type:'RecruitTask',IsEnable:true,MaxTimes:7,Level3Choose:true,Level4Choose:true,Level5Choose:false,Level3Time:460,PreserveTagEnabled:true,PreserveTagList:['支援机械']},
  {$type:'FightTask',IsEnable:false,UseMedicine:false,MedicineCount:20,UseStone:true,StoneCount:3,UseStoneAllowSave:false,UseExpiringMedicine:true,MedicineExpireDays:2,StagePlan:['CE-6','1-7']},
  {$type:'InfrastTask',Mode:'Normal',DormThreshold:25,RoomList:[{Room:'Mfg'},{Room:'Trade',IsEnabled:false}]},
 ]}}}
 assert.deepEqual(windowsProfiles(raw),['phone'])
 const {config,warnings}=importMAAConfig(raw,base())
 assert.equal(config.client,'Bilibili');assert.deepEqual(config.enabled,['recruit','infrast'])
 assert.equal(config.timeout_minutes,90);assert.equal(config.params.startup.start_game_enabled,false)
 assert.equal(config.params.fight.medicine,0);assert.equal(config.params.fight.stone,0);assert.equal(config.params.fight.stage,'CE-6')
 assert.equal(config.params.infrast.threshold,0.25);assert.deepEqual(config.params.infrast.facility,['Mfg'])
 assert.equal(config.params.recruit.recruitment_time['3'],460);assert.deepEqual(config.params.recruit.confirm,[3,4])
 assert.ok(warnings.some(s=>s.includes('备选关卡')));assert.ok(warnings.some(s=>s.includes('碎石')))
 assert.deepEqual(selectedPlan(config,config.enabled).enabled,['recruit','infrast'])
})
test('custom base file needs separate upload and duplicate or unsupported tasks are reported',()=>{
 const {config,warnings}=importMAAConfig({TaskQueue:[{$type:'InfrastTask',Mode:'Custom',Filename:'C:\\MAA\\base.json',PlanSelect:2},{$type:'FightTask'},{$type:'FightTask'},{$type:'CustomTask'}]},base())
 assert.equal(config.params.infrast.filename,'base.json');assert.equal(config.params.infrast.plan_index,2)
 assert.ok(warnings.some(s=>s.includes('上传')));assert.ok(warnings.some(s=>s.includes('重复')));assert.ok(warnings.some(s=>s.includes('Custom')))
})
test('web backup round trip is lossless and bad formats are rejected',()=>{
 const c=normalizePlan(base());c.order=['fight','startup']
 assert.deepEqual(importMAAConfig({format:'maa-web',version:1,config:c},base()).config,c)
 assert.throws(()=>importMAAConfig({format:'maa-web',version:9},base()))
 assert.throws(()=>importMAAConfig({random:'file'},base()))
})
test('Windows UserDataUpdate imports recognition switches and interval',()=>{
 const result=importMAAConfig({TaskQueue:[{$type:'StartUpTask'},{$type:'UserDataUpdateTask',UpdateOperBox:true,UpdateDepot:false,TriggerInterval:'Daily'}]},base())
 assert.deepEqual(result.config.enabled,['launch','startup','update']);assert.equal(result.config.params.update.depot,false);assert.equal(result.config.params.update.interval,'daily')
 assert.ok(!result.warnings.some(s=>s.includes('UserDataUpdate')))
})
