// Parameter names and choices follow official MAA v6.20.1 integration.md.
const bool=(key,label)=>[key,label,'bool']
const num=(key,label,min=0,max=2147483647,step=1)=>[key,label,'number',min,max,step]
const list=(key,label)=>[key,label,'list']
const select=(key,label,values)=>[key,label,'select',values]
export const advancedFields={
 fight:[num('medicine_expire_days','使用几天内到期的理智药（0 关闭）',0,30),bool('DrGrandet','节省理智碎石模式')],
 infrast:[bool('replenish','源石碎片自动补货'),bool('dorm_notstationed_enabled','宿舍仅选择未进驻干员'),bool('reception_message_board','领取会客室信息板信用'),bool('reception_clue_exchange','进行线索交流'),bool('reception_send_clue','赠送线索'),bool('continue_training','继续未完成的专精'),bool('fiammetta_recovery_enabled','菲亚梅塔恢复心情'),list('fiammetta_targets','菲亚梅塔恢复目标（依次填写）'),bool('use_pinus_sylvestris','红松骑士团组合'),bool('use_perception_information','感知信息组合'),bool('use_worldly_plight','人间烟火组合'),bool('use_abyssal_hunter','深海猎人组合')],
 recruit:[num('level3_recruitment_permit_reserve','三星保留招聘许可数量'),select('extra_tags_mode','更多标签策略',[[0,'默认'],[1,'尽量选择三个标签'],[2,'更多高星标签组合']]),list('first_tags','三星优先标签'),list('preserve_tags','保留并跳过的标签'),bool('set_time','公招计算时设置时限')],
 mall:[bool('only_buy_discount','只买打折商品'),bool('reserve_max_credit','低于 300 信用停止购物'),bool('credit_fight','进行信用作战'),num('formation_index','信用作战编队（0 当前编队）',0,4)],
 award:[bool('recruit','领取每日免费单抽'),bool('orundum','领取幸运墙合成玉'),bool('mining','领取挖矿合成玉'),bool('specialaccess','领取特殊月卡奖励'),bool('signinevent','领取限时签到活动奖励')],
 rogue:[num('difficulty','指定难度（-1 不指定）',-1,99),bool('use_support','开局使用助战'),bool('use_nonfriend_support','允许非好友助战'),bool('investment_enabled','投资源石锭'),num('investments_count','最大投资次数'),bool('stop_when_investment_full','投资达到上限时停止'),bool('investment_with_more_score','投资后购物刷分'),bool('stop_at_final_boss','最终首领前停止（傀影除外）'),bool('stop_at_max_level','等级满后停止'),bool('start_with_elite_two','凹精二直升'),bool('only_start_with_elite_two','仅接受精二直升'),bool('refresh_trader_with_dice','使用骰子刷新商店'),bool('monthly_squad_auto_iterate','自动切换月度小队'),bool('monthly_squad_check_comms','检查月度小队通讯'),bool('deep_exploration_auto_iterate','自动切换深入调查')],
 reclamation:[list('tools_to_craft','组装道具名称'),num('num_craft_batches','组装批次数',1,999),bool('clear_store','清空商店'),select('increment_mode','组装递增模式',[[0,'点击'],[1,'长按']])],
 copilot:[bool('add_trust','自动编队空位补信赖'),bool('ignore_requirements','忽略作业干员属性要求'),select('support_unit_usage','助战策略',[[0,'不使用'],[1,'仅补齐缺失干员'],[2,'补齐或使用指定助战'],[3,'补齐或使用随机助战']]),['support_unit_name','指定助战干员','text']],
}
export const reportingFields=[bool('report_to_penguin','汇报企鹅物流掉落数据'),['penguin_id','企鹅物流用户 ID','text'],bool('report_to_yituliu','汇报一图流数据'),['yituliu_id','一图流用户 ID','text'],select('server','数据服务器',[['CN','国服'],['US','美服'],['JP','日服'],['KR','韩服']])]
export const extraDefaults={
 launch:{package:'com.hypergryph.arknights'},update:{operbox:true,depot:true,interval:'every'},
 fight:{medicine_expire_days:0,DrGrandet:false,report_to_penguin:false,penguin_id:'',report_to_yituliu:false,yituliu_id:'',server:'CN'},
 infrast:{replenish:false,dorm_notstationed_enabled:false,reception_message_board:true,reception_clue_exchange:true,reception_send_clue:true,continue_training:false,fiammetta_recovery_enabled:false,fiammetta_targets:['清流','可露希尔','但书'],use_pinus_sylvestris:false,use_perception_information:false,use_worldly_plight:false,use_abyssal_hunter:false},
 recruit:{level3_recruitment_permit_reserve:0,extra_tags_mode:0,first_tags:[],preserve_tags:['支援机械'],set_time:true,recruitment_time:{'3':540,'4':540},report_to_penguin:false,penguin_id:'',report_to_yituliu:false,yituliu_id:'',server:'CN'},
 mall:{only_buy_discount:false,reserve_max_credit:false,credit_fight:false,formation_index:0},
 award:{recruit:false,orundum:false,mining:false,specialaccess:false,signinevent:false},
 rogue:{difficulty:-1,use_support:false,use_nonfriend_support:false,investment_enabled:true,investments_count:2147483647,stop_when_investment_full:false,investment_with_more_score:false,stop_at_final_boss:false,stop_at_max_level:false,start_with_elite_two:false,only_start_with_elite_two:false,refresh_trader_with_dice:false,monthly_squad_auto_iterate:true,monthly_squad_check_comms:true,deep_exploration_auto_iterate:true},
 reclamation:{tools_to_craft:[],num_craft_batches:16,clear_store:false,increment_mode:0},
 ssscopilot:{loop_times:1},paradoxcopilot:{},recruitcalc:{},
 copilot:{add_trust:false,ignore_requirements:false,support_unit_usage:0,support_unit_name:''},
}
export function normalizePlan(input){
 const config=JSON.parse(JSON.stringify(input))
 if(!config||!Array.isArray(config.enabled)||!config.params||typeof config.params!=='object'||Array.isArray(config.params))throw new Error('不是有效的网页 MAA 配置')
 const ids=['launch','startup','fight','infrast','recruit','mall','award','rogue','reclamation','update','copilot','ssscopilot','paradoxcopilot','recruitcalc','depot','operbox']
 if(!['Official','Bilibili'].includes(config.client)||!['none','close_game'].includes(config.after))throw new Error('客户端或完成后操作无效')
 for(const values of [config.enabled,config.order||[]])if(!Array.isArray(values)||values.length>ids.length||new Set(values).size!==values.length||values.some(id=>!ids.includes(id)))throw new Error('任务列表或顺序无效')
 for(const [id,p] of Object.entries(config.params))if(!ids.includes(id)||!p||typeof p!=='object'||Array.isArray(p))throw new Error('任务参数必须为对象')
 if(config.timeout_minutes!==undefined&&(!Number.isInteger(config.timeout_minutes)||config.timeout_minutes<0||config.timeout_minutes>1440))throw new Error('运行时长应为 0 到 1440 分钟')
 for(const [id,defaults] of Object.entries(extraDefaults)){
  config.params[id]={...JSON.parse(JSON.stringify(defaults)),...(config.params[id]||{})}
  if(id==='recruit')config.params[id].recruitment_time={...defaults.recruitment_time,...config.params[id].recruitment_time}
 }
 return config
}
