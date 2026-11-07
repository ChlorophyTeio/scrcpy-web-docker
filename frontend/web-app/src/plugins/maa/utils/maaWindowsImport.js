import {normalizePlan} from './maaSettings.js'
const object=v=>v!==null&&typeof v==='object'&&!Array.isArray(v)
const split=v=>Array.isArray(v)?v:typeof v==='string'?v.split(/[,，;；\n]/).map(s=>s.trim()).filter(Boolean):[]
const ids={StartUp:'startup',Fight:'fight',Infrast:'infrast',Recruit:'recruit',Mall:'mall',Award:'award',Roguelike:'rogue',Reclamation:'reclamation',Copilot:'copilot',Depot:'depot',OperBox:'operbox',UserDataUpdate:'update'}
const maps={
 update:{UpdateOperBox:'operbox',UpdateDepot:'depot'},
 startup:{AccountName:'account_name'},
 fight:{Series:'series',IsDrGrandet:'DrGrandet'},
 infrast:{UsesOfDrones:'drones',DormTrustEnabled:'dorm_trust_enabled',OriginiumShardAutoReplenishment:'replenish',DormFilterNotStationed:'dorm_notstationed_enabled',ReceptionMessageBoard:'reception_message_board',ReceptionClueExchange:'reception_clue_exchange',SendClue:'reception_send_clue',ContinueTraining:'continue_training',FiammettaRecoveryEnabled:'fiammetta_recovery_enabled',UsePinusSylvestris:'use_pinus_sylvestris',UsePerceptionInformation:'use_perception_information',UseWorldlyPlight:'use_worldly_plight',UseAbyssalHunter:'use_abyssal_hunter'},
 recruit:{MaxTimes:'times',RefreshLevel3:'refresh',ExtraTagMode:'extra_tags_mode'},
 mall:{VisitFriends:'visit_friends',Shopping:'shopping',CreditFight:'credit_fight',CreditFightFormation:'formation_index',ShoppingIgnoreBlackListWhenFull:'force_shopping_if_credit_full',OnlyBuyDiscount:'only_buy_discount',ReserveMaxCredit:'reserve_max_credit'},
 award:{Award:'award',Mail:'mail',FreeGacha:'recruit',Orundum:'orundum',Mining:'mining',SpecialAccess:'specialaccess',SignInEvent:'signinevent'},
 rogue:{StartCount:'starts_count',Squad:'squad',Roles:'roles',UseSupportNonFriend:'use_nonfriend_support',Investment:'investment_enabled',InvestCount:'investments_count',InvestWithMoreScore:'investment_with_more_score',StopWhenDepositFull:'stop_when_investment_full',StopAtFinalBoss:'stop_at_final_boss',StopWhenLevelMax:'stop_at_max_level',RefreshTraderWithDice:'refresh_trader_with_dice',MonthlySquadAutoIterate:'monthly_squad_auto_iterate',MonthlySquadCheckComms:'monthly_squad_check_comms',DeepExplorationAutoIterate:'deep_exploration_auto_iterate'},
}
export function windowsProfiles(raw){
 if(!object(raw))throw new Error('配置文件必须是 JSON 对象')
 if(object(raw.Configurations))return Object.keys(raw.Configurations)
 return []
}
export function importMAAConfig(raw,current,profileName){
 if(!object(raw))throw new Error('配置文件必须是 JSON 对象')
 if(raw.format==='maa-web'||(Array.isArray(raw.enabled)&&object(raw.params))){
  if(raw.format==='maa-web'&&raw.version!==1)throw new Error('不支持的网页配置版本')
  return {config:normalizePlan(raw.config||raw),warnings:[],summary:'网页配置已读取，可预览后应用'}
 }
 const names=windowsProfiles(raw),profile=names.length?raw.Configurations[profileName||raw.Current||names[0]]:raw
 if(!object(profile)||!Array.isArray(profile.TaskQueue))throw new Error('未找到 Windows TaskQueue。请选择新版 MAA 的 gui.json 或网页导出配置')
 const config=normalizePlan(current),warnings=[],seen=new Set(),enabled=[],order=[]
 config.after='none'
 const warn=text=>{if(!warnings.includes(text))warnings.push(text)}
 let imported=0
 for(const task of profile.TaskQueue){
  if(!object(task)){warn('跳过无效任务项');continue}
  const type=String(task.$type||task.TaskType||'').replace(/Task$/,''),id=ids[type]
  if(type==='CloseDown'){if(task.IsEnable===true)config.after='close_game';continue}
  if(!id||!['startup','fight','infrast','recruit','mall','award','rogue','reclamation','update'].includes(id)){warn(`任务 ${type||'未知'} 未迁移，请在网页单独配置`);continue}
  if(seen.has(id)){warn(`网页暂只支持一项 ${type}，已跳过重复项`);continue}
  seen.add(id);order.push(id);if(task.IsEnable!==false&&task.IsEnable!==null)enabled.push(id)
  const p=config.params[id]||(config.params[id]={})
  for(const [source,target] of Object.entries(maps[id]||{}))if(Object.hasOwn(task,source))p[target]=task[source]
  if(task.UseWeeklySchedule)warn(`${type} 的周计划未迁移，请通过原任务调度界面配置`)
  if(id==='startup'){
   p.start_game_enabled=true
   if(task.AccountSwitchEnabled===false)p.account_name=''
  }
  if(id==='fight'){
   p.medicine=task.UseMedicine===true?task.MedicineCount??0:0
   p.stone=task.UseStone===true&&task.UseStoneAllowSave===true?task.StoneCount??0:0
   if(task.UseStone===true&&task.UseStoneAllowSave!==true)warn('未启用 Windows 碎石状态保存，源石使用量保持 0')
   p.times=task.EnableTimesLimit===true?task.TimesLimit??2147483647:2147483647
   p.medicine_expire_days=task.UseExpiringMedicine===true?task.MedicineExpireDays??2:0
   if(Array.isArray(task.StagePlan)){p.stage=task.StagePlan[0]||'';if(task.StagePlan.length>1)warn('作战关卡取 StagePlan 第一项；备选关卡未迁移')}
   if(task.EnableTargetDrop===true&&task.DropId){
    if(task.IsInventoryTarget)warn('材料库存目标需要仓库识别数据，未迁移该停止条件')
    p.drops=task.IsInventoryTarget?{}:{[task.DropId]:task.DropCount}
   }else p.drops={}
   if(task.UseExpireMedicineForActivity)warn('活动结束前用药策略未迁移，仅保留固定临期天数')
  }
  if(id==='infrast'){
   const modes={Normal:0,Custom:10000,Rotation:20000}
   if(Object.hasOwn(task,'Mode'))p.mode=typeof task.Mode==='number'?task.Mode:modes[task.Mode]??0
   if(typeof task.DormThreshold==='number')p.threshold=task.DormThreshold/100
   if(Array.isArray(task.RoomList)&&task.RoomList.every(r=>typeof r.Room==='string'))p.facility=task.RoomList.filter(r=>r.IsEnabled!==false).map(r=>r.Room)
   p.fiammetta_targets=[task.FiammettaTarget1,task.FiammettaTarget2,task.FiammettaTarget3].filter(Boolean)
   if(p.mode===10000){
    p.filename=String(task.Filename||'').split(/[\\/]/).pop()||'';p.plan_index=Math.max(0,task.PlanSelect??0)
    warn('自定义基建只迁移文件名，需另外上传排班 JSON 后才能执行')
   }
  }
  if(id==='recruit'){
   p.confirm=[3,4,5,6].filter(n=>task[`Level${n}Choose`]===true)
   p.select=[4,5,6]
   p.first_tags=task.PreferTagEnabled===false?[]:split(task.Level3PreferTags)
   p.preserve_tags=task.PreserveTagEnabled===true?split(task.PreserveTagList):[]
   p.recruitment_time={'3':task.Level3Time??540,'4':task.Level4Time??540}
   p.level3_recruitment_permit_reserve=task.Level3RecruitmentPermitReserveEnabled===true?task.Level3RecruitmentPermitReserve??8:0
   p.expedite=false // Windows deliberately does not persist expedited-permit use.
   if(task.ForceRefresh)warn('公招无许可强制刷新为 Windows 调度逻辑，网页使用核心默认行为')
  }
  if(id==='mall'){
   if(Object.hasOwn(task,'FirstList'))p.buy_first=split(task.FirstList)
   if(Object.hasOwn(task,'BlackList'))p.blacklist=split(task.BlackList)
   if(task.VisitFriendsOnceADay||task.CreditFightOnceADay)warn('信用任务每日仅一次的 Windows 历史状态未迁移，每次网页执行按所选任务运行')
  }
  if(id==='rogue'){
   if(typeof task.Theme==='string')p.theme=task.Theme
   const modes={Exp:0,Investment:1,Collectible:4,CLP_PDS:5,Squad:6,Exploration:7,FindPlaytime:20001,BlackFlowBabyAnimal:30001,BlackFlowStartRun:30003}
   if(typeof task.Mode==='number')p.mode=task.Mode
   else if(Object.hasOwn(modes,task.Mode))p.mode=modes[task.Mode]
   else if(task.Mode!==undefined)warn(`肉鸽模式 ${task.Mode} 未识别，保留网页当前模式`)
   if(typeof task.Difficulty==='number')p.difficulty=task.Difficulty===2147483647?-1:task.Difficulty
   if(task.StartingOpers?.length){p.core_char_list=task.StartingOpers.slice(0,task.UseAdditionalStartingOpers?3:1).map(o=>({name:o.Name||'',use_support:o.UseSupport===true}));p.core_char=p.core_char_list[0].name;p.use_support=p.core_char_list[0].use_support}
   warn('肉鸽主题特有的种子、藏品和开局奖励策略未自动转换，可在核心参数 JSON 中补充')
  }
  if(id==='update'){const intervals={EveryTime:'every',Daily:'daily',Weekly:'weekly'};p.interval=typeof task.TriggerInterval==='number'?['every','daily','weekly'][task.TriggerInterval]||'every':intervals[task.TriggerInterval]||'every'}
  if(id==='reclamation')warn('生息演算保留网页参数，请按 Windows 设置手动核对主题与模式')
  imported++
 }
 if(!imported)throw new Error('文件内没有可迁移的任务')
 config.enabled=enabled;config.order=order
 const gui=profile.Gui||{},client=gui.RuntimeSettings?.ClientType||gui.ConnectSettings?.ClientType
 if(typeof gui.PostActions==='string'){
  const actions=gui.PostActions.split(',').map(s=>s.trim())
  if(actions.includes('ExitArknights'))config.after='close_game'
  if(actions.some(s=>!['None','ExitArknights','SkipOnError'].includes(s)))warn('完成后仅迁移退出明日方舟；Windows 的关机、休眠及退出程序动作未迁移')
 }else if(typeof gui.PostActions==='number'){
  if(gui.PostActions&1)config.after='close_game'
  if(gui.PostActions&~257)warn('Windows 的其他完成后动作未迁移')
 }
 if(gui.RuntimeSettings?.PreRunScript||gui.RuntimeSettings?.PostRunScript)warn('运行前后脚本未迁移，服务器不会执行 Windows 脚本')
 if(typeof gui.RuntimeSettings?.StartGame==='boolean')config.params.startup={...(config.params.startup||{}),start_game_enabled:gui.RuntimeSettings.StartGame}
 config.timeout_minutes=gui.RuntimeSettings?.EnableRunDurationLimit===true?Math.max(1,Math.min(1440,gui.RuntimeSettings.RunDurationLimitMinutes||240)):0
 if(gui.RuntimeSettings?.RunDurationLimitMinutes>1440)warn('运行时长上限已限制为 1440 分钟')
 if(['Official','Bilibili'].includes(client))config.client=client
 else if(client)warn(`当前网页仅接入官服和 B 服，客户端 ${client} 未迁移`)
 if(config.enabled.includes('startup')&&config.params.startup?.start_game_enabled){config.enabled.unshift('launch');config.params.launch.package=config.client==='Bilibili'?'com.hypergryph.arknights.bilibili':'com.hypergryph.arknights'}
 const third=gui.ThirdParty
 if(object(third))for(const id of ['fight','recruit']){
  if(typeof third.ReportToPenguin==='boolean')config.params[id].report_to_penguin=third.ReportToPenguin
  if(typeof third.PenguinId==='string')config.params[id].penguin_id=third.PenguinId
  if(typeof third.ReportToYituliu==='boolean')config.params[id].report_to_yituliu=third.ReportToYituliu
  if(third.YituliuOpenApiToken)warn('一图流 OpenAPI Token 与核心汇报 ID 不同，未导入 Token')
 }
 warn('Windows 的模拟器路径、ADB 地址、热键、主题、通知密钥及开机启动不迁移；网页继续使用当前共享 USB ADB')
 if(raw.Timers)warn('定时计划未自动启用，请在“定时任务”中配置')
 return {config,warnings,summary:`已转换 ${imported} 项任务，勾选 ${enabled.length} 项；尚未保存或执行`}
}
