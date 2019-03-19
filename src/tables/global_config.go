package tables

import (
	"encoding/json"
	"io/ioutil"
	"mm_server/libs/log"
	"mm_server/src/server_config"
)

type ChatConfig struct {
	MaxMsgNum       int32 // 最大消息数
	PullMaxMsgNum   int32 // 拉取的消息数量最大值
	PullMsgCooldown int32 // 拉取CD
	MsgMaxBytes     int32 // 消息最大长度
	MsgExistTime    int32 // 消息存在时间
	SendMsgCooldown int32 // 冷却时间
}

type WeekTime struct {
	WeekDay int32
	Hour    int32
	Minute  int32
	Second  int32
}

type CfgIdNum struct {
	CfgId int32
	Num   int32
}

type DaySignSumReward struct {
	SignNum int32
	ChestId int32
}

type TimeData struct {
	Hour   int32
	Minute int32
	Second int32
}

type GlobalConfig struct {
	InitItems     []CfgIdNum
	InitItem_len  int32
	InitCats      []CfgIdNum
	InitCats_len  int32
	InitDiamond   int32
	InitCoin      int32
	InitAreas     []int32
	InitAreas_len int32
	InitFormulas  []int32
	InitBuildings []int32

	HeartbeatInterval int32 // 心跳

	MaxFriendNum int32

	DaySignSumRewards []DaySignSumReward // 累计签到奖励

	WoodChestUnlockTime     int32
	SilverChestUnlockTime   int32
	GoldenChestUnlockTime   int32
	GiantChestUnlockTime    int32
	MagicChestUnlockTime    int32
	RareChestUnlockTime     int32
	EpicChestUnlockTime     int32
	LegendryChestUnlockTime int32

	GooglePayUrl       string
	FaceBookPayUrl     string
	ApplePayUrl        string
	ApplePaySandBoxUrl string

	MaxNameLen     int32   // 最大名字长度
	ChgNameCost    []int32 // 改名消耗的钻石
	ChgNameCostLen int32   // 消耗数组的长度

	FirstPayReward int32

	ShopStartRefreshTime string
	ShopRefreshTime      string

	ExpeditionTaskCount       int32 // 日常任务的数目
	ExpeditionSPEventSec      int32 // 特殊日常任务间隔
	ExpeditionDayFreeChgCount int32 // 每日免费刷新任务次数
	ExpeditionDayChgAddCost   int32 // 免费结束之后每次增加的钻石
	ExpeditionDayMaxChgCost   int32 // 每日最大刷新花费
	ExpeditionDayStartCount   int32 // 每日探险次数

	MapBlockRefleshSec int32 // 地图障碍刷新时间间隔
	MapChestRefleshSec int32 // 地图宝箱刷新时间间隔

	NormalMailLastSec  int32 // 普通邮件持续秒数
	ReqHelpMailLastSec int32 // 帮助邮件持续秒数
	MaxMailCount       int32 // 最大邮件数目

	ChapterUnlockNeedFriendNum int32 // 解锁章节需要好友同意的数目
	ChapterUnlockSecPerDiamond int32 // 解锁章节每钻石对应的描述
	MaxHelpUnlockNum           int32 // 每天最大帮助别人的次数

	ExpeditionMultiVal []int32 // 参加任务的猫和奖励倍数对应的百分比

	RankingListOnceGetItemsNum int32 // 排行榜一次能取的最大数量

	WorldChatData   ChatConfig // 世界频道
	GuildChatData   ChatConfig // 公会频道
	RecruitChatData ChatConfig // 招募频道
	SystemChatData  ChatConfig // 系统公告频

	AnouncementMaxNum       int32 // 公告最大数量
	AnouncementSendCooldown int32 // 公告发送间隔冷却时间(分钟)
	AnouncementSendMaxNum   int32 // 公告一次发送最大数量
	AnouncementExistTime    int32 // 公告存在时间

	FriendMaxNum                    int32    // 最大好友数
	FriendRecommendNum              int32    // 好友推荐数
	FriendGivePointsRefreshTime     TimeData // 赠送友情点刷新时间
	FriendGivePointsPlayerNumOneDay int32    // 好友赠送点数每天最大人数

	MaxDayBuyTiLiCount int32 // 每天最大购买体力的次数
	DayBuyTiliAdd      int32 // 每次购买体力的体力增加值
	DayBuyTiLiCost     int32 // 每次购买体力的消耗的钻石数目

	CancelMakingFormulaReturnMaterial       int32 // 取消打造装饰物返还a%材料（百分比）
	GiveFriendPointsOnce                    int32 // 单次赠送/收取友情点数
	GiveFriendPointsPlayersCount            int32 // 每次赠送友情点好友上限
	GiveFriendPointsRefreshHours            int32 // 赠送好友友情点上限刷新时间（小时）
	GetFriendPointsOpenFriendWoodBox        int32 // 好友基地打开木质宝箱获得友情点
	GetFriendPointsOpenFriendSilverBox      int32 // 好友基地打开银质宝箱获得友情点
	GetFriendPointsOpenFriendGoldBox        int32 // 好友基地打开金质宝箱获得友情点
	FriendsMaxCount                         int32 // 好友数量上限
	FriendFosterLimit                       int32 // 好友寄养上限
	FriendFosterHours                       int32 // 好友寄养时间（小时）
	SpiritGrowPointNeedMinute               int32 // 体力自动恢复时间（分钟）
	FormulaAddNewSlotCostDiamond            int32 // 作坊增加空位消耗钻石
	FormulaSpeedupMakingBuildingCostDiamond int32 // 作坊加速打造建筑消耗钻石 t秒/钻石
	CropSpeedupCostDiamond                  int32 // 农作物加速升级钻石价格   t秒/钻石
	CatHouseSpeedupLevelCostDiamond         int32 // 猫舍加速升级钻石价格：t秒/钻石
	WorldChannelChatCooldown                int32 // 世界频道发送冷却时间：秒
	ChangeNameCostDiamond                   int32 // 改名消耗钻石
	ChangeNameFreeNum                       int32 // 免费改名次数

	FirstChargeRewards      []int32 // 首充奖励
	MonthCardSendRewardTime string  // 月卡发奖时间

	MailTitleBytes         int32 // 邮件标题最大字节数
	MailContentBytes       int32 // 邮件内容最大字节数
	MailMaxCount           int32 // 邮件最大数量
	MailNormalExistDays    int32 // 最大无附件邮件保存天数
	MailAttachExistDays    int32 // 最大附件邮件保存天数
	MailPlayerSendCooldown int32 // 个人邮件发送间隔(秒)
}

func (this *GlobalConfig) Init(conf_file string) bool {
	if conf_file == "" {
		conf_file = "global.json"
	}
	conf_path := server_config.GetGameDataPathFile(conf_file)
	data, err := ioutil.ReadFile(conf_path)
	if nil != err {
		log.Error("GlobalConfigManager::Init failed to readfile err(%s)!", err.Error())
		return false
	}

	err = json.Unmarshal(data, this)
	if nil != err {
		log.Error("GlobalConfigManager::Init json unmarshal failed err(%s)!", err.Error())
		return false
	}

	this.InitItem_len = int32(len(this.InitItems))
	this.InitCats_len = int32(len(this.InitCats))
	this.InitAreas_len = int32(len(this.InitAreas))
	this.ChgNameCostLen = int32(len(this.ChgNameCost))

	if this.InitBuildings != nil && len(this.InitBuildings)%2 != 0 {
		log.Error("GlobalConfigManager::Init json data InitBuildings invalid length")
		return false
	}

	return true
}
