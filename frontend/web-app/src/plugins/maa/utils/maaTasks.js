export const tasks=[
 {id:'launch',name:'启动游戏',icon:'▷',hint:'打开设备中的游戏应用，可填写已安装应用的包名',fields:[['package','游戏包名','text']]},
 {id:'startup',name:'开始唤醒',icon:'☀',hint:'启动已安装的明日方舟客户端',fields:[['start_game_enabled','启动游戏','bool'],['account_name','已登录账号','text']]},
 {id:'fight',name:'理智作战',icon:'⚔',hint:'关卡、理智与作战次数'},
 {id:'infrast',name:'基建换班',icon:'⌂',hint:'设施与无人机设置',fields:[['mode','换班模式','select',[[0,'默认换班'],[10000,'自定义排班'],[20000,'一键轮换']]],['facility','换班设施','multi',[['Mfg','制造站'],['Trade','贸易站'],['Power','发电站'],['Control','控制中枢'],['Reception','会客室'],['Office','办公室'],['Dorm','宿舍'],['Recycle','回收站'],['Processing','加工站'],['Training','训练室'],['AssistantChange','更换助理']]],['drones','无人机用途','select',[['_NotUse','不使用'],['Money','龙门币'],['SyntheticJade','合成玉'],['CombatRecord','作战记录'],['PureGold','赤金'],['OriginStone','源石碎片'],['Chip','芯片']]],['threshold','心情阈值','number',0,1,0.05],['dorm_trust_enabled','宿舍信赖优先','bool'],['filename','自定义排班文件名','text'],['plan_index','排班方案编号','number',0,99]]},
 {id:'recruit',name:'自动公招',icon:'♙',hint:'招募次数与标签选择',fields:[['times','招募次数','number',0,999],['refresh','刷新三星标签','bool'],['expedite','使用加急许可','bool'],['select','选择标签星级','multi',[[3,'三星'],[4,'四星'],[5,'五星'],[6,'六星']]],['confirm','确认招募星级','multi',[[3,'三星'],[4,'四星'],[5,'五星'],[6,'六星']]]]},
 {id:'mall',name:'信用收支',icon:'◇',hint:'访问好友与信用商店',fields:[['visit_friends','访问好友','bool'],['shopping','信用商店购物','bool'],['buy_first','优先购买（逗号分隔）','list'],['blacklist','不购买（逗号分隔）','list'],['force_shopping_if_credit_full','信用溢出时忽略黑名单','bool']]},
 {id:'award',name:'领取奖励',icon:'♧',hint:'领取日常与邮件奖励',fields:[['award','每日与每周任务奖励','bool'],['mail','邮件奖励','bool']]},
 {id:'rogue',name:'自动肉鸽',icon:'♜',hint:'综合战略主题与运行次数',fields:[['theme','主题','select',[['Phantom','傀影与猩红孤钻'],['Mizuki','水月与深蓝之树'],['Sami','探索者的银凇止境'],['Sarkaz','萨卡兹的无终奇语'],['JieGarden','岁的界园志异'],['BlackFlow','黑流树海']]],['mode','运行模式','select',[[0,'刷蜡烛 / 经验'],[1,'刷源石锭'],[4,'凹开局'],[5,'刷坍缩范式（萨米）'],[6,'月度小队'],[7,'深入调查'],[10001,'快速第一层（萨卡兹）'],[20001,'常乐节点（界园）'],[30001,'襁褓动物（黑流）'],[30003,'刷开局（黑流）']]],['starts_count','启动次数','number',1,999],['squad','开局分队','text'],['roles','开局职业组','text'],['core_char','开局干员（可留空）','text']]},
 {id:'reclamation',name:'生息演算',icon:'△',hint:'沙洲遗闻生息点数',fields:[['theme','主题','select',[['Tales','沙洲遗闻']]],['mode','运行模式','select',[[0,'无存档刷点数'],[1,'有存档组装道具']]]]},
 {id:'update',name:'更新数据',icon:'↻',hint:'识别干员数据与仓库，放在其他日常任务之后',fields:[['operbox','更新干员数据','bool'],['depot','更新仓库数据','bool'],['interval','执行间隔','select',[['every','每次执行'],['daily','每天一次'],['weekly','每周一次']]]]},
]
export const copilotTask={id:'copilot',name:'自动战斗',fields:[['loop_times','执行次数','number',1,999],['formation','自动编队','bool'],['formation_index','编队编号（0 当前）','number',0,4],['use_sanity_potion','允许使用理智药','bool']]}
export function clonePlan(value){return JSON.parse(JSON.stringify(value))}
export function fightLimit(params,key,enabled){params[key]=enabled?(key==='times'?999:1):(key==='times'?2147483647:0)}
export function taskOrder(config){const standard=tasks.map(t=>t.id).filter(id=>!['launch','update'].includes(id));return ['launch',...new Set([...(config.order||[]).filter(id=>standard.includes(id)),...standard]),'update']}
export function selectedPlan(config,enabled){const order=[...taskOrder(config),'copilot','ssscopilot','paradoxcopilot','recruitcalc','depot','operbox'];return {...clonePlan(config),enabled:order.filter(id=>enabled.includes(id))}}
