package main

import (
	"github.com/golang/protobuf/proto"
	_ "github.com/go-sql-driver/mysql"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"mm_server/libs/log"
	"math/rand"
	"os"
	"os/exec"
	"mm_server/proto/gen_go/db_game"
	"strings"
	"sync/atomic"
	"time"
)

type dbArgs struct {
	args  []interface{}
	count int32
}

func new_db_args(count int32) (this *dbArgs) {
	this = &dbArgs{}
	this.args = make([]interface{}, count)
	this.count = 0
	return this
}
func (this *dbArgs) Push(arg interface{}) {
	this.args[this.count] = arg
	this.count++
}
func (this *dbArgs) GetArgs() (args []interface{}) {
	return this.args[0:this.count]
}
func (this *DBC) StmtPrepare(s string) (r *sql.Stmt, e error) {
	this.m_db_lock.Lock("DBC.StmtPrepare")
	defer this.m_db_lock.Unlock()
	return this.m_db.Prepare(s)
}
func (this *DBC) StmtExec(stmt *sql.Stmt, args ...interface{}) (r sql.Result, err error) {
	this.m_db_lock.Lock("DBC.StmtExec")
	defer this.m_db_lock.Unlock()
	return stmt.Exec(args...)
}
func (this *DBC) StmtQuery(stmt *sql.Stmt, args ...interface{}) (r *sql.Rows, err error) {
	this.m_db_lock.Lock("DBC.StmtQuery")
	defer this.m_db_lock.Unlock()
	return stmt.Query(args...)
}
func (this *DBC) StmtQueryRow(stmt *sql.Stmt, args ...interface{}) (r *sql.Row) {
	this.m_db_lock.Lock("DBC.StmtQueryRow")
	defer this.m_db_lock.Unlock()
	return stmt.QueryRow(args...)
}
func (this *DBC) Query(s string, args ...interface{}) (r *sql.Rows, e error) {
	this.m_db_lock.Lock("DBC.Query")
	defer this.m_db_lock.Unlock()
	return this.m_db.Query(s, args...)
}
func (this *DBC) QueryRow(s string, args ...interface{}) (r *sql.Row) {
	this.m_db_lock.Lock("DBC.QueryRow")
	defer this.m_db_lock.Unlock()
	return this.m_db.QueryRow(s, args...)
}
func (this *DBC) Exec(s string, args ...interface{}) (r sql.Result, e error) {
	this.m_db_lock.Lock("DBC.Exec")
	defer this.m_db_lock.Unlock()
	return this.m_db.Exec(s, args...)
}
func (this *DBC) Conn(name string, addr string, acc string, pwd string, db_copy_path string) (err error) {
	log.Trace("%v %v %v %v", name, addr, acc, pwd)
	this.m_db_name = name
	source := acc + ":" + pwd + "@tcp(" + addr + ")/" + name + "?charset=utf8"
	this.m_db, err = sql.Open("mysql", source)
	if err != nil {
		log.Error("open db failed %v", err)
		return
	}
	
	this.m_db.SetConnMaxLifetime(time.Second * 5)

	this.m_db_lock = NewMutex()
	this.m_shutdown_lock = NewMutex()

	if config.DBCST_MAX-config.DBCST_MIN <= 1 {
		return errors.New("DBCST_MAX sub DBCST_MIN should greater than 1s")
	}

	err = this.init_tables()
	if err != nil {
		log.Error("init tables failed")
		return
	}

	if os.MkdirAll(db_copy_path, os.ModePerm) == nil {
		os.Chmod(db_copy_path, os.ModePerm)
	}
	
	this.m_db_last_copy_time = int32(time.Now().Hour())
	this.m_db_copy_path = db_copy_path
	addr_list := strings.Split(addr, ":")
	this.m_db_addr = addr_list[0]
	this.m_db_account = acc
	this.m_db_password = pwd
	this.m_initialized = true

	return
}
func (this *DBC) check_files_exist() (file_name string) {
	f_name := fmt.Sprintf("%v/%v_%v", this.m_db_copy_path, this.m_db_name, time.Now().Format("20060102-15"))
	num := int32(0)
	for {
		if num == 0 {
			file_name = f_name
		} else {
			file_name = f_name + fmt.Sprintf("_%v", num)
		}
		_, err := os.Lstat(file_name)
		if err != nil {
			break
		}
		num++
	}
	return file_name
}
func (this *DBC) Loop() {
	defer func() {
		if err := recover(); err != nil {
			log.Stack(err)
		}

		log.Trace("数据库主循环退出")
		this.m_shutdown_completed = true
	}()

	for {
		t := config.DBCST_MIN + rand.Intn(config.DBCST_MAX-config.DBCST_MIN)
		if t <= 0 {
			t = 600
		}

		for i := 0; i < t; i++ {
			time.Sleep(time.Second)
			if this.m_quit {
				break
			}
		}

		if this.m_quit {
			break
		}

		begin := time.Now()
		err := this.Save(false)
		if err != nil {
			log.Error("save db failed %v", err)
		}
		log.Trace("db存数据花费时长: %vms", time.Now().Sub(begin).Nanoseconds()/1000000)
		
		now_time_hour := int32(time.Now().Hour())
		if now_time_hour-24 >= this.m_db_last_copy_time {
			args := []string {
				fmt.Sprintf("-h%v", this.m_db_addr),
				fmt.Sprintf("-u%v", this.m_db_account),
				fmt.Sprintf("-p%v", this.m_db_password),
				this.m_db_name,
			}
			cmd := exec.Command("mysqldump", args...)
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd_err := cmd.Run()
			if cmd_err == nil {
				file_name := this.check_files_exist()
				file, file_err := os.OpenFile(file_name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0660)
				defer file.Close()
				if file_err == nil {
					_, write_err := file.Write(out.Bytes())
					if write_err == nil {
						log.Trace("数据库备份成功！备份文件名:%v", file_name)
					} else {
						log.Error("数据库备份文件写入失败！备份文件名%v", file_name)
					}
				} else {
					log.Error("数据库备份文件打开失败！备份文件名%v", file_name)
				}
				file.Close()
			} else {
				log.Error("数据库备份失败！")
			}
			this.m_db_last_copy_time = now_time_hour
		}
		
		if this.m_quit {
			break
		}
	}

	log.Trace("数据库缓存主循环退出，保存所有数据")

	err := this.Save(true)
	if err != nil {
		log.Error("shutdwon save db failed %v", err)
		return
	}

	err = this.m_db.Close()
	if err != nil {
		log.Error("close db failed %v", err)
		return
	}
}
func (this *DBC) Shutdown() {
	if !this.m_initialized {
		return
	}

	this.m_shutdown_lock.UnSafeLock("DBC.Shutdown")
	defer this.m_shutdown_lock.UnSafeUnlock()

	if this.m_quit {
		return
	}
	this.m_quit = true

	log.Trace("关闭数据库缓存")

	begin := time.Now()

	for {
		if this.m_shutdown_completed {
			break
		}

		time.Sleep(time.Millisecond * 100)
	}

	log.Trace("关闭数据库缓存耗时 %v 秒", time.Now().Sub(begin).Seconds())
}


const DBC_VERSION = 1
const DBC_SUB_VERSION = 0

type dbExpeditionConData struct{
	ConType int32
	ConVals []int32
}
func (this* dbExpeditionConData)from_pb(pb *db.ExpeditionCon){
	if pb == nil {
		this.ConVals = make([]int32,0)
		return
	}
	this.ConType = pb.GetConType()
	this.ConVals = make([]int32,len(pb.GetConVals()))
	for i, v := range pb.GetConVals() {
		this.ConVals[i] = v
	}
	return
}
func (this* dbExpeditionConData)to_pb()(pb *db.ExpeditionCon){
	pb = &db.ExpeditionCon{}
	pb.ConType = proto.Int32(this.ConType)
	pb.ConVals = make([]int32, len(this.ConVals))
	for i, v := range this.ConVals {
		pb.ConVals[i]=v
	}
	return
}
func (this* dbExpeditionConData)clone_to(d *dbExpeditionConData){
	d.ConType = this.ConType
	d.ConVals = make([]int32, len(this.ConVals))
	for _ii, _vv := range this.ConVals {
		d.ConVals[_ii]=_vv
	}
	return
}
type dbExpeditionEventData struct{
	EventId int32
	ClientId int32
	Sec int32
	DropIdNums []int32
}
func (this* dbExpeditionEventData)from_pb(pb *db.ExpeditionEvent){
	if pb == nil {
		this.DropIdNums = make([]int32,0)
		return
	}
	this.EventId = pb.GetEventId()
	this.ClientId = pb.GetClientId()
	this.Sec = pb.GetSec()
	this.DropIdNums = make([]int32,len(pb.GetDropIdNums()))
	for i, v := range pb.GetDropIdNums() {
		this.DropIdNums[i] = v
	}
	return
}
func (this* dbExpeditionEventData)to_pb()(pb *db.ExpeditionEvent){
	pb = &db.ExpeditionEvent{}
	pb.EventId = proto.Int32(this.EventId)
	pb.ClientId = proto.Int32(this.ClientId)
	pb.Sec = proto.Int32(this.Sec)
	pb.DropIdNums = make([]int32, len(this.DropIdNums))
	for i, v := range this.DropIdNums {
		pb.DropIdNums[i]=v
	}
	return
}
func (this* dbExpeditionEventData)clone_to(d *dbExpeditionEventData){
	d.EventId = this.EventId
	d.ClientId = this.ClientId
	d.Sec = this.Sec
	d.DropIdNums = make([]int32, len(this.DropIdNums))
	for _ii, _vv := range this.DropIdNums {
		d.DropIdNums[_ii]=_vv
	}
	return
}
type dbPlayerInfoData struct{
	Gold int32
	Diamond int32
	CurMaxStage int32
	TotalStars int32
	CurPassMaxStage int32
	MaxUnlockStage int32
	MaxChapter int32
	CreateUnix int32
	Lvl int32
	Exp int32
	FirstPayState int32
	ChangeNameCount int32
	LastDialyTaskUpUinx int32
	Head int32
	CustomIcon string
	NextBuildingId int32
	NextCatId int32
	CharmVal int32
	LastLogin int32
	Zan int32
	CatFood int32
	Spirit int32
	FriendPoints int32
	SoulStone int32
	CharmMedal int32
	SaveLastSpiritPointTime int32
	LastRefreshShopTime int32
	DayChgExpeditionCount int32
	DayChgExpeditionUpDay int32
	LastMapChestUpUnix int32
	LastMapBlockUpUnix int32
	VipLvl int32
	MakingBuildingQueue []int32
	MakedBuildingQueue []int32
	DayHelpUnlockCount int32
	DayHelpUnlockUpDay int32
	FriendMessageUnreadCurrId int32
	VipCardEndDay int32
	NextExpeditionId int32
	DayExpeditionCount int32
	DayExpeditionUpDay int32
	Channel string
	DayBuyTiLiCount int32
	DayBuyTiLiUpDay int32
	LastLogout int32
}
func (this* dbPlayerInfoData)from_pb(pb *db.PlayerInfo){
	if pb == nil {
		this.MakingBuildingQueue = make([]int32,0)
		this.MakedBuildingQueue = make([]int32,0)
		return
	}
	this.Gold = pb.GetGold()
	this.Diamond = pb.GetDiamond()
	this.CurMaxStage = pb.GetCurMaxStage()
	this.TotalStars = pb.GetTotalStars()
	this.CurPassMaxStage = pb.GetCurPassMaxStage()
	this.MaxUnlockStage = pb.GetMaxUnlockStage()
	this.MaxChapter = pb.GetMaxChapter()
	this.CreateUnix = pb.GetCreateUnix()
	this.Lvl = pb.GetLvl()
	this.Exp = pb.GetExp()
	this.FirstPayState = pb.GetFirstPayState()
	this.ChangeNameCount = pb.GetChangeNameCount()
	this.LastDialyTaskUpUinx = pb.GetLastDialyTaskUpUinx()
	this.Head = pb.GetHead()
	this.CustomIcon = pb.GetCustomIcon()
	this.NextBuildingId = pb.GetNextBuildingId()
	this.NextCatId = pb.GetNextCatId()
	this.CharmVal = pb.GetCharmVal()
	this.LastLogin = pb.GetLastLogin()
	this.Zan = pb.GetZan()
	this.CatFood = pb.GetCatFood()
	this.Spirit = pb.GetSpirit()
	this.FriendPoints = pb.GetFriendPoints()
	this.SoulStone = pb.GetSoulStone()
	this.CharmMedal = pb.GetCharmMedal()
	this.SaveLastSpiritPointTime = pb.GetSaveLastSpiritPointTime()
	this.LastRefreshShopTime = pb.GetLastRefreshShopTime()
	this.DayChgExpeditionCount = pb.GetDayChgExpeditionCount()
	this.DayChgExpeditionUpDay = pb.GetDayChgExpeditionUpDay()
	this.LastMapChestUpUnix = pb.GetLastMapChestUpUnix()
	this.LastMapBlockUpUnix = pb.GetLastMapBlockUpUnix()
	this.VipLvl = pb.GetVipLvl()
	this.MakingBuildingQueue = make([]int32,len(pb.GetMakingBuildingQueue()))
	for i, v := range pb.GetMakingBuildingQueue() {
		this.MakingBuildingQueue[i] = v
	}
	this.MakedBuildingQueue = make([]int32,len(pb.GetMakedBuildingQueue()))
	for i, v := range pb.GetMakedBuildingQueue() {
		this.MakedBuildingQueue[i] = v
	}
	this.DayHelpUnlockCount = pb.GetDayHelpUnlockCount()
	this.DayHelpUnlockUpDay = pb.GetDayHelpUnlockUpDay()
	this.FriendMessageUnreadCurrId = pb.GetFriendMessageUnreadCurrId()
	this.VipCardEndDay = pb.GetVipCardEndDay()
	this.NextExpeditionId = pb.GetNextExpeditionId()
	this.DayExpeditionCount = pb.GetDayExpeditionCount()
	this.DayExpeditionUpDay = pb.GetDayExpeditionUpDay()
	this.Channel = pb.GetChannel()
	this.DayBuyTiLiCount = pb.GetDayBuyTiLiCount()
	this.DayBuyTiLiUpDay = pb.GetDayBuyTiLiUpDay()
	this.LastLogout = pb.GetLastLogout()
	return
}
func (this* dbPlayerInfoData)to_pb()(pb *db.PlayerInfo){
	pb = &db.PlayerInfo{}
	pb.Gold = proto.Int32(this.Gold)
	pb.Diamond = proto.Int32(this.Diamond)
	pb.CurMaxStage = proto.Int32(this.CurMaxStage)
	pb.TotalStars = proto.Int32(this.TotalStars)
	pb.CurPassMaxStage = proto.Int32(this.CurPassMaxStage)
	pb.MaxUnlockStage = proto.Int32(this.MaxUnlockStage)
	pb.MaxChapter = proto.Int32(this.MaxChapter)
	pb.CreateUnix = proto.Int32(this.CreateUnix)
	pb.Lvl = proto.Int32(this.Lvl)
	pb.Exp = proto.Int32(this.Exp)
	pb.FirstPayState = proto.Int32(this.FirstPayState)
	pb.ChangeNameCount = proto.Int32(this.ChangeNameCount)
	pb.LastDialyTaskUpUinx = proto.Int32(this.LastDialyTaskUpUinx)
	pb.Head = proto.Int32(this.Head)
	pb.CustomIcon = proto.String(this.CustomIcon)
	pb.NextBuildingId = proto.Int32(this.NextBuildingId)
	pb.NextCatId = proto.Int32(this.NextCatId)
	pb.CharmVal = proto.Int32(this.CharmVal)
	pb.LastLogin = proto.Int32(this.LastLogin)
	pb.Zan = proto.Int32(this.Zan)
	pb.CatFood = proto.Int32(this.CatFood)
	pb.Spirit = proto.Int32(this.Spirit)
	pb.FriendPoints = proto.Int32(this.FriendPoints)
	pb.SoulStone = proto.Int32(this.SoulStone)
	pb.CharmMedal = proto.Int32(this.CharmMedal)
	pb.SaveLastSpiritPointTime = proto.Int32(this.SaveLastSpiritPointTime)
	pb.LastRefreshShopTime = proto.Int32(this.LastRefreshShopTime)
	pb.DayChgExpeditionCount = proto.Int32(this.DayChgExpeditionCount)
	pb.DayChgExpeditionUpDay = proto.Int32(this.DayChgExpeditionUpDay)
	pb.LastMapChestUpUnix = proto.Int32(this.LastMapChestUpUnix)
	pb.LastMapBlockUpUnix = proto.Int32(this.LastMapBlockUpUnix)
	pb.VipLvl = proto.Int32(this.VipLvl)
	pb.MakingBuildingQueue = make([]int32, len(this.MakingBuildingQueue))
	for i, v := range this.MakingBuildingQueue {
		pb.MakingBuildingQueue[i]=v
	}
	pb.MakedBuildingQueue = make([]int32, len(this.MakedBuildingQueue))
	for i, v := range this.MakedBuildingQueue {
		pb.MakedBuildingQueue[i]=v
	}
	pb.DayHelpUnlockCount = proto.Int32(this.DayHelpUnlockCount)
	pb.DayHelpUnlockUpDay = proto.Int32(this.DayHelpUnlockUpDay)
	pb.FriendMessageUnreadCurrId = proto.Int32(this.FriendMessageUnreadCurrId)
	pb.VipCardEndDay = proto.Int32(this.VipCardEndDay)
	pb.NextExpeditionId = proto.Int32(this.NextExpeditionId)
	pb.DayExpeditionCount = proto.Int32(this.DayExpeditionCount)
	pb.DayExpeditionUpDay = proto.Int32(this.DayExpeditionUpDay)
	pb.Channel = proto.String(this.Channel)
	pb.DayBuyTiLiCount = proto.Int32(this.DayBuyTiLiCount)
	pb.DayBuyTiLiUpDay = proto.Int32(this.DayBuyTiLiUpDay)
	pb.LastLogout = proto.Int32(this.LastLogout)
	return
}
func (this* dbPlayerInfoData)clone_to(d *dbPlayerInfoData){
	d.Gold = this.Gold
	d.Diamond = this.Diamond
	d.CurMaxStage = this.CurMaxStage
	d.TotalStars = this.TotalStars
	d.CurPassMaxStage = this.CurPassMaxStage
	d.MaxUnlockStage = this.MaxUnlockStage
	d.MaxChapter = this.MaxChapter
	d.CreateUnix = this.CreateUnix
	d.Lvl = this.Lvl
	d.Exp = this.Exp
	d.FirstPayState = this.FirstPayState
	d.ChangeNameCount = this.ChangeNameCount
	d.LastDialyTaskUpUinx = this.LastDialyTaskUpUinx
	d.Head = this.Head
	d.CustomIcon = this.CustomIcon
	d.NextBuildingId = this.NextBuildingId
	d.NextCatId = this.NextCatId
	d.CharmVal = this.CharmVal
	d.LastLogin = this.LastLogin
	d.Zan = this.Zan
	d.CatFood = this.CatFood
	d.Spirit = this.Spirit
	d.FriendPoints = this.FriendPoints
	d.SoulStone = this.SoulStone
	d.CharmMedal = this.CharmMedal
	d.SaveLastSpiritPointTime = this.SaveLastSpiritPointTime
	d.LastRefreshShopTime = this.LastRefreshShopTime
	d.DayChgExpeditionCount = this.DayChgExpeditionCount
	d.DayChgExpeditionUpDay = this.DayChgExpeditionUpDay
	d.LastMapChestUpUnix = this.LastMapChestUpUnix
	d.LastMapBlockUpUnix = this.LastMapBlockUpUnix
	d.VipLvl = this.VipLvl
	d.MakingBuildingQueue = make([]int32, len(this.MakingBuildingQueue))
	for _ii, _vv := range this.MakingBuildingQueue {
		d.MakingBuildingQueue[_ii]=_vv
	}
	d.MakedBuildingQueue = make([]int32, len(this.MakedBuildingQueue))
	for _ii, _vv := range this.MakedBuildingQueue {
		d.MakedBuildingQueue[_ii]=_vv
	}
	d.DayHelpUnlockCount = this.DayHelpUnlockCount
	d.DayHelpUnlockUpDay = this.DayHelpUnlockUpDay
	d.FriendMessageUnreadCurrId = this.FriendMessageUnreadCurrId
	d.VipCardEndDay = this.VipCardEndDay
	d.NextExpeditionId = this.NextExpeditionId
	d.DayExpeditionCount = this.DayExpeditionCount
	d.DayExpeditionUpDay = this.DayExpeditionUpDay
	d.Channel = this.Channel
	d.DayBuyTiLiCount = this.DayBuyTiLiCount
	d.DayBuyTiLiUpDay = this.DayBuyTiLiUpDay
	d.LastLogout = this.LastLogout
	return
}
type dbPlayerStageData struct{
	StageId int32
	Stars int32
	LastFinishedUnix int32
	TopScore int32
	CatId int32
	PlayedCount int32
	PassCount int32
}
func (this* dbPlayerStageData)from_pb(pb *db.PlayerStage){
	if pb == nil {
		return
	}
	this.StageId = pb.GetStageId()
	this.Stars = pb.GetStars()
	this.LastFinishedUnix = pb.GetLastFinishedUnix()
	this.TopScore = pb.GetTopScore()
	this.CatId = pb.GetCatId()
	this.PlayedCount = pb.GetPlayedCount()
	this.PassCount = pb.GetPassCount()
	return
}
func (this* dbPlayerStageData)to_pb()(pb *db.PlayerStage){
	pb = &db.PlayerStage{}
	pb.StageId = proto.Int32(this.StageId)
	pb.Stars = proto.Int32(this.Stars)
	pb.LastFinishedUnix = proto.Int32(this.LastFinishedUnix)
	pb.TopScore = proto.Int32(this.TopScore)
	pb.CatId = proto.Int32(this.CatId)
	pb.PlayedCount = proto.Int32(this.PlayedCount)
	pb.PassCount = proto.Int32(this.PassCount)
	return
}
func (this* dbPlayerStageData)clone_to(d *dbPlayerStageData){
	d.StageId = this.StageId
	d.Stars = this.Stars
	d.LastFinishedUnix = this.LastFinishedUnix
	d.TopScore = this.TopScore
	d.CatId = this.CatId
	d.PlayedCount = this.PlayedCount
	d.PassCount = this.PassCount
	return
}
type dbPlayerChapterUnLockData struct{
	ChapterId int32
	PlayerIds []int32
	CurHelpIds []int32
	StartUnix int32
}
func (this* dbPlayerChapterUnLockData)from_pb(pb *db.PlayerChapterUnLock){
	if pb == nil {
		this.PlayerIds = make([]int32,0)
		this.CurHelpIds = make([]int32,0)
		return
	}
	this.ChapterId = pb.GetChapterId()
	this.PlayerIds = make([]int32,len(pb.GetPlayerIds()))
	for i, v := range pb.GetPlayerIds() {
		this.PlayerIds[i] = v
	}
	this.CurHelpIds = make([]int32,len(pb.GetCurHelpIds()))
	for i, v := range pb.GetCurHelpIds() {
		this.CurHelpIds[i] = v
	}
	this.StartUnix = pb.GetStartUnix()
	return
}
func (this* dbPlayerChapterUnLockData)to_pb()(pb *db.PlayerChapterUnLock){
	pb = &db.PlayerChapterUnLock{}
	pb.ChapterId = proto.Int32(this.ChapterId)
	pb.PlayerIds = make([]int32, len(this.PlayerIds))
	for i, v := range this.PlayerIds {
		pb.PlayerIds[i]=v
	}
	pb.CurHelpIds = make([]int32, len(this.CurHelpIds))
	for i, v := range this.CurHelpIds {
		pb.CurHelpIds[i]=v
	}
	pb.StartUnix = proto.Int32(this.StartUnix)
	return
}
func (this* dbPlayerChapterUnLockData)clone_to(d *dbPlayerChapterUnLockData){
	d.ChapterId = this.ChapterId
	d.PlayerIds = make([]int32, len(this.PlayerIds))
	for _ii, _vv := range this.PlayerIds {
		d.PlayerIds[_ii]=_vv
	}
	d.CurHelpIds = make([]int32, len(this.CurHelpIds))
	for _ii, _vv := range this.CurHelpIds {
		d.CurHelpIds[_ii]=_vv
	}
	d.StartUnix = this.StartUnix
	return
}
type dbPlayerItemData struct{
	ItemCfgId int32
	ItemNum int32
	StartTimeUnix int32
	RemainSeconds int32
}
func (this* dbPlayerItemData)from_pb(pb *db.PlayerItem){
	if pb == nil {
		return
	}
	this.ItemCfgId = pb.GetItemCfgId()
	this.ItemNum = pb.GetItemNum()
	this.StartTimeUnix = pb.GetStartTimeUnix()
	this.RemainSeconds = pb.GetRemainSeconds()
	return
}
func (this* dbPlayerItemData)to_pb()(pb *db.PlayerItem){
	pb = &db.PlayerItem{}
	pb.ItemCfgId = proto.Int32(this.ItemCfgId)
	pb.ItemNum = proto.Int32(this.ItemNum)
	pb.StartTimeUnix = proto.Int32(this.StartTimeUnix)
	pb.RemainSeconds = proto.Int32(this.RemainSeconds)
	return
}
func (this* dbPlayerItemData)clone_to(d *dbPlayerItemData){
	d.ItemCfgId = this.ItemCfgId
	d.ItemNum = this.ItemNum
	d.StartTimeUnix = this.StartTimeUnix
	d.RemainSeconds = this.RemainSeconds
	return
}
type dbPlayerAreaData struct{
	CfgId int32
}
func (this* dbPlayerAreaData)from_pb(pb *db.PlayerArea){
	if pb == nil {
		return
	}
	this.CfgId = pb.GetCfgId()
	return
}
func (this* dbPlayerAreaData)to_pb()(pb *db.PlayerArea){
	pb = &db.PlayerArea{}
	pb.CfgId = proto.Int32(this.CfgId)
	return
}
func (this* dbPlayerAreaData)clone_to(d *dbPlayerAreaData){
	d.CfgId = this.CfgId
	return
}
type dbPlayerBuildingData struct{
	Id int32
	CfgId int32
	X int32
	Y int32
	Dir int32
	CreateUnix int32
	OverUnix int32
}
func (this* dbPlayerBuildingData)from_pb(pb *db.PlayerBuilding){
	if pb == nil {
		return
	}
	this.Id = pb.GetId()
	this.CfgId = pb.GetCfgId()
	this.X = pb.GetX()
	this.Y = pb.GetY()
	this.Dir = pb.GetDir()
	this.CreateUnix = pb.GetCreateUnix()
	this.OverUnix = pb.GetOverUnix()
	return
}
func (this* dbPlayerBuildingData)to_pb()(pb *db.PlayerBuilding){
	pb = &db.PlayerBuilding{}
	pb.Id = proto.Int32(this.Id)
	pb.CfgId = proto.Int32(this.CfgId)
	pb.X = proto.Int32(this.X)
	pb.Y = proto.Int32(this.Y)
	pb.Dir = proto.Int32(this.Dir)
	pb.CreateUnix = proto.Int32(this.CreateUnix)
	pb.OverUnix = proto.Int32(this.OverUnix)
	return
}
func (this* dbPlayerBuildingData)clone_to(d *dbPlayerBuildingData){
	d.Id = this.Id
	d.CfgId = this.CfgId
	d.X = this.X
	d.Y = this.Y
	d.Dir = this.Dir
	d.CreateUnix = this.CreateUnix
	d.OverUnix = this.OverUnix
	return
}
type dbPlayerBuildingDepotData struct{
	CfgId int32
	Num int32
}
func (this* dbPlayerBuildingDepotData)from_pb(pb *db.PlayerBuildingDepot){
	if pb == nil {
		return
	}
	this.CfgId = pb.GetCfgId()
	this.Num = pb.GetNum()
	return
}
func (this* dbPlayerBuildingDepotData)to_pb()(pb *db.PlayerBuildingDepot){
	pb = &db.PlayerBuildingDepot{}
	pb.CfgId = proto.Int32(this.CfgId)
	pb.Num = proto.Int32(this.Num)
	return
}
func (this* dbPlayerBuildingDepotData)clone_to(d *dbPlayerBuildingDepotData){
	d.CfgId = this.CfgId
	d.Num = this.Num
	return
}
type dbPlayerDepotBuildingFormulaData struct{
	Id int32
}
func (this* dbPlayerDepotBuildingFormulaData)from_pb(pb *db.PlayerDepotBuildingFormula){
	if pb == nil {
		return
	}
	this.Id = pb.GetId()
	return
}
func (this* dbPlayerDepotBuildingFormulaData)to_pb()(pb *db.PlayerDepotBuildingFormula){
	pb = &db.PlayerDepotBuildingFormula{}
	pb.Id = proto.Int32(this.Id)
	return
}
func (this* dbPlayerDepotBuildingFormulaData)clone_to(d *dbPlayerDepotBuildingFormulaData){
	d.Id = this.Id
	return
}
type dbPlayerMakingFormulaBuildingData struct{
	SlotId int32
	CanUse int32
	FormulaId int32
	StartTime int32
}
func (this* dbPlayerMakingFormulaBuildingData)from_pb(pb *db.PlayerMakingFormulaBuilding){
	if pb == nil {
		return
	}
	this.SlotId = pb.GetSlotId()
	this.CanUse = pb.GetCanUse()
	this.FormulaId = pb.GetFormulaId()
	this.StartTime = pb.GetStartTime()
	return
}
func (this* dbPlayerMakingFormulaBuildingData)to_pb()(pb *db.PlayerMakingFormulaBuilding){
	pb = &db.PlayerMakingFormulaBuilding{}
	pb.SlotId = proto.Int32(this.SlotId)
	pb.CanUse = proto.Int32(this.CanUse)
	pb.FormulaId = proto.Int32(this.FormulaId)
	pb.StartTime = proto.Int32(this.StartTime)
	return
}
func (this* dbPlayerMakingFormulaBuildingData)clone_to(d *dbPlayerMakingFormulaBuildingData){
	d.SlotId = this.SlotId
	d.CanUse = this.CanUse
	d.FormulaId = this.FormulaId
	d.StartTime = this.StartTime
	return
}
type dbPlayerCropData struct{
	BuildingId int32
	Id int32
	PlantTime int32
	BuildingTableId int32
}
func (this* dbPlayerCropData)from_pb(pb *db.PlayerCrop){
	if pb == nil {
		return
	}
	this.BuildingId = pb.GetBuildingId()
	this.Id = pb.GetId()
	this.PlantTime = pb.GetPlantTime()
	this.BuildingTableId = pb.GetBuildingTableId()
	return
}
func (this* dbPlayerCropData)to_pb()(pb *db.PlayerCrop){
	pb = &db.PlayerCrop{}
	pb.BuildingId = proto.Int32(this.BuildingId)
	pb.Id = proto.Int32(this.Id)
	pb.PlantTime = proto.Int32(this.PlantTime)
	pb.BuildingTableId = proto.Int32(this.BuildingTableId)
	return
}
func (this* dbPlayerCropData)clone_to(d *dbPlayerCropData){
	d.BuildingId = this.BuildingId
	d.Id = this.Id
	d.PlantTime = this.PlantTime
	d.BuildingTableId = this.BuildingTableId
	return
}
type dbPlayerCatData struct{
	Id int32
	CfgId int32
	Exp int32
	Level int32
	Star int32
	Nick string
	SkillLevel int32
	Locked int32
	CoinAbility int32
	ExploreAbility int32
	MatchAbility int32
	CathouseId int32
	State int32
	StateValue int32
}
func (this* dbPlayerCatData)from_pb(pb *db.PlayerCat){
	if pb == nil {
		return
	}
	this.Id = pb.GetId()
	this.CfgId = pb.GetCfgId()
	this.Exp = pb.GetExp()
	this.Level = pb.GetLevel()
	this.Star = pb.GetStar()
	this.Nick = pb.GetNick()
	this.SkillLevel = pb.GetSkillLevel()
	this.Locked = pb.GetLocked()
	this.CoinAbility = pb.GetCoinAbility()
	this.ExploreAbility = pb.GetExploreAbility()
	this.MatchAbility = pb.GetMatchAbility()
	this.CathouseId = pb.GetCathouseId()
	this.State = pb.GetState()
	this.StateValue = pb.GetStateValue()
	return
}
func (this* dbPlayerCatData)to_pb()(pb *db.PlayerCat){
	pb = &db.PlayerCat{}
	pb.Id = proto.Int32(this.Id)
	pb.CfgId = proto.Int32(this.CfgId)
	pb.Exp = proto.Int32(this.Exp)
	pb.Level = proto.Int32(this.Level)
	pb.Star = proto.Int32(this.Star)
	pb.Nick = proto.String(this.Nick)
	pb.SkillLevel = proto.Int32(this.SkillLevel)
	pb.Locked = proto.Int32(this.Locked)
	pb.CoinAbility = proto.Int32(this.CoinAbility)
	pb.ExploreAbility = proto.Int32(this.ExploreAbility)
	pb.MatchAbility = proto.Int32(this.MatchAbility)
	pb.CathouseId = proto.Int32(this.CathouseId)
	pb.State = proto.Int32(this.State)
	pb.StateValue = proto.Int32(this.StateValue)
	return
}
func (this* dbPlayerCatData)clone_to(d *dbPlayerCatData){
	d.Id = this.Id
	d.CfgId = this.CfgId
	d.Exp = this.Exp
	d.Level = this.Level
	d.Star = this.Star
	d.Nick = this.Nick
	d.SkillLevel = this.SkillLevel
	d.Locked = this.Locked
	d.CoinAbility = this.CoinAbility
	d.ExploreAbility = this.ExploreAbility
	d.MatchAbility = this.MatchAbility
	d.CathouseId = this.CathouseId
	d.State = this.State
	d.StateValue = this.StateValue
	return
}
type dbPlayerCatHouseData struct{
	BuildingId int32
	CfgId int32
	Level int32
	CatIds []int32
	LastGetGoldTime int32
	CurrGold int32
	LevelupStartTime int32
	IsDone int32
}
func (this* dbPlayerCatHouseData)from_pb(pb *db.PlayerCatHouse){
	if pb == nil {
		this.CatIds = make([]int32,0)
		return
	}
	this.BuildingId = pb.GetBuildingId()
	this.CfgId = pb.GetCfgId()
	this.Level = pb.GetLevel()
	this.CatIds = make([]int32,len(pb.GetCatIds()))
	for i, v := range pb.GetCatIds() {
		this.CatIds[i] = v
	}
	this.LastGetGoldTime = pb.GetLastGetGoldTime()
	this.CurrGold = pb.GetCurrGold()
	this.LevelupStartTime = pb.GetLevelupStartTime()
	this.IsDone = pb.GetIsDone()
	return
}
func (this* dbPlayerCatHouseData)to_pb()(pb *db.PlayerCatHouse){
	pb = &db.PlayerCatHouse{}
	pb.BuildingId = proto.Int32(this.BuildingId)
	pb.CfgId = proto.Int32(this.CfgId)
	pb.Level = proto.Int32(this.Level)
	pb.CatIds = make([]int32, len(this.CatIds))
	for i, v := range this.CatIds {
		pb.CatIds[i]=v
	}
	pb.LastGetGoldTime = proto.Int32(this.LastGetGoldTime)
	pb.CurrGold = proto.Int32(this.CurrGold)
	pb.LevelupStartTime = proto.Int32(this.LevelupStartTime)
	pb.IsDone = proto.Int32(this.IsDone)
	return
}
func (this* dbPlayerCatHouseData)clone_to(d *dbPlayerCatHouseData){
	d.BuildingId = this.BuildingId
	d.CfgId = this.CfgId
	d.Level = this.Level
	d.CatIds = make([]int32, len(this.CatIds))
	for _ii, _vv := range this.CatIds {
		d.CatIds[_ii]=_vv
	}
	d.LastGetGoldTime = this.LastGetGoldTime
	d.CurrGold = this.CurrGold
	d.LevelupStartTime = this.LevelupStartTime
	d.IsDone = this.IsDone
	return
}
type dbPlayerShopItemData struct{
	Id int32
	LeftNum int32
}
func (this* dbPlayerShopItemData)from_pb(pb *db.PlayerShopItem){
	if pb == nil {
		return
	}
	this.Id = pb.GetId()
	this.LeftNum = pb.GetLeftNum()
	return
}
func (this* dbPlayerShopItemData)to_pb()(pb *db.PlayerShopItem){
	pb = &db.PlayerShopItem{}
	pb.Id = proto.Int32(this.Id)
	pb.LeftNum = proto.Int32(this.LeftNum)
	return
}
func (this* dbPlayerShopItemData)clone_to(d *dbPlayerShopItemData){
	d.Id = this.Id
	d.LeftNum = this.LeftNum
	return
}
type dbPlayerShopLimitedInfoData struct{
	LimitedDays int32
	LastSaveTime int32
}
func (this* dbPlayerShopLimitedInfoData)from_pb(pb *db.PlayerShopLimitedInfo){
	if pb == nil {
		return
	}
	this.LimitedDays = pb.GetLimitedDays()
	this.LastSaveTime = pb.GetLastSaveTime()
	return
}
func (this* dbPlayerShopLimitedInfoData)to_pb()(pb *db.PlayerShopLimitedInfo){
	pb = &db.PlayerShopLimitedInfo{}
	pb.LimitedDays = proto.Int32(this.LimitedDays)
	pb.LastSaveTime = proto.Int32(this.LastSaveTime)
	return
}
func (this* dbPlayerShopLimitedInfoData)clone_to(d *dbPlayerShopLimitedInfoData){
	d.LimitedDays = this.LimitedDays
	d.LastSaveTime = this.LastSaveTime
	return
}
type dbPlayerChestData struct{
	Pos int32
	ChestId int32
	OpenSec int32
}
func (this* dbPlayerChestData)from_pb(pb *db.PlayerChest){
	if pb == nil {
		return
	}
	this.Pos = pb.GetPos()
	this.ChestId = pb.GetChestId()
	this.OpenSec = pb.GetOpenSec()
	return
}
func (this* dbPlayerChestData)to_pb()(pb *db.PlayerChest){
	pb = &db.PlayerChest{}
	pb.Pos = proto.Int32(this.Pos)
	pb.ChestId = proto.Int32(this.ChestId)
	pb.OpenSec = proto.Int32(this.OpenSec)
	return
}
func (this* dbPlayerChestData)clone_to(d *dbPlayerChestData){
	d.Pos = this.Pos
	d.ChestId = this.ChestId
	d.OpenSec = this.OpenSec
	return
}
type dbPlayerPayBackData struct{
	PayBackId int32
	Value string
}
func (this* dbPlayerPayBackData)from_pb(pb *db.PlayerPayBack){
	if pb == nil {
		return
	}
	this.PayBackId = pb.GetPayBackId()
	this.Value = pb.GetValue()
	return
}
func (this* dbPlayerPayBackData)to_pb()(pb *db.PlayerPayBack){
	pb = &db.PlayerPayBack{}
	pb.PayBackId = proto.Int32(this.PayBackId)
	pb.Value = proto.String(this.Value)
	return
}
func (this* dbPlayerPayBackData)clone_to(d *dbPlayerPayBackData){
	d.PayBackId = this.PayBackId
	d.Value = this.Value
	return
}
type dbPlayerOptionsData struct{
	Values []int32
}
func (this* dbPlayerOptionsData)from_pb(pb *db.PlayerOptions){
	if pb == nil {
		this.Values = make([]int32,0)
		return
	}
	this.Values = make([]int32,len(pb.GetValues()))
	for i, v := range pb.GetValues() {
		this.Values[i] = v
	}
	return
}
func (this* dbPlayerOptionsData)to_pb()(pb *db.PlayerOptions){
	pb = &db.PlayerOptions{}
	pb.Values = make([]int32, len(this.Values))
	for i, v := range this.Values {
		pb.Values[i]=v
	}
	return
}
func (this* dbPlayerOptionsData)clone_to(d *dbPlayerOptionsData){
	d.Values = make([]int32, len(this.Values))
	for _ii, _vv := range this.Values {
		d.Values[_ii]=_vv
	}
	return
}
type dbPlayerTaskCommonData struct{
	LastRefreshTime int32
}
func (this* dbPlayerTaskCommonData)from_pb(pb *db.PlayerTaskCommon){
	if pb == nil {
		return
	}
	this.LastRefreshTime = pb.GetLastRefreshTime()
	return
}
func (this* dbPlayerTaskCommonData)to_pb()(pb *db.PlayerTaskCommon){
	pb = &db.PlayerTaskCommon{}
	pb.LastRefreshTime = proto.Int32(this.LastRefreshTime)
	return
}
func (this* dbPlayerTaskCommonData)clone_to(d *dbPlayerTaskCommonData){
	d.LastRefreshTime = this.LastRefreshTime
	return
}
type dbPlayerTaskData struct{
	Id int32
	Value int32
	State int32
	Param int32
}
func (this* dbPlayerTaskData)from_pb(pb *db.PlayerTask){
	if pb == nil {
		return
	}
	this.Id = pb.GetId()
	this.Value = pb.GetValue()
	this.State = pb.GetState()
	this.Param = pb.GetParam()
	return
}
func (this* dbPlayerTaskData)to_pb()(pb *db.PlayerTask){
	pb = &db.PlayerTask{}
	pb.Id = proto.Int32(this.Id)
	pb.Value = proto.Int32(this.Value)
	pb.State = proto.Int32(this.State)
	pb.Param = proto.Int32(this.Param)
	return
}
func (this* dbPlayerTaskData)clone_to(d *dbPlayerTaskData){
	d.Id = this.Id
	d.Value = this.Value
	d.State = this.State
	d.Param = this.Param
	return
}
type dbPlayerFinishedTaskData struct{
	Id int32
}
func (this* dbPlayerFinishedTaskData)from_pb(pb *db.PlayerFinishedTask){
	if pb == nil {
		return
	}
	this.Id = pb.GetId()
	return
}
func (this* dbPlayerFinishedTaskData)to_pb()(pb *db.PlayerFinishedTask){
	pb = &db.PlayerFinishedTask{}
	pb.Id = proto.Int32(this.Id)
	return
}
func (this* dbPlayerFinishedTaskData)clone_to(d *dbPlayerFinishedTaskData){
	d.Id = this.Id
	return
}
type dbPlayerDailyTaskAllDailyData struct{
	CompleteTaskId int32
}
func (this* dbPlayerDailyTaskAllDailyData)from_pb(pb *db.PlayerDailyTaskAllDaily){
	if pb == nil {
		return
	}
	this.CompleteTaskId = pb.GetCompleteTaskId()
	return
}
func (this* dbPlayerDailyTaskAllDailyData)to_pb()(pb *db.PlayerDailyTaskAllDaily){
	pb = &db.PlayerDailyTaskAllDaily{}
	pb.CompleteTaskId = proto.Int32(this.CompleteTaskId)
	return
}
func (this* dbPlayerDailyTaskAllDailyData)clone_to(d *dbPlayerDailyTaskAllDailyData){
	d.CompleteTaskId = this.CompleteTaskId
	return
}
type dbPlayerSevenActivityData struct{
	ActivityId int32
	Value int32
	RewardUnix int32
}
func (this* dbPlayerSevenActivityData)from_pb(pb *db.PlayerSevenActivity){
	if pb == nil {
		return
	}
	this.ActivityId = pb.GetActivityId()
	this.Value = pb.GetValue()
	this.RewardUnix = pb.GetRewardUnix()
	return
}
func (this* dbPlayerSevenActivityData)to_pb()(pb *db.PlayerSevenActivity){
	pb = &db.PlayerSevenActivity{}
	pb.ActivityId = proto.Int32(this.ActivityId)
	pb.Value = proto.Int32(this.Value)
	pb.RewardUnix = proto.Int32(this.RewardUnix)
	return
}
func (this* dbPlayerSevenActivityData)clone_to(d *dbPlayerSevenActivityData){
	d.ActivityId = this.ActivityId
	d.Value = this.Value
	d.RewardUnix = this.RewardUnix
	return
}
type dbPlayerSignInfoData struct{
	LastSignDay int32
	CurSignSum int32
	CurSignSumMonth int32
	CurSignDays []int32
	RewardSignSum []int32
}
func (this* dbPlayerSignInfoData)from_pb(pb *db.PlayerSignInfo){
	if pb == nil {
		this.CurSignDays = make([]int32,0)
		this.RewardSignSum = make([]int32,0)
		return
	}
	this.LastSignDay = pb.GetLastSignDay()
	this.CurSignSum = pb.GetCurSignSum()
	this.CurSignSumMonth = pb.GetCurSignSumMonth()
	this.CurSignDays = make([]int32,len(pb.GetCurSignDays()))
	for i, v := range pb.GetCurSignDays() {
		this.CurSignDays[i] = v
	}
	this.RewardSignSum = make([]int32,len(pb.GetRewardSignSum()))
	for i, v := range pb.GetRewardSignSum() {
		this.RewardSignSum[i] = v
	}
	return
}
func (this* dbPlayerSignInfoData)to_pb()(pb *db.PlayerSignInfo){
	pb = &db.PlayerSignInfo{}
	pb.LastSignDay = proto.Int32(this.LastSignDay)
	pb.CurSignSum = proto.Int32(this.CurSignSum)
	pb.CurSignSumMonth = proto.Int32(this.CurSignSumMonth)
	pb.CurSignDays = make([]int32, len(this.CurSignDays))
	for i, v := range this.CurSignDays {
		pb.CurSignDays[i]=v
	}
	pb.RewardSignSum = make([]int32, len(this.RewardSignSum))
	for i, v := range this.RewardSignSum {
		pb.RewardSignSum[i]=v
	}
	return
}
func (this* dbPlayerSignInfoData)clone_to(d *dbPlayerSignInfoData){
	d.LastSignDay = this.LastSignDay
	d.CurSignSum = this.CurSignSum
	d.CurSignSumMonth = this.CurSignSumMonth
	d.CurSignDays = make([]int32, len(this.CurSignDays))
	for _ii, _vv := range this.CurSignDays {
		d.CurSignDays[_ii]=_vv
	}
	d.RewardSignSum = make([]int32, len(this.RewardSignSum))
	for _ii, _vv := range this.RewardSignSum {
		d.RewardSignSum[_ii]=_vv
	}
	return
}
type dbPlayerGuidesData struct{
	GuideId int32
	SetUnix int32
}
func (this* dbPlayerGuidesData)from_pb(pb *db.PlayerGuides){
	if pb == nil {
		return
	}
	this.GuideId = pb.GetGuideId()
	this.SetUnix = pb.GetSetUnix()
	return
}
func (this* dbPlayerGuidesData)to_pb()(pb *db.PlayerGuides){
	pb = &db.PlayerGuides{}
	pb.GuideId = proto.Int32(this.GuideId)
	pb.SetUnix = proto.Int32(this.SetUnix)
	return
}
func (this* dbPlayerGuidesData)clone_to(d *dbPlayerGuidesData){
	d.GuideId = this.GuideId
	d.SetUnix = this.SetUnix
	return
}
type dbPlayerFriendRelativeData struct{
	LastGiveFriendPointsTime int32
	GiveNumToday int32
	LastRefreshTime int32
}
func (this* dbPlayerFriendRelativeData)from_pb(pb *db.PlayerFriendRelative){
	if pb == nil {
		return
	}
	this.LastGiveFriendPointsTime = pb.GetLastGiveFriendPointsTime()
	this.GiveNumToday = pb.GetGiveNumToday()
	this.LastRefreshTime = pb.GetLastRefreshTime()
	return
}
func (this* dbPlayerFriendRelativeData)to_pb()(pb *db.PlayerFriendRelative){
	pb = &db.PlayerFriendRelative{}
	pb.LastGiveFriendPointsTime = proto.Int32(this.LastGiveFriendPointsTime)
	pb.GiveNumToday = proto.Int32(this.GiveNumToday)
	pb.LastRefreshTime = proto.Int32(this.LastRefreshTime)
	return
}
func (this* dbPlayerFriendRelativeData)clone_to(d *dbPlayerFriendRelativeData){
	d.LastGiveFriendPointsTime = this.LastGiveFriendPointsTime
	d.GiveNumToday = this.GiveNumToday
	d.LastRefreshTime = this.LastRefreshTime
	return
}
type dbPlayerFriendData struct{
	FriendId int32
	FriendName string
	Head int32
	Level int32
	VipLevel int32
	LastLogin int32
	LastGivePointsTime int32
}
func (this* dbPlayerFriendData)from_pb(pb *db.PlayerFriend){
	if pb == nil {
		return
	}
	this.FriendId = pb.GetFriendId()
	this.FriendName = pb.GetFriendName()
	this.Head = pb.GetHead()
	this.Level = pb.GetLevel()
	this.VipLevel = pb.GetVipLevel()
	this.LastLogin = pb.GetLastLogin()
	this.LastGivePointsTime = pb.GetLastGivePointsTime()
	return
}
func (this* dbPlayerFriendData)to_pb()(pb *db.PlayerFriend){
	pb = &db.PlayerFriend{}
	pb.FriendId = proto.Int32(this.FriendId)
	pb.FriendName = proto.String(this.FriendName)
	pb.Head = proto.Int32(this.Head)
	pb.Level = proto.Int32(this.Level)
	pb.VipLevel = proto.Int32(this.VipLevel)
	pb.LastLogin = proto.Int32(this.LastLogin)
	pb.LastGivePointsTime = proto.Int32(this.LastGivePointsTime)
	return
}
func (this* dbPlayerFriendData)clone_to(d *dbPlayerFriendData){
	d.FriendId = this.FriendId
	d.FriendName = this.FriendName
	d.Head = this.Head
	d.Level = this.Level
	d.VipLevel = this.VipLevel
	d.LastLogin = this.LastLogin
	d.LastGivePointsTime = this.LastGivePointsTime
	return
}
type dbPlayerFriendRecommendData struct{
	PlayerId int32
}
func (this* dbPlayerFriendRecommendData)from_pb(pb *db.PlayerFriendRecommend){
	if pb == nil {
		return
	}
	this.PlayerId = pb.GetPlayerId()
	return
}
func (this* dbPlayerFriendRecommendData)to_pb()(pb *db.PlayerFriendRecommend){
	pb = &db.PlayerFriendRecommend{}
	pb.PlayerId = proto.Int32(this.PlayerId)
	return
}
func (this* dbPlayerFriendRecommendData)clone_to(d *dbPlayerFriendRecommendData){
	d.PlayerId = this.PlayerId
	return
}
type dbPlayerFriendAskData struct{
	PlayerId int32
}
func (this* dbPlayerFriendAskData)from_pb(pb *db.PlayerFriendAsk){
	if pb == nil {
		return
	}
	this.PlayerId = pb.GetPlayerId()
	return
}
func (this* dbPlayerFriendAskData)to_pb()(pb *db.PlayerFriendAsk){
	pb = &db.PlayerFriendAsk{}
	pb.PlayerId = proto.Int32(this.PlayerId)
	return
}
func (this* dbPlayerFriendAskData)clone_to(d *dbPlayerFriendAskData){
	d.PlayerId = this.PlayerId
	return
}
type dbPlayerFriendReqData struct{
	PlayerId int32
	PlayerName string
	ReqUnix int32
}
func (this* dbPlayerFriendReqData)from_pb(pb *db.PlayerFriendReq){
	if pb == nil {
		return
	}
	this.PlayerId = pb.GetPlayerId()
	this.PlayerName = pb.GetPlayerName()
	this.ReqUnix = pb.GetReqUnix()
	return
}
func (this* dbPlayerFriendReqData)to_pb()(pb *db.PlayerFriendReq){
	pb = &db.PlayerFriendReq{}
	pb.PlayerId = proto.Int32(this.PlayerId)
	pb.PlayerName = proto.String(this.PlayerName)
	pb.ReqUnix = proto.Int32(this.ReqUnix)
	return
}
func (this* dbPlayerFriendReqData)clone_to(d *dbPlayerFriendReqData){
	d.PlayerId = this.PlayerId
	d.PlayerName = this.PlayerName
	d.ReqUnix = this.ReqUnix
	return
}
type dbPlayerFriendPointData struct{
	FromPlayerId int32
	GivePoints int32
	LastGiveTime int32
	IsTodayGive int32
}
func (this* dbPlayerFriendPointData)from_pb(pb *db.PlayerFriendPoint){
	if pb == nil {
		return
	}
	this.FromPlayerId = pb.GetFromPlayerId()
	this.GivePoints = pb.GetGivePoints()
	this.LastGiveTime = pb.GetLastGiveTime()
	this.IsTodayGive = pb.GetIsTodayGive()
	return
}
func (this* dbPlayerFriendPointData)to_pb()(pb *db.PlayerFriendPoint){
	pb = &db.PlayerFriendPoint{}
	pb.FromPlayerId = proto.Int32(this.FromPlayerId)
	pb.GivePoints = proto.Int32(this.GivePoints)
	pb.LastGiveTime = proto.Int32(this.LastGiveTime)
	pb.IsTodayGive = proto.Int32(this.IsTodayGive)
	return
}
func (this* dbPlayerFriendPointData)clone_to(d *dbPlayerFriendPointData){
	d.FromPlayerId = this.FromPlayerId
	d.GivePoints = this.GivePoints
	d.LastGiveTime = this.LastGiveTime
	d.IsTodayGive = this.IsTodayGive
	return
}
type dbPlayerFriendChatUnreadIdData struct{
	FriendId int32
	MessageIds []int32
	CurrMessageId int32
}
func (this* dbPlayerFriendChatUnreadIdData)from_pb(pb *db.PlayerFriendChatUnreadId){
	if pb == nil {
		this.MessageIds = make([]int32,0)
		return
	}
	this.FriendId = pb.GetFriendId()
	this.MessageIds = make([]int32,len(pb.GetMessageIds()))
	for i, v := range pb.GetMessageIds() {
		this.MessageIds[i] = v
	}
	this.CurrMessageId = pb.GetCurrMessageId()
	return
}
func (this* dbPlayerFriendChatUnreadIdData)to_pb()(pb *db.PlayerFriendChatUnreadId){
	pb = &db.PlayerFriendChatUnreadId{}
	pb.FriendId = proto.Int32(this.FriendId)
	pb.MessageIds = make([]int32, len(this.MessageIds))
	for i, v := range this.MessageIds {
		pb.MessageIds[i]=v
	}
	pb.CurrMessageId = proto.Int32(this.CurrMessageId)
	return
}
func (this* dbPlayerFriendChatUnreadIdData)clone_to(d *dbPlayerFriendChatUnreadIdData){
	d.FriendId = this.FriendId
	d.MessageIds = make([]int32, len(this.MessageIds))
	for _ii, _vv := range this.MessageIds {
		d.MessageIds[_ii]=_vv
	}
	d.CurrMessageId = this.CurrMessageId
	return
}
type dbPlayerFriendChatUnreadMessageData struct{
	PlayerMessageId int64
	Message []byte
	SendTime int32
	IsRead int32
}
func (this* dbPlayerFriendChatUnreadMessageData)from_pb(pb *db.PlayerFriendChatUnreadMessage){
	if pb == nil {
		return
	}
	this.PlayerMessageId = pb.GetPlayerMessageId()
	this.Message = pb.GetMessage()
	this.SendTime = pb.GetSendTime()
	this.IsRead = pb.GetIsRead()
	return
}
func (this* dbPlayerFriendChatUnreadMessageData)to_pb()(pb *db.PlayerFriendChatUnreadMessage){
	pb = &db.PlayerFriendChatUnreadMessage{}
	pb.PlayerMessageId = proto.Int64(this.PlayerMessageId)
	pb.Message = this.Message
	pb.SendTime = proto.Int32(this.SendTime)
	pb.IsRead = proto.Int32(this.IsRead)
	return
}
func (this* dbPlayerFriendChatUnreadMessageData)clone_to(d *dbPlayerFriendChatUnreadMessageData){
	d.PlayerMessageId = this.PlayerMessageId
	d.Message = make([]byte, len(this.Message))
	for _ii, _vv := range this.Message {
		d.Message[_ii]=_vv
	}
	d.SendTime = this.SendTime
	d.IsRead = this.IsRead
	return
}
type dbPlayerFocusPlayerData struct{
	FriendId int32
	FriendName string
}
func (this* dbPlayerFocusPlayerData)from_pb(pb *db.PlayerFocusPlayer){
	if pb == nil {
		return
	}
	this.FriendId = pb.GetFriendId()
	this.FriendName = pb.GetFriendName()
	return
}
func (this* dbPlayerFocusPlayerData)to_pb()(pb *db.PlayerFocusPlayer){
	pb = &db.PlayerFocusPlayer{}
	pb.FriendId = proto.Int32(this.FriendId)
	pb.FriendName = proto.String(this.FriendName)
	return
}
func (this* dbPlayerFocusPlayerData)clone_to(d *dbPlayerFocusPlayerData){
	d.FriendId = this.FriendId
	d.FriendName = this.FriendName
	return
}
type dbPlayerBeFocusPlayerData struct{
	FriendId int32
	FriendName string
}
func (this* dbPlayerBeFocusPlayerData)from_pb(pb *db.PlayerBeFocusPlayer){
	if pb == nil {
		return
	}
	this.FriendId = pb.GetFriendId()
	this.FriendName = pb.GetFriendName()
	return
}
func (this* dbPlayerBeFocusPlayerData)to_pb()(pb *db.PlayerBeFocusPlayer){
	pb = &db.PlayerBeFocusPlayer{}
	pb.FriendId = proto.Int32(this.FriendId)
	pb.FriendName = proto.String(this.FriendName)
	return
}
func (this* dbPlayerBeFocusPlayerData)clone_to(d *dbPlayerBeFocusPlayerData){
	d.FriendId = this.FriendId
	d.FriendName = this.FriendName
	return
}
type dbPlayerCustomDataData struct{
	CustomData []byte
}
func (this* dbPlayerCustomDataData)from_pb(pb *db.PlayerCustomData){
	if pb == nil {
		return
	}
	this.CustomData = pb.GetCustomData()
	return
}
func (this* dbPlayerCustomDataData)to_pb()(pb *db.PlayerCustomData){
	pb = &db.PlayerCustomData{}
	pb.CustomData = this.CustomData
	return
}
func (this* dbPlayerCustomDataData)clone_to(d *dbPlayerCustomDataData){
	d.CustomData = make([]byte, len(this.CustomData))
	for _ii, _vv := range this.CustomData {
		d.CustomData[_ii]=_vv
	}
	return
}
type dbPlayerChaterOpenRequestData struct{
	CustomData []byte
}
func (this* dbPlayerChaterOpenRequestData)from_pb(pb *db.PlayerChaterOpenRequest){
	if pb == nil {
		return
	}
	this.CustomData = pb.GetCustomData()
	return
}
func (this* dbPlayerChaterOpenRequestData)to_pb()(pb *db.PlayerChaterOpenRequest){
	pb = &db.PlayerChaterOpenRequest{}
	pb.CustomData = this.CustomData
	return
}
func (this* dbPlayerChaterOpenRequestData)clone_to(d *dbPlayerChaterOpenRequestData){
	d.CustomData = make([]byte, len(this.CustomData))
	for _ii, _vv := range this.CustomData {
		d.CustomData[_ii]=_vv
	}
	return
}
type dbPlayerExpeditionData struct{
	Id int32
	TaskId int32
	StartUnix int32
	EndUnix int32
	InCatIds []int32
	CurState int32
	Result int32
	TaskLeftSec int32
	TaskLeftSecLastUpUnix int32
	Conditions []dbExpeditionConData
	EventIds []dbExpeditionEventData
	TotalSpecials int32
}
func (this* dbPlayerExpeditionData)from_pb(pb *db.PlayerExpedition){
	if pb == nil {
		this.InCatIds = make([]int32,0)
		this.Conditions = make([]dbExpeditionConData,0)
		this.EventIds = make([]dbExpeditionEventData,0)
		return
	}
	this.Id = pb.GetId()
	this.TaskId = pb.GetTaskId()
	this.StartUnix = pb.GetStartUnix()
	this.EndUnix = pb.GetEndUnix()
	this.InCatIds = make([]int32,len(pb.GetInCatIds()))
	for i, v := range pb.GetInCatIds() {
		this.InCatIds[i] = v
	}
	this.CurState = pb.GetCurState()
	this.Result = pb.GetResult()
	this.TaskLeftSec = pb.GetTaskLeftSec()
	this.TaskLeftSecLastUpUnix = pb.GetTaskLeftSecLastUpUnix()
	this.Conditions = make([]dbExpeditionConData,len(pb.GetConditions()))
	for i, v := range pb.GetConditions() {
		this.Conditions[i].from_pb(v)
	}
	this.EventIds = make([]dbExpeditionEventData,len(pb.GetEventIds()))
	for i, v := range pb.GetEventIds() {
		this.EventIds[i].from_pb(v)
	}
	this.TotalSpecials = pb.GetTotalSpecials()
	return
}
func (this* dbPlayerExpeditionData)to_pb()(pb *db.PlayerExpedition){
	pb = &db.PlayerExpedition{}
	pb.Id = proto.Int32(this.Id)
	pb.TaskId = proto.Int32(this.TaskId)
	pb.StartUnix = proto.Int32(this.StartUnix)
	pb.EndUnix = proto.Int32(this.EndUnix)
	pb.InCatIds = make([]int32, len(this.InCatIds))
	for i, v := range this.InCatIds {
		pb.InCatIds[i]=v
	}
	pb.CurState = proto.Int32(this.CurState)
	pb.Result = proto.Int32(this.Result)
	pb.TaskLeftSec = proto.Int32(this.TaskLeftSec)
	pb.TaskLeftSecLastUpUnix = proto.Int32(this.TaskLeftSecLastUpUnix)
	pb.Conditions = make([]*db.ExpeditionCon, len(this.Conditions))
	for i, v := range this.Conditions {
		pb.Conditions[i]=v.to_pb()
	}
	pb.EventIds = make([]*db.ExpeditionEvent, len(this.EventIds))
	for i, v := range this.EventIds {
		pb.EventIds[i]=v.to_pb()
	}
	pb.TotalSpecials = proto.Int32(this.TotalSpecials)
	return
}
func (this* dbPlayerExpeditionData)clone_to(d *dbPlayerExpeditionData){
	d.Id = this.Id
	d.TaskId = this.TaskId
	d.StartUnix = this.StartUnix
	d.EndUnix = this.EndUnix
	d.InCatIds = make([]int32, len(this.InCatIds))
	for _ii, _vv := range this.InCatIds {
		d.InCatIds[_ii]=_vv
	}
	d.CurState = this.CurState
	d.Result = this.Result
	d.TaskLeftSec = this.TaskLeftSec
	d.TaskLeftSecLastUpUnix = this.TaskLeftSecLastUpUnix
	d.Conditions = make([]dbExpeditionConData, len(this.Conditions))
	for _ii, _vv := range this.Conditions {
		_vv.clone_to(&d.Conditions[_ii])
	}
	d.EventIds = make([]dbExpeditionEventData, len(this.EventIds))
	for _ii, _vv := range this.EventIds {
		_vv.clone_to(&d.EventIds[_ii])
	}
	d.TotalSpecials = this.TotalSpecials
	return
}
type dbPlayerHandbookItemData struct{
	Id int32
}
func (this* dbPlayerHandbookItemData)from_pb(pb *db.PlayerHandbookItem){
	if pb == nil {
		return
	}
	this.Id = pb.GetId()
	return
}
func (this* dbPlayerHandbookItemData)to_pb()(pb *db.PlayerHandbookItem){
	pb = &db.PlayerHandbookItem{}
	pb.Id = proto.Int32(this.Id)
	return
}
func (this* dbPlayerHandbookItemData)clone_to(d *dbPlayerHandbookItemData){
	d.Id = this.Id
	return
}
type dbPlayerHeadItemData struct{
	Id int32
}
func (this* dbPlayerHeadItemData)from_pb(pb *db.PlayerHeadItem){
	if pb == nil {
		return
	}
	this.Id = pb.GetId()
	return
}
func (this* dbPlayerHeadItemData)to_pb()(pb *db.PlayerHeadItem){
	pb = &db.PlayerHeadItem{}
	pb.Id = proto.Int32(this.Id)
	return
}
func (this* dbPlayerHeadItemData)clone_to(d *dbPlayerHeadItemData){
	d.Id = this.Id
	return
}
type dbPlayerActivityData struct{
	CfgId int32
	States []int32
	Vals []int32
}
func (this* dbPlayerActivityData)from_pb(pb *db.PlayerActivity){
	if pb == nil {
		this.States = make([]int32,0)
		this.Vals = make([]int32,0)
		return
	}
	this.CfgId = pb.GetCfgId()
	this.States = make([]int32,len(pb.GetStates()))
	for i, v := range pb.GetStates() {
		this.States[i] = v
	}
	this.Vals = make([]int32,len(pb.GetVals()))
	for i, v := range pb.GetVals() {
		this.Vals[i] = v
	}
	return
}
func (this* dbPlayerActivityData)to_pb()(pb *db.PlayerActivity){
	pb = &db.PlayerActivity{}
	pb.CfgId = proto.Int32(this.CfgId)
	pb.States = make([]int32, len(this.States))
	for i, v := range this.States {
		pb.States[i]=v
	}
	pb.Vals = make([]int32, len(this.Vals))
	for i, v := range this.Vals {
		pb.Vals[i]=v
	}
	return
}
func (this* dbPlayerActivityData)clone_to(d *dbPlayerActivityData){
	d.CfgId = this.CfgId
	d.States = make([]int32, len(this.States))
	for _ii, _vv := range this.States {
		d.States[_ii]=_vv
	}
	d.Vals = make([]int32, len(this.Vals))
	for _ii, _vv := range this.Vals {
		d.Vals[_ii]=_vv
	}
	return
}
type dbPlayerSuitAwardData struct{
	Id int32
	AwardTime int32
}
func (this* dbPlayerSuitAwardData)from_pb(pb *db.PlayerSuitAward){
	if pb == nil {
		return
	}
	this.Id = pb.GetId()
	this.AwardTime = pb.GetAwardTime()
	return
}
func (this* dbPlayerSuitAwardData)to_pb()(pb *db.PlayerSuitAward){
	pb = &db.PlayerSuitAward{}
	pb.Id = proto.Int32(this.Id)
	pb.AwardTime = proto.Int32(this.AwardTime)
	return
}
func (this* dbPlayerSuitAwardData)clone_to(d *dbPlayerSuitAwardData){
	d.Id = this.Id
	d.AwardTime = this.AwardTime
	return
}
type dbPlayerZanData struct{
	PlayerId int32
	ZanTime int32
	ZanNum int32
}
func (this* dbPlayerZanData)from_pb(pb *db.PlayerZan){
	if pb == nil {
		return
	}
	this.PlayerId = pb.GetPlayerId()
	this.ZanTime = pb.GetZanTime()
	this.ZanNum = pb.GetZanNum()
	return
}
func (this* dbPlayerZanData)to_pb()(pb *db.PlayerZan){
	pb = &db.PlayerZan{}
	pb.PlayerId = proto.Int32(this.PlayerId)
	pb.ZanTime = proto.Int32(this.ZanTime)
	pb.ZanNum = proto.Int32(this.ZanNum)
	return
}
func (this* dbPlayerZanData)clone_to(d *dbPlayerZanData){
	d.PlayerId = this.PlayerId
	d.ZanTime = this.ZanTime
	d.ZanNum = this.ZanNum
	return
}
type dbPlayerFosterData struct{
	BuildingId int32
	EquippedCardId int32
	StartTime int32
	CatIds []int32
	PlayerCatIds []int64
}
func (this* dbPlayerFosterData)from_pb(pb *db.PlayerFoster){
	if pb == nil {
		this.CatIds = make([]int32,0)
		this.PlayerCatIds = make([]int64,0)
		return
	}
	this.BuildingId = pb.GetBuildingId()
	this.EquippedCardId = pb.GetEquippedCardId()
	this.StartTime = pb.GetStartTime()
	this.CatIds = make([]int32,len(pb.GetCatIds()))
	for i, v := range pb.GetCatIds() {
		this.CatIds[i] = v
	}
	this.PlayerCatIds = make([]int64,len(pb.GetPlayerCatIds()))
	for i, v := range pb.GetPlayerCatIds() {
		this.PlayerCatIds[i] = v
	}
	return
}
func (this* dbPlayerFosterData)to_pb()(pb *db.PlayerFoster){
	pb = &db.PlayerFoster{}
	pb.BuildingId = proto.Int32(this.BuildingId)
	pb.EquippedCardId = proto.Int32(this.EquippedCardId)
	pb.StartTime = proto.Int32(this.StartTime)
	pb.CatIds = make([]int32, len(this.CatIds))
	for i, v := range this.CatIds {
		pb.CatIds[i]=v
	}
	pb.PlayerCatIds = make([]int64, len(this.PlayerCatIds))
	for i, v := range this.PlayerCatIds {
		pb.PlayerCatIds[i]=v
	}
	return
}
func (this* dbPlayerFosterData)clone_to(d *dbPlayerFosterData){
	d.BuildingId = this.BuildingId
	d.EquippedCardId = this.EquippedCardId
	d.StartTime = this.StartTime
	d.CatIds = make([]int32, len(this.CatIds))
	for _ii, _vv := range this.CatIds {
		d.CatIds[_ii]=_vv
	}
	d.PlayerCatIds = make([]int64, len(this.PlayerCatIds))
	for _ii, _vv := range this.PlayerCatIds {
		d.PlayerCatIds[_ii]=_vv
	}
	return
}
type dbPlayerFosterCatData struct{
	CatId int32
	StartTime int32
	RemainSeconds int32
}
func (this* dbPlayerFosterCatData)from_pb(pb *db.PlayerFosterCat){
	if pb == nil {
		return
	}
	this.CatId = pb.GetCatId()
	this.StartTime = pb.GetStartTime()
	this.RemainSeconds = pb.GetRemainSeconds()
	return
}
func (this* dbPlayerFosterCatData)to_pb()(pb *db.PlayerFosterCat){
	pb = &db.PlayerFosterCat{}
	pb.CatId = proto.Int32(this.CatId)
	pb.StartTime = proto.Int32(this.StartTime)
	pb.RemainSeconds = proto.Int32(this.RemainSeconds)
	return
}
func (this* dbPlayerFosterCatData)clone_to(d *dbPlayerFosterCatData){
	d.CatId = this.CatId
	d.StartTime = this.StartTime
	d.RemainSeconds = this.RemainSeconds
	return
}
type dbPlayerFosterCatOnFriendData struct{
	CatId int32
	FriendId int32
}
func (this* dbPlayerFosterCatOnFriendData)from_pb(pb *db.PlayerFosterCatOnFriend){
	if pb == nil {
		return
	}
	this.CatId = pb.GetCatId()
	this.FriendId = pb.GetFriendId()
	return
}
func (this* dbPlayerFosterCatOnFriendData)to_pb()(pb *db.PlayerFosterCatOnFriend){
	pb = &db.PlayerFosterCatOnFriend{}
	pb.CatId = proto.Int32(this.CatId)
	pb.FriendId = proto.Int32(this.FriendId)
	return
}
func (this* dbPlayerFosterCatOnFriendData)clone_to(d *dbPlayerFosterCatOnFriendData){
	d.CatId = this.CatId
	d.FriendId = this.FriendId
	return
}
type dbPlayerFosterFriendCatData struct{
	PlayerId int32
	CatId int32
	CatTableId int32
	StartTime int32
	StartCardId int32
	PlayerName string
	PlayerLevel int32
	PlayerHead int32
	CatLevel int32
	CatStar int32
	CatNick string
}
func (this* dbPlayerFosterFriendCatData)from_pb(pb *db.PlayerFosterFriendCat){
	if pb == nil {
		return
	}
	this.PlayerId = pb.GetPlayerId()
	this.CatId = pb.GetCatId()
	this.CatTableId = pb.GetCatTableId()
	this.StartTime = pb.GetStartTime()
	this.StartCardId = pb.GetStartCardId()
	this.PlayerName = pb.GetPlayerName()
	this.PlayerLevel = pb.GetPlayerLevel()
	this.PlayerHead = pb.GetPlayerHead()
	this.CatLevel = pb.GetCatLevel()
	this.CatStar = pb.GetCatStar()
	this.CatNick = pb.GetCatNick()
	return
}
func (this* dbPlayerFosterFriendCatData)to_pb()(pb *db.PlayerFosterFriendCat){
	pb = &db.PlayerFosterFriendCat{}
	pb.PlayerId = proto.Int32(this.PlayerId)
	pb.CatId = proto.Int32(this.CatId)
	pb.CatTableId = proto.Int32(this.CatTableId)
	pb.StartTime = proto.Int32(this.StartTime)
	pb.StartCardId = proto.Int32(this.StartCardId)
	pb.PlayerName = proto.String(this.PlayerName)
	pb.PlayerLevel = proto.Int32(this.PlayerLevel)
	pb.PlayerHead = proto.Int32(this.PlayerHead)
	pb.CatLevel = proto.Int32(this.CatLevel)
	pb.CatStar = proto.Int32(this.CatStar)
	pb.CatNick = proto.String(this.CatNick)
	return
}
func (this* dbPlayerFosterFriendCatData)clone_to(d *dbPlayerFosterFriendCatData){
	d.PlayerId = this.PlayerId
	d.CatId = this.CatId
	d.CatTableId = this.CatTableId
	d.StartTime = this.StartTime
	d.StartCardId = this.StartCardId
	d.PlayerName = this.PlayerName
	d.PlayerLevel = this.PlayerLevel
	d.PlayerHead = this.PlayerHead
	d.CatLevel = this.CatLevel
	d.CatStar = this.CatStar
	d.CatNick = this.CatNick
	return
}
type dbPlayerChatData struct{
	Channel int32
	LastChatTime int32
	LastPullTime int32
	LastMsgIndex int32
}
func (this* dbPlayerChatData)from_pb(pb *db.PlayerChat){
	if pb == nil {
		return
	}
	this.Channel = pb.GetChannel()
	this.LastChatTime = pb.GetLastChatTime()
	this.LastPullTime = pb.GetLastPullTime()
	this.LastMsgIndex = pb.GetLastMsgIndex()
	return
}
func (this* dbPlayerChatData)to_pb()(pb *db.PlayerChat){
	pb = &db.PlayerChat{}
	pb.Channel = proto.Int32(this.Channel)
	pb.LastChatTime = proto.Int32(this.LastChatTime)
	pb.LastPullTime = proto.Int32(this.LastPullTime)
	pb.LastMsgIndex = proto.Int32(this.LastMsgIndex)
	return
}
func (this* dbPlayerChatData)clone_to(d *dbPlayerChatData){
	d.Channel = this.Channel
	d.LastChatTime = this.LastChatTime
	d.LastPullTime = this.LastPullTime
	d.LastMsgIndex = this.LastMsgIndex
	return
}
type dbPlayerAnouncementData struct{
	LastSendTime int32
}
func (this* dbPlayerAnouncementData)from_pb(pb *db.PlayerAnouncement){
	if pb == nil {
		return
	}
	this.LastSendTime = pb.GetLastSendTime()
	return
}
func (this* dbPlayerAnouncementData)to_pb()(pb *db.PlayerAnouncement){
	pb = &db.PlayerAnouncement{}
	pb.LastSendTime = proto.Int32(this.LastSendTime)
	return
}
func (this* dbPlayerAnouncementData)clone_to(d *dbPlayerAnouncementData){
	d.LastSendTime = this.LastSendTime
	return
}
type dbPlayerFirstDrawCardData struct{
	Id int32
	Drawed int32
}
func (this* dbPlayerFirstDrawCardData)from_pb(pb *db.PlayerFirstDrawCard){
	if pb == nil {
		return
	}
	this.Id = pb.GetId()
	this.Drawed = pb.GetDrawed()
	return
}
func (this* dbPlayerFirstDrawCardData)to_pb()(pb *db.PlayerFirstDrawCard){
	pb = &db.PlayerFirstDrawCard{}
	pb.Id = proto.Int32(this.Id)
	pb.Drawed = proto.Int32(this.Drawed)
	return
}
func (this* dbPlayerFirstDrawCardData)clone_to(d *dbPlayerFirstDrawCardData){
	d.Id = this.Id
	d.Drawed = this.Drawed
	return
}
type dbPlayerTalkForbidData struct{
	EndUnix int32
	ForbidReason string
}
func (this* dbPlayerTalkForbidData)from_pb(pb *db.PlayerTalkForbid){
	if pb == nil {
		return
	}
	this.EndUnix = pb.GetEndUnix()
	this.ForbidReason = pb.GetForbidReason()
	return
}
func (this* dbPlayerTalkForbidData)to_pb()(pb *db.PlayerTalkForbid){
	pb = &db.PlayerTalkForbid{}
	pb.EndUnix = proto.Int32(this.EndUnix)
	pb.ForbidReason = proto.String(this.ForbidReason)
	return
}
func (this* dbPlayerTalkForbidData)clone_to(d *dbPlayerTalkForbidData){
	d.EndUnix = this.EndUnix
	d.ForbidReason = this.ForbidReason
	return
}
type dbPlayerServerRewardData struct{
	RewardId int32
	EndUnix int32
}
func (this* dbPlayerServerRewardData)from_pb(pb *db.PlayerServerReward){
	if pb == nil {
		return
	}
	this.RewardId = pb.GetRewardId()
	this.EndUnix = pb.GetEndUnix()
	return
}
func (this* dbPlayerServerRewardData)to_pb()(pb *db.PlayerServerReward){
	pb = &db.PlayerServerReward{}
	pb.RewardId = proto.Int32(this.RewardId)
	pb.EndUnix = proto.Int32(this.EndUnix)
	return
}
func (this* dbPlayerServerRewardData)clone_to(d *dbPlayerServerRewardData){
	d.RewardId = this.RewardId
	d.EndUnix = this.EndUnix
	return
}
type dbPlayerMailCommonData struct{
	CurrId int32
	LastSendPlayerMailTime int32
}
func (this* dbPlayerMailCommonData)from_pb(pb *db.PlayerMailCommon){
	if pb == nil {
		return
	}
	this.CurrId = pb.GetCurrId()
	this.LastSendPlayerMailTime = pb.GetLastSendPlayerMailTime()
	return
}
func (this* dbPlayerMailCommonData)to_pb()(pb *db.PlayerMailCommon){
	pb = &db.PlayerMailCommon{}
	pb.CurrId = proto.Int32(this.CurrId)
	pb.LastSendPlayerMailTime = proto.Int32(this.LastSendPlayerMailTime)
	return
}
func (this* dbPlayerMailCommonData)clone_to(d *dbPlayerMailCommonData){
	d.CurrId = this.CurrId
	d.LastSendPlayerMailTime = this.LastSendPlayerMailTime
	return
}
type dbPlayerMailData struct{
	Id int32
	Type int8
	Title string
	Content string
	SendUnix int32
	AttachItemIds []int32
	AttachItemNums []int32
	IsRead int32
	IsGetAttached int32
	SenderId int32
	SenderName string
	Subtype int32
	ExtraValue int32
}
func (this* dbPlayerMailData)from_pb(pb *db.PlayerMail){
	if pb == nil {
		this.AttachItemIds = make([]int32,0)
		this.AttachItemNums = make([]int32,0)
		return
	}
	this.Id = pb.GetId()
	this.Type = int8(pb.GetType())
	this.Title = pb.GetTitle()
	this.Content = pb.GetContent()
	this.SendUnix = pb.GetSendUnix()
	this.AttachItemIds = make([]int32,len(pb.GetAttachItemIds()))
	for i, v := range pb.GetAttachItemIds() {
		this.AttachItemIds[i] = v
	}
	this.AttachItemNums = make([]int32,len(pb.GetAttachItemNums()))
	for i, v := range pb.GetAttachItemNums() {
		this.AttachItemNums[i] = v
	}
	this.IsRead = pb.GetIsRead()
	this.IsGetAttached = pb.GetIsGetAttached()
	this.SenderId = pb.GetSenderId()
	this.SenderName = pb.GetSenderName()
	this.Subtype = pb.GetSubtype()
	this.ExtraValue = pb.GetExtraValue()
	return
}
func (this* dbPlayerMailData)to_pb()(pb *db.PlayerMail){
	pb = &db.PlayerMail{}
	pb.Id = proto.Int32(this.Id)
	temp_Type:=int32(this.Type)
	pb.Type = proto.Int32(temp_Type)
	pb.Title = proto.String(this.Title)
	pb.Content = proto.String(this.Content)
	pb.SendUnix = proto.Int32(this.SendUnix)
	pb.AttachItemIds = make([]int32, len(this.AttachItemIds))
	for i, v := range this.AttachItemIds {
		pb.AttachItemIds[i]=v
	}
	pb.AttachItemNums = make([]int32, len(this.AttachItemNums))
	for i, v := range this.AttachItemNums {
		pb.AttachItemNums[i]=v
	}
	pb.IsRead = proto.Int32(this.IsRead)
	pb.IsGetAttached = proto.Int32(this.IsGetAttached)
	pb.SenderId = proto.Int32(this.SenderId)
	pb.SenderName = proto.String(this.SenderName)
	pb.Subtype = proto.Int32(this.Subtype)
	pb.ExtraValue = proto.Int32(this.ExtraValue)
	return
}
func (this* dbPlayerMailData)clone_to(d *dbPlayerMailData){
	d.Id = this.Id
	d.Type = int8(this.Type)
	d.Title = this.Title
	d.Content = this.Content
	d.SendUnix = this.SendUnix
	d.AttachItemIds = make([]int32, len(this.AttachItemIds))
	for _ii, _vv := range this.AttachItemIds {
		d.AttachItemIds[_ii]=_vv
	}
	d.AttachItemNums = make([]int32, len(this.AttachItemNums))
	for _ii, _vv := range this.AttachItemNums {
		d.AttachItemNums[_ii]=_vv
	}
	d.IsRead = this.IsRead
	d.IsGetAttached = this.IsGetAttached
	d.SenderId = this.SenderId
	d.SenderName = this.SenderName
	d.Subtype = this.Subtype
	d.ExtraValue = this.ExtraValue
	return
}
type dbPlayerPayCommonData struct{
	FirstPayState int32
}
func (this* dbPlayerPayCommonData)from_pb(pb *db.PlayerPayCommon){
	if pb == nil {
		return
	}
	this.FirstPayState = pb.GetFirstPayState()
	return
}
func (this* dbPlayerPayCommonData)to_pb()(pb *db.PlayerPayCommon){
	pb = &db.PlayerPayCommon{}
	pb.FirstPayState = proto.Int32(this.FirstPayState)
	return
}
func (this* dbPlayerPayCommonData)clone_to(d *dbPlayerPayCommonData){
	d.FirstPayState = this.FirstPayState
	return
}
type dbPlayerPayData struct{
	BundleId string
	LastPayedTime int32
	LastAwardTime int32
	SendMailNum int32
	ChargeNum int32
}
func (this* dbPlayerPayData)from_pb(pb *db.PlayerPay){
	if pb == nil {
		return
	}
	this.BundleId = pb.GetBundleId()
	this.LastPayedTime = pb.GetLastPayedTime()
	this.LastAwardTime = pb.GetLastAwardTime()
	this.SendMailNum = pb.GetSendMailNum()
	this.ChargeNum = pb.GetChargeNum()
	return
}
func (this* dbPlayerPayData)to_pb()(pb *db.PlayerPay){
	pb = &db.PlayerPay{}
	pb.BundleId = proto.String(this.BundleId)
	pb.LastPayedTime = proto.Int32(this.LastPayedTime)
	pb.LastAwardTime = proto.Int32(this.LastAwardTime)
	pb.SendMailNum = proto.Int32(this.SendMailNum)
	pb.ChargeNum = proto.Int32(this.ChargeNum)
	return
}
func (this* dbPlayerPayData)clone_to(d *dbPlayerPayData){
	d.BundleId = this.BundleId
	d.LastPayedTime = this.LastPayedTime
	d.LastAwardTime = this.LastAwardTime
	d.SendMailNum = this.SendMailNum
	d.ChargeNum = this.ChargeNum
	return
}
type dbPlayerGuideDataData struct{
	Data []byte
}
func (this* dbPlayerGuideDataData)from_pb(pb *db.PlayerGuideData){
	if pb == nil {
		return
	}
	this.Data = pb.GetData()
	return
}
func (this* dbPlayerGuideDataData)to_pb()(pb *db.PlayerGuideData){
	pb = &db.PlayerGuideData{}
	pb.Data = this.Data
	return
}
func (this* dbPlayerGuideDataData)clone_to(d *dbPlayerGuideDataData){
	d.Data = make([]byte, len(this.Data))
	for _ii, _vv := range this.Data {
		d.Data[_ii]=_vv
	}
	return
}
type dbPlayerActivityDataData struct{
	Id int32
	SubIds []int32
	SubValues []int32
	SubNum int32
}
func (this* dbPlayerActivityDataData)from_pb(pb *db.PlayerActivityData){
	if pb == nil {
		this.SubIds = make([]int32,0)
		this.SubValues = make([]int32,0)
		return
	}
	this.Id = pb.GetId()
	this.SubIds = make([]int32,len(pb.GetSubIds()))
	for i, v := range pb.GetSubIds() {
		this.SubIds[i] = v
	}
	this.SubValues = make([]int32,len(pb.GetSubValues()))
	for i, v := range pb.GetSubValues() {
		this.SubValues[i] = v
	}
	this.SubNum = pb.GetSubNum()
	return
}
func (this* dbPlayerActivityDataData)to_pb()(pb *db.PlayerActivityData){
	pb = &db.PlayerActivityData{}
	pb.Id = proto.Int32(this.Id)
	pb.SubIds = make([]int32, len(this.SubIds))
	for i, v := range this.SubIds {
		pb.SubIds[i]=v
	}
	pb.SubValues = make([]int32, len(this.SubValues))
	for i, v := range this.SubValues {
		pb.SubValues[i]=v
	}
	pb.SubNum = proto.Int32(this.SubNum)
	return
}
func (this* dbPlayerActivityDataData)clone_to(d *dbPlayerActivityDataData){
	d.Id = this.Id
	d.SubIds = make([]int32, len(this.SubIds))
	for _ii, _vv := range this.SubIds {
		d.SubIds[_ii]=_vv
	}
	d.SubValues = make([]int32, len(this.SubValues))
	for _ii, _vv := range this.SubValues {
		d.SubValues[_ii]=_vv
	}
	d.SubNum = this.SubNum
	return
}
type dbPlayerSysMailData struct{
	CurrId int32
}
func (this* dbPlayerSysMailData)from_pb(pb *db.PlayerSysMail){
	if pb == nil {
		return
	}
	this.CurrId = pb.GetCurrId()
	return
}
func (this* dbPlayerSysMailData)to_pb()(pb *db.PlayerSysMail){
	pb = &db.PlayerSysMail{}
	pb.CurrId = proto.Int32(this.CurrId)
	return
}
func (this* dbPlayerSysMailData)clone_to(d *dbPlayerSysMailData){
	d.CurrId = this.CurrId
	return
}
type dbSysMailAttachedItemsData struct{
	ItemList []int32
}
func (this* dbSysMailAttachedItemsData)from_pb(pb *db.SysMailAttachedItems){
	if pb == nil {
		this.ItemList = make([]int32,0)
		return
	}
	this.ItemList = make([]int32,len(pb.GetItemList()))
	for i, v := range pb.GetItemList() {
		this.ItemList[i] = v
	}
	return
}
func (this* dbSysMailAttachedItemsData)to_pb()(pb *db.SysMailAttachedItems){
	pb = &db.SysMailAttachedItems{}
	pb.ItemList = make([]int32, len(this.ItemList))
	for i, v := range this.ItemList {
		pb.ItemList[i]=v
	}
	return
}
func (this* dbSysMailAttachedItemsData)clone_to(d *dbSysMailAttachedItemsData){
	d.ItemList = make([]int32, len(this.ItemList))
	for _ii, _vv := range this.ItemList {
		d.ItemList[_ii]=_vv
	}
	return
}

func (this *dbGlobalRow)GetCurrentPlayerId( )(r int32 ){
	this.m_lock.UnSafeRLock("dbGlobalRow.GetdbGlobalCurrentPlayerIdColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_CurrentPlayerId)
}
func (this *dbGlobalRow)SetCurrentPlayerId(v int32){
	this.m_lock.UnSafeLock("dbGlobalRow.SetdbGlobalCurrentPlayerIdColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_CurrentPlayerId=int32(v)
	this.m_CurrentPlayerId_changed=true
	return
}
func (this *dbGlobalRow)GetCurrentGuildId( )(r int32 ){
	this.m_lock.UnSafeRLock("dbGlobalRow.GetdbGlobalCurrentGuildIdColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_CurrentGuildId)
}
func (this *dbGlobalRow)SetCurrentGuildId(v int32){
	this.m_lock.UnSafeLock("dbGlobalRow.SetdbGlobalCurrentGuildIdColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_CurrentGuildId=int32(v)
	this.m_CurrentGuildId_changed=true
	return
}
type dbGlobalRow struct {
	m_table *dbGlobalTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_Id        int32
	m_CurrentPlayerId_changed bool
	m_CurrentPlayerId int32
	m_CurrentGuildId_changed bool
	m_CurrentGuildId int32
}
func new_dbGlobalRow(table *dbGlobalTable, Id int32) (r *dbGlobalRow) {
	this := &dbGlobalRow{}
	this.m_table = table
	this.m_Id = Id
	this.m_lock = NewRWMutex()
	this.m_CurrentPlayerId_changed=true
	this.m_CurrentGuildId_changed=true
	return this
}
func (this *dbGlobalRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbGlobalRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(3)
		db_args.Push(this.m_Id)
		db_args.Push(this.m_CurrentPlayerId)
		db_args.Push(this.m_CurrentGuildId)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_CurrentPlayerId_changed||this.m_CurrentGuildId_changed{
			update_string = "UPDATE Global SET "
			db_args:=new_db_args(3)
			if this.m_CurrentPlayerId_changed{
				update_string+="CurrentPlayerId=?,"
				db_args.Push(this.m_CurrentPlayerId)
			}
			if this.m_CurrentGuildId_changed{
				update_string+="CurrentGuildId=?,"
				db_args.Push(this.m_CurrentGuildId)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE Id=?"
			db_args.Push(this.m_Id)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_CurrentPlayerId_changed = false
	this.m_CurrentGuildId_changed = false
	if release && this.m_loaded {
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbGlobalRow) Save(release bool) (err error, d bool, released bool) {
	err,released, state, update_string, args := this.save_data(release)
	if err != nil {
		log.Error("save data failed")
		return err, false, false
	}
	if state == 0 {
		d = false
	} else if state == 1 {
		_, err = this.m_table.m_dbc.StmtExec(this.m_table.m_save_insert_stmt, args...)
		if err != nil {
			log.Error("INSERT Global exec failed %v ", this.m_Id)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE Global exec failed %v", this.m_Id)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
type dbGlobalTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_row *dbGlobalRow
	m_preload_select_stmt *sql.Stmt
	m_save_insert_stmt *sql.Stmt
}
func new_dbGlobalTable(dbc *DBC) (this *dbGlobalTable) {
	this = &dbGlobalTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	return this
}
func (this *dbGlobalTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS Global(Id int(11),PRIMARY KEY (Id))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS Global failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='Global'", this.m_dbc.m_db_name)
	if err != nil {
		log.Error("SELECT information_schema failed")
		return
	}
	columns := make(map[string]int32)
	for rows.Next() {
		var column_name string
		var ordinal_position int32
		err = rows.Scan(&column_name, &ordinal_position)
		if err != nil {
			log.Error("scan information_schema row failed")
			return
		}
		if ordinal_position < 1 {
			log.Error("col ordinal out of range")
			continue
		}
		columns[column_name] = ordinal_position
	}
	_, hasCurrentPlayerId := columns["CurrentPlayerId"]
	if !hasCurrentPlayerId {
		_, err = this.m_dbc.Exec("ALTER TABLE Global ADD COLUMN CurrentPlayerId int(11) DEFAULT 0")
		if err != nil {
			log.Error("ADD COLUMN CurrentPlayerId failed")
			return
		}
	}
	_, hasCurrentGuildId := columns["CurrentGuildId"]
	if !hasCurrentGuildId {
		_, err = this.m_dbc.Exec("ALTER TABLE Global ADD COLUMN CurrentGuildId int(11) DEFAULT 0")
		if err != nil {
			log.Error("ADD COLUMN CurrentGuildId failed")
			return
		}
	}
	return
}
func (this *dbGlobalTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT CurrentPlayerId,CurrentGuildId FROM Global WHERE Id=0")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbGlobalTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO Global (Id,CurrentPlayerId,CurrentGuildId) VALUES (?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbGlobalTable) Init() (err error) {
	err=this.check_create_table()
	if err!=nil{
		log.Error("check_create_table failed")
		return
	}
	err=this.prepare_preload_select_stmt()
	if err!=nil{
		log.Error("prepare_preload_select_stmt failed")
		return
	}
	err=this.prepare_save_insert_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	return
}
func (this *dbGlobalTable) Preload() (err error) {
	r := this.m_dbc.StmtQueryRow(this.m_preload_select_stmt)
	var dCurrentPlayerId int32
	var dCurrentGuildId int32
	err = r.Scan(&dCurrentPlayerId,&dCurrentGuildId)
	if err!=nil{
		if err!=sql.ErrNoRows{
			log.Error("Scan failed")
			return
		}
	}else{
		row := new_dbGlobalRow(this,0)
		row.m_CurrentPlayerId=dCurrentPlayerId
		row.m_CurrentGuildId=dCurrentGuildId
		row.m_CurrentPlayerId_changed=false
		row.m_CurrentGuildId_changed=false
		row.m_valid = true
		row.m_loaded=true
		this.m_row=row
	}
	if this.m_row == nil {
		this.m_row = new_dbGlobalRow(this, 0)
		this.m_row.m_new = true
		this.m_row.m_valid = true
		err = this.Save(false)
		if err != nil {
			log.Error("save failed")
			return
		}
		this.m_row.m_loaded = true
	}
	return
}
func (this *dbGlobalTable) Save(quick bool) (err error) {
	if this.m_row==nil{
		return errors.New("row nil")
	}
	err, _, _ = this.m_row.Save(false)
	return err
}
func (this *dbGlobalTable) GetRow( ) (row *dbGlobalRow) {
	return this.m_row
}
func (this *dbPlayerRow)GetUniqueId( )(r string ){
	this.m_lock.UnSafeRLock("dbPlayerRow.GetdbPlayerUniqueIdColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_UniqueId)
}
func (this *dbPlayerRow)SetUniqueId(v string){
	this.m_lock.UnSafeLock("dbPlayerRow.SetdbPlayerUniqueIdColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_UniqueId=string(v)
	this.m_UniqueId_changed=true
	return
}
func (this *dbPlayerRow)GetAccount( )(r string ){
	this.m_lock.UnSafeRLock("dbPlayerRow.GetdbPlayerAccountColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_Account)
}
func (this *dbPlayerRow)SetAccount(v string){
	this.m_lock.UnSafeLock("dbPlayerRow.SetdbPlayerAccountColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Account=string(v)
	this.m_Account_changed=true
	return
}
func (this *dbPlayerRow)GetName( )(r string ){
	this.m_lock.UnSafeRLock("dbPlayerRow.GetdbPlayerNameColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_Name)
}
func (this *dbPlayerRow)SetName(v string){
	this.m_lock.UnSafeLock("dbPlayerRow.SetdbPlayerNameColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Name=string(v)
	this.m_Name_changed=true
	return
}
func (this *dbPlayerRow)GetToken( )(r string ){
	this.m_lock.UnSafeRLock("dbPlayerRow.GetdbPlayerTokenColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_Token)
}
func (this *dbPlayerRow)SetToken(v string){
	this.m_lock.UnSafeLock("dbPlayerRow.SetdbPlayerTokenColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Token=string(v)
	this.m_Token_changed=true
	return
}
func (this *dbPlayerRow)GetLevel( )(r int32 ){
	this.m_lock.UnSafeRLock("dbPlayerRow.GetdbPlayerLevelColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_Level)
}
func (this *dbPlayerRow)SetLevel(v int32){
	this.m_lock.UnSafeLock("dbPlayerRow.SetdbPlayerLevelColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Level=int32(v)
	this.m_Level_changed=true
	return
}
type dbPlayerInfoColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerInfoData
	m_changed bool
}
func (this *dbPlayerInfoColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerInfoData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerInfo{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerInfoData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerInfoColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerInfoColumn)Get( )(v *dbPlayerInfoData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerInfoData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerInfoColumn)Set(v dbPlayerInfoData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerInfoData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerInfoColumn)GetGold( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetGold")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Gold
	return
}
func (this *dbPlayerInfoColumn)SetGold(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetGold")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Gold = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyGold(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyGold")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Gold += v
	this.m_changed = true
	return this.m_data.Gold
}
func (this *dbPlayerInfoColumn)GetDiamond( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetDiamond")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Diamond
	return
}
func (this *dbPlayerInfoColumn)SetDiamond(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetDiamond")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Diamond = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyDiamond(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyDiamond")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Diamond += v
	this.m_changed = true
	return this.m_data.Diamond
}
func (this *dbPlayerInfoColumn)GetCurMaxStage( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetCurMaxStage")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.CurMaxStage
	return
}
func (this *dbPlayerInfoColumn)SetCurMaxStage(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetCurMaxStage")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CurMaxStage = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetTotalStars( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetTotalStars")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.TotalStars
	return
}
func (this *dbPlayerInfoColumn)SetTotalStars(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetTotalStars")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.TotalStars = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyTotalStars(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyTotalStars")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.TotalStars += v
	this.m_changed = true
	return this.m_data.TotalStars
}
func (this *dbPlayerInfoColumn)GetCurPassMaxStage( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetCurPassMaxStage")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.CurPassMaxStage
	return
}
func (this *dbPlayerInfoColumn)SetCurPassMaxStage(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetCurPassMaxStage")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CurPassMaxStage = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetMaxUnlockStage( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetMaxUnlockStage")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.MaxUnlockStage
	return
}
func (this *dbPlayerInfoColumn)SetMaxUnlockStage(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetMaxUnlockStage")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.MaxUnlockStage = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetMaxChapter( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetMaxChapter")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.MaxChapter
	return
}
func (this *dbPlayerInfoColumn)SetMaxChapter(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetMaxChapter")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.MaxChapter = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetCreateUnix( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetCreateUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.CreateUnix
	return
}
func (this *dbPlayerInfoColumn)SetCreateUnix(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetCreateUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CreateUnix = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetLvl( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetLvl")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Lvl
	return
}
func (this *dbPlayerInfoColumn)SetLvl(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetLvl")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Lvl = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetExp( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetExp")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Exp
	return
}
func (this *dbPlayerInfoColumn)SetExp(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetExp")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Exp = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetFirstPayState( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetFirstPayState")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.FirstPayState
	return
}
func (this *dbPlayerInfoColumn)SetFirstPayState(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetFirstPayState")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.FirstPayState = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetChangeNameCount( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetChangeNameCount")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.ChangeNameCount
	return
}
func (this *dbPlayerInfoColumn)SetChangeNameCount(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetChangeNameCount")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.ChangeNameCount = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyChangeNameCount(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyChangeNameCount")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.ChangeNameCount += v
	this.m_changed = true
	return this.m_data.ChangeNameCount
}
func (this *dbPlayerInfoColumn)GetLastDialyTaskUpUinx( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetLastDialyTaskUpUinx")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastDialyTaskUpUinx
	return
}
func (this *dbPlayerInfoColumn)SetLastDialyTaskUpUinx(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetLastDialyTaskUpUinx")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastDialyTaskUpUinx = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetHead( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetHead")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Head
	return
}
func (this *dbPlayerInfoColumn)SetHead(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetHead")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Head = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetCustomIcon( )(v string ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetCustomIcon")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.CustomIcon
	return
}
func (this *dbPlayerInfoColumn)SetCustomIcon(v string){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetCustomIcon")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CustomIcon = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetNextBuildingId( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetNextBuildingId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.NextBuildingId
	return
}
func (this *dbPlayerInfoColumn)SetNextBuildingId(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetNextBuildingId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.NextBuildingId = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyNextBuildingId(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyNextBuildingId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.NextBuildingId += v
	this.m_changed = true
	return this.m_data.NextBuildingId
}
func (this *dbPlayerInfoColumn)GetNextCatId( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetNextCatId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.NextCatId
	return
}
func (this *dbPlayerInfoColumn)SetNextCatId(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetNextCatId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.NextCatId = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyNextCatId(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyNextCatId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.NextCatId += v
	this.m_changed = true
	return this.m_data.NextCatId
}
func (this *dbPlayerInfoColumn)GetCharmVal( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetCharmVal")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.CharmVal
	return
}
func (this *dbPlayerInfoColumn)SetCharmVal(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetCharmVal")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CharmVal = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyCharmVal(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyCharmVal")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CharmVal += v
	this.m_changed = true
	return this.m_data.CharmVal
}
func (this *dbPlayerInfoColumn)GetLastLogin( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetLastLogin")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastLogin
	return
}
func (this *dbPlayerInfoColumn)SetLastLogin(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetLastLogin")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastLogin = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetZan( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetZan")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Zan
	return
}
func (this *dbPlayerInfoColumn)SetZan(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetZan")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Zan = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyZan(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyZan")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Zan += v
	this.m_changed = true
	return this.m_data.Zan
}
func (this *dbPlayerInfoColumn)GetCatFood( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetCatFood")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.CatFood
	return
}
func (this *dbPlayerInfoColumn)SetCatFood(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetCatFood")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CatFood = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyCatFood(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyCatFood")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CatFood += v
	this.m_changed = true
	return this.m_data.CatFood
}
func (this *dbPlayerInfoColumn)GetSpirit( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetSpirit")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Spirit
	return
}
func (this *dbPlayerInfoColumn)SetSpirit(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetSpirit")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Spirit = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbySpirit(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbySpirit")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Spirit += v
	this.m_changed = true
	return this.m_data.Spirit
}
func (this *dbPlayerInfoColumn)GetFriendPoints( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetFriendPoints")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.FriendPoints
	return
}
func (this *dbPlayerInfoColumn)SetFriendPoints(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetFriendPoints")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.FriendPoints = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyFriendPoints(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyFriendPoints")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.FriendPoints += v
	this.m_changed = true
	return this.m_data.FriendPoints
}
func (this *dbPlayerInfoColumn)GetSoulStone( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetSoulStone")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.SoulStone
	return
}
func (this *dbPlayerInfoColumn)SetSoulStone(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetSoulStone")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.SoulStone = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbySoulStone(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbySoulStone")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.SoulStone += v
	this.m_changed = true
	return this.m_data.SoulStone
}
func (this *dbPlayerInfoColumn)GetCharmMedal( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetCharmMedal")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.CharmMedal
	return
}
func (this *dbPlayerInfoColumn)SetCharmMedal(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetCharmMedal")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CharmMedal = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyCharmMedal(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyCharmMedal")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CharmMedal += v
	this.m_changed = true
	return this.m_data.CharmMedal
}
func (this *dbPlayerInfoColumn)GetSaveLastSpiritPointTime( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetSaveLastSpiritPointTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.SaveLastSpiritPointTime
	return
}
func (this *dbPlayerInfoColumn)SetSaveLastSpiritPointTime(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetSaveLastSpiritPointTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.SaveLastSpiritPointTime = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetLastRefreshShopTime( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetLastRefreshShopTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastRefreshShopTime
	return
}
func (this *dbPlayerInfoColumn)SetLastRefreshShopTime(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetLastRefreshShopTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastRefreshShopTime = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetDayChgExpeditionCount( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetDayChgExpeditionCount")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.DayChgExpeditionCount
	return
}
func (this *dbPlayerInfoColumn)SetDayChgExpeditionCount(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetDayChgExpeditionCount")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.DayChgExpeditionCount = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetDayChgExpeditionUpDay( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetDayChgExpeditionUpDay")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.DayChgExpeditionUpDay
	return
}
func (this *dbPlayerInfoColumn)SetDayChgExpeditionUpDay(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetDayChgExpeditionUpDay")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.DayChgExpeditionUpDay = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetLastMapChestUpUnix( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetLastMapChestUpUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastMapChestUpUnix
	return
}
func (this *dbPlayerInfoColumn)SetLastMapChestUpUnix(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetLastMapChestUpUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastMapChestUpUnix = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetLastMapBlockUpUnix( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetLastMapBlockUpUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastMapBlockUpUnix
	return
}
func (this *dbPlayerInfoColumn)SetLastMapBlockUpUnix(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetLastMapBlockUpUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastMapBlockUpUnix = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetVipLvl( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetVipLvl")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.VipLvl
	return
}
func (this *dbPlayerInfoColumn)SetVipLvl(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetVipLvl")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.VipLvl = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetMakingBuildingQueue( )(v []int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetMakingBuildingQueue")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]int32, len(this.m_data.MakingBuildingQueue))
	for _ii, _vv := range this.m_data.MakingBuildingQueue {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerInfoColumn)SetMakingBuildingQueue(v []int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetMakingBuildingQueue")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.MakingBuildingQueue = make([]int32, len(v))
	for _ii, _vv := range v {
		this.m_data.MakingBuildingQueue[_ii]=_vv
	}
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetMakedBuildingQueue( )(v []int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetMakedBuildingQueue")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]int32, len(this.m_data.MakedBuildingQueue))
	for _ii, _vv := range this.m_data.MakedBuildingQueue {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerInfoColumn)SetMakedBuildingQueue(v []int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetMakedBuildingQueue")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.MakedBuildingQueue = make([]int32, len(v))
	for _ii, _vv := range v {
		this.m_data.MakedBuildingQueue[_ii]=_vv
	}
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetDayHelpUnlockCount( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetDayHelpUnlockCount")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.DayHelpUnlockCount
	return
}
func (this *dbPlayerInfoColumn)SetDayHelpUnlockCount(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetDayHelpUnlockCount")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.DayHelpUnlockCount = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetDayHelpUnlockUpDay( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetDayHelpUnlockUpDay")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.DayHelpUnlockUpDay
	return
}
func (this *dbPlayerInfoColumn)SetDayHelpUnlockUpDay(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetDayHelpUnlockUpDay")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.DayHelpUnlockUpDay = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetFriendMessageUnreadCurrId( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetFriendMessageUnreadCurrId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.FriendMessageUnreadCurrId
	return
}
func (this *dbPlayerInfoColumn)SetFriendMessageUnreadCurrId(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetFriendMessageUnreadCurrId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.FriendMessageUnreadCurrId = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyFriendMessageUnreadCurrId(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyFriendMessageUnreadCurrId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.FriendMessageUnreadCurrId += v
	this.m_changed = true
	return this.m_data.FriendMessageUnreadCurrId
}
func (this *dbPlayerInfoColumn)GetVipCardEndDay( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetVipCardEndDay")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.VipCardEndDay
	return
}
func (this *dbPlayerInfoColumn)SetVipCardEndDay(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetVipCardEndDay")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.VipCardEndDay = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetNextExpeditionId( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetNextExpeditionId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.NextExpeditionId
	return
}
func (this *dbPlayerInfoColumn)SetNextExpeditionId(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetNextExpeditionId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.NextExpeditionId = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)IncbyNextExpeditionId(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.IncbyNextExpeditionId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.NextExpeditionId += v
	this.m_changed = true
	return this.m_data.NextExpeditionId
}
func (this *dbPlayerInfoColumn)GetDayExpeditionCount( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetDayExpeditionCount")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.DayExpeditionCount
	return
}
func (this *dbPlayerInfoColumn)SetDayExpeditionCount(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetDayExpeditionCount")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.DayExpeditionCount = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetDayExpeditionUpDay( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetDayExpeditionUpDay")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.DayExpeditionUpDay
	return
}
func (this *dbPlayerInfoColumn)SetDayExpeditionUpDay(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetDayExpeditionUpDay")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.DayExpeditionUpDay = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetChannel( )(v string ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetChannel")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Channel
	return
}
func (this *dbPlayerInfoColumn)SetChannel(v string){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetChannel")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Channel = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetDayBuyTiLiCount( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetDayBuyTiLiCount")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.DayBuyTiLiCount
	return
}
func (this *dbPlayerInfoColumn)SetDayBuyTiLiCount(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetDayBuyTiLiCount")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.DayBuyTiLiCount = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetDayBuyTiLiUpDay( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetDayBuyTiLiUpDay")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.DayBuyTiLiUpDay
	return
}
func (this *dbPlayerInfoColumn)SetDayBuyTiLiUpDay(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetDayBuyTiLiUpDay")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.DayBuyTiLiUpDay = v
	this.m_changed = true
	return
}
func (this *dbPlayerInfoColumn)GetLastLogout( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerInfoColumn.GetLastLogout")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastLogout
	return
}
func (this *dbPlayerInfoColumn)SetLastLogout(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerInfoColumn.SetLastLogout")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastLogout = v
	this.m_changed = true
	return
}
type dbPlayerStageColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerStageData
	m_changed bool
}
func (this *dbPlayerStageColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerStageList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerStageData{}
		d.from_pb(v)
		this.m_data[int32(d.StageId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerStageColumn)save( )(data []byte,err error){
	pb := &db.PlayerStageList{}
	pb.List=make([]*db.PlayerStage,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerStageColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerStageColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerStageColumn)GetAll()(list []dbPlayerStageData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerStageData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerStageColumn)Get(id int32)(v *dbPlayerStageData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerStageData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerStageColumn)Set(v dbPlayerStageData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.StageId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.StageId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerStageColumn)Add(v *dbPlayerStageData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.StageId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.StageId)
		return false
	}
	d:=&dbPlayerStageData{}
	v.clone_to(d)
	this.m_data[int32(v.StageId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerStageColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerStageColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerStageData)
	this.m_changed = true
	return
}
func (this *dbPlayerStageColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerStageColumn)GetStars(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.GetStars")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Stars
	return v,true
}
func (this *dbPlayerStageColumn)SetStars(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.SetStars")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Stars = v
	this.m_changed = true
	return true
}
func (this *dbPlayerStageColumn)GetLastFinishedUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.GetLastFinishedUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LastFinishedUnix
	return v,true
}
func (this *dbPlayerStageColumn)SetLastFinishedUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.SetLastFinishedUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LastFinishedUnix = v
	this.m_changed = true
	return true
}
func (this *dbPlayerStageColumn)GetTopScore(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.GetTopScore")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.TopScore
	return v,true
}
func (this *dbPlayerStageColumn)SetTopScore(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.SetTopScore")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.TopScore = v
	this.m_changed = true
	return true
}
func (this *dbPlayerStageColumn)GetCatId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.GetCatId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CatId
	return v,true
}
func (this *dbPlayerStageColumn)SetCatId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.SetCatId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CatId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerStageColumn)GetPlayedCount(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.GetPlayedCount")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.PlayedCount
	return v,true
}
func (this *dbPlayerStageColumn)SetPlayedCount(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.SetPlayedCount")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.PlayedCount = v
	this.m_changed = true
	return true
}
func (this *dbPlayerStageColumn)IncbyPlayedCount(id int32,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.IncbyPlayedCount")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerStageData{}
		this.m_data[id] = d
	}
	d.PlayedCount +=  v
	this.m_changed = true
	return d.PlayedCount
}
func (this *dbPlayerStageColumn)GetPassCount(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageColumn.GetPassCount")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.PassCount
	return v,true
}
func (this *dbPlayerStageColumn)SetPassCount(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.SetPassCount")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.PassCount = v
	this.m_changed = true
	return true
}
func (this *dbPlayerStageColumn)IncbyPassCount(id int32,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageColumn.IncbyPassCount")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerStageData{}
		this.m_data[id] = d
	}
	d.PassCount +=  v
	this.m_changed = true
	return d.PassCount
}
type dbPlayerChapterUnLockColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerChapterUnLockData
	m_changed bool
}
func (this *dbPlayerChapterUnLockColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerChapterUnLockData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerChapterUnLock{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerChapterUnLockData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerChapterUnLockColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerChapterUnLockColumn)Get( )(v *dbPlayerChapterUnLockData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChapterUnLockColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerChapterUnLockData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerChapterUnLockColumn)Set(v dbPlayerChapterUnLockData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerChapterUnLockColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerChapterUnLockData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerChapterUnLockColumn)GetChapterId( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChapterUnLockColumn.GetChapterId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.ChapterId
	return
}
func (this *dbPlayerChapterUnLockColumn)SetChapterId(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerChapterUnLockColumn.SetChapterId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.ChapterId = v
	this.m_changed = true
	return
}
func (this *dbPlayerChapterUnLockColumn)GetPlayerIds( )(v []int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChapterUnLockColumn.GetPlayerIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]int32, len(this.m_data.PlayerIds))
	for _ii, _vv := range this.m_data.PlayerIds {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerChapterUnLockColumn)SetPlayerIds(v []int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerChapterUnLockColumn.SetPlayerIds")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.PlayerIds = make([]int32, len(v))
	for _ii, _vv := range v {
		this.m_data.PlayerIds[_ii]=_vv
	}
	this.m_changed = true
	return
}
func (this *dbPlayerChapterUnLockColumn)GetCurHelpIds( )(v []int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChapterUnLockColumn.GetCurHelpIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]int32, len(this.m_data.CurHelpIds))
	for _ii, _vv := range this.m_data.CurHelpIds {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerChapterUnLockColumn)SetCurHelpIds(v []int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerChapterUnLockColumn.SetCurHelpIds")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CurHelpIds = make([]int32, len(v))
	for _ii, _vv := range v {
		this.m_data.CurHelpIds[_ii]=_vv
	}
	this.m_changed = true
	return
}
func (this *dbPlayerChapterUnLockColumn)GetStartUnix( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChapterUnLockColumn.GetStartUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.StartUnix
	return
}
func (this *dbPlayerChapterUnLockColumn)SetStartUnix(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerChapterUnLockColumn.SetStartUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.StartUnix = v
	this.m_changed = true
	return
}
type dbPlayerItemColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerItemData
	m_changed bool
}
func (this *dbPlayerItemColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerItemList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerItemData{}
		d.from_pb(v)
		this.m_data[int32(d.ItemCfgId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerItemColumn)save( )(data []byte,err error){
	pb := &db.PlayerItemList{}
	pb.List=make([]*db.PlayerItem,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerItemColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerItemColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerItemColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerItemColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerItemColumn)GetAll()(list []dbPlayerItemData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerItemColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerItemData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerItemColumn)Get(id int32)(v *dbPlayerItemData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerItemColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerItemData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerItemColumn)Set(v dbPlayerItemData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerItemColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.ItemCfgId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.ItemCfgId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerItemColumn)Add(v *dbPlayerItemData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerItemColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.ItemCfgId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.ItemCfgId)
		return false
	}
	d:=&dbPlayerItemData{}
	v.clone_to(d)
	this.m_data[int32(v.ItemCfgId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerItemColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerItemColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerItemColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerItemColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerItemData)
	this.m_changed = true
	return
}
func (this *dbPlayerItemColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerItemColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerItemColumn)GetItemNum(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerItemColumn.GetItemNum")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.ItemNum
	return v,true
}
func (this *dbPlayerItemColumn)SetItemNum(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerItemColumn.SetItemNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.ItemNum = v
	this.m_changed = true
	return true
}
func (this *dbPlayerItemColumn)GetStartTimeUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerItemColumn.GetStartTimeUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.StartTimeUnix
	return v,true
}
func (this *dbPlayerItemColumn)SetStartTimeUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerItemColumn.SetStartTimeUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.StartTimeUnix = v
	this.m_changed = true
	return true
}
func (this *dbPlayerItemColumn)GetRemainSeconds(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerItemColumn.GetRemainSeconds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.RemainSeconds
	return v,true
}
func (this *dbPlayerItemColumn)SetRemainSeconds(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerItemColumn.SetRemainSeconds")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.RemainSeconds = v
	this.m_changed = true
	return true
}
type dbPlayerAreaColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerAreaData
	m_changed bool
}
func (this *dbPlayerAreaColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerAreaList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerAreaData{}
		d.from_pb(v)
		this.m_data[int32(d.CfgId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerAreaColumn)save( )(data []byte,err error){
	pb := &db.PlayerAreaList{}
	pb.List=make([]*db.PlayerArea,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerAreaColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerAreaColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerAreaColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerAreaColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerAreaColumn)GetAll()(list []dbPlayerAreaData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerAreaColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerAreaData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerAreaColumn)Get(id int32)(v *dbPlayerAreaData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerAreaColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerAreaData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerAreaColumn)Set(v dbPlayerAreaData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerAreaColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.CfgId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.CfgId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerAreaColumn)Add(v *dbPlayerAreaData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerAreaColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.CfgId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.CfgId)
		return false
	}
	d:=&dbPlayerAreaData{}
	v.clone_to(d)
	this.m_data[int32(v.CfgId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerAreaColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerAreaColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerAreaColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerAreaColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerAreaData)
	this.m_changed = true
	return
}
func (this *dbPlayerAreaColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerAreaColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
type dbPlayerBuildingColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerBuildingData
	m_changed bool
}
func (this *dbPlayerBuildingColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerBuildingList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerBuildingData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerBuildingColumn)save( )(data []byte,err error){
	pb := &db.PlayerBuildingList{}
	pb.List=make([]*db.PlayerBuilding,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerBuildingColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerBuildingColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerBuildingColumn)GetAll()(list []dbPlayerBuildingData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerBuildingData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerBuildingColumn)Get(id int32)(v *dbPlayerBuildingData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerBuildingData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerBuildingColumn)Set(v dbPlayerBuildingData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerBuildingColumn)Add(v *dbPlayerBuildingData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerBuildingData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerBuildingColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerBuildingColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerBuildingData)
	this.m_changed = true
	return
}
func (this *dbPlayerBuildingColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerBuildingColumn)GetCfgId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.GetCfgId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CfgId
	return v,true
}
func (this *dbPlayerBuildingColumn)SetCfgId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingColumn.SetCfgId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CfgId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerBuildingColumn)GetX(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.GetX")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.X
	return v,true
}
func (this *dbPlayerBuildingColumn)SetX(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingColumn.SetX")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.X = v
	this.m_changed = true
	return true
}
func (this *dbPlayerBuildingColumn)GetY(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.GetY")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Y
	return v,true
}
func (this *dbPlayerBuildingColumn)SetY(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingColumn.SetY")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Y = v
	this.m_changed = true
	return true
}
func (this *dbPlayerBuildingColumn)GetDir(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.GetDir")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Dir
	return v,true
}
func (this *dbPlayerBuildingColumn)SetDir(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingColumn.SetDir")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Dir = v
	this.m_changed = true
	return true
}
func (this *dbPlayerBuildingColumn)GetCreateUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.GetCreateUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CreateUnix
	return v,true
}
func (this *dbPlayerBuildingColumn)SetCreateUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingColumn.SetCreateUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CreateUnix = v
	this.m_changed = true
	return true
}
func (this *dbPlayerBuildingColumn)GetOverUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingColumn.GetOverUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.OverUnix
	return v,true
}
func (this *dbPlayerBuildingColumn)SetOverUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingColumn.SetOverUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.OverUnix = v
	this.m_changed = true
	return true
}
type dbPlayerBuildingDepotColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerBuildingDepotData
	m_changed bool
}
func (this *dbPlayerBuildingDepotColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerBuildingDepotList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerBuildingDepotData{}
		d.from_pb(v)
		this.m_data[int32(d.CfgId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerBuildingDepotColumn)save( )(data []byte,err error){
	pb := &db.PlayerBuildingDepotList{}
	pb.List=make([]*db.PlayerBuildingDepot,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerBuildingDepotColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingDepotColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerBuildingDepotColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingDepotColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerBuildingDepotColumn)GetAll()(list []dbPlayerBuildingDepotData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingDepotColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerBuildingDepotData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerBuildingDepotColumn)Get(id int32)(v *dbPlayerBuildingDepotData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingDepotColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerBuildingDepotData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerBuildingDepotColumn)Set(v dbPlayerBuildingDepotData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingDepotColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.CfgId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.CfgId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerBuildingDepotColumn)Add(v *dbPlayerBuildingDepotData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingDepotColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.CfgId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.CfgId)
		return false
	}
	d:=&dbPlayerBuildingDepotData{}
	v.clone_to(d)
	this.m_data[int32(v.CfgId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerBuildingDepotColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingDepotColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerBuildingDepotColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingDepotColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerBuildingDepotData)
	this.m_changed = true
	return
}
func (this *dbPlayerBuildingDepotColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingDepotColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerBuildingDepotColumn)GetNum(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBuildingDepotColumn.GetNum")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Num
	return v,true
}
func (this *dbPlayerBuildingDepotColumn)SetNum(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingDepotColumn.SetNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Num = v
	this.m_changed = true
	return true
}
func (this *dbPlayerBuildingDepotColumn)IncbyNum(id int32,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerBuildingDepotColumn.IncbyNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerBuildingDepotData{}
		this.m_data[id] = d
	}
	d.Num +=  v
	this.m_changed = true
	return d.Num
}
type dbPlayerDepotBuildingFormulaColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerDepotBuildingFormulaData
	m_changed bool
}
func (this *dbPlayerDepotBuildingFormulaColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerDepotBuildingFormulaList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerDepotBuildingFormulaData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerDepotBuildingFormulaColumn)save( )(data []byte,err error){
	pb := &db.PlayerDepotBuildingFormulaList{}
	pb.List=make([]*db.PlayerDepotBuildingFormula,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerDepotBuildingFormulaColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerDepotBuildingFormulaColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerDepotBuildingFormulaColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerDepotBuildingFormulaColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerDepotBuildingFormulaColumn)GetAll()(list []dbPlayerDepotBuildingFormulaData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerDepotBuildingFormulaColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerDepotBuildingFormulaData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerDepotBuildingFormulaColumn)Get(id int32)(v *dbPlayerDepotBuildingFormulaData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerDepotBuildingFormulaColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerDepotBuildingFormulaData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerDepotBuildingFormulaColumn)Set(v dbPlayerDepotBuildingFormulaData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerDepotBuildingFormulaColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerDepotBuildingFormulaColumn)Add(v *dbPlayerDepotBuildingFormulaData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerDepotBuildingFormulaColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerDepotBuildingFormulaData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerDepotBuildingFormulaColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerDepotBuildingFormulaColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerDepotBuildingFormulaColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerDepotBuildingFormulaColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerDepotBuildingFormulaData)
	this.m_changed = true
	return
}
func (this *dbPlayerDepotBuildingFormulaColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerDepotBuildingFormulaColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
type dbPlayerMakingFormulaBuildingColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerMakingFormulaBuildingData
	m_changed bool
}
func (this *dbPlayerMakingFormulaBuildingColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerMakingFormulaBuildingList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerMakingFormulaBuildingData{}
		d.from_pb(v)
		this.m_data[int32(d.SlotId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerMakingFormulaBuildingColumn)save( )(data []byte,err error){
	pb := &db.PlayerMakingFormulaBuildingList{}
	pb.List=make([]*db.PlayerMakingFormulaBuilding,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerMakingFormulaBuildingColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMakingFormulaBuildingColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerMakingFormulaBuildingColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMakingFormulaBuildingColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerMakingFormulaBuildingColumn)GetAll()(list []dbPlayerMakingFormulaBuildingData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMakingFormulaBuildingColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerMakingFormulaBuildingData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerMakingFormulaBuildingColumn)Get(id int32)(v *dbPlayerMakingFormulaBuildingData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMakingFormulaBuildingColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerMakingFormulaBuildingData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerMakingFormulaBuildingColumn)Set(v dbPlayerMakingFormulaBuildingData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMakingFormulaBuildingColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.SlotId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.SlotId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerMakingFormulaBuildingColumn)Add(v *dbPlayerMakingFormulaBuildingData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMakingFormulaBuildingColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.SlotId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.SlotId)
		return false
	}
	d:=&dbPlayerMakingFormulaBuildingData{}
	v.clone_to(d)
	this.m_data[int32(v.SlotId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerMakingFormulaBuildingColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerMakingFormulaBuildingColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerMakingFormulaBuildingColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerMakingFormulaBuildingColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerMakingFormulaBuildingData)
	this.m_changed = true
	return
}
func (this *dbPlayerMakingFormulaBuildingColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMakingFormulaBuildingColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerMakingFormulaBuildingColumn)GetCanUse(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMakingFormulaBuildingColumn.GetCanUse")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CanUse
	return v,true
}
func (this *dbPlayerMakingFormulaBuildingColumn)SetCanUse(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMakingFormulaBuildingColumn.SetCanUse")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CanUse = v
	this.m_changed = true
	return true
}
func (this *dbPlayerMakingFormulaBuildingColumn)GetFormulaId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMakingFormulaBuildingColumn.GetFormulaId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.FormulaId
	return v,true
}
func (this *dbPlayerMakingFormulaBuildingColumn)SetFormulaId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMakingFormulaBuildingColumn.SetFormulaId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.FormulaId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerMakingFormulaBuildingColumn)GetStartTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMakingFormulaBuildingColumn.GetStartTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.StartTime
	return v,true
}
func (this *dbPlayerMakingFormulaBuildingColumn)SetStartTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMakingFormulaBuildingColumn.SetStartTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.StartTime = v
	this.m_changed = true
	return true
}
type dbPlayerCropColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerCropData
	m_changed bool
}
func (this *dbPlayerCropColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerCropList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerCropData{}
		d.from_pb(v)
		this.m_data[int32(d.BuildingId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerCropColumn)save( )(data []byte,err error){
	pb := &db.PlayerCropList{}
	pb.List=make([]*db.PlayerCrop,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerCropColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCropColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerCropColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCropColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerCropColumn)GetAll()(list []dbPlayerCropData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCropColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerCropData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerCropColumn)Get(id int32)(v *dbPlayerCropData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCropColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerCropData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerCropColumn)Set(v dbPlayerCropData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.BuildingId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.BuildingId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerCropColumn)Add(v *dbPlayerCropData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.BuildingId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.BuildingId)
		return false
	}
	d:=&dbPlayerCropData{}
	v.clone_to(d)
	this.m_data[int32(v.BuildingId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerCropColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerCropColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerCropData)
	this.m_changed = true
	return
}
func (this *dbPlayerCropColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCropColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerCropColumn)GetId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCropColumn.GetId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Id
	return v,true
}
func (this *dbPlayerCropColumn)SetId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.SetId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Id = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCropColumn)GetPlantTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCropColumn.GetPlantTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.PlantTime
	return v,true
}
func (this *dbPlayerCropColumn)SetPlantTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.SetPlantTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.PlantTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCropColumn)GetBuildingTableId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCropColumn.GetBuildingTableId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.BuildingTableId
	return v,true
}
func (this *dbPlayerCropColumn)SetBuildingTableId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.SetBuildingTableId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.BuildingTableId = v
	this.m_changed = true
	return true
}
type dbPlayerCatColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerCatData
	m_changed bool
}
func (this *dbPlayerCatColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerCatList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerCatData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerCatColumn)save( )(data []byte,err error){
	pb := &db.PlayerCatList{}
	pb.List=make([]*db.PlayerCat,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerCatColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerCatColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerCatColumn)GetAll()(list []dbPlayerCatData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerCatData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerCatColumn)Get(id int32)(v *dbPlayerCatData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerCatData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerCatColumn)Set(v dbPlayerCatData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)Add(v *dbPlayerCatData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerCatData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerCatColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerCatData)
	this.m_changed = true
	return
}
func (this *dbPlayerCatColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerCatColumn)GetCfgId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetCfgId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CfgId
	return v,true
}
func (this *dbPlayerCatColumn)SetCfgId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetCfgId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CfgId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetExp(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetExp")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Exp
	return v,true
}
func (this *dbPlayerCatColumn)SetExp(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetExp")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Exp = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetLevel(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetLevel")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Level
	return v,true
}
func (this *dbPlayerCatColumn)SetLevel(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetLevel")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Level = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetStar(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetStar")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Star
	return v,true
}
func (this *dbPlayerCatColumn)SetStar(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetStar")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Star = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetNick(id int32)(v string ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetNick")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Nick
	return v,true
}
func (this *dbPlayerCatColumn)SetNick(id int32,v string)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetNick")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Nick = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetSkillLevel(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetSkillLevel")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.SkillLevel
	return v,true
}
func (this *dbPlayerCatColumn)SetSkillLevel(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetSkillLevel")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.SkillLevel = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetLocked(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetLocked")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Locked
	return v,true
}
func (this *dbPlayerCatColumn)SetLocked(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetLocked")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Locked = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetCoinAbility(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetCoinAbility")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CoinAbility
	return v,true
}
func (this *dbPlayerCatColumn)SetCoinAbility(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetCoinAbility")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CoinAbility = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetExploreAbility(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetExploreAbility")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.ExploreAbility
	return v,true
}
func (this *dbPlayerCatColumn)SetExploreAbility(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetExploreAbility")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.ExploreAbility = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetMatchAbility(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetMatchAbility")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.MatchAbility
	return v,true
}
func (this *dbPlayerCatColumn)SetMatchAbility(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetMatchAbility")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.MatchAbility = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetCathouseId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetCathouseId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CathouseId
	return v,true
}
func (this *dbPlayerCatColumn)SetCathouseId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetCathouseId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CathouseId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetState(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetState")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.State
	return v,true
}
func (this *dbPlayerCatColumn)SetState(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetState")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.State = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatColumn)GetStateValue(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatColumn.GetStateValue")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.StateValue
	return v,true
}
func (this *dbPlayerCatColumn)SetStateValue(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatColumn.SetStateValue")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.StateValue = v
	this.m_changed = true
	return true
}
type dbPlayerCatHouseColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerCatHouseData
	m_changed bool
}
func (this *dbPlayerCatHouseColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerCatHouseList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerCatHouseData{}
		d.from_pb(v)
		this.m_data[int32(d.BuildingId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerCatHouseColumn)save( )(data []byte,err error){
	pb := &db.PlayerCatHouseList{}
	pb.List=make([]*db.PlayerCatHouse,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerCatHouseColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerCatHouseColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerCatHouseColumn)GetAll()(list []dbPlayerCatHouseData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerCatHouseData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerCatHouseColumn)Get(id int32)(v *dbPlayerCatHouseData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerCatHouseData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerCatHouseColumn)Set(v dbPlayerCatHouseData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.BuildingId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.BuildingId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerCatHouseColumn)Add(v *dbPlayerCatHouseData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.BuildingId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.BuildingId)
		return false
	}
	d:=&dbPlayerCatHouseData{}
	v.clone_to(d)
	this.m_data[int32(v.BuildingId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerCatHouseColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerCatHouseColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerCatHouseData)
	this.m_changed = true
	return
}
func (this *dbPlayerCatHouseColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerCatHouseColumn)GetCfgId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.GetCfgId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CfgId
	return v,true
}
func (this *dbPlayerCatHouseColumn)SetCfgId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.SetCfgId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CfgId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatHouseColumn)GetLevel(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.GetLevel")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Level
	return v,true
}
func (this *dbPlayerCatHouseColumn)SetLevel(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.SetLevel")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Level = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatHouseColumn)GetCatIds(id int32)(v []int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.GetCatIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]int32, len(d.CatIds))
	for _ii, _vv := range d.CatIds {
		v[_ii]=_vv
	}
	return v,true
}
func (this *dbPlayerCatHouseColumn)SetCatIds(id int32,v []int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.SetCatIds")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CatIds = make([]int32, len(v))
	for _ii, _vv := range v {
		d.CatIds[_ii]=_vv
	}
	this.m_changed = true
	return true
}
func (this *dbPlayerCatHouseColumn)GetLastGetGoldTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.GetLastGetGoldTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LastGetGoldTime
	return v,true
}
func (this *dbPlayerCatHouseColumn)SetLastGetGoldTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.SetLastGetGoldTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LastGetGoldTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatHouseColumn)GetCurrGold(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.GetCurrGold")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CurrGold
	return v,true
}
func (this *dbPlayerCatHouseColumn)SetCurrGold(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.SetCurrGold")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CurrGold = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatHouseColumn)IncbyCurrGold(id int32,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.IncbyCurrGold")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerCatHouseData{}
		this.m_data[id] = d
	}
	d.CurrGold +=  v
	this.m_changed = true
	return d.CurrGold
}
func (this *dbPlayerCatHouseColumn)GetLevelupStartTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.GetLevelupStartTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LevelupStartTime
	return v,true
}
func (this *dbPlayerCatHouseColumn)SetLevelupStartTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.SetLevelupStartTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LevelupStartTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatHouseColumn)GetIsDone(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatHouseColumn.GetIsDone")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.IsDone
	return v,true
}
func (this *dbPlayerCatHouseColumn)SetIsDone(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatHouseColumn.SetIsDone")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.IsDone = v
	this.m_changed = true
	return true
}
type dbPlayerShopItemColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerShopItemData
	m_changed bool
}
func (this *dbPlayerShopItemColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerShopItemList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerShopItemData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerShopItemColumn)save( )(data []byte,err error){
	pb := &db.PlayerShopItemList{}
	pb.List=make([]*db.PlayerShopItem,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerShopItemColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopItemColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerShopItemColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopItemColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerShopItemColumn)GetAll()(list []dbPlayerShopItemData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopItemColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerShopItemData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerShopItemColumn)Get(id int32)(v *dbPlayerShopItemData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopItemColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerShopItemData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerShopItemColumn)Set(v dbPlayerShopItemData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerShopItemColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerShopItemColumn)Add(v *dbPlayerShopItemData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerShopItemColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerShopItemData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerShopItemColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerShopItemColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerShopItemColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerShopItemColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerShopItemData)
	this.m_changed = true
	return
}
func (this *dbPlayerShopItemColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopItemColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerShopItemColumn)GetLeftNum(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopItemColumn.GetLeftNum")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LeftNum
	return v,true
}
func (this *dbPlayerShopItemColumn)SetLeftNum(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerShopItemColumn.SetLeftNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LeftNum = v
	this.m_changed = true
	return true
}
func (this *dbPlayerShopItemColumn)IncbyLeftNum(id int32,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerShopItemColumn.IncbyLeftNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerShopItemData{}
		this.m_data[id] = d
	}
	d.LeftNum +=  v
	this.m_changed = true
	return d.LeftNum
}
type dbPlayerShopLimitedInfoColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerShopLimitedInfoData
	m_changed bool
}
func (this *dbPlayerShopLimitedInfoColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerShopLimitedInfoList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerShopLimitedInfoData{}
		d.from_pb(v)
		this.m_data[int32(d.LimitedDays)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerShopLimitedInfoColumn)save( )(data []byte,err error){
	pb := &db.PlayerShopLimitedInfoList{}
	pb.List=make([]*db.PlayerShopLimitedInfo,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerShopLimitedInfoColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopLimitedInfoColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerShopLimitedInfoColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopLimitedInfoColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerShopLimitedInfoColumn)GetAll()(list []dbPlayerShopLimitedInfoData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopLimitedInfoColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerShopLimitedInfoData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerShopLimitedInfoColumn)Get(id int32)(v *dbPlayerShopLimitedInfoData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopLimitedInfoColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerShopLimitedInfoData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerShopLimitedInfoColumn)Set(v dbPlayerShopLimitedInfoData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerShopLimitedInfoColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.LimitedDays)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.LimitedDays)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerShopLimitedInfoColumn)Add(v *dbPlayerShopLimitedInfoData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerShopLimitedInfoColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.LimitedDays)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.LimitedDays)
		return false
	}
	d:=&dbPlayerShopLimitedInfoData{}
	v.clone_to(d)
	this.m_data[int32(v.LimitedDays)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerShopLimitedInfoColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerShopLimitedInfoColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerShopLimitedInfoColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerShopLimitedInfoColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerShopLimitedInfoData)
	this.m_changed = true
	return
}
func (this *dbPlayerShopLimitedInfoColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopLimitedInfoColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerShopLimitedInfoColumn)GetLastSaveTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerShopLimitedInfoColumn.GetLastSaveTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LastSaveTime
	return v,true
}
func (this *dbPlayerShopLimitedInfoColumn)SetLastSaveTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerShopLimitedInfoColumn.SetLastSaveTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LastSaveTime = v
	this.m_changed = true
	return true
}
type dbPlayerChestColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerChestData
	m_changed bool
}
func (this *dbPlayerChestColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerChestList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerChestData{}
		d.from_pb(v)
		this.m_data[int32(d.Pos)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerChestColumn)save( )(data []byte,err error){
	pb := &db.PlayerChestList{}
	pb.List=make([]*db.PlayerChest,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerChestColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChestColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerChestColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChestColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerChestColumn)GetAll()(list []dbPlayerChestData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChestColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerChestData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerChestColumn)Get(id int32)(v *dbPlayerChestData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChestColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerChestData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerChestColumn)Set(v dbPlayerChestData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerChestColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Pos)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Pos)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerChestColumn)Add(v *dbPlayerChestData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerChestColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Pos)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Pos)
		return false
	}
	d:=&dbPlayerChestData{}
	v.clone_to(d)
	this.m_data[int32(v.Pos)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerChestColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerChestColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerChestColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerChestColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerChestData)
	this.m_changed = true
	return
}
func (this *dbPlayerChestColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChestColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerChestColumn)GetChestId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChestColumn.GetChestId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.ChestId
	return v,true
}
func (this *dbPlayerChestColumn)SetChestId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerChestColumn.SetChestId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.ChestId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerChestColumn)GetOpenSec(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChestColumn.GetOpenSec")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.OpenSec
	return v,true
}
func (this *dbPlayerChestColumn)SetOpenSec(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerChestColumn.SetOpenSec")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.OpenSec = v
	this.m_changed = true
	return true
}
type dbPlayerPayBackColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerPayBackData
	m_changed bool
}
func (this *dbPlayerPayBackColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerPayBackList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerPayBackData{}
		d.from_pb(v)
		this.m_data[int32(d.PayBackId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerPayBackColumn)save( )(data []byte,err error){
	pb := &db.PlayerPayBackList{}
	pb.List=make([]*db.PlayerPayBack,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerPayBackColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayBackColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerPayBackColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayBackColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerPayBackColumn)GetAll()(list []dbPlayerPayBackData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayBackColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerPayBackData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerPayBackColumn)Get(id int32)(v *dbPlayerPayBackData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayBackColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerPayBackData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerPayBackColumn)Set(v dbPlayerPayBackData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayBackColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.PayBackId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.PayBackId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerPayBackColumn)Add(v *dbPlayerPayBackData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayBackColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.PayBackId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.PayBackId)
		return false
	}
	d:=&dbPlayerPayBackData{}
	v.clone_to(d)
	this.m_data[int32(v.PayBackId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerPayBackColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayBackColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerPayBackColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayBackColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerPayBackData)
	this.m_changed = true
	return
}
func (this *dbPlayerPayBackColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayBackColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerPayBackColumn)GetValue(id int32)(v string ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayBackColumn.GetValue")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Value
	return v,true
}
func (this *dbPlayerPayBackColumn)SetValue(id int32,v string)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayBackColumn.SetValue")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Value = v
	this.m_changed = true
	return true
}
type dbPlayerOptionsColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerOptionsData
	m_changed bool
}
func (this *dbPlayerOptionsColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerOptionsData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerOptions{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerOptionsData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerOptionsColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerOptionsColumn)Get( )(v *dbPlayerOptionsData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerOptionsColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerOptionsData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerOptionsColumn)Set(v dbPlayerOptionsData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerOptionsColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerOptionsData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerOptionsColumn)GetValues( )(v []int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerOptionsColumn.GetValues")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]int32, len(this.m_data.Values))
	for _ii, _vv := range this.m_data.Values {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerOptionsColumn)SetValues(v []int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerOptionsColumn.SetValues")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Values = make([]int32, len(v))
	for _ii, _vv := range v {
		this.m_data.Values[_ii]=_vv
	}
	this.m_changed = true
	return
}
type dbPlayerTaskCommonColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerTaskCommonData
	m_changed bool
}
func (this *dbPlayerTaskCommonColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerTaskCommonData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerTaskCommon{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerTaskCommonData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerTaskCommonColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerTaskCommonColumn)Get( )(v *dbPlayerTaskCommonData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTaskCommonColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerTaskCommonData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerTaskCommonColumn)Set(v dbPlayerTaskCommonData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerTaskCommonColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerTaskCommonData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerTaskCommonColumn)GetLastRefreshTime( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTaskCommonColumn.GetLastRefreshTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastRefreshTime
	return
}
func (this *dbPlayerTaskCommonColumn)SetLastRefreshTime(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerTaskCommonColumn.SetLastRefreshTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastRefreshTime = v
	this.m_changed = true
	return
}
type dbPlayerTaskColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerTaskData
	m_changed bool
}
func (this *dbPlayerTaskColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerTaskList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerTaskData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerTaskColumn)save( )(data []byte,err error){
	pb := &db.PlayerTaskList{}
	pb.List=make([]*db.PlayerTask,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerTaskColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTaskColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerTaskColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTaskColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerTaskColumn)GetAll()(list []dbPlayerTaskData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTaskColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerTaskData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerTaskColumn)Get(id int32)(v *dbPlayerTaskData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTaskColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerTaskData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerTaskColumn)Set(v dbPlayerTaskData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerTaskColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerTaskColumn)Add(v *dbPlayerTaskData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerTaskColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerTaskData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerTaskColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerTaskColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerTaskColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerTaskColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerTaskData)
	this.m_changed = true
	return
}
func (this *dbPlayerTaskColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTaskColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerTaskColumn)GetValue(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTaskColumn.GetValue")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Value
	return v,true
}
func (this *dbPlayerTaskColumn)SetValue(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerTaskColumn.SetValue")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Value = v
	this.m_changed = true
	return true
}
func (this *dbPlayerTaskColumn)IncbyValue(id int32,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerTaskColumn.IncbyValue")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerTaskData{}
		this.m_data[id] = d
	}
	d.Value +=  v
	this.m_changed = true
	return d.Value
}
func (this *dbPlayerTaskColumn)GetState(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTaskColumn.GetState")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.State
	return v,true
}
func (this *dbPlayerTaskColumn)SetState(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerTaskColumn.SetState")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.State = v
	this.m_changed = true
	return true
}
func (this *dbPlayerTaskColumn)GetParam(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTaskColumn.GetParam")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Param
	return v,true
}
func (this *dbPlayerTaskColumn)SetParam(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerTaskColumn.SetParam")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Param = v
	this.m_changed = true
	return true
}
type dbPlayerFinishedTaskColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFinishedTaskData
	m_changed bool
}
func (this *dbPlayerFinishedTaskColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFinishedTaskList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFinishedTaskData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFinishedTaskColumn)save( )(data []byte,err error){
	pb := &db.PlayerFinishedTaskList{}
	pb.List=make([]*db.PlayerFinishedTask,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFinishedTaskColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFinishedTaskColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFinishedTaskColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFinishedTaskColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFinishedTaskColumn)GetAll()(list []dbPlayerFinishedTaskData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFinishedTaskColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFinishedTaskData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFinishedTaskColumn)Get(id int32)(v *dbPlayerFinishedTaskData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFinishedTaskColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFinishedTaskData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFinishedTaskColumn)Set(v dbPlayerFinishedTaskData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFinishedTaskColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFinishedTaskColumn)Add(v *dbPlayerFinishedTaskData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFinishedTaskColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerFinishedTaskData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFinishedTaskColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFinishedTaskColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFinishedTaskColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFinishedTaskColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFinishedTaskData)
	this.m_changed = true
	return
}
func (this *dbPlayerFinishedTaskColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFinishedTaskColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
type dbPlayerDailyTaskAllDailyColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerDailyTaskAllDailyData
	m_changed bool
}
func (this *dbPlayerDailyTaskAllDailyColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerDailyTaskAllDailyList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerDailyTaskAllDailyData{}
		d.from_pb(v)
		this.m_data[int32(d.CompleteTaskId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerDailyTaskAllDailyColumn)save( )(data []byte,err error){
	pb := &db.PlayerDailyTaskAllDailyList{}
	pb.List=make([]*db.PlayerDailyTaskAllDaily,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerDailyTaskAllDailyColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerDailyTaskAllDailyColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerDailyTaskAllDailyColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerDailyTaskAllDailyColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerDailyTaskAllDailyColumn)GetAll()(list []dbPlayerDailyTaskAllDailyData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerDailyTaskAllDailyColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerDailyTaskAllDailyData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerDailyTaskAllDailyColumn)Get(id int32)(v *dbPlayerDailyTaskAllDailyData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerDailyTaskAllDailyColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerDailyTaskAllDailyData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerDailyTaskAllDailyColumn)Set(v dbPlayerDailyTaskAllDailyData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerDailyTaskAllDailyColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.CompleteTaskId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.CompleteTaskId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerDailyTaskAllDailyColumn)Add(v *dbPlayerDailyTaskAllDailyData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerDailyTaskAllDailyColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.CompleteTaskId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.CompleteTaskId)
		return false
	}
	d:=&dbPlayerDailyTaskAllDailyData{}
	v.clone_to(d)
	this.m_data[int32(v.CompleteTaskId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerDailyTaskAllDailyColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerDailyTaskAllDailyColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerDailyTaskAllDailyColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerDailyTaskAllDailyColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerDailyTaskAllDailyData)
	this.m_changed = true
	return
}
func (this *dbPlayerDailyTaskAllDailyColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerDailyTaskAllDailyColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
type dbPlayerSevenActivityColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerSevenActivityData
	m_changed bool
}
func (this *dbPlayerSevenActivityColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerSevenActivityList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerSevenActivityData{}
		d.from_pb(v)
		this.m_data[int32(d.ActivityId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerSevenActivityColumn)save( )(data []byte,err error){
	pb := &db.PlayerSevenActivityList{}
	pb.List=make([]*db.PlayerSevenActivity,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerSevenActivityColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSevenActivityColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerSevenActivityColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSevenActivityColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerSevenActivityColumn)GetAll()(list []dbPlayerSevenActivityData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSevenActivityColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerSevenActivityData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerSevenActivityColumn)Get(id int32)(v *dbPlayerSevenActivityData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSevenActivityColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerSevenActivityData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerSevenActivityColumn)Set(v dbPlayerSevenActivityData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerSevenActivityColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.ActivityId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.ActivityId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerSevenActivityColumn)Add(v *dbPlayerSevenActivityData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerSevenActivityColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.ActivityId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.ActivityId)
		return false
	}
	d:=&dbPlayerSevenActivityData{}
	v.clone_to(d)
	this.m_data[int32(v.ActivityId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerSevenActivityColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerSevenActivityColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerSevenActivityColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerSevenActivityColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerSevenActivityData)
	this.m_changed = true
	return
}
func (this *dbPlayerSevenActivityColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSevenActivityColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerSevenActivityColumn)GetValue(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSevenActivityColumn.GetValue")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Value
	return v,true
}
func (this *dbPlayerSevenActivityColumn)SetValue(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerSevenActivityColumn.SetValue")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Value = v
	this.m_changed = true
	return true
}
func (this *dbPlayerSevenActivityColumn)IncbyValue(id int32,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerSevenActivityColumn.IncbyValue")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerSevenActivityData{}
		this.m_data[id] = d
	}
	d.Value +=  v
	this.m_changed = true
	return d.Value
}
func (this *dbPlayerSevenActivityColumn)GetRewardUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSevenActivityColumn.GetRewardUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.RewardUnix
	return v,true
}
func (this *dbPlayerSevenActivityColumn)SetRewardUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerSevenActivityColumn.SetRewardUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.RewardUnix = v
	this.m_changed = true
	return true
}
type dbPlayerSignInfoColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerSignInfoData
	m_changed bool
}
func (this *dbPlayerSignInfoColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerSignInfoData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerSignInfo{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerSignInfoData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerSignInfoColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerSignInfoColumn)Get( )(v *dbPlayerSignInfoData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSignInfoColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerSignInfoData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerSignInfoColumn)Set(v dbPlayerSignInfoData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerSignInfoColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerSignInfoData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerSignInfoColumn)GetLastSignDay( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSignInfoColumn.GetLastSignDay")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastSignDay
	return
}
func (this *dbPlayerSignInfoColumn)SetLastSignDay(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerSignInfoColumn.SetLastSignDay")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastSignDay = v
	this.m_changed = true
	return
}
func (this *dbPlayerSignInfoColumn)GetCurSignSum( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSignInfoColumn.GetCurSignSum")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.CurSignSum
	return
}
func (this *dbPlayerSignInfoColumn)SetCurSignSum(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerSignInfoColumn.SetCurSignSum")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CurSignSum = v
	this.m_changed = true
	return
}
func (this *dbPlayerSignInfoColumn)IncbyCurSignSum(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerSignInfoColumn.IncbyCurSignSum")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CurSignSum += v
	this.m_changed = true
	return this.m_data.CurSignSum
}
func (this *dbPlayerSignInfoColumn)GetCurSignSumMonth( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSignInfoColumn.GetCurSignSumMonth")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.CurSignSumMonth
	return
}
func (this *dbPlayerSignInfoColumn)SetCurSignSumMonth(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerSignInfoColumn.SetCurSignSumMonth")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CurSignSumMonth = v
	this.m_changed = true
	return
}
func (this *dbPlayerSignInfoColumn)GetCurSignDays( )(v []int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSignInfoColumn.GetCurSignDays")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]int32, len(this.m_data.CurSignDays))
	for _ii, _vv := range this.m_data.CurSignDays {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerSignInfoColumn)SetCurSignDays(v []int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerSignInfoColumn.SetCurSignDays")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CurSignDays = make([]int32, len(v))
	for _ii, _vv := range v {
		this.m_data.CurSignDays[_ii]=_vv
	}
	this.m_changed = true
	return
}
func (this *dbPlayerSignInfoColumn)GetRewardSignSum( )(v []int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSignInfoColumn.GetRewardSignSum")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]int32, len(this.m_data.RewardSignSum))
	for _ii, _vv := range this.m_data.RewardSignSum {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerSignInfoColumn)SetRewardSignSum(v []int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerSignInfoColumn.SetRewardSignSum")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.RewardSignSum = make([]int32, len(v))
	for _ii, _vv := range v {
		this.m_data.RewardSignSum[_ii]=_vv
	}
	this.m_changed = true
	return
}
type dbPlayerGuidesColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerGuidesData
	m_changed bool
}
func (this *dbPlayerGuidesColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerGuidesList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerGuidesData{}
		d.from_pb(v)
		this.m_data[int32(d.GuideId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerGuidesColumn)save( )(data []byte,err error){
	pb := &db.PlayerGuidesList{}
	pb.List=make([]*db.PlayerGuides,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerGuidesColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerGuidesColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerGuidesColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerGuidesColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerGuidesColumn)GetAll()(list []dbPlayerGuidesData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerGuidesColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerGuidesData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerGuidesColumn)Get(id int32)(v *dbPlayerGuidesData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerGuidesColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerGuidesData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerGuidesColumn)Set(v dbPlayerGuidesData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerGuidesColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.GuideId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.GuideId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerGuidesColumn)Add(v *dbPlayerGuidesData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerGuidesColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.GuideId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.GuideId)
		return false
	}
	d:=&dbPlayerGuidesData{}
	v.clone_to(d)
	this.m_data[int32(v.GuideId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerGuidesColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerGuidesColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerGuidesColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerGuidesColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerGuidesData)
	this.m_changed = true
	return
}
func (this *dbPlayerGuidesColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerGuidesColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerGuidesColumn)GetSetUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerGuidesColumn.GetSetUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.SetUnix
	return v,true
}
func (this *dbPlayerGuidesColumn)SetSetUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerGuidesColumn.SetSetUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.SetUnix = v
	this.m_changed = true
	return true
}
type dbPlayerFriendRelativeColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerFriendRelativeData
	m_changed bool
}
func (this *dbPlayerFriendRelativeColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerFriendRelativeData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFriendRelative{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerFriendRelativeData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerFriendRelativeColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendRelativeColumn)Get( )(v *dbPlayerFriendRelativeData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendRelativeColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerFriendRelativeData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerFriendRelativeColumn)Set(v dbPlayerFriendRelativeData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendRelativeColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerFriendRelativeData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerFriendRelativeColumn)GetLastGiveFriendPointsTime( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendRelativeColumn.GetLastGiveFriendPointsTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastGiveFriendPointsTime
	return
}
func (this *dbPlayerFriendRelativeColumn)SetLastGiveFriendPointsTime(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendRelativeColumn.SetLastGiveFriendPointsTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastGiveFriendPointsTime = v
	this.m_changed = true
	return
}
func (this *dbPlayerFriendRelativeColumn)GetGiveNumToday( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendRelativeColumn.GetGiveNumToday")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.GiveNumToday
	return
}
func (this *dbPlayerFriendRelativeColumn)SetGiveNumToday(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendRelativeColumn.SetGiveNumToday")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.GiveNumToday = v
	this.m_changed = true
	return
}
func (this *dbPlayerFriendRelativeColumn)IncbyGiveNumToday(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendRelativeColumn.IncbyGiveNumToday")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.GiveNumToday += v
	this.m_changed = true
	return this.m_data.GiveNumToday
}
func (this *dbPlayerFriendRelativeColumn)GetLastRefreshTime( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendRelativeColumn.GetLastRefreshTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastRefreshTime
	return
}
func (this *dbPlayerFriendRelativeColumn)SetLastRefreshTime(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendRelativeColumn.SetLastRefreshTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastRefreshTime = v
	this.m_changed = true
	return
}
type dbPlayerFriendColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFriendData
	m_changed bool
}
func (this *dbPlayerFriendColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFriendList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFriendData{}
		d.from_pb(v)
		this.m_data[int32(d.FriendId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendColumn)save( )(data []byte,err error){
	pb := &db.PlayerFriendList{}
	pb.List=make([]*db.PlayerFriend,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFriendColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFriendColumn)GetAll()(list []dbPlayerFriendData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFriendData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFriendColumn)Get(id int32)(v *dbPlayerFriendData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFriendData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFriendColumn)Set(v dbPlayerFriendData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.FriendId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.FriendId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendColumn)Add(v *dbPlayerFriendData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.FriendId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.FriendId)
		return false
	}
	d:=&dbPlayerFriendData{}
	v.clone_to(d)
	this.m_data[int32(v.FriendId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFriendColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFriendData)
	this.m_changed = true
	return
}
func (this *dbPlayerFriendColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerFriendColumn)GetFriendName(id int32)(v string ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.GetFriendName")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.FriendName
	return v,true
}
func (this *dbPlayerFriendColumn)SetFriendName(id int32,v string)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendColumn.SetFriendName")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.FriendName = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendColumn)GetHead(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.GetHead")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Head
	return v,true
}
func (this *dbPlayerFriendColumn)SetHead(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendColumn.SetHead")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Head = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendColumn)GetLevel(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.GetLevel")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Level
	return v,true
}
func (this *dbPlayerFriendColumn)SetLevel(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendColumn.SetLevel")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Level = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendColumn)GetVipLevel(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.GetVipLevel")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.VipLevel
	return v,true
}
func (this *dbPlayerFriendColumn)SetVipLevel(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendColumn.SetVipLevel")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.VipLevel = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendColumn)GetLastLogin(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.GetLastLogin")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LastLogin
	return v,true
}
func (this *dbPlayerFriendColumn)SetLastLogin(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendColumn.SetLastLogin")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LastLogin = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendColumn)GetLastGivePointsTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendColumn.GetLastGivePointsTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LastGivePointsTime
	return v,true
}
func (this *dbPlayerFriendColumn)SetLastGivePointsTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendColumn.SetLastGivePointsTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LastGivePointsTime = v
	this.m_changed = true
	return true
}
type dbPlayerFriendRecommendColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFriendRecommendData
	m_changed bool
}
func (this *dbPlayerFriendRecommendColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFriendRecommendList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFriendRecommendData{}
		d.from_pb(v)
		this.m_data[int32(d.PlayerId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendRecommendColumn)save( )(data []byte,err error){
	pb := &db.PlayerFriendRecommendList{}
	pb.List=make([]*db.PlayerFriendRecommend,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendRecommendColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendRecommendColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFriendRecommendColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendRecommendColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFriendRecommendColumn)GetAll()(list []dbPlayerFriendRecommendData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendRecommendColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFriendRecommendData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFriendRecommendColumn)Get(id int32)(v *dbPlayerFriendRecommendData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendRecommendColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFriendRecommendData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFriendRecommendColumn)Set(v dbPlayerFriendRecommendData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendRecommendColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.PlayerId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.PlayerId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendRecommendColumn)Add(v *dbPlayerFriendRecommendData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendRecommendColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.PlayerId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.PlayerId)
		return false
	}
	d:=&dbPlayerFriendRecommendData{}
	v.clone_to(d)
	this.m_data[int32(v.PlayerId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendRecommendColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendRecommendColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFriendRecommendColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendRecommendColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFriendRecommendData)
	this.m_changed = true
	return
}
func (this *dbPlayerFriendRecommendColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendRecommendColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
type dbPlayerFriendAskColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFriendAskData
	m_changed bool
}
func (this *dbPlayerFriendAskColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFriendAskList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFriendAskData{}
		d.from_pb(v)
		this.m_data[int32(d.PlayerId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendAskColumn)save( )(data []byte,err error){
	pb := &db.PlayerFriendAskList{}
	pb.List=make([]*db.PlayerFriendAsk,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendAskColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendAskColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFriendAskColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendAskColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFriendAskColumn)GetAll()(list []dbPlayerFriendAskData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendAskColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFriendAskData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFriendAskColumn)Get(id int32)(v *dbPlayerFriendAskData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendAskColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFriendAskData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFriendAskColumn)Set(v dbPlayerFriendAskData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendAskColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.PlayerId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.PlayerId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendAskColumn)Add(v *dbPlayerFriendAskData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendAskColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.PlayerId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.PlayerId)
		return false
	}
	d:=&dbPlayerFriendAskData{}
	v.clone_to(d)
	this.m_data[int32(v.PlayerId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendAskColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendAskColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFriendAskColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendAskColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFriendAskData)
	this.m_changed = true
	return
}
func (this *dbPlayerFriendAskColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendAskColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
type dbPlayerFriendReqColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFriendReqData
	m_changed bool
}
func (this *dbPlayerFriendReqColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFriendReqList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFriendReqData{}
		d.from_pb(v)
		this.m_data[int32(d.PlayerId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendReqColumn)save( )(data []byte,err error){
	pb := &db.PlayerFriendReqList{}
	pb.List=make([]*db.PlayerFriendReq,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendReqColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendReqColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFriendReqColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendReqColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFriendReqColumn)GetAll()(list []dbPlayerFriendReqData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendReqColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFriendReqData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFriendReqColumn)Get(id int32)(v *dbPlayerFriendReqData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendReqColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFriendReqData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFriendReqColumn)Set(v dbPlayerFriendReqData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendReqColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.PlayerId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.PlayerId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendReqColumn)Add(v *dbPlayerFriendReqData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendReqColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.PlayerId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.PlayerId)
		return false
	}
	d:=&dbPlayerFriendReqData{}
	v.clone_to(d)
	this.m_data[int32(v.PlayerId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendReqColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendReqColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFriendReqColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendReqColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFriendReqData)
	this.m_changed = true
	return
}
func (this *dbPlayerFriendReqColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendReqColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerFriendReqColumn)GetPlayerName(id int32)(v string ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendReqColumn.GetPlayerName")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.PlayerName
	return v,true
}
func (this *dbPlayerFriendReqColumn)SetPlayerName(id int32,v string)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendReqColumn.SetPlayerName")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.PlayerName = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendReqColumn)GetReqUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendReqColumn.GetReqUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.ReqUnix
	return v,true
}
func (this *dbPlayerFriendReqColumn)SetReqUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendReqColumn.SetReqUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.ReqUnix = v
	this.m_changed = true
	return true
}
type dbPlayerFriendPointColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFriendPointData
	m_changed bool
}
func (this *dbPlayerFriendPointColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFriendPointList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFriendPointData{}
		d.from_pb(v)
		this.m_data[int32(d.FromPlayerId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendPointColumn)save( )(data []byte,err error){
	pb := &db.PlayerFriendPointList{}
	pb.List=make([]*db.PlayerFriendPoint,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendPointColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendPointColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFriendPointColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendPointColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFriendPointColumn)GetAll()(list []dbPlayerFriendPointData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendPointColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFriendPointData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFriendPointColumn)Get(id int32)(v *dbPlayerFriendPointData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendPointColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFriendPointData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFriendPointColumn)Set(v dbPlayerFriendPointData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendPointColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.FromPlayerId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.FromPlayerId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendPointColumn)Add(v *dbPlayerFriendPointData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendPointColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.FromPlayerId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.FromPlayerId)
		return false
	}
	d:=&dbPlayerFriendPointData{}
	v.clone_to(d)
	this.m_data[int32(v.FromPlayerId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendPointColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendPointColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFriendPointColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendPointColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFriendPointData)
	this.m_changed = true
	return
}
func (this *dbPlayerFriendPointColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendPointColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerFriendPointColumn)GetGivePoints(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendPointColumn.GetGivePoints")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.GivePoints
	return v,true
}
func (this *dbPlayerFriendPointColumn)SetGivePoints(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendPointColumn.SetGivePoints")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.GivePoints = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendPointColumn)GetLastGiveTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendPointColumn.GetLastGiveTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LastGiveTime
	return v,true
}
func (this *dbPlayerFriendPointColumn)SetLastGiveTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendPointColumn.SetLastGiveTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LastGiveTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendPointColumn)GetIsTodayGive(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendPointColumn.GetIsTodayGive")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.IsTodayGive
	return v,true
}
func (this *dbPlayerFriendPointColumn)SetIsTodayGive(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendPointColumn.SetIsTodayGive")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.IsTodayGive = v
	this.m_changed = true
	return true
}
type dbPlayerFriendChatUnreadIdColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFriendChatUnreadIdData
	m_changed bool
}
func (this *dbPlayerFriendChatUnreadIdColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFriendChatUnreadIdList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFriendChatUnreadIdData{}
		d.from_pb(v)
		this.m_data[int32(d.FriendId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendChatUnreadIdColumn)save( )(data []byte,err error){
	pb := &db.PlayerFriendChatUnreadIdList{}
	pb.List=make([]*db.PlayerFriendChatUnreadId,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendChatUnreadIdColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadIdColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFriendChatUnreadIdColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadIdColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFriendChatUnreadIdColumn)GetAll()(list []dbPlayerFriendChatUnreadIdData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadIdColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFriendChatUnreadIdData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFriendChatUnreadIdColumn)Get(id int32)(v *dbPlayerFriendChatUnreadIdData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadIdColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFriendChatUnreadIdData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFriendChatUnreadIdColumn)Set(v dbPlayerFriendChatUnreadIdData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadIdColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.FriendId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.FriendId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendChatUnreadIdColumn)Add(v *dbPlayerFriendChatUnreadIdData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadIdColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.FriendId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.FriendId)
		return false
	}
	d:=&dbPlayerFriendChatUnreadIdData{}
	v.clone_to(d)
	this.m_data[int32(v.FriendId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendChatUnreadIdColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadIdColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFriendChatUnreadIdColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadIdColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFriendChatUnreadIdData)
	this.m_changed = true
	return
}
func (this *dbPlayerFriendChatUnreadIdColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadIdColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerFriendChatUnreadIdColumn)GetMessageIds(id int32)(v []int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadIdColumn.GetMessageIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]int32, len(d.MessageIds))
	for _ii, _vv := range d.MessageIds {
		v[_ii]=_vv
	}
	return v,true
}
func (this *dbPlayerFriendChatUnreadIdColumn)SetMessageIds(id int32,v []int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadIdColumn.SetMessageIds")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.MessageIds = make([]int32, len(v))
	for _ii, _vv := range v {
		d.MessageIds[_ii]=_vv
	}
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendChatUnreadIdColumn)GetCurrMessageId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadIdColumn.GetCurrMessageId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CurrMessageId
	return v,true
}
func (this *dbPlayerFriendChatUnreadIdColumn)SetCurrMessageId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadIdColumn.SetCurrMessageId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CurrMessageId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendChatUnreadIdColumn)IncbyCurrMessageId(id int32,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadIdColumn.IncbyCurrMessageId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerFriendChatUnreadIdData{}
		this.m_data[id] = d
	}
	d.CurrMessageId +=  v
	this.m_changed = true
	return d.CurrMessageId
}
type dbPlayerFriendChatUnreadMessageColumn struct{
	m_row *dbPlayerRow
	m_data map[int64]*dbPlayerFriendChatUnreadMessageData
	m_changed bool
}
func (this *dbPlayerFriendChatUnreadMessageColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFriendChatUnreadMessageList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFriendChatUnreadMessageData{}
		d.from_pb(v)
		this.m_data[int64(d.PlayerMessageId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendChatUnreadMessageColumn)save( )(data []byte,err error){
	pb := &db.PlayerFriendChatUnreadMessageList{}
	pb.List=make([]*db.PlayerFriendChatUnreadMessage,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFriendChatUnreadMessageColumn)HasIndex(id int64)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadMessageColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFriendChatUnreadMessageColumn)GetAllIndex()(list []int64){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadMessageColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int64, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFriendChatUnreadMessageColumn)GetAll()(list []dbPlayerFriendChatUnreadMessageData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadMessageColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFriendChatUnreadMessageData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFriendChatUnreadMessageColumn)Get(id int64)(v *dbPlayerFriendChatUnreadMessageData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadMessageColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFriendChatUnreadMessageData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFriendChatUnreadMessageColumn)Set(v dbPlayerFriendChatUnreadMessageData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadMessageColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int64(v.PlayerMessageId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.PlayerMessageId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendChatUnreadMessageColumn)Add(v *dbPlayerFriendChatUnreadMessageData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadMessageColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int64(v.PlayerMessageId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.PlayerMessageId)
		return false
	}
	d:=&dbPlayerFriendChatUnreadMessageData{}
	v.clone_to(d)
	this.m_data[int64(v.PlayerMessageId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendChatUnreadMessageColumn)Remove(id int64){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadMessageColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFriendChatUnreadMessageColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadMessageColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int64]*dbPlayerFriendChatUnreadMessageData)
	this.m_changed = true
	return
}
func (this *dbPlayerFriendChatUnreadMessageColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadMessageColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerFriendChatUnreadMessageColumn)GetMessage(id int64)(v []byte,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadMessageColumn.GetMessage")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]byte, len(d.Message))
	for _ii, _vv := range d.Message {
		v[_ii]=_vv
	}
	return v,true
}
func (this *dbPlayerFriendChatUnreadMessageColumn)SetMessage(id int64,v []byte)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadMessageColumn.SetMessage")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Message = make([]byte, len(v))
	for _ii, _vv := range v {
		d.Message[_ii]=_vv
	}
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendChatUnreadMessageColumn)GetSendTime(id int64)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadMessageColumn.GetSendTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.SendTime
	return v,true
}
func (this *dbPlayerFriendChatUnreadMessageColumn)SetSendTime(id int64,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadMessageColumn.SetSendTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.SendTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFriendChatUnreadMessageColumn)GetIsRead(id int64)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFriendChatUnreadMessageColumn.GetIsRead")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.IsRead
	return v,true
}
func (this *dbPlayerFriendChatUnreadMessageColumn)SetIsRead(id int64,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFriendChatUnreadMessageColumn.SetIsRead")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.IsRead = v
	this.m_changed = true
	return true
}
type dbPlayerFocusPlayerColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFocusPlayerData
	m_changed bool
}
func (this *dbPlayerFocusPlayerColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFocusPlayerList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFocusPlayerData{}
		d.from_pb(v)
		this.m_data[int32(d.FriendId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFocusPlayerColumn)save( )(data []byte,err error){
	pb := &db.PlayerFocusPlayerList{}
	pb.List=make([]*db.PlayerFocusPlayer,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFocusPlayerColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFocusPlayerColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFocusPlayerColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFocusPlayerColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFocusPlayerColumn)GetAll()(list []dbPlayerFocusPlayerData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFocusPlayerColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFocusPlayerData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFocusPlayerColumn)Get(id int32)(v *dbPlayerFocusPlayerData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFocusPlayerColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFocusPlayerData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFocusPlayerColumn)Set(v dbPlayerFocusPlayerData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFocusPlayerColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.FriendId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.FriendId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFocusPlayerColumn)Add(v *dbPlayerFocusPlayerData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFocusPlayerColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.FriendId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.FriendId)
		return false
	}
	d:=&dbPlayerFocusPlayerData{}
	v.clone_to(d)
	this.m_data[int32(v.FriendId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFocusPlayerColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFocusPlayerColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFocusPlayerColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFocusPlayerColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFocusPlayerData)
	this.m_changed = true
	return
}
func (this *dbPlayerFocusPlayerColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFocusPlayerColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerFocusPlayerColumn)GetFriendName(id int32)(v string ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFocusPlayerColumn.GetFriendName")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.FriendName
	return v,true
}
func (this *dbPlayerFocusPlayerColumn)SetFriendName(id int32,v string)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFocusPlayerColumn.SetFriendName")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.FriendName = v
	this.m_changed = true
	return true
}
type dbPlayerBeFocusPlayerColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerBeFocusPlayerData
	m_changed bool
}
func (this *dbPlayerBeFocusPlayerColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerBeFocusPlayerList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerBeFocusPlayerData{}
		d.from_pb(v)
		this.m_data[int32(d.FriendId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerBeFocusPlayerColumn)save( )(data []byte,err error){
	pb := &db.PlayerBeFocusPlayerList{}
	pb.List=make([]*db.PlayerBeFocusPlayer,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerBeFocusPlayerColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBeFocusPlayerColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerBeFocusPlayerColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBeFocusPlayerColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerBeFocusPlayerColumn)GetAll()(list []dbPlayerBeFocusPlayerData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBeFocusPlayerColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerBeFocusPlayerData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerBeFocusPlayerColumn)Get(id int32)(v *dbPlayerBeFocusPlayerData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBeFocusPlayerColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerBeFocusPlayerData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerBeFocusPlayerColumn)Set(v dbPlayerBeFocusPlayerData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBeFocusPlayerColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.FriendId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.FriendId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerBeFocusPlayerColumn)Add(v *dbPlayerBeFocusPlayerData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBeFocusPlayerColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.FriendId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.FriendId)
		return false
	}
	d:=&dbPlayerBeFocusPlayerData{}
	v.clone_to(d)
	this.m_data[int32(v.FriendId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerBeFocusPlayerColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerBeFocusPlayerColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerBeFocusPlayerColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerBeFocusPlayerColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerBeFocusPlayerData)
	this.m_changed = true
	return
}
func (this *dbPlayerBeFocusPlayerColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBeFocusPlayerColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerBeFocusPlayerColumn)GetFriendName(id int32)(v string ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBeFocusPlayerColumn.GetFriendName")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.FriendName
	return v,true
}
func (this *dbPlayerBeFocusPlayerColumn)SetFriendName(id int32,v string)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerBeFocusPlayerColumn.SetFriendName")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.FriendName = v
	this.m_changed = true
	return true
}
type dbPlayerCustomDataColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerCustomDataData
	m_changed bool
}
func (this *dbPlayerCustomDataColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerCustomDataData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerCustomData{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerCustomDataData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerCustomDataColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerCustomDataColumn)Get( )(v *dbPlayerCustomDataData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCustomDataColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerCustomDataData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerCustomDataColumn)Set(v dbPlayerCustomDataData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerCustomDataColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerCustomDataData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerCustomDataColumn)GetCustomData( )(v []byte){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCustomDataColumn.GetCustomData")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]byte, len(this.m_data.CustomData))
	for _ii, _vv := range this.m_data.CustomData {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerCustomDataColumn)SetCustomData(v []byte){
	this.m_row.m_lock.UnSafeLock("dbPlayerCustomDataColumn.SetCustomData")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CustomData = make([]byte, len(v))
	for _ii, _vv := range v {
		this.m_data.CustomData[_ii]=_vv
	}
	this.m_changed = true
	return
}
type dbPlayerChaterOpenRequestColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerChaterOpenRequestData
	m_changed bool
}
func (this *dbPlayerChaterOpenRequestColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerChaterOpenRequestData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerChaterOpenRequest{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerChaterOpenRequestData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerChaterOpenRequestColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerChaterOpenRequestColumn)Get( )(v *dbPlayerChaterOpenRequestData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChaterOpenRequestColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerChaterOpenRequestData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerChaterOpenRequestColumn)Set(v dbPlayerChaterOpenRequestData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerChaterOpenRequestColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerChaterOpenRequestData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerChaterOpenRequestColumn)GetCustomData( )(v []byte){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChaterOpenRequestColumn.GetCustomData")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]byte, len(this.m_data.CustomData))
	for _ii, _vv := range this.m_data.CustomData {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerChaterOpenRequestColumn)SetCustomData(v []byte){
	this.m_row.m_lock.UnSafeLock("dbPlayerChaterOpenRequestColumn.SetCustomData")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CustomData = make([]byte, len(v))
	for _ii, _vv := range v {
		this.m_data.CustomData[_ii]=_vv
	}
	this.m_changed = true
	return
}
type dbPlayerExpeditionColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerExpeditionData
	m_changed bool
}
func (this *dbPlayerExpeditionColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerExpeditionList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerExpeditionData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerExpeditionColumn)save( )(data []byte,err error){
	pb := &db.PlayerExpeditionList{}
	pb.List=make([]*db.PlayerExpedition,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerExpeditionColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerExpeditionColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerExpeditionColumn)GetAll()(list []dbPlayerExpeditionData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerExpeditionData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerExpeditionColumn)Get(id int32)(v *dbPlayerExpeditionData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerExpeditionData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerExpeditionColumn)Set(v dbPlayerExpeditionData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)Add(v *dbPlayerExpeditionData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerExpeditionData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerExpeditionColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerExpeditionData)
	this.m_changed = true
	return
}
func (this *dbPlayerExpeditionColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerExpeditionColumn)GetTaskId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetTaskId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.TaskId
	return v,true
}
func (this *dbPlayerExpeditionColumn)SetTaskId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.SetTaskId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.TaskId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)GetStartUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetStartUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.StartUnix
	return v,true
}
func (this *dbPlayerExpeditionColumn)SetStartUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.SetStartUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.StartUnix = v
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)GetEndUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetEndUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.EndUnix
	return v,true
}
func (this *dbPlayerExpeditionColumn)SetEndUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.SetEndUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.EndUnix = v
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)GetInCatIds(id int32)(v []int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetInCatIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]int32, len(d.InCatIds))
	for _ii, _vv := range d.InCatIds {
		v[_ii]=_vv
	}
	return v,true
}
func (this *dbPlayerExpeditionColumn)SetInCatIds(id int32,v []int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.SetInCatIds")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.InCatIds = make([]int32, len(v))
	for _ii, _vv := range v {
		d.InCatIds[_ii]=_vv
	}
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)GetCurState(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetCurState")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CurState
	return v,true
}
func (this *dbPlayerExpeditionColumn)SetCurState(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.SetCurState")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CurState = v
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)GetResult(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetResult")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Result
	return v,true
}
func (this *dbPlayerExpeditionColumn)SetResult(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.SetResult")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Result = v
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)GetTaskLeftSec(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetTaskLeftSec")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.TaskLeftSec
	return v,true
}
func (this *dbPlayerExpeditionColumn)SetTaskLeftSec(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.SetTaskLeftSec")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.TaskLeftSec = v
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)GetTaskLeftSecLastUpUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetTaskLeftSecLastUpUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.TaskLeftSecLastUpUnix
	return v,true
}
func (this *dbPlayerExpeditionColumn)SetTaskLeftSecLastUpUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.SetTaskLeftSecLastUpUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.TaskLeftSecLastUpUnix = v
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)GetConditions(id int32)(v []dbExpeditionConData,has bool ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetConditions")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]dbExpeditionConData, len(d.Conditions))
	for _ii, _vv := range d.Conditions {
		_vv.clone_to(&v[_ii])
	}
	return v,true
}
func (this *dbPlayerExpeditionColumn)SetConditions(id int32,v []dbExpeditionConData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.SetConditions")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Conditions = make([]dbExpeditionConData, len(v))
	for _ii, _vv := range v {
		_vv.clone_to(&d.Conditions[_ii])
	}
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)GetEventIds(id int32)(v []dbExpeditionEventData,has bool ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetEventIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]dbExpeditionEventData, len(d.EventIds))
	for _ii, _vv := range d.EventIds {
		_vv.clone_to(&v[_ii])
	}
	return v,true
}
func (this *dbPlayerExpeditionColumn)SetEventIds(id int32,v []dbExpeditionEventData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.SetEventIds")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.EventIds = make([]dbExpeditionEventData, len(v))
	for _ii, _vv := range v {
		_vv.clone_to(&d.EventIds[_ii])
	}
	this.m_changed = true
	return true
}
func (this *dbPlayerExpeditionColumn)GetTotalSpecials(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerExpeditionColumn.GetTotalSpecials")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.TotalSpecials
	return v,true
}
func (this *dbPlayerExpeditionColumn)SetTotalSpecials(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerExpeditionColumn.SetTotalSpecials")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.TotalSpecials = v
	this.m_changed = true
	return true
}
type dbPlayerHandbookItemColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerHandbookItemData
	m_changed bool
}
func (this *dbPlayerHandbookItemColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerHandbookItemList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerHandbookItemData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerHandbookItemColumn)save( )(data []byte,err error){
	pb := &db.PlayerHandbookItemList{}
	pb.List=make([]*db.PlayerHandbookItem,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerHandbookItemColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerHandbookItemColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerHandbookItemColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerHandbookItemColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerHandbookItemColumn)GetAll()(list []dbPlayerHandbookItemData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerHandbookItemColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerHandbookItemData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerHandbookItemColumn)Get(id int32)(v *dbPlayerHandbookItemData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerHandbookItemColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerHandbookItemData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerHandbookItemColumn)Set(v dbPlayerHandbookItemData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerHandbookItemColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerHandbookItemColumn)Add(v *dbPlayerHandbookItemData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerHandbookItemColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerHandbookItemData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerHandbookItemColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerHandbookItemColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerHandbookItemColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerHandbookItemColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerHandbookItemData)
	this.m_changed = true
	return
}
func (this *dbPlayerHandbookItemColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerHandbookItemColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
type dbPlayerHeadItemColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerHeadItemData
	m_changed bool
}
func (this *dbPlayerHeadItemColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerHeadItemList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerHeadItemData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerHeadItemColumn)save( )(data []byte,err error){
	pb := &db.PlayerHeadItemList{}
	pb.List=make([]*db.PlayerHeadItem,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerHeadItemColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerHeadItemColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerHeadItemColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerHeadItemColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerHeadItemColumn)GetAll()(list []dbPlayerHeadItemData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerHeadItemColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerHeadItemData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerHeadItemColumn)Get(id int32)(v *dbPlayerHeadItemData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerHeadItemColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerHeadItemData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerHeadItemColumn)Set(v dbPlayerHeadItemData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerHeadItemColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerHeadItemColumn)Add(v *dbPlayerHeadItemData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerHeadItemColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerHeadItemData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerHeadItemColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerHeadItemColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerHeadItemColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerHeadItemColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerHeadItemData)
	this.m_changed = true
	return
}
func (this *dbPlayerHeadItemColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerHeadItemColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
type dbPlayerActivityColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerActivityData
	m_changed bool
}
func (this *dbPlayerActivityColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerActivityList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerActivityData{}
		d.from_pb(v)
		this.m_data[int32(d.CfgId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerActivityColumn)save( )(data []byte,err error){
	pb := &db.PlayerActivityList{}
	pb.List=make([]*db.PlayerActivity,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerActivityColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerActivityColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerActivityColumn)GetAll()(list []dbPlayerActivityData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerActivityData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerActivityColumn)Get(id int32)(v *dbPlayerActivityData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerActivityData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerActivityColumn)Set(v dbPlayerActivityData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.CfgId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.CfgId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerActivityColumn)Add(v *dbPlayerActivityData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.CfgId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.CfgId)
		return false
	}
	d:=&dbPlayerActivityData{}
	v.clone_to(d)
	this.m_data[int32(v.CfgId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerActivityColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerActivityColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerActivityData)
	this.m_changed = true
	return
}
func (this *dbPlayerActivityColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerActivityColumn)GetStates(id int32)(v []int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.GetStates")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]int32, len(d.States))
	for _ii, _vv := range d.States {
		v[_ii]=_vv
	}
	return v,true
}
func (this *dbPlayerActivityColumn)SetStates(id int32,v []int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.SetStates")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.States = make([]int32, len(v))
	for _ii, _vv := range v {
		d.States[_ii]=_vv
	}
	this.m_changed = true
	return true
}
func (this *dbPlayerActivityColumn)GetVals(id int32)(v []int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityColumn.GetVals")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]int32, len(d.Vals))
	for _ii, _vv := range d.Vals {
		v[_ii]=_vv
	}
	return v,true
}
func (this *dbPlayerActivityColumn)SetVals(id int32,v []int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityColumn.SetVals")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Vals = make([]int32, len(v))
	for _ii, _vv := range v {
		d.Vals[_ii]=_vv
	}
	this.m_changed = true
	return true
}
type dbPlayerSuitAwardColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerSuitAwardData
	m_changed bool
}
func (this *dbPlayerSuitAwardColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerSuitAwardList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerSuitAwardData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerSuitAwardColumn)save( )(data []byte,err error){
	pb := &db.PlayerSuitAwardList{}
	pb.List=make([]*db.PlayerSuitAward,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerSuitAwardColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSuitAwardColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerSuitAwardColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSuitAwardColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerSuitAwardColumn)GetAll()(list []dbPlayerSuitAwardData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSuitAwardColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerSuitAwardData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerSuitAwardColumn)Get(id int32)(v *dbPlayerSuitAwardData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSuitAwardColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerSuitAwardData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerSuitAwardColumn)Set(v dbPlayerSuitAwardData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerSuitAwardColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerSuitAwardColumn)Add(v *dbPlayerSuitAwardData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerSuitAwardColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerSuitAwardData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerSuitAwardColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerSuitAwardColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerSuitAwardColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerSuitAwardColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerSuitAwardData)
	this.m_changed = true
	return
}
func (this *dbPlayerSuitAwardColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSuitAwardColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerSuitAwardColumn)GetAwardTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSuitAwardColumn.GetAwardTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.AwardTime
	return v,true
}
func (this *dbPlayerSuitAwardColumn)SetAwardTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerSuitAwardColumn.SetAwardTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.AwardTime = v
	this.m_changed = true
	return true
}
type dbPlayerZanColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerZanData
	m_changed bool
}
func (this *dbPlayerZanColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerZanList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerZanData{}
		d.from_pb(v)
		this.m_data[int32(d.PlayerId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerZanColumn)save( )(data []byte,err error){
	pb := &db.PlayerZanList{}
	pb.List=make([]*db.PlayerZan,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerZanColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerZanColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerZanColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerZanColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerZanColumn)GetAll()(list []dbPlayerZanData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerZanColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerZanData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerZanColumn)Get(id int32)(v *dbPlayerZanData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerZanColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerZanData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerZanColumn)Set(v dbPlayerZanData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerZanColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.PlayerId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.PlayerId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerZanColumn)Add(v *dbPlayerZanData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerZanColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.PlayerId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.PlayerId)
		return false
	}
	d:=&dbPlayerZanData{}
	v.clone_to(d)
	this.m_data[int32(v.PlayerId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerZanColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerZanColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerZanColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerZanColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerZanData)
	this.m_changed = true
	return
}
func (this *dbPlayerZanColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerZanColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerZanColumn)GetZanTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerZanColumn.GetZanTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.ZanTime
	return v,true
}
func (this *dbPlayerZanColumn)SetZanTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerZanColumn.SetZanTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.ZanTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerZanColumn)GetZanNum(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerZanColumn.GetZanNum")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.ZanNum
	return v,true
}
func (this *dbPlayerZanColumn)SetZanNum(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerZanColumn.SetZanNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.ZanNum = v
	this.m_changed = true
	return true
}
func (this *dbPlayerZanColumn)IncbyZanNum(id int32,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerZanColumn.IncbyZanNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerZanData{}
		this.m_data[id] = d
	}
	d.ZanNum +=  v
	this.m_changed = true
	return d.ZanNum
}
type dbPlayerFosterColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerFosterData
	m_changed bool
}
func (this *dbPlayerFosterColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerFosterData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFoster{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerFosterData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerFosterColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFosterColumn)Get( )(v *dbPlayerFosterData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerFosterData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerFosterColumn)Set(v dbPlayerFosterData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerFosterData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerFosterColumn)GetBuildingId( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterColumn.GetBuildingId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.BuildingId
	return
}
func (this *dbPlayerFosterColumn)SetBuildingId(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterColumn.SetBuildingId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.BuildingId = v
	this.m_changed = true
	return
}
func (this *dbPlayerFosterColumn)GetEquippedCardId( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterColumn.GetEquippedCardId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.EquippedCardId
	return
}
func (this *dbPlayerFosterColumn)SetEquippedCardId(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterColumn.SetEquippedCardId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.EquippedCardId = v
	this.m_changed = true
	return
}
func (this *dbPlayerFosterColumn)GetStartTime( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterColumn.GetStartTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.StartTime
	return
}
func (this *dbPlayerFosterColumn)SetStartTime(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterColumn.SetStartTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.StartTime = v
	this.m_changed = true
	return
}
func (this *dbPlayerFosterColumn)GetCatIds( )(v []int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterColumn.GetCatIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]int32, len(this.m_data.CatIds))
	for _ii, _vv := range this.m_data.CatIds {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerFosterColumn)SetCatIds(v []int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterColumn.SetCatIds")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CatIds = make([]int32, len(v))
	for _ii, _vv := range v {
		this.m_data.CatIds[_ii]=_vv
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFosterColumn)GetPlayerCatIds( )(v []int64 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterColumn.GetPlayerCatIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]int64, len(this.m_data.PlayerCatIds))
	for _ii, _vv := range this.m_data.PlayerCatIds {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerFosterColumn)SetPlayerCatIds(v []int64){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterColumn.SetPlayerCatIds")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.PlayerCatIds = make([]int64, len(v))
	for _ii, _vv := range v {
		this.m_data.PlayerCatIds[_ii]=_vv
	}
	this.m_changed = true
	return
}
type dbPlayerFosterCatColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFosterCatData
	m_changed bool
}
func (this *dbPlayerFosterCatColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFosterCatList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFosterCatData{}
		d.from_pb(v)
		this.m_data[int32(d.CatId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFosterCatColumn)save( )(data []byte,err error){
	pb := &db.PlayerFosterCatList{}
	pb.List=make([]*db.PlayerFosterCat,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFosterCatColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFosterCatColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFosterCatColumn)GetAll()(list []dbPlayerFosterCatData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFosterCatData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFosterCatColumn)Get(id int32)(v *dbPlayerFosterCatData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFosterCatData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFosterCatColumn)Set(v dbPlayerFosterCatData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.CatId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.CatId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterCatColumn)Add(v *dbPlayerFosterCatData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.CatId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.CatId)
		return false
	}
	d:=&dbPlayerFosterCatData{}
	v.clone_to(d)
	this.m_data[int32(v.CatId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterCatColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFosterCatColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFosterCatData)
	this.m_changed = true
	return
}
func (this *dbPlayerFosterCatColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerFosterCatColumn)GetStartTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatColumn.GetStartTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.StartTime
	return v,true
}
func (this *dbPlayerFosterCatColumn)SetStartTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatColumn.SetStartTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.StartTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterCatColumn)GetRemainSeconds(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatColumn.GetRemainSeconds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.RemainSeconds
	return v,true
}
func (this *dbPlayerFosterCatColumn)SetRemainSeconds(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatColumn.SetRemainSeconds")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.RemainSeconds = v
	this.m_changed = true
	return true
}
type dbPlayerFosterCatOnFriendColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFosterCatOnFriendData
	m_changed bool
}
func (this *dbPlayerFosterCatOnFriendColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFosterCatOnFriendList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFosterCatOnFriendData{}
		d.from_pb(v)
		this.m_data[int32(d.CatId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFosterCatOnFriendColumn)save( )(data []byte,err error){
	pb := &db.PlayerFosterCatOnFriendList{}
	pb.List=make([]*db.PlayerFosterCatOnFriend,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFosterCatOnFriendColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatOnFriendColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFosterCatOnFriendColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatOnFriendColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFosterCatOnFriendColumn)GetAll()(list []dbPlayerFosterCatOnFriendData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatOnFriendColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFosterCatOnFriendData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFosterCatOnFriendColumn)Get(id int32)(v *dbPlayerFosterCatOnFriendData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatOnFriendColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFosterCatOnFriendData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFosterCatOnFriendColumn)Set(v dbPlayerFosterCatOnFriendData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatOnFriendColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.CatId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.CatId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterCatOnFriendColumn)Add(v *dbPlayerFosterCatOnFriendData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatOnFriendColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.CatId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.CatId)
		return false
	}
	d:=&dbPlayerFosterCatOnFriendData{}
	v.clone_to(d)
	this.m_data[int32(v.CatId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterCatOnFriendColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatOnFriendColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFosterCatOnFriendColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatOnFriendColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFosterCatOnFriendData)
	this.m_changed = true
	return
}
func (this *dbPlayerFosterCatOnFriendColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatOnFriendColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerFosterCatOnFriendColumn)GetFriendId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterCatOnFriendColumn.GetFriendId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.FriendId
	return v,true
}
func (this *dbPlayerFosterCatOnFriendColumn)SetFriendId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterCatOnFriendColumn.SetFriendId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.FriendId = v
	this.m_changed = true
	return true
}
type dbPlayerFosterFriendCatColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFosterFriendCatData
	m_changed bool
}
func (this *dbPlayerFosterFriendCatColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFosterFriendCatList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFosterFriendCatData{}
		d.from_pb(v)
		this.m_data[int32(d.PlayerId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFosterFriendCatColumn)save( )(data []byte,err error){
	pb := &db.PlayerFosterFriendCatList{}
	pb.List=make([]*db.PlayerFosterFriendCat,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFosterFriendCatColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFosterFriendCatColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFosterFriendCatColumn)GetAll()(list []dbPlayerFosterFriendCatData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFosterFriendCatData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFosterFriendCatColumn)Get(id int32)(v *dbPlayerFosterFriendCatData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFosterFriendCatData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFosterFriendCatColumn)Set(v dbPlayerFosterFriendCatData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.PlayerId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.PlayerId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterFriendCatColumn)Add(v *dbPlayerFosterFriendCatData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.PlayerId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.PlayerId)
		return false
	}
	d:=&dbPlayerFosterFriendCatData{}
	v.clone_to(d)
	this.m_data[int32(v.PlayerId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterFriendCatColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFosterFriendCatColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFosterFriendCatData)
	this.m_changed = true
	return
}
func (this *dbPlayerFosterFriendCatColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerFosterFriendCatColumn)GetCatId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetCatId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CatId
	return v,true
}
func (this *dbPlayerFosterFriendCatColumn)SetCatId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.SetCatId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CatId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterFriendCatColumn)GetCatTableId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetCatTableId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CatTableId
	return v,true
}
func (this *dbPlayerFosterFriendCatColumn)SetCatTableId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.SetCatTableId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CatTableId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterFriendCatColumn)GetStartTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetStartTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.StartTime
	return v,true
}
func (this *dbPlayerFosterFriendCatColumn)SetStartTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.SetStartTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.StartTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterFriendCatColumn)GetStartCardId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetStartCardId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.StartCardId
	return v,true
}
func (this *dbPlayerFosterFriendCatColumn)SetStartCardId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.SetStartCardId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.StartCardId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterFriendCatColumn)GetPlayerName(id int32)(v string ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetPlayerName")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.PlayerName
	return v,true
}
func (this *dbPlayerFosterFriendCatColumn)SetPlayerName(id int32,v string)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.SetPlayerName")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.PlayerName = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterFriendCatColumn)GetPlayerLevel(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetPlayerLevel")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.PlayerLevel
	return v,true
}
func (this *dbPlayerFosterFriendCatColumn)SetPlayerLevel(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.SetPlayerLevel")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.PlayerLevel = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterFriendCatColumn)GetPlayerHead(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetPlayerHead")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.PlayerHead
	return v,true
}
func (this *dbPlayerFosterFriendCatColumn)SetPlayerHead(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.SetPlayerHead")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.PlayerHead = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterFriendCatColumn)GetCatLevel(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetCatLevel")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CatLevel
	return v,true
}
func (this *dbPlayerFosterFriendCatColumn)SetCatLevel(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.SetCatLevel")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CatLevel = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterFriendCatColumn)GetCatStar(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetCatStar")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CatStar
	return v,true
}
func (this *dbPlayerFosterFriendCatColumn)SetCatStar(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.SetCatStar")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CatStar = v
	this.m_changed = true
	return true
}
func (this *dbPlayerFosterFriendCatColumn)GetCatNick(id int32)(v string ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFosterFriendCatColumn.GetCatNick")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.CatNick
	return v,true
}
func (this *dbPlayerFosterFriendCatColumn)SetCatNick(id int32,v string)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFosterFriendCatColumn.SetCatNick")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.CatNick = v
	this.m_changed = true
	return true
}
type dbPlayerChatColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerChatData
	m_changed bool
}
func (this *dbPlayerChatColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerChatList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerChatData{}
		d.from_pb(v)
		this.m_data[int32(d.Channel)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerChatColumn)save( )(data []byte,err error){
	pb := &db.PlayerChatList{}
	pb.List=make([]*db.PlayerChat,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerChatColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChatColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerChatColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChatColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerChatColumn)GetAll()(list []dbPlayerChatData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChatColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerChatData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerChatColumn)Get(id int32)(v *dbPlayerChatData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChatColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerChatData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerChatColumn)Set(v dbPlayerChatData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerChatColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Channel)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Channel)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerChatColumn)Add(v *dbPlayerChatData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerChatColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Channel)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Channel)
		return false
	}
	d:=&dbPlayerChatData{}
	v.clone_to(d)
	this.m_data[int32(v.Channel)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerChatColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerChatColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerChatColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerChatColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerChatData)
	this.m_changed = true
	return
}
func (this *dbPlayerChatColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChatColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerChatColumn)GetLastChatTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChatColumn.GetLastChatTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LastChatTime
	return v,true
}
func (this *dbPlayerChatColumn)SetLastChatTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerChatColumn.SetLastChatTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LastChatTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerChatColumn)GetLastPullTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChatColumn.GetLastPullTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LastPullTime
	return v,true
}
func (this *dbPlayerChatColumn)SetLastPullTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerChatColumn.SetLastPullTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LastPullTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerChatColumn)GetLastMsgIndex(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerChatColumn.GetLastMsgIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LastMsgIndex
	return v,true
}
func (this *dbPlayerChatColumn)SetLastMsgIndex(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerChatColumn.SetLastMsgIndex")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LastMsgIndex = v
	this.m_changed = true
	return true
}
type dbPlayerAnouncementColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerAnouncementData
	m_changed bool
}
func (this *dbPlayerAnouncementColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerAnouncementData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerAnouncement{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerAnouncementData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerAnouncementColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerAnouncementColumn)Get( )(v *dbPlayerAnouncementData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerAnouncementColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerAnouncementData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerAnouncementColumn)Set(v dbPlayerAnouncementData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerAnouncementColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerAnouncementData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerAnouncementColumn)GetLastSendTime( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerAnouncementColumn.GetLastSendTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastSendTime
	return
}
func (this *dbPlayerAnouncementColumn)SetLastSendTime(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerAnouncementColumn.SetLastSendTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastSendTime = v
	this.m_changed = true
	return
}
type dbPlayerFirstDrawCardColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerFirstDrawCardData
	m_changed bool
}
func (this *dbPlayerFirstDrawCardColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerFirstDrawCardList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerFirstDrawCardData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFirstDrawCardColumn)save( )(data []byte,err error){
	pb := &db.PlayerFirstDrawCardList{}
	pb.List=make([]*db.PlayerFirstDrawCard,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerFirstDrawCardColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFirstDrawCardColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerFirstDrawCardColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFirstDrawCardColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerFirstDrawCardColumn)GetAll()(list []dbPlayerFirstDrawCardData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFirstDrawCardColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerFirstDrawCardData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerFirstDrawCardColumn)Get(id int32)(v *dbPlayerFirstDrawCardData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFirstDrawCardColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerFirstDrawCardData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerFirstDrawCardColumn)Set(v dbPlayerFirstDrawCardData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFirstDrawCardColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerFirstDrawCardColumn)Add(v *dbPlayerFirstDrawCardData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFirstDrawCardColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerFirstDrawCardData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerFirstDrawCardColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerFirstDrawCardColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerFirstDrawCardColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerFirstDrawCardColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerFirstDrawCardData)
	this.m_changed = true
	return
}
func (this *dbPlayerFirstDrawCardColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFirstDrawCardColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerFirstDrawCardColumn)GetDrawed(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerFirstDrawCardColumn.GetDrawed")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Drawed
	return v,true
}
func (this *dbPlayerFirstDrawCardColumn)SetDrawed(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerFirstDrawCardColumn.SetDrawed")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Drawed = v
	this.m_changed = true
	return true
}
type dbPlayerTalkForbidColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerTalkForbidData
	m_changed bool
}
func (this *dbPlayerTalkForbidColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerTalkForbidData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerTalkForbid{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerTalkForbidData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerTalkForbidColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerTalkForbidColumn)Get( )(v *dbPlayerTalkForbidData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTalkForbidColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerTalkForbidData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerTalkForbidColumn)Set(v dbPlayerTalkForbidData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerTalkForbidColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerTalkForbidData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerTalkForbidColumn)GetEndUnix( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTalkForbidColumn.GetEndUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.EndUnix
	return
}
func (this *dbPlayerTalkForbidColumn)SetEndUnix(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerTalkForbidColumn.SetEndUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.EndUnix = v
	this.m_changed = true
	return
}
func (this *dbPlayerTalkForbidColumn)GetForbidReason( )(v string ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerTalkForbidColumn.GetForbidReason")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.ForbidReason
	return
}
func (this *dbPlayerTalkForbidColumn)SetForbidReason(v string){
	this.m_row.m_lock.UnSafeLock("dbPlayerTalkForbidColumn.SetForbidReason")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.ForbidReason = v
	this.m_changed = true
	return
}
type dbPlayerServerRewardColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerServerRewardData
	m_changed bool
}
func (this *dbPlayerServerRewardColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerServerRewardList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerServerRewardData{}
		d.from_pb(v)
		this.m_data[int32(d.RewardId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerServerRewardColumn)save( )(data []byte,err error){
	pb := &db.PlayerServerRewardList{}
	pb.List=make([]*db.PlayerServerReward,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerServerRewardColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerServerRewardColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerServerRewardColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerServerRewardColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerServerRewardColumn)GetAll()(list []dbPlayerServerRewardData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerServerRewardColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerServerRewardData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerServerRewardColumn)Get(id int32)(v *dbPlayerServerRewardData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerServerRewardColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerServerRewardData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerServerRewardColumn)Set(v dbPlayerServerRewardData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerServerRewardColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.RewardId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.RewardId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerServerRewardColumn)Add(v *dbPlayerServerRewardData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerServerRewardColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.RewardId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.RewardId)
		return false
	}
	d:=&dbPlayerServerRewardData{}
	v.clone_to(d)
	this.m_data[int32(v.RewardId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerServerRewardColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerServerRewardColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerServerRewardColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerServerRewardColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerServerRewardData)
	this.m_changed = true
	return
}
func (this *dbPlayerServerRewardColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerServerRewardColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerServerRewardColumn)GetEndUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerServerRewardColumn.GetEndUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.EndUnix
	return v,true
}
func (this *dbPlayerServerRewardColumn)SetEndUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerServerRewardColumn.SetEndUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.EndUnix = v
	this.m_changed = true
	return true
}
type dbPlayerMailCommonColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerMailCommonData
	m_changed bool
}
func (this *dbPlayerMailCommonColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerMailCommonData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerMailCommon{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerMailCommonData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerMailCommonColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerMailCommonColumn)Get( )(v *dbPlayerMailCommonData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailCommonColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerMailCommonData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerMailCommonColumn)Set(v dbPlayerMailCommonData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailCommonColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerMailCommonData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerMailCommonColumn)GetCurrId( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailCommonColumn.GetCurrId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.CurrId
	return
}
func (this *dbPlayerMailCommonColumn)SetCurrId(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailCommonColumn.SetCurrId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CurrId = v
	this.m_changed = true
	return
}
func (this *dbPlayerMailCommonColumn)IncbyCurrId(v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailCommonColumn.IncbyCurrId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CurrId += v
	this.m_changed = true
	return this.m_data.CurrId
}
func (this *dbPlayerMailCommonColumn)GetLastSendPlayerMailTime( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailCommonColumn.GetLastSendPlayerMailTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.LastSendPlayerMailTime
	return
}
func (this *dbPlayerMailCommonColumn)SetLastSendPlayerMailTime(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailCommonColumn.SetLastSendPlayerMailTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.LastSendPlayerMailTime = v
	this.m_changed = true
	return
}
type dbPlayerMailColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerMailData
	m_changed bool
}
func (this *dbPlayerMailColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerMailList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerMailData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerMailColumn)save( )(data []byte,err error){
	pb := &db.PlayerMailList{}
	pb.List=make([]*db.PlayerMail,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerMailColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerMailColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerMailColumn)GetAll()(list []dbPlayerMailData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerMailData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerMailColumn)Get(id int32)(v *dbPlayerMailData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerMailData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerMailColumn)Set(v dbPlayerMailData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)Add(v *dbPlayerMailData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerMailData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerMailColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerMailData)
	this.m_changed = true
	return
}
func (this *dbPlayerMailColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerMailColumn)GetType(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetType")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = int32(d.Type)
	return v,true
}
func (this *dbPlayerMailColumn)SetType(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetType")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Type = int8(v)
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)GetTitle(id int32)(v string ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetTitle")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Title
	return v,true
}
func (this *dbPlayerMailColumn)SetTitle(id int32,v string)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetTitle")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Title = v
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)GetContent(id int32)(v string ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetContent")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Content
	return v,true
}
func (this *dbPlayerMailColumn)SetContent(id int32,v string)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetContent")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Content = v
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)GetSendUnix(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetSendUnix")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.SendUnix
	return v,true
}
func (this *dbPlayerMailColumn)SetSendUnix(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetSendUnix")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.SendUnix = v
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)GetAttachItemIds(id int32)(v []int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetAttachItemIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]int32, len(d.AttachItemIds))
	for _ii, _vv := range d.AttachItemIds {
		v[_ii]=_vv
	}
	return v,true
}
func (this *dbPlayerMailColumn)SetAttachItemIds(id int32,v []int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetAttachItemIds")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.AttachItemIds = make([]int32, len(v))
	for _ii, _vv := range v {
		d.AttachItemIds[_ii]=_vv
	}
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)GetAttachItemNums(id int32)(v []int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetAttachItemNums")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]int32, len(d.AttachItemNums))
	for _ii, _vv := range d.AttachItemNums {
		v[_ii]=_vv
	}
	return v,true
}
func (this *dbPlayerMailColumn)SetAttachItemNums(id int32,v []int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetAttachItemNums")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.AttachItemNums = make([]int32, len(v))
	for _ii, _vv := range v {
		d.AttachItemNums[_ii]=_vv
	}
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)GetIsRead(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetIsRead")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.IsRead
	return v,true
}
func (this *dbPlayerMailColumn)SetIsRead(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetIsRead")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.IsRead = v
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)GetIsGetAttached(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetIsGetAttached")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.IsGetAttached
	return v,true
}
func (this *dbPlayerMailColumn)SetIsGetAttached(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetIsGetAttached")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.IsGetAttached = v
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)GetSenderId(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetSenderId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.SenderId
	return v,true
}
func (this *dbPlayerMailColumn)SetSenderId(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetSenderId")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.SenderId = v
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)GetSenderName(id int32)(v string ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetSenderName")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.SenderName
	return v,true
}
func (this *dbPlayerMailColumn)SetSenderName(id int32,v string)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetSenderName")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.SenderName = v
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)GetSubtype(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetSubtype")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Subtype
	return v,true
}
func (this *dbPlayerMailColumn)SetSubtype(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetSubtype")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Subtype = v
	this.m_changed = true
	return true
}
func (this *dbPlayerMailColumn)GetExtraValue(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerMailColumn.GetExtraValue")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.ExtraValue
	return v,true
}
func (this *dbPlayerMailColumn)SetExtraValue(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerMailColumn.SetExtraValue")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.ExtraValue = v
	this.m_changed = true
	return true
}
type dbPlayerPayCommonColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerPayCommonData
	m_changed bool
}
func (this *dbPlayerPayCommonColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerPayCommonData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerPayCommon{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerPayCommonData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerPayCommonColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerPayCommonColumn)Get( )(v *dbPlayerPayCommonData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayCommonColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerPayCommonData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerPayCommonColumn)Set(v dbPlayerPayCommonData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayCommonColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerPayCommonData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerPayCommonColumn)GetFirstPayState( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayCommonColumn.GetFirstPayState")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.FirstPayState
	return
}
func (this *dbPlayerPayCommonColumn)SetFirstPayState(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayCommonColumn.SetFirstPayState")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.FirstPayState = v
	this.m_changed = true
	return
}
type dbPlayerPayColumn struct{
	m_row *dbPlayerRow
	m_data map[string]*dbPlayerPayData
	m_changed bool
}
func (this *dbPlayerPayColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerPayList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerPayData{}
		d.from_pb(v)
		this.m_data[string(d.BundleId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerPayColumn)save( )(data []byte,err error){
	pb := &db.PlayerPayList{}
	pb.List=make([]*db.PlayerPay,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerPayColumn)HasIndex(id string)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerPayColumn)GetAllIndex()(list []string){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]string, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerPayColumn)GetAll()(list []dbPlayerPayData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerPayData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerPayColumn)Get(id string)(v *dbPlayerPayData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerPayData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerPayColumn)Set(v dbPlayerPayData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[string(v.BundleId)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.BundleId)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerPayColumn)Add(v *dbPlayerPayData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[string(v.BundleId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.BundleId)
		return false
	}
	d:=&dbPlayerPayData{}
	v.clone_to(d)
	this.m_data[string(v.BundleId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerPayColumn)Remove(id string){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerPayColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[string]*dbPlayerPayData)
	this.m_changed = true
	return
}
func (this *dbPlayerPayColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerPayColumn)GetLastPayedTime(id string)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayColumn.GetLastPayedTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LastPayedTime
	return v,true
}
func (this *dbPlayerPayColumn)SetLastPayedTime(id string,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayColumn.SetLastPayedTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LastPayedTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerPayColumn)GetLastAwardTime(id string)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayColumn.GetLastAwardTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.LastAwardTime
	return v,true
}
func (this *dbPlayerPayColumn)SetLastAwardTime(id string,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayColumn.SetLastAwardTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.LastAwardTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerPayColumn)GetSendMailNum(id string)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayColumn.GetSendMailNum")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.SendMailNum
	return v,true
}
func (this *dbPlayerPayColumn)SetSendMailNum(id string,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayColumn.SetSendMailNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.SendMailNum = v
	this.m_changed = true
	return true
}
func (this *dbPlayerPayColumn)IncbySendMailNum(id string,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayColumn.IncbySendMailNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerPayData{}
		this.m_data[id] = d
	}
	d.SendMailNum +=  v
	this.m_changed = true
	return d.SendMailNum
}
func (this *dbPlayerPayColumn)GetChargeNum(id string)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerPayColumn.GetChargeNum")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.ChargeNum
	return v,true
}
func (this *dbPlayerPayColumn)SetChargeNum(id string,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayColumn.SetChargeNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.ChargeNum = v
	this.m_changed = true
	return true
}
func (this *dbPlayerPayColumn)IncbyChargeNum(id string,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerPayColumn.IncbyChargeNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerPayData{}
		this.m_data[id] = d
	}
	d.ChargeNum +=  v
	this.m_changed = true
	return d.ChargeNum
}
type dbPlayerGuideDataColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerGuideDataData
	m_changed bool
}
func (this *dbPlayerGuideDataColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerGuideDataData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerGuideData{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerGuideDataData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerGuideDataColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerGuideDataColumn)Get( )(v *dbPlayerGuideDataData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerGuideDataColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerGuideDataData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerGuideDataColumn)Set(v dbPlayerGuideDataData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerGuideDataColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerGuideDataData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerGuideDataColumn)GetData( )(v []byte){
	this.m_row.m_lock.UnSafeRLock("dbPlayerGuideDataColumn.GetData")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]byte, len(this.m_data.Data))
	for _ii, _vv := range this.m_data.Data {
		v[_ii]=_vv
	}
	return
}
func (this *dbPlayerGuideDataColumn)SetData(v []byte){
	this.m_row.m_lock.UnSafeLock("dbPlayerGuideDataColumn.SetData")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Data = make([]byte, len(v))
	for _ii, _vv := range v {
		this.m_data.Data[_ii]=_vv
	}
	this.m_changed = true
	return
}
type dbPlayerActivityDataColumn struct{
	m_row *dbPlayerRow
	m_data map[int32]*dbPlayerActivityDataData
	m_changed bool
}
func (this *dbPlayerActivityDataColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerActivityDataList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerActivityDataData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerActivityDataColumn)save( )(data []byte,err error){
	pb := &db.PlayerActivityDataList{}
	pb.List=make([]*db.PlayerActivityData,len(this.m_data))
	i:=0
	for _, v := range this.m_data {
		pb.List[i] = v.to_pb()
		i++
	}
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerActivityDataColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityDataColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerActivityDataColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityDataColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerActivityDataColumn)GetAll()(list []dbPlayerActivityDataData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityDataColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerActivityDataData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerActivityDataColumn)Get(id int32)(v *dbPlayerActivityDataData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityDataColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerActivityDataData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerActivityDataColumn)Set(v dbPlayerActivityDataData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityDataColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[int32(v.Id)]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	v.clone_to(d)
	this.m_changed = true
	return true
}
func (this *dbPlayerActivityDataColumn)Add(v *dbPlayerActivityDataData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityDataColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerActivityDataData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerActivityDataColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityDataColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerActivityDataColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityDataColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerActivityDataData)
	this.m_changed = true
	return
}
func (this *dbPlayerActivityDataColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityDataColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerActivityDataColumn)GetSubIds(id int32)(v []int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityDataColumn.GetSubIds")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]int32, len(d.SubIds))
	for _ii, _vv := range d.SubIds {
		v[_ii]=_vv
	}
	return v,true
}
func (this *dbPlayerActivityDataColumn)SetSubIds(id int32,v []int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityDataColumn.SetSubIds")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.SubIds = make([]int32, len(v))
	for _ii, _vv := range v {
		d.SubIds[_ii]=_vv
	}
	this.m_changed = true
	return true
}
func (this *dbPlayerActivityDataColumn)GetSubValues(id int32)(v []int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityDataColumn.GetSubValues")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = make([]int32, len(d.SubValues))
	for _ii, _vv := range d.SubValues {
		v[_ii]=_vv
	}
	return v,true
}
func (this *dbPlayerActivityDataColumn)SetSubValues(id int32,v []int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityDataColumn.SetSubValues")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.SubValues = make([]int32, len(v))
	for _ii, _vv := range v {
		d.SubValues[_ii]=_vv
	}
	this.m_changed = true
	return true
}
func (this *dbPlayerActivityDataColumn)GetSubNum(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerActivityDataColumn.GetSubNum")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.SubNum
	return v,true
}
func (this *dbPlayerActivityDataColumn)SetSubNum(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityDataColumn.SetSubNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.SubNum = v
	this.m_changed = true
	return true
}
func (this *dbPlayerActivityDataColumn)IncbySubNum(id int32,v int32)(r int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerActivityDataColumn.IncbySubNum")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		d = &dbPlayerActivityDataData{}
		this.m_data[id] = d
	}
	d.SubNum +=  v
	this.m_changed = true
	return d.SubNum
}
type dbPlayerSysMailColumn struct{
	m_row *dbPlayerRow
	m_data *dbPlayerSysMailData
	m_changed bool
}
func (this *dbPlayerSysMailColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerSysMailData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerSysMail{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerSysMailData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerSysMailColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerSysMailColumn)Get( )(v *dbPlayerSysMailData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSysMailColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerSysMailData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerSysMailColumn)Set(v dbPlayerSysMailData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerSysMailColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerSysMailData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerSysMailColumn)GetCurrId( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerSysMailColumn.GetCurrId")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.CurrId
	return
}
func (this *dbPlayerSysMailColumn)SetCurrId(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerSysMailColumn.SetCurrId")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.CurrId = v
	this.m_changed = true
	return
}
type dbPlayerRow struct {
	m_table *dbPlayerTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_PlayerId        int32
	m_UniqueId_changed bool
	m_UniqueId string
	m_Account_changed bool
	m_Account string
	m_Name_changed bool
	m_Name string
	m_Token_changed bool
	m_Token string
	m_Level_changed bool
	m_Level int32
	Info dbPlayerInfoColumn
	Stages dbPlayerStageColumn
	ChapterUnLock dbPlayerChapterUnLockColumn
	Items dbPlayerItemColumn
	Areas dbPlayerAreaColumn
	Buildings dbPlayerBuildingColumn
	BuildingDepots dbPlayerBuildingDepotColumn
	DepotBuildingFormulas dbPlayerDepotBuildingFormulaColumn
	MakingFormulaBuildings dbPlayerMakingFormulaBuildingColumn
	Crops dbPlayerCropColumn
	Cats dbPlayerCatColumn
	CatHouses dbPlayerCatHouseColumn
	ShopItems dbPlayerShopItemColumn
	ShopLimitedInfos dbPlayerShopLimitedInfoColumn
	Chests dbPlayerChestColumn
	PayBacks dbPlayerPayBackColumn
	Options dbPlayerOptionsColumn
	TaskCommon dbPlayerTaskCommonColumn
	Tasks dbPlayerTaskColumn
	FinishedTasks dbPlayerFinishedTaskColumn
	DailyTaskAllDailys dbPlayerDailyTaskAllDailyColumn
	SevenActivitys dbPlayerSevenActivityColumn
	SignInfo dbPlayerSignInfoColumn
	Guidess dbPlayerGuidesColumn
	FriendRelative dbPlayerFriendRelativeColumn
	Friends dbPlayerFriendColumn
	FriendRecommends dbPlayerFriendRecommendColumn
	FriendAsks dbPlayerFriendAskColumn
	FriendReqs dbPlayerFriendReqColumn
	FriendPoints dbPlayerFriendPointColumn
	FriendChatUnreadIds dbPlayerFriendChatUnreadIdColumn
	FriendChatUnreadMessages dbPlayerFriendChatUnreadMessageColumn
	FocusPlayers dbPlayerFocusPlayerColumn
	BeFocusPlayers dbPlayerBeFocusPlayerColumn
	CustomData dbPlayerCustomDataColumn
	ChaterOpenRequest dbPlayerChaterOpenRequestColumn
	Expeditions dbPlayerExpeditionColumn
	HandbookItems dbPlayerHandbookItemColumn
	HeadItems dbPlayerHeadItemColumn
	Activitys dbPlayerActivityColumn
	SuitAwards dbPlayerSuitAwardColumn
	Zans dbPlayerZanColumn
	Foster dbPlayerFosterColumn
	FosterCats dbPlayerFosterCatColumn
	FosterCatOnFriends dbPlayerFosterCatOnFriendColumn
	FosterFriendCats dbPlayerFosterFriendCatColumn
	Chats dbPlayerChatColumn
	Anouncement dbPlayerAnouncementColumn
	FirstDrawCards dbPlayerFirstDrawCardColumn
	TalkForbid dbPlayerTalkForbidColumn
	ServerRewards dbPlayerServerRewardColumn
	MailCommon dbPlayerMailCommonColumn
	Mails dbPlayerMailColumn
	PayCommon dbPlayerPayCommonColumn
	Pays dbPlayerPayColumn
	GuideData dbPlayerGuideDataColumn
	ActivityDatas dbPlayerActivityDataColumn
	SysMail dbPlayerSysMailColumn
}
func new_dbPlayerRow(table *dbPlayerTable, PlayerId int32) (r *dbPlayerRow) {
	this := &dbPlayerRow{}
	this.m_table = table
	this.m_PlayerId = PlayerId
	this.m_lock = NewRWMutex()
	this.m_UniqueId_changed=true
	this.m_Account_changed=true
	this.m_Name_changed=true
	this.m_Token_changed=true
	this.m_Level_changed=true
	this.Info.m_row=this
	this.Info.m_data=&dbPlayerInfoData{}
	this.Stages.m_row=this
	this.Stages.m_data=make(map[int32]*dbPlayerStageData)
	this.ChapterUnLock.m_row=this
	this.ChapterUnLock.m_data=&dbPlayerChapterUnLockData{}
	this.Items.m_row=this
	this.Items.m_data=make(map[int32]*dbPlayerItemData)
	this.Areas.m_row=this
	this.Areas.m_data=make(map[int32]*dbPlayerAreaData)
	this.Buildings.m_row=this
	this.Buildings.m_data=make(map[int32]*dbPlayerBuildingData)
	this.BuildingDepots.m_row=this
	this.BuildingDepots.m_data=make(map[int32]*dbPlayerBuildingDepotData)
	this.DepotBuildingFormulas.m_row=this
	this.DepotBuildingFormulas.m_data=make(map[int32]*dbPlayerDepotBuildingFormulaData)
	this.MakingFormulaBuildings.m_row=this
	this.MakingFormulaBuildings.m_data=make(map[int32]*dbPlayerMakingFormulaBuildingData)
	this.Crops.m_row=this
	this.Crops.m_data=make(map[int32]*dbPlayerCropData)
	this.Cats.m_row=this
	this.Cats.m_data=make(map[int32]*dbPlayerCatData)
	this.CatHouses.m_row=this
	this.CatHouses.m_data=make(map[int32]*dbPlayerCatHouseData)
	this.ShopItems.m_row=this
	this.ShopItems.m_data=make(map[int32]*dbPlayerShopItemData)
	this.ShopLimitedInfos.m_row=this
	this.ShopLimitedInfos.m_data=make(map[int32]*dbPlayerShopLimitedInfoData)
	this.Chests.m_row=this
	this.Chests.m_data=make(map[int32]*dbPlayerChestData)
	this.PayBacks.m_row=this
	this.PayBacks.m_data=make(map[int32]*dbPlayerPayBackData)
	this.Options.m_row=this
	this.Options.m_data=&dbPlayerOptionsData{}
	this.TaskCommon.m_row=this
	this.TaskCommon.m_data=&dbPlayerTaskCommonData{}
	this.Tasks.m_row=this
	this.Tasks.m_data=make(map[int32]*dbPlayerTaskData)
	this.FinishedTasks.m_row=this
	this.FinishedTasks.m_data=make(map[int32]*dbPlayerFinishedTaskData)
	this.DailyTaskAllDailys.m_row=this
	this.DailyTaskAllDailys.m_data=make(map[int32]*dbPlayerDailyTaskAllDailyData)
	this.SevenActivitys.m_row=this
	this.SevenActivitys.m_data=make(map[int32]*dbPlayerSevenActivityData)
	this.SignInfo.m_row=this
	this.SignInfo.m_data=&dbPlayerSignInfoData{}
	this.Guidess.m_row=this
	this.Guidess.m_data=make(map[int32]*dbPlayerGuidesData)
	this.FriendRelative.m_row=this
	this.FriendRelative.m_data=&dbPlayerFriendRelativeData{}
	this.Friends.m_row=this
	this.Friends.m_data=make(map[int32]*dbPlayerFriendData)
	this.FriendRecommends.m_row=this
	this.FriendRecommends.m_data=make(map[int32]*dbPlayerFriendRecommendData)
	this.FriendAsks.m_row=this
	this.FriendAsks.m_data=make(map[int32]*dbPlayerFriendAskData)
	this.FriendReqs.m_row=this
	this.FriendReqs.m_data=make(map[int32]*dbPlayerFriendReqData)
	this.FriendPoints.m_row=this
	this.FriendPoints.m_data=make(map[int32]*dbPlayerFriendPointData)
	this.FriendChatUnreadIds.m_row=this
	this.FriendChatUnreadIds.m_data=make(map[int32]*dbPlayerFriendChatUnreadIdData)
	this.FriendChatUnreadMessages.m_row=this
	this.FriendChatUnreadMessages.m_data=make(map[int64]*dbPlayerFriendChatUnreadMessageData)
	this.FocusPlayers.m_row=this
	this.FocusPlayers.m_data=make(map[int32]*dbPlayerFocusPlayerData)
	this.BeFocusPlayers.m_row=this
	this.BeFocusPlayers.m_data=make(map[int32]*dbPlayerBeFocusPlayerData)
	this.CustomData.m_row=this
	this.CustomData.m_data=&dbPlayerCustomDataData{}
	this.ChaterOpenRequest.m_row=this
	this.ChaterOpenRequest.m_data=&dbPlayerChaterOpenRequestData{}
	this.Expeditions.m_row=this
	this.Expeditions.m_data=make(map[int32]*dbPlayerExpeditionData)
	this.HandbookItems.m_row=this
	this.HandbookItems.m_data=make(map[int32]*dbPlayerHandbookItemData)
	this.HeadItems.m_row=this
	this.HeadItems.m_data=make(map[int32]*dbPlayerHeadItemData)
	this.Activitys.m_row=this
	this.Activitys.m_data=make(map[int32]*dbPlayerActivityData)
	this.SuitAwards.m_row=this
	this.SuitAwards.m_data=make(map[int32]*dbPlayerSuitAwardData)
	this.Zans.m_row=this
	this.Zans.m_data=make(map[int32]*dbPlayerZanData)
	this.Foster.m_row=this
	this.Foster.m_data=&dbPlayerFosterData{}
	this.FosterCats.m_row=this
	this.FosterCats.m_data=make(map[int32]*dbPlayerFosterCatData)
	this.FosterCatOnFriends.m_row=this
	this.FosterCatOnFriends.m_data=make(map[int32]*dbPlayerFosterCatOnFriendData)
	this.FosterFriendCats.m_row=this
	this.FosterFriendCats.m_data=make(map[int32]*dbPlayerFosterFriendCatData)
	this.Chats.m_row=this
	this.Chats.m_data=make(map[int32]*dbPlayerChatData)
	this.Anouncement.m_row=this
	this.Anouncement.m_data=&dbPlayerAnouncementData{}
	this.FirstDrawCards.m_row=this
	this.FirstDrawCards.m_data=make(map[int32]*dbPlayerFirstDrawCardData)
	this.TalkForbid.m_row=this
	this.TalkForbid.m_data=&dbPlayerTalkForbidData{}
	this.ServerRewards.m_row=this
	this.ServerRewards.m_data=make(map[int32]*dbPlayerServerRewardData)
	this.MailCommon.m_row=this
	this.MailCommon.m_data=&dbPlayerMailCommonData{}
	this.Mails.m_row=this
	this.Mails.m_data=make(map[int32]*dbPlayerMailData)
	this.PayCommon.m_row=this
	this.PayCommon.m_data=&dbPlayerPayCommonData{}
	this.Pays.m_row=this
	this.Pays.m_data=make(map[string]*dbPlayerPayData)
	this.GuideData.m_row=this
	this.GuideData.m_data=&dbPlayerGuideDataData{}
	this.ActivityDatas.m_row=this
	this.ActivityDatas.m_data=make(map[int32]*dbPlayerActivityDataData)
	this.SysMail.m_row=this
	this.SysMail.m_data=&dbPlayerSysMailData{}
	return this
}
func (this *dbPlayerRow) GetPlayerId() (r int32) {
	return this.m_PlayerId
}
func (this *dbPlayerRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbPlayerRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(64)
		db_args.Push(this.m_PlayerId)
		db_args.Push(this.m_UniqueId)
		db_args.Push(this.m_Account)
		db_args.Push(this.m_Name)
		db_args.Push(this.m_Token)
		db_args.Push(this.m_Level)
		dInfo,db_err:=this.Info.save()
		if db_err!=nil{
			log.Error("insert save Info failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dInfo)
		dStages,db_err:=this.Stages.save()
		if db_err!=nil{
			log.Error("insert save Stage failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dStages)
		dChapterUnLock,db_err:=this.ChapterUnLock.save()
		if db_err!=nil{
			log.Error("insert save ChapterUnLock failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dChapterUnLock)
		dItems,db_err:=this.Items.save()
		if db_err!=nil{
			log.Error("insert save Item failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dItems)
		dAreas,db_err:=this.Areas.save()
		if db_err!=nil{
			log.Error("insert save Area failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dAreas)
		dBuildings,db_err:=this.Buildings.save()
		if db_err!=nil{
			log.Error("insert save Building failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dBuildings)
		dBuildingDepots,db_err:=this.BuildingDepots.save()
		if db_err!=nil{
			log.Error("insert save BuildingDepot failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dBuildingDepots)
		dDepotBuildingFormulas,db_err:=this.DepotBuildingFormulas.save()
		if db_err!=nil{
			log.Error("insert save DepotBuildingFormula failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dDepotBuildingFormulas)
		dMakingFormulaBuildings,db_err:=this.MakingFormulaBuildings.save()
		if db_err!=nil{
			log.Error("insert save MakingFormulaBuilding failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dMakingFormulaBuildings)
		dCrops,db_err:=this.Crops.save()
		if db_err!=nil{
			log.Error("insert save Crop failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dCrops)
		dCats,db_err:=this.Cats.save()
		if db_err!=nil{
			log.Error("insert save Cat failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dCats)
		dCatHouses,db_err:=this.CatHouses.save()
		if db_err!=nil{
			log.Error("insert save CatHouse failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dCatHouses)
		dShopItems,db_err:=this.ShopItems.save()
		if db_err!=nil{
			log.Error("insert save ShopItem failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dShopItems)
		dShopLimitedInfos,db_err:=this.ShopLimitedInfos.save()
		if db_err!=nil{
			log.Error("insert save ShopLimitedInfo failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dShopLimitedInfos)
		dChests,db_err:=this.Chests.save()
		if db_err!=nil{
			log.Error("insert save Chest failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dChests)
		dPayBacks,db_err:=this.PayBacks.save()
		if db_err!=nil{
			log.Error("insert save PayBack failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dPayBacks)
		dOptions,db_err:=this.Options.save()
		if db_err!=nil{
			log.Error("insert save Options failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dOptions)
		dTaskCommon,db_err:=this.TaskCommon.save()
		if db_err!=nil{
			log.Error("insert save TaskCommon failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dTaskCommon)
		dTasks,db_err:=this.Tasks.save()
		if db_err!=nil{
			log.Error("insert save Task failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dTasks)
		dFinishedTasks,db_err:=this.FinishedTasks.save()
		if db_err!=nil{
			log.Error("insert save FinishedTask failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFinishedTasks)
		dDailyTaskAllDailys,db_err:=this.DailyTaskAllDailys.save()
		if db_err!=nil{
			log.Error("insert save DailyTaskAllDaily failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dDailyTaskAllDailys)
		dSevenActivitys,db_err:=this.SevenActivitys.save()
		if db_err!=nil{
			log.Error("insert save SevenActivity failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dSevenActivitys)
		dSignInfo,db_err:=this.SignInfo.save()
		if db_err!=nil{
			log.Error("insert save SignInfo failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dSignInfo)
		dGuidess,db_err:=this.Guidess.save()
		if db_err!=nil{
			log.Error("insert save Guides failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dGuidess)
		dFriendRelative,db_err:=this.FriendRelative.save()
		if db_err!=nil{
			log.Error("insert save FriendRelative failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFriendRelative)
		dFriends,db_err:=this.Friends.save()
		if db_err!=nil{
			log.Error("insert save Friend failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFriends)
		dFriendRecommends,db_err:=this.FriendRecommends.save()
		if db_err!=nil{
			log.Error("insert save FriendRecommend failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFriendRecommends)
		dFriendAsks,db_err:=this.FriendAsks.save()
		if db_err!=nil{
			log.Error("insert save FriendAsk failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFriendAsks)
		dFriendReqs,db_err:=this.FriendReqs.save()
		if db_err!=nil{
			log.Error("insert save FriendReq failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFriendReqs)
		dFriendPoints,db_err:=this.FriendPoints.save()
		if db_err!=nil{
			log.Error("insert save FriendPoint failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFriendPoints)
		dFriendChatUnreadIds,db_err:=this.FriendChatUnreadIds.save()
		if db_err!=nil{
			log.Error("insert save FriendChatUnreadId failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFriendChatUnreadIds)
		dFriendChatUnreadMessages,db_err:=this.FriendChatUnreadMessages.save()
		if db_err!=nil{
			log.Error("insert save FriendChatUnreadMessage failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFriendChatUnreadMessages)
		dFocusPlayers,db_err:=this.FocusPlayers.save()
		if db_err!=nil{
			log.Error("insert save FocusPlayer failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFocusPlayers)
		dBeFocusPlayers,db_err:=this.BeFocusPlayers.save()
		if db_err!=nil{
			log.Error("insert save BeFocusPlayer failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dBeFocusPlayers)
		dCustomData,db_err:=this.CustomData.save()
		if db_err!=nil{
			log.Error("insert save CustomData failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dCustomData)
		dChaterOpenRequest,db_err:=this.ChaterOpenRequest.save()
		if db_err!=nil{
			log.Error("insert save ChaterOpenRequest failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dChaterOpenRequest)
		dExpeditions,db_err:=this.Expeditions.save()
		if db_err!=nil{
			log.Error("insert save Expedition failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dExpeditions)
		dHandbookItems,db_err:=this.HandbookItems.save()
		if db_err!=nil{
			log.Error("insert save HandbookItem failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dHandbookItems)
		dHeadItems,db_err:=this.HeadItems.save()
		if db_err!=nil{
			log.Error("insert save HeadItem failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dHeadItems)
		dActivitys,db_err:=this.Activitys.save()
		if db_err!=nil{
			log.Error("insert save Activity failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dActivitys)
		dSuitAwards,db_err:=this.SuitAwards.save()
		if db_err!=nil{
			log.Error("insert save SuitAward failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dSuitAwards)
		dZans,db_err:=this.Zans.save()
		if db_err!=nil{
			log.Error("insert save Zan failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dZans)
		dFoster,db_err:=this.Foster.save()
		if db_err!=nil{
			log.Error("insert save Foster failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFoster)
		dFosterCats,db_err:=this.FosterCats.save()
		if db_err!=nil{
			log.Error("insert save FosterCat failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFosterCats)
		dFosterCatOnFriends,db_err:=this.FosterCatOnFriends.save()
		if db_err!=nil{
			log.Error("insert save FosterCatOnFriend failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFosterCatOnFriends)
		dFosterFriendCats,db_err:=this.FosterFriendCats.save()
		if db_err!=nil{
			log.Error("insert save FosterFriendCat failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFosterFriendCats)
		dChats,db_err:=this.Chats.save()
		if db_err!=nil{
			log.Error("insert save Chat failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dChats)
		dAnouncement,db_err:=this.Anouncement.save()
		if db_err!=nil{
			log.Error("insert save Anouncement failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dAnouncement)
		dFirstDrawCards,db_err:=this.FirstDrawCards.save()
		if db_err!=nil{
			log.Error("insert save FirstDrawCard failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dFirstDrawCards)
		dTalkForbid,db_err:=this.TalkForbid.save()
		if db_err!=nil{
			log.Error("insert save TalkForbid failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dTalkForbid)
		dServerRewards,db_err:=this.ServerRewards.save()
		if db_err!=nil{
			log.Error("insert save ServerReward failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dServerRewards)
		dMailCommon,db_err:=this.MailCommon.save()
		if db_err!=nil{
			log.Error("insert save MailCommon failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dMailCommon)
		dMails,db_err:=this.Mails.save()
		if db_err!=nil{
			log.Error("insert save Mail failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dMails)
		dPayCommon,db_err:=this.PayCommon.save()
		if db_err!=nil{
			log.Error("insert save PayCommon failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dPayCommon)
		dPays,db_err:=this.Pays.save()
		if db_err!=nil{
			log.Error("insert save Pay failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dPays)
		dGuideData,db_err:=this.GuideData.save()
		if db_err!=nil{
			log.Error("insert save GuideData failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dGuideData)
		dActivityDatas,db_err:=this.ActivityDatas.save()
		if db_err!=nil{
			log.Error("insert save ActivityData failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dActivityDatas)
		dSysMail,db_err:=this.SysMail.save()
		if db_err!=nil{
			log.Error("insert save SysMail failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dSysMail)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_UniqueId_changed||this.m_Account_changed||this.m_Name_changed||this.m_Token_changed||this.m_Level_changed||this.Info.m_changed||this.Stages.m_changed||this.ChapterUnLock.m_changed||this.Items.m_changed||this.Areas.m_changed||this.Buildings.m_changed||this.BuildingDepots.m_changed||this.DepotBuildingFormulas.m_changed||this.MakingFormulaBuildings.m_changed||this.Crops.m_changed||this.Cats.m_changed||this.CatHouses.m_changed||this.ShopItems.m_changed||this.ShopLimitedInfos.m_changed||this.Chests.m_changed||this.PayBacks.m_changed||this.Options.m_changed||this.TaskCommon.m_changed||this.Tasks.m_changed||this.FinishedTasks.m_changed||this.DailyTaskAllDailys.m_changed||this.SevenActivitys.m_changed||this.SignInfo.m_changed||this.Guidess.m_changed||this.FriendRelative.m_changed||this.Friends.m_changed||this.FriendRecommends.m_changed||this.FriendAsks.m_changed||this.FriendReqs.m_changed||this.FriendPoints.m_changed||this.FriendChatUnreadIds.m_changed||this.FriendChatUnreadMessages.m_changed||this.FocusPlayers.m_changed||this.BeFocusPlayers.m_changed||this.CustomData.m_changed||this.ChaterOpenRequest.m_changed||this.Expeditions.m_changed||this.HandbookItems.m_changed||this.HeadItems.m_changed||this.Activitys.m_changed||this.SuitAwards.m_changed||this.Zans.m_changed||this.Foster.m_changed||this.FosterCats.m_changed||this.FosterCatOnFriends.m_changed||this.FosterFriendCats.m_changed||this.Chats.m_changed||this.Anouncement.m_changed||this.FirstDrawCards.m_changed||this.TalkForbid.m_changed||this.ServerRewards.m_changed||this.MailCommon.m_changed||this.Mails.m_changed||this.PayCommon.m_changed||this.Pays.m_changed||this.GuideData.m_changed||this.ActivityDatas.m_changed||this.SysMail.m_changed{
			update_string = "UPDATE Players SET "
			db_args:=new_db_args(64)
			if this.m_UniqueId_changed{
				update_string+="UniqueId=?,"
				db_args.Push(this.m_UniqueId)
			}
			if this.m_Account_changed{
				update_string+="Account=?,"
				db_args.Push(this.m_Account)
			}
			if this.m_Name_changed{
				update_string+="Name=?,"
				db_args.Push(this.m_Name)
			}
			if this.m_Token_changed{
				update_string+="Token=?,"
				db_args.Push(this.m_Token)
			}
			if this.m_Level_changed{
				update_string+="Level=?,"
				db_args.Push(this.m_Level)
			}
			if this.Info.m_changed{
				update_string+="Info=?,"
				dInfo,err:=this.Info.save()
				if err!=nil{
					log.Error("update save Info failed")
					return err,false,0,"",nil
				}
				db_args.Push(dInfo)
			}
			if this.Stages.m_changed{
				update_string+="Stages=?,"
				dStages,err:=this.Stages.save()
				if err!=nil{
					log.Error("insert save Stage failed")
					return err,false,0,"",nil
				}
				db_args.Push(dStages)
			}
			if this.ChapterUnLock.m_changed{
				update_string+="ChapterUnLock=?,"
				dChapterUnLock,err:=this.ChapterUnLock.save()
				if err!=nil{
					log.Error("update save ChapterUnLock failed")
					return err,false,0,"",nil
				}
				db_args.Push(dChapterUnLock)
			}
			if this.Items.m_changed{
				update_string+="Items=?,"
				dItems,err:=this.Items.save()
				if err!=nil{
					log.Error("insert save Item failed")
					return err,false,0,"",nil
				}
				db_args.Push(dItems)
			}
			if this.Areas.m_changed{
				update_string+="Areas=?,"
				dAreas,err:=this.Areas.save()
				if err!=nil{
					log.Error("insert save Area failed")
					return err,false,0,"",nil
				}
				db_args.Push(dAreas)
			}
			if this.Buildings.m_changed{
				update_string+="Buildings=?,"
				dBuildings,err:=this.Buildings.save()
				if err!=nil{
					log.Error("insert save Building failed")
					return err,false,0,"",nil
				}
				db_args.Push(dBuildings)
			}
			if this.BuildingDepots.m_changed{
				update_string+="BuildingDepots=?,"
				dBuildingDepots,err:=this.BuildingDepots.save()
				if err!=nil{
					log.Error("insert save BuildingDepot failed")
					return err,false,0,"",nil
				}
				db_args.Push(dBuildingDepots)
			}
			if this.DepotBuildingFormulas.m_changed{
				update_string+="DepotBuildingFormulas=?,"
				dDepotBuildingFormulas,err:=this.DepotBuildingFormulas.save()
				if err!=nil{
					log.Error("insert save DepotBuildingFormula failed")
					return err,false,0,"",nil
				}
				db_args.Push(dDepotBuildingFormulas)
			}
			if this.MakingFormulaBuildings.m_changed{
				update_string+="MakingFormulaBuildings=?,"
				dMakingFormulaBuildings,err:=this.MakingFormulaBuildings.save()
				if err!=nil{
					log.Error("insert save MakingFormulaBuilding failed")
					return err,false,0,"",nil
				}
				db_args.Push(dMakingFormulaBuildings)
			}
			if this.Crops.m_changed{
				update_string+="Crops=?,"
				dCrops,err:=this.Crops.save()
				if err!=nil{
					log.Error("insert save Crop failed")
					return err,false,0,"",nil
				}
				db_args.Push(dCrops)
			}
			if this.Cats.m_changed{
				update_string+="Cats=?,"
				dCats,err:=this.Cats.save()
				if err!=nil{
					log.Error("insert save Cat failed")
					return err,false,0,"",nil
				}
				db_args.Push(dCats)
			}
			if this.CatHouses.m_changed{
				update_string+="CatHouses=?,"
				dCatHouses,err:=this.CatHouses.save()
				if err!=nil{
					log.Error("insert save CatHouse failed")
					return err,false,0,"",nil
				}
				db_args.Push(dCatHouses)
			}
			if this.ShopItems.m_changed{
				update_string+="ShopItems=?,"
				dShopItems,err:=this.ShopItems.save()
				if err!=nil{
					log.Error("insert save ShopItem failed")
					return err,false,0,"",nil
				}
				db_args.Push(dShopItems)
			}
			if this.ShopLimitedInfos.m_changed{
				update_string+="ShopLimitedInfos=?,"
				dShopLimitedInfos,err:=this.ShopLimitedInfos.save()
				if err!=nil{
					log.Error("insert save ShopLimitedInfo failed")
					return err,false,0,"",nil
				}
				db_args.Push(dShopLimitedInfos)
			}
			if this.Chests.m_changed{
				update_string+="Chests=?,"
				dChests,err:=this.Chests.save()
				if err!=nil{
					log.Error("insert save Chest failed")
					return err,false,0,"",nil
				}
				db_args.Push(dChests)
			}
			if this.PayBacks.m_changed{
				update_string+="PayBacks=?,"
				dPayBacks,err:=this.PayBacks.save()
				if err!=nil{
					log.Error("insert save PayBack failed")
					return err,false,0,"",nil
				}
				db_args.Push(dPayBacks)
			}
			if this.Options.m_changed{
				update_string+="Options=?,"
				dOptions,err:=this.Options.save()
				if err!=nil{
					log.Error("update save Options failed")
					return err,false,0,"",nil
				}
				db_args.Push(dOptions)
			}
			if this.TaskCommon.m_changed{
				update_string+="TaskCommon=?,"
				dTaskCommon,err:=this.TaskCommon.save()
				if err!=nil{
					log.Error("update save TaskCommon failed")
					return err,false,0,"",nil
				}
				db_args.Push(dTaskCommon)
			}
			if this.Tasks.m_changed{
				update_string+="Tasks=?,"
				dTasks,err:=this.Tasks.save()
				if err!=nil{
					log.Error("insert save Task failed")
					return err,false,0,"",nil
				}
				db_args.Push(dTasks)
			}
			if this.FinishedTasks.m_changed{
				update_string+="FinishedTasks=?,"
				dFinishedTasks,err:=this.FinishedTasks.save()
				if err!=nil{
					log.Error("insert save FinishedTask failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFinishedTasks)
			}
			if this.DailyTaskAllDailys.m_changed{
				update_string+="DailyTaskAllDailys=?,"
				dDailyTaskAllDailys,err:=this.DailyTaskAllDailys.save()
				if err!=nil{
					log.Error("insert save DailyTaskAllDaily failed")
					return err,false,0,"",nil
				}
				db_args.Push(dDailyTaskAllDailys)
			}
			if this.SevenActivitys.m_changed{
				update_string+="SevenActivitys=?,"
				dSevenActivitys,err:=this.SevenActivitys.save()
				if err!=nil{
					log.Error("insert save SevenActivity failed")
					return err,false,0,"",nil
				}
				db_args.Push(dSevenActivitys)
			}
			if this.SignInfo.m_changed{
				update_string+="SignInfo=?,"
				dSignInfo,err:=this.SignInfo.save()
				if err!=nil{
					log.Error("update save SignInfo failed")
					return err,false,0,"",nil
				}
				db_args.Push(dSignInfo)
			}
			if this.Guidess.m_changed{
				update_string+="Guidess=?,"
				dGuidess,err:=this.Guidess.save()
				if err!=nil{
					log.Error("insert save Guides failed")
					return err,false,0,"",nil
				}
				db_args.Push(dGuidess)
			}
			if this.FriendRelative.m_changed{
				update_string+="FriendRelative=?,"
				dFriendRelative,err:=this.FriendRelative.save()
				if err!=nil{
					log.Error("update save FriendRelative failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFriendRelative)
			}
			if this.Friends.m_changed{
				update_string+="Friends=?,"
				dFriends,err:=this.Friends.save()
				if err!=nil{
					log.Error("insert save Friend failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFriends)
			}
			if this.FriendRecommends.m_changed{
				update_string+="FriendRecommends=?,"
				dFriendRecommends,err:=this.FriendRecommends.save()
				if err!=nil{
					log.Error("insert save FriendRecommend failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFriendRecommends)
			}
			if this.FriendAsks.m_changed{
				update_string+="FriendAsks=?,"
				dFriendAsks,err:=this.FriendAsks.save()
				if err!=nil{
					log.Error("insert save FriendAsk failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFriendAsks)
			}
			if this.FriendReqs.m_changed{
				update_string+="FriendReqs=?,"
				dFriendReqs,err:=this.FriendReqs.save()
				if err!=nil{
					log.Error("insert save FriendReq failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFriendReqs)
			}
			if this.FriendPoints.m_changed{
				update_string+="FriendPoints=?,"
				dFriendPoints,err:=this.FriendPoints.save()
				if err!=nil{
					log.Error("insert save FriendPoint failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFriendPoints)
			}
			if this.FriendChatUnreadIds.m_changed{
				update_string+="FriendChatUnreadIds=?,"
				dFriendChatUnreadIds,err:=this.FriendChatUnreadIds.save()
				if err!=nil{
					log.Error("insert save FriendChatUnreadId failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFriendChatUnreadIds)
			}
			if this.FriendChatUnreadMessages.m_changed{
				update_string+="FriendChatUnreadMessages=?,"
				dFriendChatUnreadMessages,err:=this.FriendChatUnreadMessages.save()
				if err!=nil{
					log.Error("insert save FriendChatUnreadMessage failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFriendChatUnreadMessages)
			}
			if this.FocusPlayers.m_changed{
				update_string+="FocusPlayers=?,"
				dFocusPlayers,err:=this.FocusPlayers.save()
				if err!=nil{
					log.Error("insert save FocusPlayer failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFocusPlayers)
			}
			if this.BeFocusPlayers.m_changed{
				update_string+="BeFocusPlayers=?,"
				dBeFocusPlayers,err:=this.BeFocusPlayers.save()
				if err!=nil{
					log.Error("insert save BeFocusPlayer failed")
					return err,false,0,"",nil
				}
				db_args.Push(dBeFocusPlayers)
			}
			if this.CustomData.m_changed{
				update_string+="CustomData=?,"
				dCustomData,err:=this.CustomData.save()
				if err!=nil{
					log.Error("update save CustomData failed")
					return err,false,0,"",nil
				}
				db_args.Push(dCustomData)
			}
			if this.ChaterOpenRequest.m_changed{
				update_string+="ChaterOpenRequest=?,"
				dChaterOpenRequest,err:=this.ChaterOpenRequest.save()
				if err!=nil{
					log.Error("update save ChaterOpenRequest failed")
					return err,false,0,"",nil
				}
				db_args.Push(dChaterOpenRequest)
			}
			if this.Expeditions.m_changed{
				update_string+="Expeditions=?,"
				dExpeditions,err:=this.Expeditions.save()
				if err!=nil{
					log.Error("insert save Expedition failed")
					return err,false,0,"",nil
				}
				db_args.Push(dExpeditions)
			}
			if this.HandbookItems.m_changed{
				update_string+="HandbookItems=?,"
				dHandbookItems,err:=this.HandbookItems.save()
				if err!=nil{
					log.Error("insert save HandbookItem failed")
					return err,false,0,"",nil
				}
				db_args.Push(dHandbookItems)
			}
			if this.HeadItems.m_changed{
				update_string+="HeadItems=?,"
				dHeadItems,err:=this.HeadItems.save()
				if err!=nil{
					log.Error("insert save HeadItem failed")
					return err,false,0,"",nil
				}
				db_args.Push(dHeadItems)
			}
			if this.Activitys.m_changed{
				update_string+="Activitys=?,"
				dActivitys,err:=this.Activitys.save()
				if err!=nil{
					log.Error("insert save Activity failed")
					return err,false,0,"",nil
				}
				db_args.Push(dActivitys)
			}
			if this.SuitAwards.m_changed{
				update_string+="SuitAwards=?,"
				dSuitAwards,err:=this.SuitAwards.save()
				if err!=nil{
					log.Error("insert save SuitAward failed")
					return err,false,0,"",nil
				}
				db_args.Push(dSuitAwards)
			}
			if this.Zans.m_changed{
				update_string+="Zans=?,"
				dZans,err:=this.Zans.save()
				if err!=nil{
					log.Error("insert save Zan failed")
					return err,false,0,"",nil
				}
				db_args.Push(dZans)
			}
			if this.Foster.m_changed{
				update_string+="Foster=?,"
				dFoster,err:=this.Foster.save()
				if err!=nil{
					log.Error("update save Foster failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFoster)
			}
			if this.FosterCats.m_changed{
				update_string+="FosterCats=?,"
				dFosterCats,err:=this.FosterCats.save()
				if err!=nil{
					log.Error("insert save FosterCat failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFosterCats)
			}
			if this.FosterCatOnFriends.m_changed{
				update_string+="FosterCatOnFriends=?,"
				dFosterCatOnFriends,err:=this.FosterCatOnFriends.save()
				if err!=nil{
					log.Error("insert save FosterCatOnFriend failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFosterCatOnFriends)
			}
			if this.FosterFriendCats.m_changed{
				update_string+="FosterFriendCats=?,"
				dFosterFriendCats,err:=this.FosterFriendCats.save()
				if err!=nil{
					log.Error("insert save FosterFriendCat failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFosterFriendCats)
			}
			if this.Chats.m_changed{
				update_string+="Chats=?,"
				dChats,err:=this.Chats.save()
				if err!=nil{
					log.Error("insert save Chat failed")
					return err,false,0,"",nil
				}
				db_args.Push(dChats)
			}
			if this.Anouncement.m_changed{
				update_string+="Anouncement=?,"
				dAnouncement,err:=this.Anouncement.save()
				if err!=nil{
					log.Error("update save Anouncement failed")
					return err,false,0,"",nil
				}
				db_args.Push(dAnouncement)
			}
			if this.FirstDrawCards.m_changed{
				update_string+="FirstDrawCards=?,"
				dFirstDrawCards,err:=this.FirstDrawCards.save()
				if err!=nil{
					log.Error("insert save FirstDrawCard failed")
					return err,false,0,"",nil
				}
				db_args.Push(dFirstDrawCards)
			}
			if this.TalkForbid.m_changed{
				update_string+="TalkForbid=?,"
				dTalkForbid,err:=this.TalkForbid.save()
				if err!=nil{
					log.Error("update save TalkForbid failed")
					return err,false,0,"",nil
				}
				db_args.Push(dTalkForbid)
			}
			if this.ServerRewards.m_changed{
				update_string+="ServerRewards=?,"
				dServerRewards,err:=this.ServerRewards.save()
				if err!=nil{
					log.Error("insert save ServerReward failed")
					return err,false,0,"",nil
				}
				db_args.Push(dServerRewards)
			}
			if this.MailCommon.m_changed{
				update_string+="MailCommon=?,"
				dMailCommon,err:=this.MailCommon.save()
				if err!=nil{
					log.Error("update save MailCommon failed")
					return err,false,0,"",nil
				}
				db_args.Push(dMailCommon)
			}
			if this.Mails.m_changed{
				update_string+="Mails=?,"
				dMails,err:=this.Mails.save()
				if err!=nil{
					log.Error("insert save Mail failed")
					return err,false,0,"",nil
				}
				db_args.Push(dMails)
			}
			if this.PayCommon.m_changed{
				update_string+="PayCommon=?,"
				dPayCommon,err:=this.PayCommon.save()
				if err!=nil{
					log.Error("update save PayCommon failed")
					return err,false,0,"",nil
				}
				db_args.Push(dPayCommon)
			}
			if this.Pays.m_changed{
				update_string+="Pays=?,"
				dPays,err:=this.Pays.save()
				if err!=nil{
					log.Error("insert save Pay failed")
					return err,false,0,"",nil
				}
				db_args.Push(dPays)
			}
			if this.GuideData.m_changed{
				update_string+="GuideData=?,"
				dGuideData,err:=this.GuideData.save()
				if err!=nil{
					log.Error("update save GuideData failed")
					return err,false,0,"",nil
				}
				db_args.Push(dGuideData)
			}
			if this.ActivityDatas.m_changed{
				update_string+="ActivityDatas=?,"
				dActivityDatas,err:=this.ActivityDatas.save()
				if err!=nil{
					log.Error("insert save ActivityData failed")
					return err,false,0,"",nil
				}
				db_args.Push(dActivityDatas)
			}
			if this.SysMail.m_changed{
				update_string+="SysMail=?,"
				dSysMail,err:=this.SysMail.save()
				if err!=nil{
					log.Error("update save SysMail failed")
					return err,false,0,"",nil
				}
				db_args.Push(dSysMail)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE PlayerId=?"
			db_args.Push(this.m_PlayerId)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_UniqueId_changed = false
	this.m_Account_changed = false
	this.m_Name_changed = false
	this.m_Token_changed = false
	this.m_Level_changed = false
	this.Info.m_changed = false
	this.Stages.m_changed = false
	this.ChapterUnLock.m_changed = false
	this.Items.m_changed = false
	this.Areas.m_changed = false
	this.Buildings.m_changed = false
	this.BuildingDepots.m_changed = false
	this.DepotBuildingFormulas.m_changed = false
	this.MakingFormulaBuildings.m_changed = false
	this.Crops.m_changed = false
	this.Cats.m_changed = false
	this.CatHouses.m_changed = false
	this.ShopItems.m_changed = false
	this.ShopLimitedInfos.m_changed = false
	this.Chests.m_changed = false
	this.PayBacks.m_changed = false
	this.Options.m_changed = false
	this.TaskCommon.m_changed = false
	this.Tasks.m_changed = false
	this.FinishedTasks.m_changed = false
	this.DailyTaskAllDailys.m_changed = false
	this.SevenActivitys.m_changed = false
	this.SignInfo.m_changed = false
	this.Guidess.m_changed = false
	this.FriendRelative.m_changed = false
	this.Friends.m_changed = false
	this.FriendRecommends.m_changed = false
	this.FriendAsks.m_changed = false
	this.FriendReqs.m_changed = false
	this.FriendPoints.m_changed = false
	this.FriendChatUnreadIds.m_changed = false
	this.FriendChatUnreadMessages.m_changed = false
	this.FocusPlayers.m_changed = false
	this.BeFocusPlayers.m_changed = false
	this.CustomData.m_changed = false
	this.ChaterOpenRequest.m_changed = false
	this.Expeditions.m_changed = false
	this.HandbookItems.m_changed = false
	this.HeadItems.m_changed = false
	this.Activitys.m_changed = false
	this.SuitAwards.m_changed = false
	this.Zans.m_changed = false
	this.Foster.m_changed = false
	this.FosterCats.m_changed = false
	this.FosterCatOnFriends.m_changed = false
	this.FosterFriendCats.m_changed = false
	this.Chats.m_changed = false
	this.Anouncement.m_changed = false
	this.FirstDrawCards.m_changed = false
	this.TalkForbid.m_changed = false
	this.ServerRewards.m_changed = false
	this.MailCommon.m_changed = false
	this.Mails.m_changed = false
	this.PayCommon.m_changed = false
	this.Pays.m_changed = false
	this.GuideData.m_changed = false
	this.ActivityDatas.m_changed = false
	this.SysMail.m_changed = false
	if release && this.m_loaded {
		atomic.AddInt32(&this.m_table.m_gc_n, -1)
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbPlayerRow) Save(release bool) (err error, d bool, released bool) {
	err,released, state, update_string, args := this.save_data(release)
	if err != nil {
		log.Error("save data failed")
		return err, false, false
	}
	if state == 0 {
		d = false
	} else if state == 1 {
		_, err = this.m_table.m_dbc.StmtExec(this.m_table.m_save_insert_stmt, args...)
		if err != nil {
			log.Error("INSERT Players exec failed %v ", this.m_PlayerId)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE Players exec failed %v", this.m_PlayerId)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbPlayerRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbPlayerRowSort struct {
	rows []*dbPlayerRow
}
func (this *dbPlayerRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbPlayerRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbPlayerRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbPlayerTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[int32]*dbPlayerRow
	m_new_rows map[int32]*dbPlayerRow
	m_removed_rows map[int32]*dbPlayerRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
}
func new_dbPlayerTable(dbc *DBC) (this *dbPlayerTable) {
	this = &dbPlayerTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[int32]*dbPlayerRow)
	this.m_new_rows = make(map[int32]*dbPlayerRow)
	this.m_removed_rows = make(map[int32]*dbPlayerRow)
	return this
}
func (this *dbPlayerTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS Players(PlayerId int(11),PRIMARY KEY (PlayerId))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS Players failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='Players'", this.m_dbc.m_db_name)
	if err != nil {
		log.Error("SELECT information_schema failed")
		return
	}
	columns := make(map[string]int32)
	for rows.Next() {
		var column_name string
		var ordinal_position int32
		err = rows.Scan(&column_name, &ordinal_position)
		if err != nil {
			log.Error("scan information_schema row failed")
			return
		}
		if ordinal_position < 1 {
			log.Error("col ordinal out of range")
			continue
		}
		columns[column_name] = ordinal_position
	}
	_, hasUniqueId := columns["UniqueId"]
	if !hasUniqueId {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN UniqueId varchar(45) DEFAULT ''")
		if err != nil {
			log.Error("ADD COLUMN UniqueId failed")
			return
		}
	}
	_, hasAccount := columns["Account"]
	if !hasAccount {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Account varchar(45)")
		if err != nil {
			log.Error("ADD COLUMN Account failed")
			return
		}
	}
	_, hasName := columns["Name"]
	if !hasName {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Name varchar(45)")
		if err != nil {
			log.Error("ADD COLUMN Name failed")
			return
		}
	}
	_, hasToken := columns["Token"]
	if !hasToken {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Token varchar(45) DEFAULT ''")
		if err != nil {
			log.Error("ADD COLUMN Token failed")
			return
		}
	}
	_, hasLevel := columns["Level"]
	if !hasLevel {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Level int(11) DEFAULT 0")
		if err != nil {
			log.Error("ADD COLUMN Level failed")
			return
		}
	}
	_, hasInfo := columns["Info"]
	if !hasInfo {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Info LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Info failed")
			return
		}
	}
	_, hasStage := columns["Stages"]
	if !hasStage {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Stages LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Stages failed")
			return
		}
	}
	_, hasChapterUnLock := columns["ChapterUnLock"]
	if !hasChapterUnLock {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN ChapterUnLock LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN ChapterUnLock failed")
			return
		}
	}
	_, hasItem := columns["Items"]
	if !hasItem {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Items LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Items failed")
			return
		}
	}
	_, hasArea := columns["Areas"]
	if !hasArea {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Areas LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Areas failed")
			return
		}
	}
	_, hasBuilding := columns["Buildings"]
	if !hasBuilding {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Buildings LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Buildings failed")
			return
		}
	}
	_, hasBuildingDepot := columns["BuildingDepots"]
	if !hasBuildingDepot {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN BuildingDepots LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN BuildingDepots failed")
			return
		}
	}
	_, hasDepotBuildingFormula := columns["DepotBuildingFormulas"]
	if !hasDepotBuildingFormula {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN DepotBuildingFormulas LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN DepotBuildingFormulas failed")
			return
		}
	}
	_, hasMakingFormulaBuilding := columns["MakingFormulaBuildings"]
	if !hasMakingFormulaBuilding {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN MakingFormulaBuildings LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN MakingFormulaBuildings failed")
			return
		}
	}
	_, hasCrop := columns["Crops"]
	if !hasCrop {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Crops LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Crops failed")
			return
		}
	}
	_, hasCat := columns["Cats"]
	if !hasCat {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Cats LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Cats failed")
			return
		}
	}
	_, hasCatHouse := columns["CatHouses"]
	if !hasCatHouse {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN CatHouses LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN CatHouses failed")
			return
		}
	}
	_, hasShopItem := columns["ShopItems"]
	if !hasShopItem {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN ShopItems LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN ShopItems failed")
			return
		}
	}
	_, hasShopLimitedInfo := columns["ShopLimitedInfos"]
	if !hasShopLimitedInfo {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN ShopLimitedInfos LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN ShopLimitedInfos failed")
			return
		}
	}
	_, hasChest := columns["Chests"]
	if !hasChest {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Chests LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Chests failed")
			return
		}
	}
	_, hasPayBack := columns["PayBacks"]
	if !hasPayBack {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN PayBacks LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN PayBacks failed")
			return
		}
	}
	_, hasOptions := columns["Options"]
	if !hasOptions {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Options LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Options failed")
			return
		}
	}
	_, hasTaskCommon := columns["TaskCommon"]
	if !hasTaskCommon {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN TaskCommon LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN TaskCommon failed")
			return
		}
	}
	_, hasTask := columns["Tasks"]
	if !hasTask {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Tasks LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Tasks failed")
			return
		}
	}
	_, hasFinishedTask := columns["FinishedTasks"]
	if !hasFinishedTask {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FinishedTasks LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FinishedTasks failed")
			return
		}
	}
	_, hasDailyTaskAllDaily := columns["DailyTaskAllDailys"]
	if !hasDailyTaskAllDaily {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN DailyTaskAllDailys LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN DailyTaskAllDailys failed")
			return
		}
	}
	_, hasSevenActivity := columns["SevenActivitys"]
	if !hasSevenActivity {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN SevenActivitys LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN SevenActivitys failed")
			return
		}
	}
	_, hasSignInfo := columns["SignInfo"]
	if !hasSignInfo {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN SignInfo LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN SignInfo failed")
			return
		}
	}
	_, hasGuides := columns["Guidess"]
	if !hasGuides {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Guidess LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Guidess failed")
			return
		}
	}
	_, hasFriendRelative := columns["FriendRelative"]
	if !hasFriendRelative {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FriendRelative LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FriendRelative failed")
			return
		}
	}
	_, hasFriend := columns["Friends"]
	if !hasFriend {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Friends LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Friends failed")
			return
		}
	}
	_, hasFriendRecommend := columns["FriendRecommends"]
	if !hasFriendRecommend {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FriendRecommends LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FriendRecommends failed")
			return
		}
	}
	_, hasFriendAsk := columns["FriendAsks"]
	if !hasFriendAsk {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FriendAsks LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FriendAsks failed")
			return
		}
	}
	_, hasFriendReq := columns["FriendReqs"]
	if !hasFriendReq {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FriendReqs LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FriendReqs failed")
			return
		}
	}
	_, hasFriendPoint := columns["FriendPoints"]
	if !hasFriendPoint {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FriendPoints LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FriendPoints failed")
			return
		}
	}
	_, hasFriendChatUnreadId := columns["FriendChatUnreadIds"]
	if !hasFriendChatUnreadId {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FriendChatUnreadIds LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FriendChatUnreadIds failed")
			return
		}
	}
	_, hasFriendChatUnreadMessage := columns["FriendChatUnreadMessages"]
	if !hasFriendChatUnreadMessage {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FriendChatUnreadMessages LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FriendChatUnreadMessages failed")
			return
		}
	}
	_, hasFocusPlayer := columns["FocusPlayers"]
	if !hasFocusPlayer {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FocusPlayers LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FocusPlayers failed")
			return
		}
	}
	_, hasBeFocusPlayer := columns["BeFocusPlayers"]
	if !hasBeFocusPlayer {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN BeFocusPlayers LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN BeFocusPlayers failed")
			return
		}
	}
	_, hasCustomData := columns["CustomData"]
	if !hasCustomData {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN CustomData LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN CustomData failed")
			return
		}
	}
	_, hasChaterOpenRequest := columns["ChaterOpenRequest"]
	if !hasChaterOpenRequest {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN ChaterOpenRequest LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN ChaterOpenRequest failed")
			return
		}
	}
	_, hasExpedition := columns["Expeditions"]
	if !hasExpedition {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Expeditions LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Expeditions failed")
			return
		}
	}
	_, hasHandbookItem := columns["HandbookItems"]
	if !hasHandbookItem {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN HandbookItems LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN HandbookItems failed")
			return
		}
	}
	_, hasHeadItem := columns["HeadItems"]
	if !hasHeadItem {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN HeadItems LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN HeadItems failed")
			return
		}
	}
	_, hasActivity := columns["Activitys"]
	if !hasActivity {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Activitys LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Activitys failed")
			return
		}
	}
	_, hasSuitAward := columns["SuitAwards"]
	if !hasSuitAward {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN SuitAwards LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN SuitAwards failed")
			return
		}
	}
	_, hasZan := columns["Zans"]
	if !hasZan {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Zans LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Zans failed")
			return
		}
	}
	_, hasFoster := columns["Foster"]
	if !hasFoster {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Foster LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Foster failed")
			return
		}
	}
	_, hasFosterCat := columns["FosterCats"]
	if !hasFosterCat {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FosterCats LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FosterCats failed")
			return
		}
	}
	_, hasFosterCatOnFriend := columns["FosterCatOnFriends"]
	if !hasFosterCatOnFriend {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FosterCatOnFriends LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FosterCatOnFriends failed")
			return
		}
	}
	_, hasFosterFriendCat := columns["FosterFriendCats"]
	if !hasFosterFriendCat {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FosterFriendCats LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FosterFriendCats failed")
			return
		}
	}
	_, hasChat := columns["Chats"]
	if !hasChat {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Chats LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Chats failed")
			return
		}
	}
	_, hasAnouncement := columns["Anouncement"]
	if !hasAnouncement {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Anouncement LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Anouncement failed")
			return
		}
	}
	_, hasFirstDrawCard := columns["FirstDrawCards"]
	if !hasFirstDrawCard {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN FirstDrawCards LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN FirstDrawCards failed")
			return
		}
	}
	_, hasTalkForbid := columns["TalkForbid"]
	if !hasTalkForbid {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN TalkForbid LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN TalkForbid failed")
			return
		}
	}
	_, hasServerReward := columns["ServerRewards"]
	if !hasServerReward {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN ServerRewards LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN ServerRewards failed")
			return
		}
	}
	_, hasMailCommon := columns["MailCommon"]
	if !hasMailCommon {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN MailCommon LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN MailCommon failed")
			return
		}
	}
	_, hasMail := columns["Mails"]
	if !hasMail {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Mails LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Mails failed")
			return
		}
	}
	_, hasPayCommon := columns["PayCommon"]
	if !hasPayCommon {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN PayCommon LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN PayCommon failed")
			return
		}
	}
	_, hasPay := columns["Pays"]
	if !hasPay {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN Pays LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Pays failed")
			return
		}
	}
	_, hasGuideData := columns["GuideData"]
	if !hasGuideData {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN GuideData LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN GuideData failed")
			return
		}
	}
	_, hasActivityData := columns["ActivityDatas"]
	if !hasActivityData {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN ActivityDatas LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN ActivityDatas failed")
			return
		}
	}
	_, hasSysMail := columns["SysMail"]
	if !hasSysMail {
		_, err = this.m_dbc.Exec("ALTER TABLE Players ADD COLUMN SysMail LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN SysMail failed")
			return
		}
	}
	return
}
func (this *dbPlayerTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT PlayerId,UniqueId,Account,Name,Token,Level,Info,Stages,ChapterUnLock,Items,Areas,Buildings,BuildingDepots,DepotBuildingFormulas,MakingFormulaBuildings,Crops,Cats,CatHouses,ShopItems,ShopLimitedInfos,Chests,PayBacks,Options,TaskCommon,Tasks,FinishedTasks,DailyTaskAllDailys,SevenActivitys,SignInfo,Guidess,FriendRelative,Friends,FriendRecommends,FriendAsks,FriendReqs,FriendPoints,FriendChatUnreadIds,FriendChatUnreadMessages,FocusPlayers,BeFocusPlayers,CustomData,ChaterOpenRequest,Expeditions,HandbookItems,HeadItems,Activitys,SuitAwards,Zans,Foster,FosterCats,FosterCatOnFriends,FosterFriendCats,Chats,Anouncement,FirstDrawCards,TalkForbid,ServerRewards,MailCommon,Mails,PayCommon,Pays,GuideData,ActivityDatas,SysMail FROM Players")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO Players (PlayerId,UniqueId,Account,Name,Token,Level,Info,Stages,ChapterUnLock,Items,Areas,Buildings,BuildingDepots,DepotBuildingFormulas,MakingFormulaBuildings,Crops,Cats,CatHouses,ShopItems,ShopLimitedInfos,Chests,PayBacks,Options,TaskCommon,Tasks,FinishedTasks,DailyTaskAllDailys,SevenActivitys,SignInfo,Guidess,FriendRelative,Friends,FriendRecommends,FriendAsks,FriendReqs,FriendPoints,FriendChatUnreadIds,FriendChatUnreadMessages,FocusPlayers,BeFocusPlayers,CustomData,ChaterOpenRequest,Expeditions,HandbookItems,HeadItems,Activitys,SuitAwards,Zans,Foster,FosterCats,FosterCatOnFriends,FosterFriendCats,Chats,Anouncement,FirstDrawCards,TalkForbid,ServerRewards,MailCommon,Mails,PayCommon,Pays,GuideData,ActivityDatas,SysMail) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM Players WHERE PlayerId=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerTable) Init() (err error) {
	err=this.check_create_table()
	if err!=nil{
		log.Error("check_create_table failed")
		return
	}
	err=this.prepare_preload_select_stmt()
	if err!=nil{
		log.Error("prepare_preload_select_stmt failed")
		return
	}
	err=this.prepare_save_insert_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	err=this.prepare_delete_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	return
}
func (this *dbPlayerTable) Preload() (err error) {
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var PlayerId int32
	var dUniqueId string
	var dAccount string
	var dName string
	var dToken string
	var dLevel int32
	var dInfo []byte
	var dStages []byte
	var dChapterUnLock []byte
	var dItems []byte
	var dAreas []byte
	var dBuildings []byte
	var dBuildingDepots []byte
	var dDepotBuildingFormulas []byte
	var dMakingFormulaBuildings []byte
	var dCrops []byte
	var dCats []byte
	var dCatHouses []byte
	var dShopItems []byte
	var dShopLimitedInfos []byte
	var dChests []byte
	var dPayBacks []byte
	var dOptions []byte
	var dTaskCommon []byte
	var dTasks []byte
	var dFinishedTasks []byte
	var dDailyTaskAllDailys []byte
	var dSevenActivitys []byte
	var dSignInfo []byte
	var dGuidess []byte
	var dFriendRelative []byte
	var dFriends []byte
	var dFriendRecommends []byte
	var dFriendAsks []byte
	var dFriendReqs []byte
	var dFriendPoints []byte
	var dFriendChatUnreadIds []byte
	var dFriendChatUnreadMessages []byte
	var dFocusPlayers []byte
	var dBeFocusPlayers []byte
	var dCustomData []byte
	var dChaterOpenRequest []byte
	var dExpeditions []byte
	var dHandbookItems []byte
	var dHeadItems []byte
	var dActivitys []byte
	var dSuitAwards []byte
	var dZans []byte
	var dFoster []byte
	var dFosterCats []byte
	var dFosterCatOnFriends []byte
	var dFosterFriendCats []byte
	var dChats []byte
	var dAnouncement []byte
	var dFirstDrawCards []byte
	var dTalkForbid []byte
	var dServerRewards []byte
	var dMailCommon []byte
	var dMails []byte
	var dPayCommon []byte
	var dPays []byte
	var dGuideData []byte
	var dActivityDatas []byte
	var dSysMail []byte
		this.m_preload_max_id = 0
	for r.Next() {
		err = r.Scan(&PlayerId,&dUniqueId,&dAccount,&dName,&dToken,&dLevel,&dInfo,&dStages,&dChapterUnLock,&dItems,&dAreas,&dBuildings,&dBuildingDepots,&dDepotBuildingFormulas,&dMakingFormulaBuildings,&dCrops,&dCats,&dCatHouses,&dShopItems,&dShopLimitedInfos,&dChests,&dPayBacks,&dOptions,&dTaskCommon,&dTasks,&dFinishedTasks,&dDailyTaskAllDailys,&dSevenActivitys,&dSignInfo,&dGuidess,&dFriendRelative,&dFriends,&dFriendRecommends,&dFriendAsks,&dFriendReqs,&dFriendPoints,&dFriendChatUnreadIds,&dFriendChatUnreadMessages,&dFocusPlayers,&dBeFocusPlayers,&dCustomData,&dChaterOpenRequest,&dExpeditions,&dHandbookItems,&dHeadItems,&dActivitys,&dSuitAwards,&dZans,&dFoster,&dFosterCats,&dFosterCatOnFriends,&dFosterFriendCats,&dChats,&dAnouncement,&dFirstDrawCards,&dTalkForbid,&dServerRewards,&dMailCommon,&dMails,&dPayCommon,&dPays,&dGuideData,&dActivityDatas,&dSysMail)
		if err != nil {
			log.Error("Scan err[%v]", err.Error())
			return
		}
		if PlayerId>this.m_preload_max_id{
			this.m_preload_max_id =PlayerId
		}
		row := new_dbPlayerRow(this,PlayerId)
		row.m_UniqueId=dUniqueId
		row.m_Account=dAccount
		row.m_Name=dName
		row.m_Token=dToken
		row.m_Level=dLevel
		err = row.Info.load(dInfo)
		if err != nil {
			log.Error("Info %v", PlayerId)
			return
		}
		err = row.Stages.load(dStages)
		if err != nil {
			log.Error("Stages %v", PlayerId)
			return
		}
		err = row.ChapterUnLock.load(dChapterUnLock)
		if err != nil {
			log.Error("ChapterUnLock %v", PlayerId)
			return
		}
		err = row.Items.load(dItems)
		if err != nil {
			log.Error("Items %v", PlayerId)
			return
		}
		err = row.Areas.load(dAreas)
		if err != nil {
			log.Error("Areas %v", PlayerId)
			return
		}
		err = row.Buildings.load(dBuildings)
		if err != nil {
			log.Error("Buildings %v", PlayerId)
			return
		}
		err = row.BuildingDepots.load(dBuildingDepots)
		if err != nil {
			log.Error("BuildingDepots %v", PlayerId)
			return
		}
		err = row.DepotBuildingFormulas.load(dDepotBuildingFormulas)
		if err != nil {
			log.Error("DepotBuildingFormulas %v", PlayerId)
			return
		}
		err = row.MakingFormulaBuildings.load(dMakingFormulaBuildings)
		if err != nil {
			log.Error("MakingFormulaBuildings %v", PlayerId)
			return
		}
		err = row.Crops.load(dCrops)
		if err != nil {
			log.Error("Crops %v", PlayerId)
			return
		}
		err = row.Cats.load(dCats)
		if err != nil {
			log.Error("Cats %v", PlayerId)
			return
		}
		err = row.CatHouses.load(dCatHouses)
		if err != nil {
			log.Error("CatHouses %v", PlayerId)
			return
		}
		err = row.ShopItems.load(dShopItems)
		if err != nil {
			log.Error("ShopItems %v", PlayerId)
			return
		}
		err = row.ShopLimitedInfos.load(dShopLimitedInfos)
		if err != nil {
			log.Error("ShopLimitedInfos %v", PlayerId)
			return
		}
		err = row.Chests.load(dChests)
		if err != nil {
			log.Error("Chests %v", PlayerId)
			return
		}
		err = row.PayBacks.load(dPayBacks)
		if err != nil {
			log.Error("PayBacks %v", PlayerId)
			return
		}
		err = row.Options.load(dOptions)
		if err != nil {
			log.Error("Options %v", PlayerId)
			return
		}
		err = row.TaskCommon.load(dTaskCommon)
		if err != nil {
			log.Error("TaskCommon %v", PlayerId)
			return
		}
		err = row.Tasks.load(dTasks)
		if err != nil {
			log.Error("Tasks %v", PlayerId)
			return
		}
		err = row.FinishedTasks.load(dFinishedTasks)
		if err != nil {
			log.Error("FinishedTasks %v", PlayerId)
			return
		}
		err = row.DailyTaskAllDailys.load(dDailyTaskAllDailys)
		if err != nil {
			log.Error("DailyTaskAllDailys %v", PlayerId)
			return
		}
		err = row.SevenActivitys.load(dSevenActivitys)
		if err != nil {
			log.Error("SevenActivitys %v", PlayerId)
			return
		}
		err = row.SignInfo.load(dSignInfo)
		if err != nil {
			log.Error("SignInfo %v", PlayerId)
			return
		}
		err = row.Guidess.load(dGuidess)
		if err != nil {
			log.Error("Guidess %v", PlayerId)
			return
		}
		err = row.FriendRelative.load(dFriendRelative)
		if err != nil {
			log.Error("FriendRelative %v", PlayerId)
			return
		}
		err = row.Friends.load(dFriends)
		if err != nil {
			log.Error("Friends %v", PlayerId)
			return
		}
		err = row.FriendRecommends.load(dFriendRecommends)
		if err != nil {
			log.Error("FriendRecommends %v", PlayerId)
			return
		}
		err = row.FriendAsks.load(dFriendAsks)
		if err != nil {
			log.Error("FriendAsks %v", PlayerId)
			return
		}
		err = row.FriendReqs.load(dFriendReqs)
		if err != nil {
			log.Error("FriendReqs %v", PlayerId)
			return
		}
		err = row.FriendPoints.load(dFriendPoints)
		if err != nil {
			log.Error("FriendPoints %v", PlayerId)
			return
		}
		err = row.FriendChatUnreadIds.load(dFriendChatUnreadIds)
		if err != nil {
			log.Error("FriendChatUnreadIds %v", PlayerId)
			return
		}
		err = row.FriendChatUnreadMessages.load(dFriendChatUnreadMessages)
		if err != nil {
			log.Error("FriendChatUnreadMessages %v", PlayerId)
			return
		}
		err = row.FocusPlayers.load(dFocusPlayers)
		if err != nil {
			log.Error("FocusPlayers %v", PlayerId)
			return
		}
		err = row.BeFocusPlayers.load(dBeFocusPlayers)
		if err != nil {
			log.Error("BeFocusPlayers %v", PlayerId)
			return
		}
		err = row.CustomData.load(dCustomData)
		if err != nil {
			log.Error("CustomData %v", PlayerId)
			return
		}
		err = row.ChaterOpenRequest.load(dChaterOpenRequest)
		if err != nil {
			log.Error("ChaterOpenRequest %v", PlayerId)
			return
		}
		err = row.Expeditions.load(dExpeditions)
		if err != nil {
			log.Error("Expeditions %v", PlayerId)
			return
		}
		err = row.HandbookItems.load(dHandbookItems)
		if err != nil {
			log.Error("HandbookItems %v", PlayerId)
			return
		}
		err = row.HeadItems.load(dHeadItems)
		if err != nil {
			log.Error("HeadItems %v", PlayerId)
			return
		}
		err = row.Activitys.load(dActivitys)
		if err != nil {
			log.Error("Activitys %v", PlayerId)
			return
		}
		err = row.SuitAwards.load(dSuitAwards)
		if err != nil {
			log.Error("SuitAwards %v", PlayerId)
			return
		}
		err = row.Zans.load(dZans)
		if err != nil {
			log.Error("Zans %v", PlayerId)
			return
		}
		err = row.Foster.load(dFoster)
		if err != nil {
			log.Error("Foster %v", PlayerId)
			return
		}
		err = row.FosterCats.load(dFosterCats)
		if err != nil {
			log.Error("FosterCats %v", PlayerId)
			return
		}
		err = row.FosterCatOnFriends.load(dFosterCatOnFriends)
		if err != nil {
			log.Error("FosterCatOnFriends %v", PlayerId)
			return
		}
		err = row.FosterFriendCats.load(dFosterFriendCats)
		if err != nil {
			log.Error("FosterFriendCats %v", PlayerId)
			return
		}
		err = row.Chats.load(dChats)
		if err != nil {
			log.Error("Chats %v", PlayerId)
			return
		}
		err = row.Anouncement.load(dAnouncement)
		if err != nil {
			log.Error("Anouncement %v", PlayerId)
			return
		}
		err = row.FirstDrawCards.load(dFirstDrawCards)
		if err != nil {
			log.Error("FirstDrawCards %v", PlayerId)
			return
		}
		err = row.TalkForbid.load(dTalkForbid)
		if err != nil {
			log.Error("TalkForbid %v", PlayerId)
			return
		}
		err = row.ServerRewards.load(dServerRewards)
		if err != nil {
			log.Error("ServerRewards %v", PlayerId)
			return
		}
		err = row.MailCommon.load(dMailCommon)
		if err != nil {
			log.Error("MailCommon %v", PlayerId)
			return
		}
		err = row.Mails.load(dMails)
		if err != nil {
			log.Error("Mails %v", PlayerId)
			return
		}
		err = row.PayCommon.load(dPayCommon)
		if err != nil {
			log.Error("PayCommon %v", PlayerId)
			return
		}
		err = row.Pays.load(dPays)
		if err != nil {
			log.Error("Pays %v", PlayerId)
			return
		}
		err = row.GuideData.load(dGuideData)
		if err != nil {
			log.Error("GuideData %v", PlayerId)
			return
		}
		err = row.ActivityDatas.load(dActivityDatas)
		if err != nil {
			log.Error("ActivityDatas %v", PlayerId)
			return
		}
		err = row.SysMail.load(dSysMail)
		if err != nil {
			log.Error("SysMail %v", PlayerId)
			return
		}
		row.m_UniqueId_changed=false
		row.m_Account_changed=false
		row.m_Name_changed=false
		row.m_Token_changed=false
		row.m_Level_changed=false
		row.m_valid = true
		this.m_rows[PlayerId]=row
	}
	return
}
func (this *dbPlayerTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbPlayerTable) fetch_rows(rows map[int32]*dbPlayerRow) (r map[int32]*dbPlayerRow) {
	this.m_lock.UnSafeLock("dbPlayerTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[int32]*dbPlayerRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbPlayerTable) fetch_new_rows() (new_rows map[int32]*dbPlayerRow) {
	this.m_lock.UnSafeLock("dbPlayerTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[int32]*dbPlayerRow)
	for i, v := range this.m_new_rows {
		_, has := this.m_rows[i]
		if has {
			log.Error("rows already has new rows %v", i)
			continue
		}
		this.m_rows[i] = v
		new_rows[i] = v
	}
	for i, _ := range new_rows {
		delete(this.m_new_rows, i)
	}
	return
}
func (this *dbPlayerTable) save_rows(rows map[int32]*dbPlayerRow, quick bool) {
	for _, v := range rows {
		if this.m_dbc.m_quit && !quick {
			return
		}
		err, delay, _ := v.Save(false)
		if err != nil {
			log.Error("save failed %v", err)
		}
		if this.m_dbc.m_quit && !quick {
			return
		}
		if delay&&!quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
}
func (this *dbPlayerTable) Save(quick bool) (err error){
	removed_rows := this.fetch_rows(this.m_removed_rows)
	for _, v := range removed_rows {
		_, err := this.m_dbc.StmtExec(this.m_delete_stmt, v.GetPlayerId())
		if err != nil {
			log.Error("exec delete stmt failed %v", err)
		}
		v.m_valid = false
		if !quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
	this.m_removed_rows = make(map[int32]*dbPlayerRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbPlayerTable) AddRow(PlayerId int32) (row *dbPlayerRow) {
	this.m_lock.UnSafeLock("dbPlayerTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	row = new_dbPlayerRow(this,PlayerId)
	row.m_new = true
	row.m_loaded = true
	row.m_valid = true
	_, has := this.m_new_rows[PlayerId]
	if has{
		log.Error("已经存在 %v", PlayerId)
		return nil
	}
	this.m_new_rows[PlayerId] = row
	atomic.AddInt32(&this.m_gc_n,1)
	return row
}
func (this *dbPlayerTable) RemoveRow(PlayerId int32) {
	this.m_lock.UnSafeLock("dbPlayerTable.RemoveRow")
	defer this.m_lock.UnSafeUnlock()
	row := this.m_rows[PlayerId]
	if row != nil {
		row.m_remove = true
		delete(this.m_rows, PlayerId)
		rm_row := this.m_removed_rows[PlayerId]
		if rm_row != nil {
			log.Error("rows and removed rows both has %v", PlayerId)
		}
		this.m_removed_rows[PlayerId] = row
		_, has_new := this.m_new_rows[PlayerId]
		if has_new {
			delete(this.m_new_rows, PlayerId)
			log.Error("rows and new_rows both has %v", PlayerId)
		}
	} else {
		row = this.m_removed_rows[PlayerId]
		if row == nil {
			_, has_new := this.m_new_rows[PlayerId]
			if has_new {
				delete(this.m_new_rows, PlayerId)
			} else {
				log.Error("row not exist %v", PlayerId)
			}
		} else {
			log.Error("already removed %v", PlayerId)
			_, has_new := this.m_new_rows[PlayerId]
			if has_new {
				delete(this.m_new_rows, PlayerId)
				log.Error("removed rows and new_rows both has %v", PlayerId)
			}
		}
	}
}
func (this *dbPlayerTable) GetRow(PlayerId int32) (row *dbPlayerRow) {
	this.m_lock.UnSafeRLock("dbPlayerTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[PlayerId]
	if row == nil {
		row = this.m_new_rows[PlayerId]
	}
	return row
}
func (this *dbActivitysToDeleteRow)GetStartTime( )(r int32 ){
	this.m_lock.UnSafeRLock("dbActivitysToDeleteRow.GetdbActivitysToDeleteStartTimeColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_StartTime)
}
func (this *dbActivitysToDeleteRow)SetStartTime(v int32){
	this.m_lock.UnSafeLock("dbActivitysToDeleteRow.SetdbActivitysToDeleteStartTimeColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_StartTime=int32(v)
	this.m_StartTime_changed=true
	return
}
func (this *dbActivitysToDeleteRow)GetEndTime( )(r int32 ){
	this.m_lock.UnSafeRLock("dbActivitysToDeleteRow.GetdbActivitysToDeleteEndTimeColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_EndTime)
}
func (this *dbActivitysToDeleteRow)SetEndTime(v int32){
	this.m_lock.UnSafeLock("dbActivitysToDeleteRow.SetdbActivitysToDeleteEndTimeColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_EndTime=int32(v)
	this.m_EndTime_changed=true
	return
}
type dbActivitysToDeleteRow struct {
	m_table *dbActivitysToDeleteTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_Id        int32
	m_StartTime_changed bool
	m_StartTime int32
	m_EndTime_changed bool
	m_EndTime int32
}
func new_dbActivitysToDeleteRow(table *dbActivitysToDeleteTable, Id int32) (r *dbActivitysToDeleteRow) {
	this := &dbActivitysToDeleteRow{}
	this.m_table = table
	this.m_Id = Id
	this.m_lock = NewRWMutex()
	this.m_StartTime_changed=true
	this.m_EndTime_changed=true
	return this
}
func (this *dbActivitysToDeleteRow) GetId() (r int32) {
	return this.m_Id
}
func (this *dbActivitysToDeleteRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbActivitysToDeleteRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(3)
		db_args.Push(this.m_Id)
		db_args.Push(this.m_StartTime)
		db_args.Push(this.m_EndTime)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_StartTime_changed||this.m_EndTime_changed{
			update_string = "UPDATE ActivitysToDeletes SET "
			db_args:=new_db_args(3)
			if this.m_StartTime_changed{
				update_string+="StartTime=?,"
				db_args.Push(this.m_StartTime)
			}
			if this.m_EndTime_changed{
				update_string+="EndTime=?,"
				db_args.Push(this.m_EndTime)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE Id=?"
			db_args.Push(this.m_Id)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_StartTime_changed = false
	this.m_EndTime_changed = false
	if release && this.m_loaded {
		atomic.AddInt32(&this.m_table.m_gc_n, -1)
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbActivitysToDeleteRow) Save(release bool) (err error, d bool, released bool) {
	err,released, state, update_string, args := this.save_data(release)
	if err != nil {
		log.Error("save data failed")
		return err, false, false
	}
	if state == 0 {
		d = false
	} else if state == 1 {
		_, err = this.m_table.m_dbc.StmtExec(this.m_table.m_save_insert_stmt, args...)
		if err != nil {
			log.Error("INSERT ActivitysToDeletes exec failed %v ", this.m_Id)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE ActivitysToDeletes exec failed %v", this.m_Id)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbActivitysToDeleteRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbActivitysToDeleteRowSort struct {
	rows []*dbActivitysToDeleteRow
}
func (this *dbActivitysToDeleteRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbActivitysToDeleteRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbActivitysToDeleteRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbActivitysToDeleteTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[int32]*dbActivitysToDeleteRow
	m_new_rows map[int32]*dbActivitysToDeleteRow
	m_removed_rows map[int32]*dbActivitysToDeleteRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
}
func new_dbActivitysToDeleteTable(dbc *DBC) (this *dbActivitysToDeleteTable) {
	this = &dbActivitysToDeleteTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[int32]*dbActivitysToDeleteRow)
	this.m_new_rows = make(map[int32]*dbActivitysToDeleteRow)
	this.m_removed_rows = make(map[int32]*dbActivitysToDeleteRow)
	return this
}
func (this *dbActivitysToDeleteTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS ActivitysToDeletes(Id int(11),PRIMARY KEY (Id))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS ActivitysToDeletes failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='ActivitysToDeletes'", this.m_dbc.m_db_name)
	if err != nil {
		log.Error("SELECT information_schema failed")
		return
	}
	columns := make(map[string]int32)
	for rows.Next() {
		var column_name string
		var ordinal_position int32
		err = rows.Scan(&column_name, &ordinal_position)
		if err != nil {
			log.Error("scan information_schema row failed")
			return
		}
		if ordinal_position < 1 {
			log.Error("col ordinal out of range")
			continue
		}
		columns[column_name] = ordinal_position
	}
	_, hasStartTime := columns["StartTime"]
	if !hasStartTime {
		_, err = this.m_dbc.Exec("ALTER TABLE ActivitysToDeletes ADD COLUMN StartTime int(11) DEFAULT 0")
		if err != nil {
			log.Error("ADD COLUMN StartTime failed")
			return
		}
	}
	_, hasEndTime := columns["EndTime"]
	if !hasEndTime {
		_, err = this.m_dbc.Exec("ALTER TABLE ActivitysToDeletes ADD COLUMN EndTime int(11) DEFAULT 0")
		if err != nil {
			log.Error("ADD COLUMN EndTime failed")
			return
		}
	}
	return
}
func (this *dbActivitysToDeleteTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT Id,StartTime,EndTime FROM ActivitysToDeletes")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbActivitysToDeleteTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO ActivitysToDeletes (Id,StartTime,EndTime) VALUES (?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbActivitysToDeleteTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM ActivitysToDeletes WHERE Id=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbActivitysToDeleteTable) Init() (err error) {
	err=this.check_create_table()
	if err!=nil{
		log.Error("check_create_table failed")
		return
	}
	err=this.prepare_preload_select_stmt()
	if err!=nil{
		log.Error("prepare_preload_select_stmt failed")
		return
	}
	err=this.prepare_save_insert_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	err=this.prepare_delete_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	return
}
func (this *dbActivitysToDeleteTable) Preload() (err error) {
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var Id int32
	var dStartTime int32
	var dEndTime int32
		this.m_preload_max_id = 0
	for r.Next() {
		err = r.Scan(&Id,&dStartTime,&dEndTime)
		if err != nil {
			log.Error("Scan err[%v]", err.Error())
			return
		}
		if Id>this.m_preload_max_id{
			this.m_preload_max_id =Id
		}
		row := new_dbActivitysToDeleteRow(this,Id)
		row.m_StartTime=dStartTime
		row.m_EndTime=dEndTime
		row.m_StartTime_changed=false
		row.m_EndTime_changed=false
		row.m_valid = true
		this.m_rows[Id]=row
	}
	return
}
func (this *dbActivitysToDeleteTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbActivitysToDeleteTable) fetch_rows(rows map[int32]*dbActivitysToDeleteRow) (r map[int32]*dbActivitysToDeleteRow) {
	this.m_lock.UnSafeLock("dbActivitysToDeleteTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[int32]*dbActivitysToDeleteRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbActivitysToDeleteTable) fetch_new_rows() (new_rows map[int32]*dbActivitysToDeleteRow) {
	this.m_lock.UnSafeLock("dbActivitysToDeleteTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[int32]*dbActivitysToDeleteRow)
	for i, v := range this.m_new_rows {
		_, has := this.m_rows[i]
		if has {
			log.Error("rows already has new rows %v", i)
			continue
		}
		this.m_rows[i] = v
		new_rows[i] = v
	}
	for i, _ := range new_rows {
		delete(this.m_new_rows, i)
	}
	return
}
func (this *dbActivitysToDeleteTable) save_rows(rows map[int32]*dbActivitysToDeleteRow, quick bool) {
	for _, v := range rows {
		if this.m_dbc.m_quit && !quick {
			return
		}
		err, delay, _ := v.Save(false)
		if err != nil {
			log.Error("save failed %v", err)
		}
		if this.m_dbc.m_quit && !quick {
			return
		}
		if delay&&!quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
}
func (this *dbActivitysToDeleteTable) Save(quick bool) (err error){
	removed_rows := this.fetch_rows(this.m_removed_rows)
	for _, v := range removed_rows {
		_, err := this.m_dbc.StmtExec(this.m_delete_stmt, v.GetId())
		if err != nil {
			log.Error("exec delete stmt failed %v", err)
		}
		v.m_valid = false
		if !quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
	this.m_removed_rows = make(map[int32]*dbActivitysToDeleteRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbActivitysToDeleteTable) AddRow(Id int32) (row *dbActivitysToDeleteRow) {
	this.m_lock.UnSafeLock("dbActivitysToDeleteTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	row = new_dbActivitysToDeleteRow(this,Id)
	row.m_new = true
	row.m_loaded = true
	row.m_valid = true
	_, has := this.m_new_rows[Id]
	if has{
		log.Error("已经存在 %v", Id)
		return nil
	}
	this.m_new_rows[Id] = row
	atomic.AddInt32(&this.m_gc_n,1)
	return row
}
func (this *dbActivitysToDeleteTable) RemoveRow(Id int32) {
	this.m_lock.UnSafeLock("dbActivitysToDeleteTable.RemoveRow")
	defer this.m_lock.UnSafeUnlock()
	row := this.m_rows[Id]
	if row != nil {
		row.m_remove = true
		delete(this.m_rows, Id)
		rm_row := this.m_removed_rows[Id]
		if rm_row != nil {
			log.Error("rows and removed rows both has %v", Id)
		}
		this.m_removed_rows[Id] = row
		_, has_new := this.m_new_rows[Id]
		if has_new {
			delete(this.m_new_rows, Id)
			log.Error("rows and new_rows both has %v", Id)
		}
	} else {
		row = this.m_removed_rows[Id]
		if row == nil {
			_, has_new := this.m_new_rows[Id]
			if has_new {
				delete(this.m_new_rows, Id)
			} else {
				log.Error("row not exist %v", Id)
			}
		} else {
			log.Error("already removed %v", Id)
			_, has_new := this.m_new_rows[Id]
			if has_new {
				delete(this.m_new_rows, Id)
				log.Error("removed rows and new_rows both has %v", Id)
			}
		}
	}
}
func (this *dbActivitysToDeleteTable) GetRow(Id int32) (row *dbActivitysToDeleteRow) {
	this.m_lock.UnSafeRLock("dbActivitysToDeleteTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[Id]
	if row == nil {
		row = this.m_new_rows[Id]
	}
	return row
}
func (this *dbSysMailCommonRow)GetCurrMailId( )(r int32 ){
	this.m_lock.UnSafeRLock("dbSysMailCommonRow.GetdbSysMailCommonCurrMailIdColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_CurrMailId)
}
func (this *dbSysMailCommonRow)SetCurrMailId(v int32){
	this.m_lock.UnSafeLock("dbSysMailCommonRow.SetdbSysMailCommonCurrMailIdColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_CurrMailId=int32(v)
	this.m_CurrMailId_changed=true
	return
}
type dbSysMailCommonRow struct {
	m_table *dbSysMailCommonTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_Id        int32
	m_CurrMailId_changed bool
	m_CurrMailId int32
}
func new_dbSysMailCommonRow(table *dbSysMailCommonTable, Id int32) (r *dbSysMailCommonRow) {
	this := &dbSysMailCommonRow{}
	this.m_table = table
	this.m_Id = Id
	this.m_lock = NewRWMutex()
	this.m_CurrMailId_changed=true
	return this
}
func (this *dbSysMailCommonRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbSysMailCommonRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(2)
		db_args.Push(this.m_Id)
		db_args.Push(this.m_CurrMailId)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_CurrMailId_changed{
			update_string = "UPDATE SysMailCommon SET "
			db_args:=new_db_args(2)
			if this.m_CurrMailId_changed{
				update_string+="CurrMailId=?,"
				db_args.Push(this.m_CurrMailId)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE Id=?"
			db_args.Push(this.m_Id)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_CurrMailId_changed = false
	if release && this.m_loaded {
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbSysMailCommonRow) Save(release bool) (err error, d bool, released bool) {
	err,released, state, update_string, args := this.save_data(release)
	if err != nil {
		log.Error("save data failed")
		return err, false, false
	}
	if state == 0 {
		d = false
	} else if state == 1 {
		_, err = this.m_table.m_dbc.StmtExec(this.m_table.m_save_insert_stmt, args...)
		if err != nil {
			log.Error("INSERT SysMailCommon exec failed %v ", this.m_Id)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE SysMailCommon exec failed %v", this.m_Id)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
type dbSysMailCommonTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_row *dbSysMailCommonRow
	m_preload_select_stmt *sql.Stmt
	m_save_insert_stmt *sql.Stmt
}
func new_dbSysMailCommonTable(dbc *DBC) (this *dbSysMailCommonTable) {
	this = &dbSysMailCommonTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	return this
}
func (this *dbSysMailCommonTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS SysMailCommon(Id int(11),PRIMARY KEY (Id))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS SysMailCommon failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='SysMailCommon'", this.m_dbc.m_db_name)
	if err != nil {
		log.Error("SELECT information_schema failed")
		return
	}
	columns := make(map[string]int32)
	for rows.Next() {
		var column_name string
		var ordinal_position int32
		err = rows.Scan(&column_name, &ordinal_position)
		if err != nil {
			log.Error("scan information_schema row failed")
			return
		}
		if ordinal_position < 1 {
			log.Error("col ordinal out of range")
			continue
		}
		columns[column_name] = ordinal_position
	}
	_, hasCurrMailId := columns["CurrMailId"]
	if !hasCurrMailId {
		_, err = this.m_dbc.Exec("ALTER TABLE SysMailCommon ADD COLUMN CurrMailId int(11) DEFAULT 0")
		if err != nil {
			log.Error("ADD COLUMN CurrMailId failed")
			return
		}
	}
	return
}
func (this *dbSysMailCommonTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT CurrMailId FROM SysMailCommon WHERE Id=0")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbSysMailCommonTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO SysMailCommon (Id,CurrMailId) VALUES (?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbSysMailCommonTable) Init() (err error) {
	err=this.check_create_table()
	if err!=nil{
		log.Error("check_create_table failed")
		return
	}
	err=this.prepare_preload_select_stmt()
	if err!=nil{
		log.Error("prepare_preload_select_stmt failed")
		return
	}
	err=this.prepare_save_insert_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	return
}
func (this *dbSysMailCommonTable) Preload() (err error) {
	r := this.m_dbc.StmtQueryRow(this.m_preload_select_stmt)
	var dCurrMailId int32
	err = r.Scan(&dCurrMailId)
	if err!=nil{
		if err!=sql.ErrNoRows{
			log.Error("Scan failed")
			return
		}
	}else{
		row := new_dbSysMailCommonRow(this,0)
		row.m_CurrMailId=dCurrMailId
		row.m_CurrMailId_changed=false
		row.m_valid = true
		row.m_loaded=true
		this.m_row=row
	}
	if this.m_row == nil {
		this.m_row = new_dbSysMailCommonRow(this, 0)
		this.m_row.m_new = true
		this.m_row.m_valid = true
		err = this.Save(false)
		if err != nil {
			log.Error("save failed")
			return
		}
		this.m_row.m_loaded = true
	}
	return
}
func (this *dbSysMailCommonTable) Save(quick bool) (err error) {
	if this.m_row==nil{
		return errors.New("row nil")
	}
	err, _, _ = this.m_row.Save(false)
	return err
}
func (this *dbSysMailCommonTable) GetRow( ) (row *dbSysMailCommonRow) {
	return this.m_row
}
func (this *dbSysMailRow)GetTableId( )(r int32 ){
	this.m_lock.UnSafeRLock("dbSysMailRow.GetdbSysMailTableIdColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_TableId)
}
func (this *dbSysMailRow)SetTableId(v int32){
	this.m_lock.UnSafeLock("dbSysMailRow.SetdbSysMailTableIdColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_TableId=int32(v)
	this.m_TableId_changed=true
	return
}
type dbSysMailAttachedItemsColumn struct{
	m_row *dbSysMailRow
	m_data *dbSysMailAttachedItemsData
	m_changed bool
}
func (this *dbSysMailAttachedItemsColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbSysMailAttachedItemsData{}
		this.m_changed = false
		return nil
	}
	pb := &db.SysMailAttachedItems{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetId())
		return
	}
	this.m_data = &dbSysMailAttachedItemsData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbSysMailAttachedItemsColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbSysMailAttachedItemsColumn)Get( )(v *dbSysMailAttachedItemsData ){
	this.m_row.m_lock.UnSafeRLock("dbSysMailAttachedItemsColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbSysMailAttachedItemsData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbSysMailAttachedItemsColumn)Set(v dbSysMailAttachedItemsData ){
	this.m_row.m_lock.UnSafeLock("dbSysMailAttachedItemsColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbSysMailAttachedItemsData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbSysMailAttachedItemsColumn)GetItemList( )(v []int32 ){
	this.m_row.m_lock.UnSafeRLock("dbSysMailAttachedItemsColumn.GetItemList")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = make([]int32, len(this.m_data.ItemList))
	for _ii, _vv := range this.m_data.ItemList {
		v[_ii]=_vv
	}
	return
}
func (this *dbSysMailAttachedItemsColumn)SetItemList(v []int32){
	this.m_row.m_lock.UnSafeLock("dbSysMailAttachedItemsColumn.SetItemList")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.ItemList = make([]int32, len(v))
	for _ii, _vv := range v {
		this.m_data.ItemList[_ii]=_vv
	}
	this.m_changed = true
	return
}
func (this *dbSysMailRow)GetSendTime( )(r int32 ){
	this.m_lock.UnSafeRLock("dbSysMailRow.GetdbSysMailSendTimeColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_SendTime)
}
func (this *dbSysMailRow)SetSendTime(v int32){
	this.m_lock.UnSafeLock("dbSysMailRow.SetdbSysMailSendTimeColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_SendTime=int32(v)
	this.m_SendTime_changed=true
	return
}
type dbSysMailRow struct {
	m_table *dbSysMailTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_Id        int32
	m_TableId_changed bool
	m_TableId int32
	AttachedItems dbSysMailAttachedItemsColumn
	m_SendTime_changed bool
	m_SendTime int32
}
func new_dbSysMailRow(table *dbSysMailTable, Id int32) (r *dbSysMailRow) {
	this := &dbSysMailRow{}
	this.m_table = table
	this.m_Id = Id
	this.m_lock = NewRWMutex()
	this.m_TableId_changed=true
	this.m_SendTime_changed=true
	this.AttachedItems.m_row=this
	this.AttachedItems.m_data=&dbSysMailAttachedItemsData{}
	return this
}
func (this *dbSysMailRow) GetId() (r int32) {
	return this.m_Id
}
func (this *dbSysMailRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbSysMailRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(4)
		db_args.Push(this.m_Id)
		db_args.Push(this.m_TableId)
		dAttachedItems,db_err:=this.AttachedItems.save()
		if db_err!=nil{
			log.Error("insert save AttachedItems failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dAttachedItems)
		db_args.Push(this.m_SendTime)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_TableId_changed||this.AttachedItems.m_changed||this.m_SendTime_changed{
			update_string = "UPDATE SysMails SET "
			db_args:=new_db_args(4)
			if this.m_TableId_changed{
				update_string+="TableId=?,"
				db_args.Push(this.m_TableId)
			}
			if this.AttachedItems.m_changed{
				update_string+="AttachedItems=?,"
				dAttachedItems,err:=this.AttachedItems.save()
				if err!=nil{
					log.Error("update save AttachedItems failed")
					return err,false,0,"",nil
				}
				db_args.Push(dAttachedItems)
			}
			if this.m_SendTime_changed{
				update_string+="SendTime=?,"
				db_args.Push(this.m_SendTime)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE Id=?"
			db_args.Push(this.m_Id)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_TableId_changed = false
	this.AttachedItems.m_changed = false
	this.m_SendTime_changed = false
	if release && this.m_loaded {
		atomic.AddInt32(&this.m_table.m_gc_n, -1)
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbSysMailRow) Save(release bool) (err error, d bool, released bool) {
	err,released, state, update_string, args := this.save_data(release)
	if err != nil {
		log.Error("save data failed")
		return err, false, false
	}
	if state == 0 {
		d = false
	} else if state == 1 {
		_, err = this.m_table.m_dbc.StmtExec(this.m_table.m_save_insert_stmt, args...)
		if err != nil {
			log.Error("INSERT SysMails exec failed %v ", this.m_Id)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE SysMails exec failed %v", this.m_Id)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbSysMailRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbSysMailRowSort struct {
	rows []*dbSysMailRow
}
func (this *dbSysMailRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbSysMailRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbSysMailRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbSysMailTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[int32]*dbSysMailRow
	m_new_rows map[int32]*dbSysMailRow
	m_removed_rows map[int32]*dbSysMailRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
	m_max_id int32
	m_max_id_changed bool
}
func new_dbSysMailTable(dbc *DBC) (this *dbSysMailTable) {
	this = &dbSysMailTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[int32]*dbSysMailRow)
	this.m_new_rows = make(map[int32]*dbSysMailRow)
	this.m_removed_rows = make(map[int32]*dbSysMailRow)
	return this
}
func (this *dbSysMailTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS SysMailsMaxId(PlaceHolder int(11),MaxId int(11),PRIMARY KEY (PlaceHolder))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS SysMailsMaxId failed")
		return
	}
	r := this.m_dbc.QueryRow("SELECT Count(*) FROM SysMailsMaxId WHERE PlaceHolder=0")
	if r != nil {
		var count int32
		err = r.Scan(&count)
		if err != nil {
			log.Error("scan count failed")
			return
		}
		if count == 0 {
		_, err = this.m_dbc.Exec("INSERT INTO SysMailsMaxId (PlaceHolder,MaxId) VALUES (0,0)")
			if err != nil {
				log.Error("INSERTSysMailsMaxId failed")
				return
			}
		}
	}
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS SysMails(Id int(11),PRIMARY KEY (Id))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS SysMails failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='SysMails'", this.m_dbc.m_db_name)
	if err != nil {
		log.Error("SELECT information_schema failed")
		return
	}
	columns := make(map[string]int32)
	for rows.Next() {
		var column_name string
		var ordinal_position int32
		err = rows.Scan(&column_name, &ordinal_position)
		if err != nil {
			log.Error("scan information_schema row failed")
			return
		}
		if ordinal_position < 1 {
			log.Error("col ordinal out of range")
			continue
		}
		columns[column_name] = ordinal_position
	}
	_, hasTableId := columns["TableId"]
	if !hasTableId {
		_, err = this.m_dbc.Exec("ALTER TABLE SysMails ADD COLUMN TableId int(11) DEFAULT 0")
		if err != nil {
			log.Error("ADD COLUMN TableId failed")
			return
		}
	}
	_, hasAttachedItems := columns["AttachedItems"]
	if !hasAttachedItems {
		_, err = this.m_dbc.Exec("ALTER TABLE SysMails ADD COLUMN AttachedItems LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN AttachedItems failed")
			return
		}
	}
	_, hasSendTime := columns["SendTime"]
	if !hasSendTime {
		_, err = this.m_dbc.Exec("ALTER TABLE SysMails ADD COLUMN SendTime int(11) DEFAULT 0")
		if err != nil {
			log.Error("ADD COLUMN SendTime failed")
			return
		}
	}
	return
}
func (this *dbSysMailTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT Id,TableId,AttachedItems,SendTime FROM SysMails")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbSysMailTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO SysMails (Id,TableId,AttachedItems,SendTime) VALUES (?,?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbSysMailTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM SysMails WHERE Id=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbSysMailTable) Init() (err error) {
	err=this.check_create_table()
	if err!=nil{
		log.Error("check_create_table failed")
		return
	}
	err=this.prepare_preload_select_stmt()
	if err!=nil{
		log.Error("prepare_preload_select_stmt failed")
		return
	}
	err=this.prepare_save_insert_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	err=this.prepare_delete_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	return
}
func (this *dbSysMailTable) Preload() (err error) {
	r_max_id := this.m_dbc.QueryRow("SELECT MaxId FROM SysMailsMaxId WHERE PLACEHOLDER=0")
	if r_max_id != nil {
		err = r_max_id.Scan(&this.m_max_id)
		if err != nil {
			log.Error("scan max id failed")
			return
		}
	}
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var Id int32
	var dTableId int32
	var dAttachedItems []byte
	var dSendTime int32
	for r.Next() {
		err = r.Scan(&Id,&dTableId,&dAttachedItems,&dSendTime)
		if err != nil {
			log.Error("Scan err[%v]", err.Error())
			return
		}
		if Id>this.m_max_id{
			log.Error("max id ext")
			this.m_max_id = Id
			this.m_max_id_changed = true
		}
		row := new_dbSysMailRow(this,Id)
		row.m_TableId=dTableId
		err = row.AttachedItems.load(dAttachedItems)
		if err != nil {
			log.Error("AttachedItems %v", Id)
			return
		}
		row.m_SendTime=dSendTime
		row.m_TableId_changed=false
		row.m_SendTime_changed=false
		row.m_valid = true
		this.m_rows[Id]=row
	}
	return
}
func (this *dbSysMailTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbSysMailTable) fetch_rows(rows map[int32]*dbSysMailRow) (r map[int32]*dbSysMailRow) {
	this.m_lock.UnSafeLock("dbSysMailTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[int32]*dbSysMailRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbSysMailTable) fetch_new_rows() (new_rows map[int32]*dbSysMailRow) {
	this.m_lock.UnSafeLock("dbSysMailTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[int32]*dbSysMailRow)
	for i, v := range this.m_new_rows {
		_, has := this.m_rows[i]
		if has {
			log.Error("rows already has new rows %v", i)
			continue
		}
		this.m_rows[i] = v
		new_rows[i] = v
	}
	for i, _ := range new_rows {
		delete(this.m_new_rows, i)
	}
	return
}
func (this *dbSysMailTable) save_rows(rows map[int32]*dbSysMailRow, quick bool) {
	for _, v := range rows {
		if this.m_dbc.m_quit && !quick {
			return
		}
		err, delay, _ := v.Save(false)
		if err != nil {
			log.Error("save failed %v", err)
		}
		if this.m_dbc.m_quit && !quick {
			return
		}
		if delay&&!quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
}
func (this *dbSysMailTable) Save(quick bool) (err error){
	if this.m_max_id_changed {
		max_id := atomic.LoadInt32(&this.m_max_id)
		_, err := this.m_dbc.Exec("UPDATE SysMailsMaxId SET MaxId=?", max_id)
		if err != nil {
			log.Error("save max id failed %v", err)
		}
	}
	removed_rows := this.fetch_rows(this.m_removed_rows)
	for _, v := range removed_rows {
		_, err := this.m_dbc.StmtExec(this.m_delete_stmt, v.GetId())
		if err != nil {
			log.Error("exec delete stmt failed %v", err)
		}
		v.m_valid = false
		if !quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
	this.m_removed_rows = make(map[int32]*dbSysMailRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbSysMailTable) AddRow() (row *dbSysMailRow) {
	this.m_lock.UnSafeLock("dbSysMailTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	Id := atomic.AddInt32(&this.m_max_id, 1)
	this.m_max_id_changed = true
	row = new_dbSysMailRow(this,Id)
	row.m_new = true
	row.m_loaded = true
	row.m_valid = true
	this.m_new_rows[Id] = row
	atomic.AddInt32(&this.m_gc_n,1)
	return row
}
func (this *dbSysMailTable) RemoveRow(Id int32) {
	this.m_lock.UnSafeLock("dbSysMailTable.RemoveRow")
	defer this.m_lock.UnSafeUnlock()
	row := this.m_rows[Id]
	if row != nil {
		row.m_remove = true
		delete(this.m_rows, Id)
		rm_row := this.m_removed_rows[Id]
		if rm_row != nil {
			log.Error("rows and removed rows both has %v", Id)
		}
		this.m_removed_rows[Id] = row
		_, has_new := this.m_new_rows[Id]
		if has_new {
			delete(this.m_new_rows, Id)
			log.Error("rows and new_rows both has %v", Id)
		}
	} else {
		row = this.m_removed_rows[Id]
		if row == nil {
			_, has_new := this.m_new_rows[Id]
			if has_new {
				delete(this.m_new_rows, Id)
			} else {
				log.Error("row not exist %v", Id)
			}
		} else {
			log.Error("already removed %v", Id)
			_, has_new := this.m_new_rows[Id]
			if has_new {
				delete(this.m_new_rows, Id)
				log.Error("removed rows and new_rows both has %v", Id)
			}
		}
	}
}
func (this *dbSysMailTable) GetRow(Id int32) (row *dbSysMailRow) {
	this.m_lock.UnSafeRLock("dbSysMailTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[Id]
	if row == nil {
		row = this.m_new_rows[Id]
	}
	return row
}
func (this *dbBanPlayerRow)GetStartTime( )(r int32 ){
	this.m_lock.UnSafeRLock("dbBanPlayerRow.GetdbBanPlayerStartTimeColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_StartTime)
}
func (this *dbBanPlayerRow)SetStartTime(v int32){
	this.m_lock.UnSafeLock("dbBanPlayerRow.SetdbBanPlayerStartTimeColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_StartTime=int32(v)
	this.m_StartTime_changed=true
	return
}
func (this *dbBanPlayerRow)GetStartTimeStr( )(r string ){
	this.m_lock.UnSafeRLock("dbBanPlayerRow.GetdbBanPlayerStartTimeStrColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_StartTimeStr)
}
func (this *dbBanPlayerRow)SetStartTimeStr(v string){
	this.m_lock.UnSafeLock("dbBanPlayerRow.SetdbBanPlayerStartTimeStrColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_StartTimeStr=string(v)
	this.m_StartTimeStr_changed=true
	return
}
func (this *dbBanPlayerRow)GetDuration( )(r int32 ){
	this.m_lock.UnSafeRLock("dbBanPlayerRow.GetdbBanPlayerDurationColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_Duration)
}
func (this *dbBanPlayerRow)SetDuration(v int32){
	this.m_lock.UnSafeLock("dbBanPlayerRow.SetdbBanPlayerDurationColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Duration=int32(v)
	this.m_Duration_changed=true
	return
}
func (this *dbBanPlayerRow)GetPlayerId( )(r int32 ){
	this.m_lock.UnSafeRLock("dbBanPlayerRow.GetdbBanPlayerPlayerIdColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_PlayerId)
}
func (this *dbBanPlayerRow)SetPlayerId(v int32){
	this.m_lock.UnSafeLock("dbBanPlayerRow.SetdbBanPlayerPlayerIdColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_PlayerId=int32(v)
	this.m_PlayerId_changed=true
	return
}
func (this *dbBanPlayerRow)GetAccount( )(r string ){
	this.m_lock.UnSafeRLock("dbBanPlayerRow.GetdbBanPlayerAccountColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_Account)
}
func (this *dbBanPlayerRow)SetAccount(v string){
	this.m_lock.UnSafeLock("dbBanPlayerRow.SetdbBanPlayerAccountColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Account=string(v)
	this.m_Account_changed=true
	return
}
type dbBanPlayerRow struct {
	m_table *dbBanPlayerTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_UniqueId        string
	m_StartTime_changed bool
	m_StartTime int32
	m_StartTimeStr_changed bool
	m_StartTimeStr string
	m_Duration_changed bool
	m_Duration int32
	m_PlayerId_changed bool
	m_PlayerId int32
	m_Account_changed bool
	m_Account string
}
func new_dbBanPlayerRow(table *dbBanPlayerTable, UniqueId string) (r *dbBanPlayerRow) {
	this := &dbBanPlayerRow{}
	this.m_table = table
	this.m_UniqueId = UniqueId
	this.m_lock = NewRWMutex()
	this.m_StartTime_changed=true
	this.m_StartTimeStr_changed=true
	this.m_Duration_changed=true
	this.m_PlayerId_changed=true
	this.m_Account_changed=true
	return this
}
func (this *dbBanPlayerRow) GetUniqueId() (r string) {
	return this.m_UniqueId
}
func (this *dbBanPlayerRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbBanPlayerRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(6)
		db_args.Push(this.m_UniqueId)
		db_args.Push(this.m_StartTime)
		db_args.Push(this.m_StartTimeStr)
		db_args.Push(this.m_Duration)
		db_args.Push(this.m_PlayerId)
		db_args.Push(this.m_Account)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_StartTime_changed||this.m_StartTimeStr_changed||this.m_Duration_changed||this.m_PlayerId_changed||this.m_Account_changed{
			update_string = "UPDATE BanPlayers SET "
			db_args:=new_db_args(6)
			if this.m_StartTime_changed{
				update_string+="StartTime=?,"
				db_args.Push(this.m_StartTime)
			}
			if this.m_StartTimeStr_changed{
				update_string+="StartTimeStr=?,"
				db_args.Push(this.m_StartTimeStr)
			}
			if this.m_Duration_changed{
				update_string+="Duration=?,"
				db_args.Push(this.m_Duration)
			}
			if this.m_PlayerId_changed{
				update_string+="PlayerId=?,"
				db_args.Push(this.m_PlayerId)
			}
			if this.m_Account_changed{
				update_string+="Account=?,"
				db_args.Push(this.m_Account)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE UniqueId=?"
			db_args.Push(this.m_UniqueId)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_StartTime_changed = false
	this.m_StartTimeStr_changed = false
	this.m_Duration_changed = false
	this.m_PlayerId_changed = false
	this.m_Account_changed = false
	if release && this.m_loaded {
		atomic.AddInt32(&this.m_table.m_gc_n, -1)
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbBanPlayerRow) Save(release bool) (err error, d bool, released bool) {
	err,released, state, update_string, args := this.save_data(release)
	if err != nil {
		log.Error("save data failed")
		return err, false, false
	}
	if state == 0 {
		d = false
	} else if state == 1 {
		_, err = this.m_table.m_dbc.StmtExec(this.m_table.m_save_insert_stmt, args...)
		if err != nil {
			log.Error("INSERT BanPlayers exec failed %v ", this.m_UniqueId)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE BanPlayers exec failed %v", this.m_UniqueId)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbBanPlayerRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbBanPlayerRowSort struct {
	rows []*dbBanPlayerRow
}
func (this *dbBanPlayerRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbBanPlayerRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbBanPlayerRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbBanPlayerTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[string]*dbBanPlayerRow
	m_new_rows map[string]*dbBanPlayerRow
	m_removed_rows map[string]*dbBanPlayerRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
}
func new_dbBanPlayerTable(dbc *DBC) (this *dbBanPlayerTable) {
	this = &dbBanPlayerTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[string]*dbBanPlayerRow)
	this.m_new_rows = make(map[string]*dbBanPlayerRow)
	this.m_removed_rows = make(map[string]*dbBanPlayerRow)
	return this
}
func (this *dbBanPlayerTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS BanPlayers(UniqueId varchar(64),PRIMARY KEY (UniqueId))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS BanPlayers failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='BanPlayers'", this.m_dbc.m_db_name)
	if err != nil {
		log.Error("SELECT information_schema failed")
		return
	}
	columns := make(map[string]int32)
	for rows.Next() {
		var column_name string
		var ordinal_position int32
		err = rows.Scan(&column_name, &ordinal_position)
		if err != nil {
			log.Error("scan information_schema row failed")
			return
		}
		if ordinal_position < 1 {
			log.Error("col ordinal out of range")
			continue
		}
		columns[column_name] = ordinal_position
	}
	_, hasStartTime := columns["StartTime"]
	if !hasStartTime {
		_, err = this.m_dbc.Exec("ALTER TABLE BanPlayers ADD COLUMN StartTime int(11) DEFAULT 0")
		if err != nil {
			log.Error("ADD COLUMN StartTime failed")
			return
		}
	}
	_, hasStartTimeStr := columns["StartTimeStr"]
	if !hasStartTimeStr {
		_, err = this.m_dbc.Exec("ALTER TABLE BanPlayers ADD COLUMN StartTimeStr varchar(45) DEFAULT ''")
		if err != nil {
			log.Error("ADD COLUMN StartTimeStr failed")
			return
		}
	}
	_, hasDuration := columns["Duration"]
	if !hasDuration {
		_, err = this.m_dbc.Exec("ALTER TABLE BanPlayers ADD COLUMN Duration int(11) DEFAULT 0")
		if err != nil {
			log.Error("ADD COLUMN Duration failed")
			return
		}
	}
	_, hasPlayerId := columns["PlayerId"]
	if !hasPlayerId {
		_, err = this.m_dbc.Exec("ALTER TABLE BanPlayers ADD COLUMN PlayerId int(11) DEFAULT 0")
		if err != nil {
			log.Error("ADD COLUMN PlayerId failed")
			return
		}
	}
	_, hasAccount := columns["Account"]
	if !hasAccount {
		_, err = this.m_dbc.Exec("ALTER TABLE BanPlayers ADD COLUMN Account varchar(45) DEFAULT ''")
		if err != nil {
			log.Error("ADD COLUMN Account failed")
			return
		}
	}
	return
}
func (this *dbBanPlayerTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT UniqueId,StartTime,StartTimeStr,Duration,PlayerId,Account FROM BanPlayers")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbBanPlayerTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO BanPlayers (UniqueId,StartTime,StartTimeStr,Duration,PlayerId,Account) VALUES (?,?,?,?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbBanPlayerTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM BanPlayers WHERE UniqueId=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbBanPlayerTable) Init() (err error) {
	err=this.check_create_table()
	if err!=nil{
		log.Error("check_create_table failed")
		return
	}
	err=this.prepare_preload_select_stmt()
	if err!=nil{
		log.Error("prepare_preload_select_stmt failed")
		return
	}
	err=this.prepare_save_insert_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	err=this.prepare_delete_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	return
}
func (this *dbBanPlayerTable) Preload() (err error) {
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var UniqueId string
	var dStartTime int32
	var dStartTimeStr string
	var dDuration int32
	var dPlayerId int32
	var dAccount string
	for r.Next() {
		err = r.Scan(&UniqueId,&dStartTime,&dStartTimeStr,&dDuration,&dPlayerId,&dAccount)
		if err != nil {
			log.Error("Scan err[%v]", err.Error())
			return
		}
		row := new_dbBanPlayerRow(this,UniqueId)
		row.m_StartTime=dStartTime
		row.m_StartTimeStr=dStartTimeStr
		row.m_Duration=dDuration
		row.m_PlayerId=dPlayerId
		row.m_Account=dAccount
		row.m_StartTime_changed=false
		row.m_StartTimeStr_changed=false
		row.m_Duration_changed=false
		row.m_PlayerId_changed=false
		row.m_Account_changed=false
		row.m_valid = true
		this.m_rows[UniqueId]=row
	}
	return
}
func (this *dbBanPlayerTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbBanPlayerTable) fetch_rows(rows map[string]*dbBanPlayerRow) (r map[string]*dbBanPlayerRow) {
	this.m_lock.UnSafeLock("dbBanPlayerTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[string]*dbBanPlayerRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbBanPlayerTable) fetch_new_rows() (new_rows map[string]*dbBanPlayerRow) {
	this.m_lock.UnSafeLock("dbBanPlayerTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[string]*dbBanPlayerRow)
	for i, v := range this.m_new_rows {
		_, has := this.m_rows[i]
		if has {
			log.Error("rows already has new rows %v", i)
			continue
		}
		this.m_rows[i] = v
		new_rows[i] = v
	}
	for i, _ := range new_rows {
		delete(this.m_new_rows, i)
	}
	return
}
func (this *dbBanPlayerTable) save_rows(rows map[string]*dbBanPlayerRow, quick bool) {
	for _, v := range rows {
		if this.m_dbc.m_quit && !quick {
			return
		}
		err, delay, _ := v.Save(false)
		if err != nil {
			log.Error("save failed %v", err)
		}
		if this.m_dbc.m_quit && !quick {
			return
		}
		if delay&&!quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
}
func (this *dbBanPlayerTable) Save(quick bool) (err error){
	removed_rows := this.fetch_rows(this.m_removed_rows)
	for _, v := range removed_rows {
		_, err := this.m_dbc.StmtExec(this.m_delete_stmt, v.GetUniqueId())
		if err != nil {
			log.Error("exec delete stmt failed %v", err)
		}
		v.m_valid = false
		if !quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
	this.m_removed_rows = make(map[string]*dbBanPlayerRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbBanPlayerTable) AddRow(UniqueId string) (row *dbBanPlayerRow) {
	this.m_lock.UnSafeLock("dbBanPlayerTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	row = new_dbBanPlayerRow(this,UniqueId)
	row.m_new = true
	row.m_loaded = true
	row.m_valid = true
	_, has := this.m_new_rows[UniqueId]
	if has{
		log.Error("已经存在 %v", UniqueId)
		return nil
	}
	this.m_new_rows[UniqueId] = row
	atomic.AddInt32(&this.m_gc_n,1)
	return row
}
func (this *dbBanPlayerTable) RemoveRow(UniqueId string) {
	this.m_lock.UnSafeLock("dbBanPlayerTable.RemoveRow")
	defer this.m_lock.UnSafeUnlock()
	row := this.m_rows[UniqueId]
	if row != nil {
		row.m_remove = true
		delete(this.m_rows, UniqueId)
		rm_row := this.m_removed_rows[UniqueId]
		if rm_row != nil {
			log.Error("rows and removed rows both has %v", UniqueId)
		}
		this.m_removed_rows[UniqueId] = row
		_, has_new := this.m_new_rows[UniqueId]
		if has_new {
			delete(this.m_new_rows, UniqueId)
			log.Error("rows and new_rows both has %v", UniqueId)
		}
	} else {
		row = this.m_removed_rows[UniqueId]
		if row == nil {
			_, has_new := this.m_new_rows[UniqueId]
			if has_new {
				delete(this.m_new_rows, UniqueId)
			} else {
				log.Error("row not exist %v", UniqueId)
			}
		} else {
			log.Error("already removed %v", UniqueId)
			_, has_new := this.m_new_rows[UniqueId]
			if has_new {
				delete(this.m_new_rows, UniqueId)
				log.Error("removed rows and new_rows both has %v", UniqueId)
			}
		}
	}
}
func (this *dbBanPlayerTable) GetRow(UniqueId string) (row *dbBanPlayerRow) {
	this.m_lock.UnSafeRLock("dbBanPlayerTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[UniqueId]
	if row == nil {
		row = this.m_new_rows[UniqueId]
	}
	return row
}
func (this *dbServerInfoRow)GetCreateUnix( )(r int32 ){
	this.m_lock.UnSafeRLock("dbServerInfoRow.GetdbServerInfoCreateUnixColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_CreateUnix)
}
func (this *dbServerInfoRow)SetCreateUnix(v int32){
	this.m_lock.UnSafeLock("dbServerInfoRow.SetdbServerInfoCreateUnixColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_CreateUnix=int32(v)
	this.m_CreateUnix_changed=true
	return
}
func (this *dbServerInfoRow)GetCurStartUnix( )(r int32 ){
	this.m_lock.UnSafeRLock("dbServerInfoRow.GetdbServerInfoCurStartUnixColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_CurStartUnix)
}
func (this *dbServerInfoRow)SetCurStartUnix(v int32){
	this.m_lock.UnSafeLock("dbServerInfoRow.SetdbServerInfoCurStartUnixColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_CurStartUnix=int32(v)
	this.m_CurStartUnix_changed=true
	return
}
type dbServerInfoRow struct {
	m_table *dbServerInfoTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_KeyId        int32
	m_CreateUnix_changed bool
	m_CreateUnix int32
	m_CurStartUnix_changed bool
	m_CurStartUnix int32
}
func new_dbServerInfoRow(table *dbServerInfoTable, KeyId int32) (r *dbServerInfoRow) {
	this := &dbServerInfoRow{}
	this.m_table = table
	this.m_KeyId = KeyId
	this.m_lock = NewRWMutex()
	this.m_CreateUnix_changed=true
	this.m_CurStartUnix_changed=true
	return this
}
func (this *dbServerInfoRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbServerInfoRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(3)
		db_args.Push(this.m_KeyId)
		db_args.Push(this.m_CreateUnix)
		db_args.Push(this.m_CurStartUnix)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_CreateUnix_changed||this.m_CurStartUnix_changed{
			update_string = "UPDATE ServerInfo SET "
			db_args:=new_db_args(3)
			if this.m_CreateUnix_changed{
				update_string+="CreateUnix=?,"
				db_args.Push(this.m_CreateUnix)
			}
			if this.m_CurStartUnix_changed{
				update_string+="CurStartUnix=?,"
				db_args.Push(this.m_CurStartUnix)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE KeyId=?"
			db_args.Push(this.m_KeyId)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_CreateUnix_changed = false
	this.m_CurStartUnix_changed = false
	if release && this.m_loaded {
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbServerInfoRow) Save(release bool) (err error, d bool, released bool) {
	err,released, state, update_string, args := this.save_data(release)
	if err != nil {
		log.Error("save data failed")
		return err, false, false
	}
	if state == 0 {
		d = false
	} else if state == 1 {
		_, err = this.m_table.m_dbc.StmtExec(this.m_table.m_save_insert_stmt, args...)
		if err != nil {
			log.Error("INSERT ServerInfo exec failed %v ", this.m_KeyId)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE ServerInfo exec failed %v", this.m_KeyId)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
type dbServerInfoTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_row *dbServerInfoRow
	m_preload_select_stmt *sql.Stmt
	m_save_insert_stmt *sql.Stmt
}
func new_dbServerInfoTable(dbc *DBC) (this *dbServerInfoTable) {
	this = &dbServerInfoTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	return this
}
func (this *dbServerInfoTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS ServerInfo(KeyId int(11),PRIMARY KEY (KeyId))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS ServerInfo failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='ServerInfo'", this.m_dbc.m_db_name)
	if err != nil {
		log.Error("SELECT information_schema failed")
		return
	}
	columns := make(map[string]int32)
	for rows.Next() {
		var column_name string
		var ordinal_position int32
		err = rows.Scan(&column_name, &ordinal_position)
		if err != nil {
			log.Error("scan information_schema row failed")
			return
		}
		if ordinal_position < 1 {
			log.Error("col ordinal out of range")
			continue
		}
		columns[column_name] = ordinal_position
	}
	_, hasCreateUnix := columns["CreateUnix"]
	if !hasCreateUnix {
		_, err = this.m_dbc.Exec("ALTER TABLE ServerInfo ADD COLUMN CreateUnix int(11)")
		if err != nil {
			log.Error("ADD COLUMN CreateUnix failed")
			return
		}
	}
	_, hasCurStartUnix := columns["CurStartUnix"]
	if !hasCurStartUnix {
		_, err = this.m_dbc.Exec("ALTER TABLE ServerInfo ADD COLUMN CurStartUnix int(11)")
		if err != nil {
			log.Error("ADD COLUMN CurStartUnix failed")
			return
		}
	}
	return
}
func (this *dbServerInfoTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT CreateUnix,CurStartUnix FROM ServerInfo WHERE KeyId=0")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbServerInfoTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO ServerInfo (KeyId,CreateUnix,CurStartUnix) VALUES (?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbServerInfoTable) Init() (err error) {
	err=this.check_create_table()
	if err!=nil{
		log.Error("check_create_table failed")
		return
	}
	err=this.prepare_preload_select_stmt()
	if err!=nil{
		log.Error("prepare_preload_select_stmt failed")
		return
	}
	err=this.prepare_save_insert_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	return
}
func (this *dbServerInfoTable) Preload() (err error) {
	r := this.m_dbc.StmtQueryRow(this.m_preload_select_stmt)
	var dCreateUnix int32
	var dCurStartUnix int32
	err = r.Scan(&dCreateUnix,&dCurStartUnix)
	if err!=nil{
		if err!=sql.ErrNoRows{
			log.Error("Scan failed")
			return
		}
	}else{
		row := new_dbServerInfoRow(this,0)
		row.m_CreateUnix=dCreateUnix
		row.m_CurStartUnix=dCurStartUnix
		row.m_CreateUnix_changed=false
		row.m_CurStartUnix_changed=false
		row.m_valid = true
		row.m_loaded=true
		this.m_row=row
	}
	if this.m_row == nil {
		this.m_row = new_dbServerInfoRow(this, 0)
		this.m_row.m_new = true
		this.m_row.m_valid = true
		err = this.Save(false)
		if err != nil {
			log.Error("save failed")
			return
		}
		this.m_row.m_loaded = true
	}
	return
}
func (this *dbServerInfoTable) Save(quick bool) (err error) {
	if this.m_row==nil{
		return errors.New("row nil")
	}
	err, _, _ = this.m_row.Save(false)
	return err
}
func (this *dbServerInfoTable) GetRow( ) (row *dbServerInfoRow) {
	return this.m_row
}
func (this *dbOtherServerPlayerRow)GetAccount( )(r string ){
	this.m_lock.UnSafeRLock("dbOtherServerPlayerRow.GetdbOtherServerPlayerAccountColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_Account)
}
func (this *dbOtherServerPlayerRow)SetAccount(v string){
	this.m_lock.UnSafeLock("dbOtherServerPlayerRow.SetdbOtherServerPlayerAccountColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Account=string(v)
	this.m_Account_changed=true
	return
}
func (this *dbOtherServerPlayerRow)GetName( )(r string ){
	this.m_lock.UnSafeRLock("dbOtherServerPlayerRow.GetdbOtherServerPlayerNameColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_Name)
}
func (this *dbOtherServerPlayerRow)SetName(v string){
	this.m_lock.UnSafeLock("dbOtherServerPlayerRow.SetdbOtherServerPlayerNameColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Name=string(v)
	this.m_Name_changed=true
	return
}
func (this *dbOtherServerPlayerRow)GetLevel( )(r int32 ){
	this.m_lock.UnSafeRLock("dbOtherServerPlayerRow.GetdbOtherServerPlayerLevelColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_Level)
}
func (this *dbOtherServerPlayerRow)SetLevel(v int32){
	this.m_lock.UnSafeLock("dbOtherServerPlayerRow.SetdbOtherServerPlayerLevelColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Level=int32(v)
	this.m_Level_changed=true
	return
}
func (this *dbOtherServerPlayerRow)GetHead( )(r string ){
	this.m_lock.UnSafeRLock("dbOtherServerPlayerRow.GetdbOtherServerPlayerHeadColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_Head)
}
func (this *dbOtherServerPlayerRow)SetHead(v string){
	this.m_lock.UnSafeLock("dbOtherServerPlayerRow.SetdbOtherServerPlayerHeadColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Head=string(v)
	this.m_Head_changed=true
	return
}
type dbOtherServerPlayerRow struct {
	m_table *dbOtherServerPlayerTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_PlayerId        int32
	m_Account_changed bool
	m_Account string
	m_Name_changed bool
	m_Name string
	m_Level_changed bool
	m_Level int32
	m_Head_changed bool
	m_Head string
}
func new_dbOtherServerPlayerRow(table *dbOtherServerPlayerTable, PlayerId int32) (r *dbOtherServerPlayerRow) {
	this := &dbOtherServerPlayerRow{}
	this.m_table = table
	this.m_PlayerId = PlayerId
	this.m_lock = NewRWMutex()
	this.m_Account_changed=true
	this.m_Name_changed=true
	this.m_Level_changed=true
	this.m_Head_changed=true
	return this
}
func (this *dbOtherServerPlayerRow) GetPlayerId() (r int32) {
	return this.m_PlayerId
}
func (this *dbOtherServerPlayerRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbOtherServerPlayerRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(5)
		db_args.Push(this.m_PlayerId)
		db_args.Push(this.m_Account)
		db_args.Push(this.m_Name)
		db_args.Push(this.m_Level)
		db_args.Push(this.m_Head)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_Account_changed||this.m_Name_changed||this.m_Level_changed||this.m_Head_changed{
			update_string = "UPDATE OtherServerPlayers SET "
			db_args:=new_db_args(5)
			if this.m_Account_changed{
				update_string+="Account=?,"
				db_args.Push(this.m_Account)
			}
			if this.m_Name_changed{
				update_string+="Name=?,"
				db_args.Push(this.m_Name)
			}
			if this.m_Level_changed{
				update_string+="Level=?,"
				db_args.Push(this.m_Level)
			}
			if this.m_Head_changed{
				update_string+="Head=?,"
				db_args.Push(this.m_Head)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE PlayerId=?"
			db_args.Push(this.m_PlayerId)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_Account_changed = false
	this.m_Name_changed = false
	this.m_Level_changed = false
	this.m_Head_changed = false
	if release && this.m_loaded {
		atomic.AddInt32(&this.m_table.m_gc_n, -1)
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbOtherServerPlayerRow) Save(release bool) (err error, d bool, released bool) {
	err,released, state, update_string, args := this.save_data(release)
	if err != nil {
		log.Error("save data failed")
		return err, false, false
	}
	if state == 0 {
		d = false
	} else if state == 1 {
		_, err = this.m_table.m_dbc.StmtExec(this.m_table.m_save_insert_stmt, args...)
		if err != nil {
			log.Error("INSERT OtherServerPlayers exec failed %v ", this.m_PlayerId)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE OtherServerPlayers exec failed %v", this.m_PlayerId)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbOtherServerPlayerRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbOtherServerPlayerRowSort struct {
	rows []*dbOtherServerPlayerRow
}
func (this *dbOtherServerPlayerRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbOtherServerPlayerRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbOtherServerPlayerRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbOtherServerPlayerTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[int32]*dbOtherServerPlayerRow
	m_new_rows map[int32]*dbOtherServerPlayerRow
	m_removed_rows map[int32]*dbOtherServerPlayerRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
}
func new_dbOtherServerPlayerTable(dbc *DBC) (this *dbOtherServerPlayerTable) {
	this = &dbOtherServerPlayerTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[int32]*dbOtherServerPlayerRow)
	this.m_new_rows = make(map[int32]*dbOtherServerPlayerRow)
	this.m_removed_rows = make(map[int32]*dbOtherServerPlayerRow)
	return this
}
func (this *dbOtherServerPlayerTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS OtherServerPlayers(PlayerId int(11),PRIMARY KEY (PlayerId))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS OtherServerPlayers failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='OtherServerPlayers'", this.m_dbc.m_db_name)
	if err != nil {
		log.Error("SELECT information_schema failed")
		return
	}
	columns := make(map[string]int32)
	for rows.Next() {
		var column_name string
		var ordinal_position int32
		err = rows.Scan(&column_name, &ordinal_position)
		if err != nil {
			log.Error("scan information_schema row failed")
			return
		}
		if ordinal_position < 1 {
			log.Error("col ordinal out of range")
			continue
		}
		columns[column_name] = ordinal_position
	}
	_, hasAccount := columns["Account"]
	if !hasAccount {
		_, err = this.m_dbc.Exec("ALTER TABLE OtherServerPlayers ADD COLUMN Account varchar(45)")
		if err != nil {
			log.Error("ADD COLUMN Account failed")
			return
		}
	}
	_, hasName := columns["Name"]
	if !hasName {
		_, err = this.m_dbc.Exec("ALTER TABLE OtherServerPlayers ADD COLUMN Name varchar(45)")
		if err != nil {
			log.Error("ADD COLUMN Name failed")
			return
		}
	}
	_, hasLevel := columns["Level"]
	if !hasLevel {
		_, err = this.m_dbc.Exec("ALTER TABLE OtherServerPlayers ADD COLUMN Level int(11)")
		if err != nil {
			log.Error("ADD COLUMN Level failed")
			return
		}
	}
	_, hasHead := columns["Head"]
	if !hasHead {
		_, err = this.m_dbc.Exec("ALTER TABLE OtherServerPlayers ADD COLUMN Head varchar(45)")
		if err != nil {
			log.Error("ADD COLUMN Head failed")
			return
		}
	}
	return
}
func (this *dbOtherServerPlayerTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT PlayerId,Account,Name,Level,Head FROM OtherServerPlayers")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbOtherServerPlayerTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO OtherServerPlayers (PlayerId,Account,Name,Level,Head) VALUES (?,?,?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbOtherServerPlayerTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM OtherServerPlayers WHERE PlayerId=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbOtherServerPlayerTable) Init() (err error) {
	err=this.check_create_table()
	if err!=nil{
		log.Error("check_create_table failed")
		return
	}
	err=this.prepare_preload_select_stmt()
	if err!=nil{
		log.Error("prepare_preload_select_stmt failed")
		return
	}
	err=this.prepare_save_insert_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	err=this.prepare_delete_stmt()
	if err!=nil{
		log.Error("prepare_save_insert_stmt failed")
		return
	}
	return
}
func (this *dbOtherServerPlayerTable) Preload() (err error) {
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var PlayerId int32
	var dAccount string
	var dName string
	var dLevel int32
	var dHead string
		this.m_preload_max_id = 0
	for r.Next() {
		err = r.Scan(&PlayerId,&dAccount,&dName,&dLevel,&dHead)
		if err != nil {
			log.Error("Scan err[%v]", err.Error())
			return
		}
		if PlayerId>this.m_preload_max_id{
			this.m_preload_max_id =PlayerId
		}
		row := new_dbOtherServerPlayerRow(this,PlayerId)
		row.m_Account=dAccount
		row.m_Name=dName
		row.m_Level=dLevel
		row.m_Head=dHead
		row.m_Account_changed=false
		row.m_Name_changed=false
		row.m_Level_changed=false
		row.m_Head_changed=false
		row.m_valid = true
		this.m_rows[PlayerId]=row
	}
	return
}
func (this *dbOtherServerPlayerTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbOtherServerPlayerTable) fetch_rows(rows map[int32]*dbOtherServerPlayerRow) (r map[int32]*dbOtherServerPlayerRow) {
	this.m_lock.UnSafeLock("dbOtherServerPlayerTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[int32]*dbOtherServerPlayerRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbOtherServerPlayerTable) fetch_new_rows() (new_rows map[int32]*dbOtherServerPlayerRow) {
	this.m_lock.UnSafeLock("dbOtherServerPlayerTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[int32]*dbOtherServerPlayerRow)
	for i, v := range this.m_new_rows {
		_, has := this.m_rows[i]
		if has {
			log.Error("rows already has new rows %v", i)
			continue
		}
		this.m_rows[i] = v
		new_rows[i] = v
	}
	for i, _ := range new_rows {
		delete(this.m_new_rows, i)
	}
	return
}
func (this *dbOtherServerPlayerTable) save_rows(rows map[int32]*dbOtherServerPlayerRow, quick bool) {
	for _, v := range rows {
		if this.m_dbc.m_quit && !quick {
			return
		}
		err, delay, _ := v.Save(false)
		if err != nil {
			log.Error("save failed %v", err)
		}
		if this.m_dbc.m_quit && !quick {
			return
		}
		if delay&&!quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
}
func (this *dbOtherServerPlayerTable) Save(quick bool) (err error){
	removed_rows := this.fetch_rows(this.m_removed_rows)
	for _, v := range removed_rows {
		_, err := this.m_dbc.StmtExec(this.m_delete_stmt, v.GetPlayerId())
		if err != nil {
			log.Error("exec delete stmt failed %v", err)
		}
		v.m_valid = false
		if !quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
	this.m_removed_rows = make(map[int32]*dbOtherServerPlayerRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbOtherServerPlayerTable) AddRow(PlayerId int32) (row *dbOtherServerPlayerRow) {
	this.m_lock.UnSafeLock("dbOtherServerPlayerTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	row = new_dbOtherServerPlayerRow(this,PlayerId)
	row.m_new = true
	row.m_loaded = true
	row.m_valid = true
	_, has := this.m_new_rows[PlayerId]
	if has{
		log.Error("已经存在 %v", PlayerId)
		return nil
	}
	this.m_new_rows[PlayerId] = row
	atomic.AddInt32(&this.m_gc_n,1)
	return row
}
func (this *dbOtherServerPlayerTable) RemoveRow(PlayerId int32) {
	this.m_lock.UnSafeLock("dbOtherServerPlayerTable.RemoveRow")
	defer this.m_lock.UnSafeUnlock()
	row := this.m_rows[PlayerId]
	if row != nil {
		row.m_remove = true
		delete(this.m_rows, PlayerId)
		rm_row := this.m_removed_rows[PlayerId]
		if rm_row != nil {
			log.Error("rows and removed rows both has %v", PlayerId)
		}
		this.m_removed_rows[PlayerId] = row
		_, has_new := this.m_new_rows[PlayerId]
		if has_new {
			delete(this.m_new_rows, PlayerId)
			log.Error("rows and new_rows both has %v", PlayerId)
		}
	} else {
		row = this.m_removed_rows[PlayerId]
		if row == nil {
			_, has_new := this.m_new_rows[PlayerId]
			if has_new {
				delete(this.m_new_rows, PlayerId)
			} else {
				log.Error("row not exist %v", PlayerId)
			}
		} else {
			log.Error("already removed %v", PlayerId)
			_, has_new := this.m_new_rows[PlayerId]
			if has_new {
				delete(this.m_new_rows, PlayerId)
				log.Error("removed rows and new_rows both has %v", PlayerId)
			}
		}
	}
}
func (this *dbOtherServerPlayerTable) GetRow(PlayerId int32) (row *dbOtherServerPlayerRow) {
	this.m_lock.UnSafeRLock("dbOtherServerPlayerTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[PlayerId]
	if row == nil {
		row = this.m_new_rows[PlayerId]
	}
	return row
}

type DBC struct {
	m_db_name            string
	m_db                 *sql.DB
	m_db_lock            *Mutex
	m_initialized        bool
	m_quit               bool
	m_shutdown_completed bool
	m_shutdown_lock      *Mutex
	m_db_last_copy_time	int32
	m_db_copy_path		string
	m_db_addr			string
	m_db_account			string
	m_db_password		string
	Global *dbGlobalTable
	Players *dbPlayerTable
	ActivitysToDeletes *dbActivitysToDeleteTable
	SysMailCommon *dbSysMailCommonTable
	SysMails *dbSysMailTable
	BanPlayers *dbBanPlayerTable
	ServerInfo *dbServerInfoTable
	OtherServerPlayers *dbOtherServerPlayerTable
}
func (this *DBC)init_tables()(err error){
	this.Global = new_dbGlobalTable(this)
	err = this.Global.Init()
	if err != nil {
		log.Error("init Global table failed")
		return
	}
	this.Players = new_dbPlayerTable(this)
	err = this.Players.Init()
	if err != nil {
		log.Error("init Players table failed")
		return
	}
	this.ActivitysToDeletes = new_dbActivitysToDeleteTable(this)
	err = this.ActivitysToDeletes.Init()
	if err != nil {
		log.Error("init ActivitysToDeletes table failed")
		return
	}
	this.SysMailCommon = new_dbSysMailCommonTable(this)
	err = this.SysMailCommon.Init()
	if err != nil {
		log.Error("init SysMailCommon table failed")
		return
	}
	this.SysMails = new_dbSysMailTable(this)
	err = this.SysMails.Init()
	if err != nil {
		log.Error("init SysMails table failed")
		return
	}
	this.BanPlayers = new_dbBanPlayerTable(this)
	err = this.BanPlayers.Init()
	if err != nil {
		log.Error("init BanPlayers table failed")
		return
	}
	this.ServerInfo = new_dbServerInfoTable(this)
	err = this.ServerInfo.Init()
	if err != nil {
		log.Error("init ServerInfo table failed")
		return
	}
	this.OtherServerPlayers = new_dbOtherServerPlayerTable(this)
	err = this.OtherServerPlayers.Init()
	if err != nil {
		log.Error("init OtherServerPlayers table failed")
		return
	}
	return
}
func (this *DBC)Preload()(err error){
	err = this.Global.Preload()
	if err != nil {
		log.Error("preload Global table failed")
		return
	}else{
		log.Info("preload Global table succeed !")
	}
	err = this.Players.Preload()
	if err != nil {
		log.Error("preload Players table failed")
		return
	}else{
		log.Info("preload Players table succeed !")
	}
	err = this.ActivitysToDeletes.Preload()
	if err != nil {
		log.Error("preload ActivitysToDeletes table failed")
		return
	}else{
		log.Info("preload ActivitysToDeletes table succeed !")
	}
	err = this.SysMailCommon.Preload()
	if err != nil {
		log.Error("preload SysMailCommon table failed")
		return
	}else{
		log.Info("preload SysMailCommon table succeed !")
	}
	err = this.SysMails.Preload()
	if err != nil {
		log.Error("preload SysMails table failed")
		return
	}else{
		log.Info("preload SysMails table succeed !")
	}
	err = this.BanPlayers.Preload()
	if err != nil {
		log.Error("preload BanPlayers table failed")
		return
	}else{
		log.Info("preload BanPlayers table succeed !")
	}
	err = this.ServerInfo.Preload()
	if err != nil {
		log.Error("preload ServerInfo table failed")
		return
	}else{
		log.Info("preload ServerInfo table succeed !")
	}
	err = this.OtherServerPlayers.Preload()
	if err != nil {
		log.Error("preload OtherServerPlayers table failed")
		return
	}else{
		log.Info("preload OtherServerPlayers table succeed !")
	}
	err = this.on_preload()
	if err != nil {
		log.Error("on_preload failed")
		return
	}
	err = this.Save(true)
	if err != nil {
		log.Error("save on preload failed")
		return
	}
	return
}
func (this *DBC)Save(quick bool)(err error){
	err = this.Global.Save(quick)
	if err != nil {
		log.Error("save Global table failed")
		return
	}
	err = this.Players.Save(quick)
	if err != nil {
		log.Error("save Players table failed")
		return
	}
	err = this.ActivitysToDeletes.Save(quick)
	if err != nil {
		log.Error("save ActivitysToDeletes table failed")
		return
	}
	err = this.SysMailCommon.Save(quick)
	if err != nil {
		log.Error("save SysMailCommon table failed")
		return
	}
	err = this.SysMails.Save(quick)
	if err != nil {
		log.Error("save SysMails table failed")
		return
	}
	err = this.BanPlayers.Save(quick)
	if err != nil {
		log.Error("save BanPlayers table failed")
		return
	}
	err = this.ServerInfo.Save(quick)
	if err != nil {
		log.Error("save ServerInfo table failed")
		return
	}
	err = this.OtherServerPlayers.Save(quick)
	if err != nil {
		log.Error("save OtherServerPlayers table failed")
		return
	}
	return
}
