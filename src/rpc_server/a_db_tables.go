package main

import (
	_ "github.com/go-sql-driver/mysql"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"mm_server/libs/log"
	"math/rand"
	"os"
	"os/exec"
	"mm_server/proto/gen_go/db_rpc"
	"strings"
	"sync/atomic"
	"time"
	
	"github.com/golang/protobuf/proto"
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
		log.Trace("db存数据花费时长: %v", time.Now().Sub(begin).Nanoseconds())

		now_time := time.Now()
		if int32(now_time.Unix()) - 24*3600 >= this.m_db_last_copy_time {
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
			this.m_db_last_copy_time = int32(now_time.Unix())
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

type dbPlayerStageTotalScoreHistoryTopDataData struct{
	Rank int32
	Score int32
}
func (this* dbPlayerStageTotalScoreHistoryTopDataData)from_pb(pb *db.PlayerStageTotalScoreHistoryTopData){
	if pb == nil {
		return
	}
	this.Rank = pb.GetRank()
	this.Score = pb.GetScore()
	return
}
func (this* dbPlayerStageTotalScoreHistoryTopDataData)to_pb()(pb *db.PlayerStageTotalScoreHistoryTopData){
	pb = &db.PlayerStageTotalScoreHistoryTopData{}
	pb.Rank = proto.Int32(this.Rank)
	pb.Score = proto.Int32(this.Score)
	return
}
func (this* dbPlayerStageTotalScoreHistoryTopDataData)clone_to(d *dbPlayerStageTotalScoreHistoryTopDataData){
	d.Rank = this.Rank
	d.Score = this.Score
	return
}
type dbPlayerStageTotalScoreStageData struct{
	Id int32
	TopScore int32
}
func (this* dbPlayerStageTotalScoreStageData)from_pb(pb *db.PlayerStageTotalScoreStage){
	if pb == nil {
		return
	}
	this.Id = pb.GetId()
	this.TopScore = pb.GetTopScore()
	return
}
func (this* dbPlayerStageTotalScoreStageData)to_pb()(pb *db.PlayerStageTotalScoreStage){
	pb = &db.PlayerStageTotalScoreStage{}
	pb.Id = proto.Int32(this.Id)
	pb.TopScore = proto.Int32(this.TopScore)
	return
}
func (this* dbPlayerStageTotalScoreStageData)clone_to(d *dbPlayerStageTotalScoreStageData){
	d.Id = this.Id
	d.TopScore = this.TopScore
	return
}
type dbPlayerCharmHistoryTopDataData struct{
	Rank int32
	Charm int32
}
func (this* dbPlayerCharmHistoryTopDataData)from_pb(pb *db.PlayerCharmHistoryTopData){
	if pb == nil {
		return
	}
	this.Rank = pb.GetRank()
	this.Charm = pb.GetCharm()
	return
}
func (this* dbPlayerCharmHistoryTopDataData)to_pb()(pb *db.PlayerCharmHistoryTopData){
	pb = &db.PlayerCharmHistoryTopData{}
	pb.Rank = proto.Int32(this.Rank)
	pb.Charm = proto.Int32(this.Charm)
	return
}
func (this* dbPlayerCharmHistoryTopDataData)clone_to(d *dbPlayerCharmHistoryTopDataData){
	d.Rank = this.Rank
	d.Charm = this.Charm
	return
}
type dbPlayerCatOuqiCatData struct{
	CatId int32
	Ouqi int32
	UpdateTime int32
	HistoryTopRank int32
}
func (this* dbPlayerCatOuqiCatData)from_pb(pb *db.PlayerCatOuqiCat){
	if pb == nil {
		return
	}
	this.CatId = pb.GetCatId()
	this.Ouqi = pb.GetOuqi()
	this.UpdateTime = pb.GetUpdateTime()
	this.HistoryTopRank = pb.GetHistoryTopRank()
	return
}
func (this* dbPlayerCatOuqiCatData)to_pb()(pb *db.PlayerCatOuqiCat){
	pb = &db.PlayerCatOuqiCat{}
	pb.CatId = proto.Int32(this.CatId)
	pb.Ouqi = proto.Int32(this.Ouqi)
	pb.UpdateTime = proto.Int32(this.UpdateTime)
	pb.HistoryTopRank = proto.Int32(this.HistoryTopRank)
	return
}
func (this* dbPlayerCatOuqiCatData)clone_to(d *dbPlayerCatOuqiCatData){
	d.CatId = this.CatId
	d.Ouqi = this.Ouqi
	d.UpdateTime = this.UpdateTime
	d.HistoryTopRank = this.HistoryTopRank
	return
}
type dbPlayerBeZanedHistoryTopDataData struct{
	Rank int32
	Zaned int32
}
func (this* dbPlayerBeZanedHistoryTopDataData)from_pb(pb *db.PlayerBeZanedHistoryTopData){
	if pb == nil {
		return
	}
	this.Rank = pb.GetRank()
	this.Zaned = pb.GetZaned()
	return
}
func (this* dbPlayerBeZanedHistoryTopDataData)to_pb()(pb *db.PlayerBeZanedHistoryTopData){
	pb = &db.PlayerBeZanedHistoryTopData{}
	pb.Rank = proto.Int32(this.Rank)
	pb.Zaned = proto.Int32(this.Zaned)
	return
}
func (this* dbPlayerBeZanedHistoryTopDataData)clone_to(d *dbPlayerBeZanedHistoryTopDataData){
	d.Rank = this.Rank
	d.Zaned = this.Zaned
	return
}

func (this *dbPlayerStageTotalScoreRow)GetScore( )(r int32 ){
	this.m_lock.UnSafeRLock("dbPlayerStageTotalScoreRow.GetdbPlayerStageTotalScoreScoreColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_Score)
}
func (this *dbPlayerStageTotalScoreRow)SetScore(v int32){
	this.m_lock.UnSafeLock("dbPlayerStageTotalScoreRow.SetdbPlayerStageTotalScoreScoreColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Score=int32(v)
	this.m_Score_changed=true
	return
}
func (this *dbPlayerStageTotalScoreRow)GetUpdateTime( )(r int32 ){
	this.m_lock.UnSafeRLock("dbPlayerStageTotalScoreRow.GetdbPlayerStageTotalScoreUpdateTimeColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_UpdateTime)
}
func (this *dbPlayerStageTotalScoreRow)SetUpdateTime(v int32){
	this.m_lock.UnSafeLock("dbPlayerStageTotalScoreRow.SetdbPlayerStageTotalScoreUpdateTimeColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_UpdateTime=int32(v)
	this.m_UpdateTime_changed=true
	return
}
type dbPlayerStageTotalScoreHistoryTopDataColumn struct{
	m_row *dbPlayerStageTotalScoreRow
	m_data *dbPlayerStageTotalScoreHistoryTopDataData
	m_changed bool
}
func (this *dbPlayerStageTotalScoreHistoryTopDataColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerStageTotalScoreHistoryTopDataData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerStageTotalScoreHistoryTopData{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerStageTotalScoreHistoryTopDataData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerStageTotalScoreHistoryTopDataColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerStageTotalScoreHistoryTopDataColumn)Get( )(v *dbPlayerStageTotalScoreHistoryTopDataData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageTotalScoreHistoryTopDataColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerStageTotalScoreHistoryTopDataData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerStageTotalScoreHistoryTopDataColumn)Set(v dbPlayerStageTotalScoreHistoryTopDataData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageTotalScoreHistoryTopDataColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerStageTotalScoreHistoryTopDataData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerStageTotalScoreHistoryTopDataColumn)GetRank( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageTotalScoreHistoryTopDataColumn.GetRank")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Rank
	return
}
func (this *dbPlayerStageTotalScoreHistoryTopDataColumn)SetRank(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageTotalScoreHistoryTopDataColumn.SetRank")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Rank = v
	this.m_changed = true
	return
}
func (this *dbPlayerStageTotalScoreHistoryTopDataColumn)GetScore( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageTotalScoreHistoryTopDataColumn.GetScore")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Score
	return
}
func (this *dbPlayerStageTotalScoreHistoryTopDataColumn)SetScore(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageTotalScoreHistoryTopDataColumn.SetScore")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Score = v
	this.m_changed = true
	return
}
type dbPlayerStageTotalScoreStageColumn struct{
	m_row *dbPlayerStageTotalScoreRow
	m_data map[int32]*dbPlayerStageTotalScoreStageData
	m_changed bool
}
func (this *dbPlayerStageTotalScoreStageColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerStageTotalScoreStageList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerStageTotalScoreStageData{}
		d.from_pb(v)
		this.m_data[int32(d.Id)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerStageTotalScoreStageColumn)save( )(data []byte,err error){
	pb := &db.PlayerStageTotalScoreStageList{}
	pb.List=make([]*db.PlayerStageTotalScoreStage,len(this.m_data))
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
func (this *dbPlayerStageTotalScoreStageColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageTotalScoreStageColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerStageTotalScoreStageColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageTotalScoreStageColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerStageTotalScoreStageColumn)GetAll()(list []dbPlayerStageTotalScoreStageData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageTotalScoreStageColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerStageTotalScoreStageData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerStageTotalScoreStageColumn)Get(id int32)(v *dbPlayerStageTotalScoreStageData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageTotalScoreStageColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerStageTotalScoreStageData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerStageTotalScoreStageColumn)Set(v dbPlayerStageTotalScoreStageData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageTotalScoreStageColumn.Set")
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
func (this *dbPlayerStageTotalScoreStageColumn)Add(v *dbPlayerStageTotalScoreStageData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageTotalScoreStageColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.Id)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.Id)
		return false
	}
	d:=&dbPlayerStageTotalScoreStageData{}
	v.clone_to(d)
	this.m_data[int32(v.Id)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerStageTotalScoreStageColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageTotalScoreStageColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerStageTotalScoreStageColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageTotalScoreStageColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerStageTotalScoreStageData)
	this.m_changed = true
	return
}
func (this *dbPlayerStageTotalScoreStageColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageTotalScoreStageColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerStageTotalScoreStageColumn)GetTopScore(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerStageTotalScoreStageColumn.GetTopScore")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.TopScore
	return v,true
}
func (this *dbPlayerStageTotalScoreStageColumn)SetTopScore(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerStageTotalScoreStageColumn.SetTopScore")
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
type dbPlayerStageTotalScoreRow struct {
	m_table *dbPlayerStageTotalScoreTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_PlayerId        int32
	m_Score_changed bool
	m_Score int32
	m_UpdateTime_changed bool
	m_UpdateTime int32
	HistoryTopData dbPlayerStageTotalScoreHistoryTopDataColumn
	Stages dbPlayerStageTotalScoreStageColumn
}
func new_dbPlayerStageTotalScoreRow(table *dbPlayerStageTotalScoreTable, PlayerId int32) (r *dbPlayerStageTotalScoreRow) {
	this := &dbPlayerStageTotalScoreRow{}
	this.m_table = table
	this.m_PlayerId = PlayerId
	this.m_lock = NewRWMutex()
	this.m_Score_changed=true
	this.m_UpdateTime_changed=true
	this.HistoryTopData.m_row=this
	this.HistoryTopData.m_data=&dbPlayerStageTotalScoreHistoryTopDataData{}
	this.Stages.m_row=this
	this.Stages.m_data=make(map[int32]*dbPlayerStageTotalScoreStageData)
	return this
}
func (this *dbPlayerStageTotalScoreRow) GetPlayerId() (r int32) {
	return this.m_PlayerId
}
func (this *dbPlayerStageTotalScoreRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbPlayerStageTotalScoreRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(5)
		db_args.Push(this.m_PlayerId)
		db_args.Push(this.m_Score)
		db_args.Push(this.m_UpdateTime)
		dHistoryTopData,db_err:=this.HistoryTopData.save()
		if db_err!=nil{
			log.Error("insert save HistoryTopData failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dHistoryTopData)
		dStages,db_err:=this.Stages.save()
		if db_err!=nil{
			log.Error("insert save Stage failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dStages)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_Score_changed||this.m_UpdateTime_changed||this.HistoryTopData.m_changed||this.Stages.m_changed{
			update_string = "UPDATE PlayerStageTotalScores SET "
			db_args:=new_db_args(5)
			if this.m_Score_changed{
				update_string+="Score=?,"
				db_args.Push(this.m_Score)
			}
			if this.m_UpdateTime_changed{
				update_string+="UpdateTime=?,"
				db_args.Push(this.m_UpdateTime)
			}
			if this.HistoryTopData.m_changed{
				update_string+="HistoryTopData=?,"
				dHistoryTopData,err:=this.HistoryTopData.save()
				if err!=nil{
					log.Error("update save HistoryTopData failed")
					return err,false,0,"",nil
				}
				db_args.Push(dHistoryTopData)
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
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE PlayerId=?"
			db_args.Push(this.m_PlayerId)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_Score_changed = false
	this.m_UpdateTime_changed = false
	this.HistoryTopData.m_changed = false
	this.Stages.m_changed = false
	if release && this.m_loaded {
		atomic.AddInt32(&this.m_table.m_gc_n, -1)
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbPlayerStageTotalScoreRow) Save(release bool) (err error, d bool, released bool) {
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
			log.Error("INSERT PlayerStageTotalScores exec failed %v ", this.m_PlayerId)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE PlayerStageTotalScores exec failed %v", this.m_PlayerId)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbPlayerStageTotalScoreRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbPlayerStageTotalScoreRowSort struct {
	rows []*dbPlayerStageTotalScoreRow
}
func (this *dbPlayerStageTotalScoreRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbPlayerStageTotalScoreRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbPlayerStageTotalScoreRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbPlayerStageTotalScoreTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[int32]*dbPlayerStageTotalScoreRow
	m_new_rows map[int32]*dbPlayerStageTotalScoreRow
	m_removed_rows map[int32]*dbPlayerStageTotalScoreRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
}
func new_dbPlayerStageTotalScoreTable(dbc *DBC) (this *dbPlayerStageTotalScoreTable) {
	this = &dbPlayerStageTotalScoreTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[int32]*dbPlayerStageTotalScoreRow)
	this.m_new_rows = make(map[int32]*dbPlayerStageTotalScoreRow)
	this.m_removed_rows = make(map[int32]*dbPlayerStageTotalScoreRow)
	return this
}
func (this *dbPlayerStageTotalScoreTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS PlayerStageTotalScores(PlayerId int(11),PRIMARY KEY (PlayerId))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS PlayerStageTotalScores failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='PlayerStageTotalScores'", this.m_dbc.m_db_name)
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
	_, hasScore := columns["Score"]
	if !hasScore {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerStageTotalScores ADD COLUMN Score int(11)")
		if err != nil {
			log.Error("ADD COLUMN Score failed")
			return
		}
	}
	_, hasUpdateTime := columns["UpdateTime"]
	if !hasUpdateTime {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerStageTotalScores ADD COLUMN UpdateTime int(11)")
		if err != nil {
			log.Error("ADD COLUMN UpdateTime failed")
			return
		}
	}
	_, hasHistoryTopData := columns["HistoryTopData"]
	if !hasHistoryTopData {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerStageTotalScores ADD COLUMN HistoryTopData LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN HistoryTopData failed")
			return
		}
	}
	_, hasStage := columns["Stages"]
	if !hasStage {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerStageTotalScores ADD COLUMN Stages LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Stages failed")
			return
		}
	}
	return
}
func (this *dbPlayerStageTotalScoreTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT PlayerId,Score,UpdateTime,HistoryTopData,Stages FROM PlayerStageTotalScores")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerStageTotalScoreTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO PlayerStageTotalScores (PlayerId,Score,UpdateTime,HistoryTopData,Stages) VALUES (?,?,?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerStageTotalScoreTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM PlayerStageTotalScores WHERE PlayerId=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerStageTotalScoreTable) Init() (err error) {
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
func (this *dbPlayerStageTotalScoreTable) Preload() (err error) {
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var PlayerId int32
	var dScore int32
	var dUpdateTime int32
	var dHistoryTopData []byte
	var dStages []byte
		this.m_preload_max_id = 0
	for r.Next() {
		err = r.Scan(&PlayerId,&dScore,&dUpdateTime,&dHistoryTopData,&dStages)
		if err != nil {
			log.Error("Scan err[%v]", err.Error())
			return
		}
		if PlayerId>this.m_preload_max_id{
			this.m_preload_max_id =PlayerId
		}
		row := new_dbPlayerStageTotalScoreRow(this,PlayerId)
		row.m_Score=dScore
		row.m_UpdateTime=dUpdateTime
		err = row.HistoryTopData.load(dHistoryTopData)
		if err != nil {
			log.Error("HistoryTopData %v", PlayerId)
			return
		}
		err = row.Stages.load(dStages)
		if err != nil {
			log.Error("Stages %v", PlayerId)
			return
		}
		row.m_Score_changed=false
		row.m_UpdateTime_changed=false
		row.m_valid = true
		this.m_rows[PlayerId]=row
	}
	return
}
func (this *dbPlayerStageTotalScoreTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbPlayerStageTotalScoreTable) fetch_rows(rows map[int32]*dbPlayerStageTotalScoreRow) (r map[int32]*dbPlayerStageTotalScoreRow) {
	this.m_lock.UnSafeLock("dbPlayerStageTotalScoreTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[int32]*dbPlayerStageTotalScoreRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbPlayerStageTotalScoreTable) fetch_new_rows() (new_rows map[int32]*dbPlayerStageTotalScoreRow) {
	this.m_lock.UnSafeLock("dbPlayerStageTotalScoreTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[int32]*dbPlayerStageTotalScoreRow)
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
func (this *dbPlayerStageTotalScoreTable) save_rows(rows map[int32]*dbPlayerStageTotalScoreRow, quick bool) {
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
func (this *dbPlayerStageTotalScoreTable) Save(quick bool) (err error){
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
	this.m_removed_rows = make(map[int32]*dbPlayerStageTotalScoreRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbPlayerStageTotalScoreTable) AddRow(PlayerId int32) (row *dbPlayerStageTotalScoreRow) {
	this.m_lock.UnSafeLock("dbPlayerStageTotalScoreTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	row = new_dbPlayerStageTotalScoreRow(this,PlayerId)
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
func (this *dbPlayerStageTotalScoreTable) RemoveRow(PlayerId int32) {
	this.m_lock.UnSafeLock("dbPlayerStageTotalScoreTable.RemoveRow")
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
func (this *dbPlayerStageTotalScoreTable) GetRow(PlayerId int32) (row *dbPlayerStageTotalScoreRow) {
	this.m_lock.UnSafeRLock("dbPlayerStageTotalScoreTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[PlayerId]
	if row == nil {
		row = this.m_new_rows[PlayerId]
	}
	return row
}
func (this *dbPlayerCharmRow)GetCharmValue( )(r int32 ){
	this.m_lock.UnSafeRLock("dbPlayerCharmRow.GetdbPlayerCharmCharmValueColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_CharmValue)
}
func (this *dbPlayerCharmRow)SetCharmValue(v int32){
	this.m_lock.UnSafeLock("dbPlayerCharmRow.SetdbPlayerCharmCharmValueColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_CharmValue=int32(v)
	this.m_CharmValue_changed=true
	return
}
func (this *dbPlayerCharmRow)GetUpdateTime( )(r int32 ){
	this.m_lock.UnSafeRLock("dbPlayerCharmRow.GetdbPlayerCharmUpdateTimeColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_UpdateTime)
}
func (this *dbPlayerCharmRow)SetUpdateTime(v int32){
	this.m_lock.UnSafeLock("dbPlayerCharmRow.SetdbPlayerCharmUpdateTimeColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_UpdateTime=int32(v)
	this.m_UpdateTime_changed=true
	return
}
type dbPlayerCharmHistoryTopDataColumn struct{
	m_row *dbPlayerCharmRow
	m_data *dbPlayerCharmHistoryTopDataData
	m_changed bool
}
func (this *dbPlayerCharmHistoryTopDataColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerCharmHistoryTopDataData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerCharmHistoryTopData{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerCharmHistoryTopDataData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerCharmHistoryTopDataColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerCharmHistoryTopDataColumn)Get( )(v *dbPlayerCharmHistoryTopDataData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCharmHistoryTopDataColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerCharmHistoryTopDataData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerCharmHistoryTopDataColumn)Set(v dbPlayerCharmHistoryTopDataData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerCharmHistoryTopDataColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerCharmHistoryTopDataData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerCharmHistoryTopDataColumn)GetRank( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCharmHistoryTopDataColumn.GetRank")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Rank
	return
}
func (this *dbPlayerCharmHistoryTopDataColumn)SetRank(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerCharmHistoryTopDataColumn.SetRank")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Rank = v
	this.m_changed = true
	return
}
func (this *dbPlayerCharmHistoryTopDataColumn)GetCharm( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCharmHistoryTopDataColumn.GetCharm")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Charm
	return
}
func (this *dbPlayerCharmHistoryTopDataColumn)SetCharm(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerCharmHistoryTopDataColumn.SetCharm")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Charm = v
	this.m_changed = true
	return
}
type dbPlayerCharmRow struct {
	m_table *dbPlayerCharmTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_PlayerId        int32
	m_CharmValue_changed bool
	m_CharmValue int32
	m_UpdateTime_changed bool
	m_UpdateTime int32
	HistoryTopData dbPlayerCharmHistoryTopDataColumn
}
func new_dbPlayerCharmRow(table *dbPlayerCharmTable, PlayerId int32) (r *dbPlayerCharmRow) {
	this := &dbPlayerCharmRow{}
	this.m_table = table
	this.m_PlayerId = PlayerId
	this.m_lock = NewRWMutex()
	this.m_CharmValue_changed=true
	this.m_UpdateTime_changed=true
	this.HistoryTopData.m_row=this
	this.HistoryTopData.m_data=&dbPlayerCharmHistoryTopDataData{}
	return this
}
func (this *dbPlayerCharmRow) GetPlayerId() (r int32) {
	return this.m_PlayerId
}
func (this *dbPlayerCharmRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbPlayerCharmRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(4)
		db_args.Push(this.m_PlayerId)
		db_args.Push(this.m_CharmValue)
		db_args.Push(this.m_UpdateTime)
		dHistoryTopData,db_err:=this.HistoryTopData.save()
		if db_err!=nil{
			log.Error("insert save HistoryTopData failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dHistoryTopData)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_CharmValue_changed||this.m_UpdateTime_changed||this.HistoryTopData.m_changed{
			update_string = "UPDATE PlayerCharms SET "
			db_args:=new_db_args(4)
			if this.m_CharmValue_changed{
				update_string+="CharmValue=?,"
				db_args.Push(this.m_CharmValue)
			}
			if this.m_UpdateTime_changed{
				update_string+="UpdateTime=?,"
				db_args.Push(this.m_UpdateTime)
			}
			if this.HistoryTopData.m_changed{
				update_string+="HistoryTopData=?,"
				dHistoryTopData,err:=this.HistoryTopData.save()
				if err!=nil{
					log.Error("update save HistoryTopData failed")
					return err,false,0,"",nil
				}
				db_args.Push(dHistoryTopData)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE PlayerId=?"
			db_args.Push(this.m_PlayerId)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_CharmValue_changed = false
	this.m_UpdateTime_changed = false
	this.HistoryTopData.m_changed = false
	if release && this.m_loaded {
		atomic.AddInt32(&this.m_table.m_gc_n, -1)
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbPlayerCharmRow) Save(release bool) (err error, d bool, released bool) {
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
			log.Error("INSERT PlayerCharms exec failed %v ", this.m_PlayerId)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE PlayerCharms exec failed %v", this.m_PlayerId)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbPlayerCharmRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbPlayerCharmRowSort struct {
	rows []*dbPlayerCharmRow
}
func (this *dbPlayerCharmRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbPlayerCharmRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbPlayerCharmRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbPlayerCharmTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[int32]*dbPlayerCharmRow
	m_new_rows map[int32]*dbPlayerCharmRow
	m_removed_rows map[int32]*dbPlayerCharmRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
}
func new_dbPlayerCharmTable(dbc *DBC) (this *dbPlayerCharmTable) {
	this = &dbPlayerCharmTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[int32]*dbPlayerCharmRow)
	this.m_new_rows = make(map[int32]*dbPlayerCharmRow)
	this.m_removed_rows = make(map[int32]*dbPlayerCharmRow)
	return this
}
func (this *dbPlayerCharmTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS PlayerCharms(PlayerId int(11),PRIMARY KEY (PlayerId))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS PlayerCharms failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='PlayerCharms'", this.m_dbc.m_db_name)
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
	_, hasCharmValue := columns["CharmValue"]
	if !hasCharmValue {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerCharms ADD COLUMN CharmValue int(11)")
		if err != nil {
			log.Error("ADD COLUMN CharmValue failed")
			return
		}
	}
	_, hasUpdateTime := columns["UpdateTime"]
	if !hasUpdateTime {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerCharms ADD COLUMN UpdateTime int(11)")
		if err != nil {
			log.Error("ADD COLUMN UpdateTime failed")
			return
		}
	}
	_, hasHistoryTopData := columns["HistoryTopData"]
	if !hasHistoryTopData {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerCharms ADD COLUMN HistoryTopData LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN HistoryTopData failed")
			return
		}
	}
	return
}
func (this *dbPlayerCharmTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT PlayerId,CharmValue,UpdateTime,HistoryTopData FROM PlayerCharms")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerCharmTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO PlayerCharms (PlayerId,CharmValue,UpdateTime,HistoryTopData) VALUES (?,?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerCharmTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM PlayerCharms WHERE PlayerId=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerCharmTable) Init() (err error) {
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
func (this *dbPlayerCharmTable) Preload() (err error) {
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var PlayerId int32
	var dCharmValue int32
	var dUpdateTime int32
	var dHistoryTopData []byte
		this.m_preload_max_id = 0
	for r.Next() {
		err = r.Scan(&PlayerId,&dCharmValue,&dUpdateTime,&dHistoryTopData)
		if err != nil {
			log.Error("Scan err[%v]", err.Error())
			return
		}
		if PlayerId>this.m_preload_max_id{
			this.m_preload_max_id =PlayerId
		}
		row := new_dbPlayerCharmRow(this,PlayerId)
		row.m_CharmValue=dCharmValue
		row.m_UpdateTime=dUpdateTime
		err = row.HistoryTopData.load(dHistoryTopData)
		if err != nil {
			log.Error("HistoryTopData %v", PlayerId)
			return
		}
		row.m_CharmValue_changed=false
		row.m_UpdateTime_changed=false
		row.m_valid = true
		this.m_rows[PlayerId]=row
	}
	return
}
func (this *dbPlayerCharmTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbPlayerCharmTable) fetch_rows(rows map[int32]*dbPlayerCharmRow) (r map[int32]*dbPlayerCharmRow) {
	this.m_lock.UnSafeLock("dbPlayerCharmTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[int32]*dbPlayerCharmRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbPlayerCharmTable) fetch_new_rows() (new_rows map[int32]*dbPlayerCharmRow) {
	this.m_lock.UnSafeLock("dbPlayerCharmTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[int32]*dbPlayerCharmRow)
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
func (this *dbPlayerCharmTable) save_rows(rows map[int32]*dbPlayerCharmRow, quick bool) {
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
func (this *dbPlayerCharmTable) Save(quick bool) (err error){
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
	this.m_removed_rows = make(map[int32]*dbPlayerCharmRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbPlayerCharmTable) AddRow(PlayerId int32) (row *dbPlayerCharmRow) {
	this.m_lock.UnSafeLock("dbPlayerCharmTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	row = new_dbPlayerCharmRow(this,PlayerId)
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
func (this *dbPlayerCharmTable) RemoveRow(PlayerId int32) {
	this.m_lock.UnSafeLock("dbPlayerCharmTable.RemoveRow")
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
func (this *dbPlayerCharmTable) GetRow(PlayerId int32) (row *dbPlayerCharmRow) {
	this.m_lock.UnSafeRLock("dbPlayerCharmTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[PlayerId]
	if row == nil {
		row = this.m_new_rows[PlayerId]
	}
	return row
}
type dbPlayerCatOuqiCatColumn struct{
	m_row *dbPlayerCatOuqiRow
	m_data map[int32]*dbPlayerCatOuqiCatData
	m_changed bool
}
func (this *dbPlayerCatOuqiCatColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerCatOuqiCatList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	for _, v := range pb.List {
		d := &dbPlayerCatOuqiCatData{}
		d.from_pb(v)
		this.m_data[int32(d.CatId)] = d
	}
	this.m_changed = false
	return
}
func (this *dbPlayerCatOuqiCatColumn)save( )(data []byte,err error){
	pb := &db.PlayerCatOuqiCatList{}
	pb.List=make([]*db.PlayerCatOuqiCat,len(this.m_data))
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
func (this *dbPlayerCatOuqiCatColumn)HasIndex(id int32)(has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatOuqiCatColumn.HasIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	_, has = this.m_data[id]
	return
}
func (this *dbPlayerCatOuqiCatColumn)GetAllIndex()(list []int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatOuqiCatColumn.GetAllIndex")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]int32, len(this.m_data))
	i := 0
	for k, _ := range this.m_data {
		list[i] = k
		i++
	}
	return
}
func (this *dbPlayerCatOuqiCatColumn)GetAll()(list []dbPlayerCatOuqiCatData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatOuqiCatColumn.GetAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	list = make([]dbPlayerCatOuqiCatData, len(this.m_data))
	i := 0
	for _, v := range this.m_data {
		v.clone_to(&list[i])
		i++
	}
	return
}
func (this *dbPlayerCatOuqiCatColumn)Get(id int32)(v *dbPlayerCatOuqiCatData){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatOuqiCatColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return nil
	}
	v=&dbPlayerCatOuqiCatData{}
	d.clone_to(v)
	return
}
func (this *dbPlayerCatOuqiCatColumn)Set(v dbPlayerCatOuqiCatData)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatOuqiCatColumn.Set")
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
func (this *dbPlayerCatOuqiCatColumn)Add(v *dbPlayerCatOuqiCatData)(ok bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatOuqiCatColumn.Add")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[int32(v.CatId)]
	if has {
		log.Error("already added %v %v",this.m_row.GetPlayerId(), v.CatId)
		return false
	}
	d:=&dbPlayerCatOuqiCatData{}
	v.clone_to(d)
	this.m_data[int32(v.CatId)]=d
	this.m_changed = true
	return true
}
func (this *dbPlayerCatOuqiCatColumn)Remove(id int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatOuqiCatColumn.Remove")
	defer this.m_row.m_lock.UnSafeUnlock()
	_, has := this.m_data[id]
	if has {
		delete(this.m_data,id)
	}
	this.m_changed = true
	return
}
func (this *dbPlayerCatOuqiCatColumn)Clear(){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatOuqiCatColumn.Clear")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=make(map[int32]*dbPlayerCatOuqiCatData)
	this.m_changed = true
	return
}
func (this *dbPlayerCatOuqiCatColumn)NumAll()(n int32){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatOuqiCatColumn.NumAll")
	defer this.m_row.m_lock.UnSafeRUnlock()
	return int32(len(this.m_data))
}
func (this *dbPlayerCatOuqiCatColumn)GetOuqi(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatOuqiCatColumn.GetOuqi")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.Ouqi
	return v,true
}
func (this *dbPlayerCatOuqiCatColumn)SetOuqi(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatOuqiCatColumn.SetOuqi")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.Ouqi = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatOuqiCatColumn)GetUpdateTime(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatOuqiCatColumn.GetUpdateTime")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.UpdateTime
	return v,true
}
func (this *dbPlayerCatOuqiCatColumn)SetUpdateTime(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatOuqiCatColumn.SetUpdateTime")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.UpdateTime = v
	this.m_changed = true
	return true
}
func (this *dbPlayerCatOuqiCatColumn)GetHistoryTopRank(id int32)(v int32 ,has bool){
	this.m_row.m_lock.UnSafeRLock("dbPlayerCatOuqiCatColumn.GetHistoryTopRank")
	defer this.m_row.m_lock.UnSafeRUnlock()
	d := this.m_data[id]
	if d==nil{
		return
	}
	v = d.HistoryTopRank
	return v,true
}
func (this *dbPlayerCatOuqiCatColumn)SetHistoryTopRank(id int32,v int32)(has bool){
	this.m_row.m_lock.UnSafeLock("dbPlayerCatOuqiCatColumn.SetHistoryTopRank")
	defer this.m_row.m_lock.UnSafeUnlock()
	d := this.m_data[id]
	if d==nil{
		log.Error("not exist %v %v",this.m_row.GetPlayerId(), id)
		return
	}
	d.HistoryTopRank = v
	this.m_changed = true
	return true
}
type dbPlayerCatOuqiRow struct {
	m_table *dbPlayerCatOuqiTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_PlayerId        int32
	Cats dbPlayerCatOuqiCatColumn
}
func new_dbPlayerCatOuqiRow(table *dbPlayerCatOuqiTable, PlayerId int32) (r *dbPlayerCatOuqiRow) {
	this := &dbPlayerCatOuqiRow{}
	this.m_table = table
	this.m_PlayerId = PlayerId
	this.m_lock = NewRWMutex()
	this.Cats.m_row=this
	this.Cats.m_data=make(map[int32]*dbPlayerCatOuqiCatData)
	return this
}
func (this *dbPlayerCatOuqiRow) GetPlayerId() (r int32) {
	return this.m_PlayerId
}
func (this *dbPlayerCatOuqiRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbPlayerCatOuqiRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(2)
		db_args.Push(this.m_PlayerId)
		dCats,db_err:=this.Cats.save()
		if db_err!=nil{
			log.Error("insert save Cat failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dCats)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.Cats.m_changed{
			update_string = "UPDATE PlayerCatOuqis SET "
			db_args:=new_db_args(2)
			if this.Cats.m_changed{
				update_string+="Cats=?,"
				dCats,err:=this.Cats.save()
				if err!=nil{
					log.Error("insert save Cat failed")
					return err,false,0,"",nil
				}
				db_args.Push(dCats)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE PlayerId=?"
			db_args.Push(this.m_PlayerId)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.Cats.m_changed = false
	if release && this.m_loaded {
		atomic.AddInt32(&this.m_table.m_gc_n, -1)
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbPlayerCatOuqiRow) Save(release bool) (err error, d bool, released bool) {
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
			log.Error("INSERT PlayerCatOuqis exec failed %v ", this.m_PlayerId)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE PlayerCatOuqis exec failed %v", this.m_PlayerId)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbPlayerCatOuqiRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbPlayerCatOuqiRowSort struct {
	rows []*dbPlayerCatOuqiRow
}
func (this *dbPlayerCatOuqiRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbPlayerCatOuqiRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbPlayerCatOuqiRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbPlayerCatOuqiTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[int32]*dbPlayerCatOuqiRow
	m_new_rows map[int32]*dbPlayerCatOuqiRow
	m_removed_rows map[int32]*dbPlayerCatOuqiRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
}
func new_dbPlayerCatOuqiTable(dbc *DBC) (this *dbPlayerCatOuqiTable) {
	this = &dbPlayerCatOuqiTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[int32]*dbPlayerCatOuqiRow)
	this.m_new_rows = make(map[int32]*dbPlayerCatOuqiRow)
	this.m_removed_rows = make(map[int32]*dbPlayerCatOuqiRow)
	return this
}
func (this *dbPlayerCatOuqiTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS PlayerCatOuqis(PlayerId int(11),PRIMARY KEY (PlayerId))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS PlayerCatOuqis failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='PlayerCatOuqis'", this.m_dbc.m_db_name)
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
	_, hasCat := columns["Cats"]
	if !hasCat {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerCatOuqis ADD COLUMN Cats LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN Cats failed")
			return
		}
	}
	return
}
func (this *dbPlayerCatOuqiTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT PlayerId,Cats FROM PlayerCatOuqis")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerCatOuqiTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO PlayerCatOuqis (PlayerId,Cats) VALUES (?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerCatOuqiTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM PlayerCatOuqis WHERE PlayerId=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerCatOuqiTable) Init() (err error) {
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
func (this *dbPlayerCatOuqiTable) Preload() (err error) {
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var PlayerId int32
	var dCats []byte
		this.m_preload_max_id = 0
	for r.Next() {
		err = r.Scan(&PlayerId,&dCats)
		if err != nil {
			log.Error("Scan err[%v]", err.Error())
			return
		}
		if PlayerId>this.m_preload_max_id{
			this.m_preload_max_id =PlayerId
		}
		row := new_dbPlayerCatOuqiRow(this,PlayerId)
		err = row.Cats.load(dCats)
		if err != nil {
			log.Error("Cats %v", PlayerId)
			return
		}
		row.m_valid = true
		this.m_rows[PlayerId]=row
	}
	return
}
func (this *dbPlayerCatOuqiTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbPlayerCatOuqiTable) fetch_rows(rows map[int32]*dbPlayerCatOuqiRow) (r map[int32]*dbPlayerCatOuqiRow) {
	this.m_lock.UnSafeLock("dbPlayerCatOuqiTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[int32]*dbPlayerCatOuqiRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbPlayerCatOuqiTable) fetch_new_rows() (new_rows map[int32]*dbPlayerCatOuqiRow) {
	this.m_lock.UnSafeLock("dbPlayerCatOuqiTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[int32]*dbPlayerCatOuqiRow)
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
func (this *dbPlayerCatOuqiTable) save_rows(rows map[int32]*dbPlayerCatOuqiRow, quick bool) {
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
func (this *dbPlayerCatOuqiTable) Save(quick bool) (err error){
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
	this.m_removed_rows = make(map[int32]*dbPlayerCatOuqiRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbPlayerCatOuqiTable) AddRow(PlayerId int32) (row *dbPlayerCatOuqiRow) {
	this.m_lock.UnSafeLock("dbPlayerCatOuqiTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	row = new_dbPlayerCatOuqiRow(this,PlayerId)
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
func (this *dbPlayerCatOuqiTable) RemoveRow(PlayerId int32) {
	this.m_lock.UnSafeLock("dbPlayerCatOuqiTable.RemoveRow")
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
func (this *dbPlayerCatOuqiTable) GetRow(PlayerId int32) (row *dbPlayerCatOuqiRow) {
	this.m_lock.UnSafeRLock("dbPlayerCatOuqiTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[PlayerId]
	if row == nil {
		row = this.m_new_rows[PlayerId]
	}
	return row
}
func (this *dbPlayerBeZanedRow)GetZaned( )(r int32 ){
	this.m_lock.UnSafeRLock("dbPlayerBeZanedRow.GetdbPlayerBeZanedZanedColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_Zaned)
}
func (this *dbPlayerBeZanedRow)SetZaned(v int32){
	this.m_lock.UnSafeLock("dbPlayerBeZanedRow.SetdbPlayerBeZanedZanedColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Zaned=int32(v)
	this.m_Zaned_changed=true
	return
}
func (this *dbPlayerBeZanedRow)GetUpdateTime( )(r int32 ){
	this.m_lock.UnSafeRLock("dbPlayerBeZanedRow.GetdbPlayerBeZanedUpdateTimeColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_UpdateTime)
}
func (this *dbPlayerBeZanedRow)SetUpdateTime(v int32){
	this.m_lock.UnSafeLock("dbPlayerBeZanedRow.SetdbPlayerBeZanedUpdateTimeColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_UpdateTime=int32(v)
	this.m_UpdateTime_changed=true
	return
}
type dbPlayerBeZanedHistoryTopDataColumn struct{
	m_row *dbPlayerBeZanedRow
	m_data *dbPlayerBeZanedHistoryTopDataData
	m_changed bool
}
func (this *dbPlayerBeZanedHistoryTopDataColumn)load(data []byte)(err error){
	if data == nil || len(data) == 0 {
		this.m_data = &dbPlayerBeZanedHistoryTopDataData{}
		this.m_changed = false
		return nil
	}
	pb := &db.PlayerBeZanedHistoryTopData{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		log.Error("Unmarshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_data = &dbPlayerBeZanedHistoryTopDataData{}
	this.m_data.from_pb(pb)
	this.m_changed = false
	return
}
func (this *dbPlayerBeZanedHistoryTopDataColumn)save( )(data []byte,err error){
	pb:=this.m_data.to_pb()
	data, err = proto.Marshal(pb)
	if err != nil {
		log.Error("Marshal %v", this.m_row.GetPlayerId())
		return
	}
	this.m_changed = false
	return
}
func (this *dbPlayerBeZanedHistoryTopDataColumn)Get( )(v *dbPlayerBeZanedHistoryTopDataData ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBeZanedHistoryTopDataColumn.Get")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v=&dbPlayerBeZanedHistoryTopDataData{}
	this.m_data.clone_to(v)
	return
}
func (this *dbPlayerBeZanedHistoryTopDataColumn)Set(v dbPlayerBeZanedHistoryTopDataData ){
	this.m_row.m_lock.UnSafeLock("dbPlayerBeZanedHistoryTopDataColumn.Set")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data=&dbPlayerBeZanedHistoryTopDataData{}
	v.clone_to(this.m_data)
	this.m_changed=true
	return
}
func (this *dbPlayerBeZanedHistoryTopDataColumn)GetRank( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBeZanedHistoryTopDataColumn.GetRank")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Rank
	return
}
func (this *dbPlayerBeZanedHistoryTopDataColumn)SetRank(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerBeZanedHistoryTopDataColumn.SetRank")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Rank = v
	this.m_changed = true
	return
}
func (this *dbPlayerBeZanedHistoryTopDataColumn)GetZaned( )(v int32 ){
	this.m_row.m_lock.UnSafeRLock("dbPlayerBeZanedHistoryTopDataColumn.GetZaned")
	defer this.m_row.m_lock.UnSafeRUnlock()
	v = this.m_data.Zaned
	return
}
func (this *dbPlayerBeZanedHistoryTopDataColumn)SetZaned(v int32){
	this.m_row.m_lock.UnSafeLock("dbPlayerBeZanedHistoryTopDataColumn.SetZaned")
	defer this.m_row.m_lock.UnSafeUnlock()
	this.m_data.Zaned = v
	this.m_changed = true
	return
}
type dbPlayerBeZanedRow struct {
	m_table *dbPlayerBeZanedTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_PlayerId        int32
	m_Zaned_changed bool
	m_Zaned int32
	m_UpdateTime_changed bool
	m_UpdateTime int32
	HistoryTopData dbPlayerBeZanedHistoryTopDataColumn
}
func new_dbPlayerBeZanedRow(table *dbPlayerBeZanedTable, PlayerId int32) (r *dbPlayerBeZanedRow) {
	this := &dbPlayerBeZanedRow{}
	this.m_table = table
	this.m_PlayerId = PlayerId
	this.m_lock = NewRWMutex()
	this.m_Zaned_changed=true
	this.m_UpdateTime_changed=true
	this.HistoryTopData.m_row=this
	this.HistoryTopData.m_data=&dbPlayerBeZanedHistoryTopDataData{}
	return this
}
func (this *dbPlayerBeZanedRow) GetPlayerId() (r int32) {
	return this.m_PlayerId
}
func (this *dbPlayerBeZanedRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbPlayerBeZanedRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(4)
		db_args.Push(this.m_PlayerId)
		db_args.Push(this.m_Zaned)
		db_args.Push(this.m_UpdateTime)
		dHistoryTopData,db_err:=this.HistoryTopData.save()
		if db_err!=nil{
			log.Error("insert save HistoryTopData failed")
			return db_err,false,0,"",nil
		}
		db_args.Push(dHistoryTopData)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_Zaned_changed||this.m_UpdateTime_changed||this.HistoryTopData.m_changed{
			update_string = "UPDATE PlayerBeZaneds SET "
			db_args:=new_db_args(4)
			if this.m_Zaned_changed{
				update_string+="Zaned=?,"
				db_args.Push(this.m_Zaned)
			}
			if this.m_UpdateTime_changed{
				update_string+="UpdateTime=?,"
				db_args.Push(this.m_UpdateTime)
			}
			if this.HistoryTopData.m_changed{
				update_string+="HistoryTopData=?,"
				dHistoryTopData,err:=this.HistoryTopData.save()
				if err!=nil{
					log.Error("update save HistoryTopData failed")
					return err,false,0,"",nil
				}
				db_args.Push(dHistoryTopData)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE PlayerId=?"
			db_args.Push(this.m_PlayerId)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_Zaned_changed = false
	this.m_UpdateTime_changed = false
	this.HistoryTopData.m_changed = false
	if release && this.m_loaded {
		atomic.AddInt32(&this.m_table.m_gc_n, -1)
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbPlayerBeZanedRow) Save(release bool) (err error, d bool, released bool) {
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
			log.Error("INSERT PlayerBeZaneds exec failed %v ", this.m_PlayerId)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE PlayerBeZaneds exec failed %v", this.m_PlayerId)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbPlayerBeZanedRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbPlayerBeZanedRowSort struct {
	rows []*dbPlayerBeZanedRow
}
func (this *dbPlayerBeZanedRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbPlayerBeZanedRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbPlayerBeZanedRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbPlayerBeZanedTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[int32]*dbPlayerBeZanedRow
	m_new_rows map[int32]*dbPlayerBeZanedRow
	m_removed_rows map[int32]*dbPlayerBeZanedRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
}
func new_dbPlayerBeZanedTable(dbc *DBC) (this *dbPlayerBeZanedTable) {
	this = &dbPlayerBeZanedTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[int32]*dbPlayerBeZanedRow)
	this.m_new_rows = make(map[int32]*dbPlayerBeZanedRow)
	this.m_removed_rows = make(map[int32]*dbPlayerBeZanedRow)
	return this
}
func (this *dbPlayerBeZanedTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS PlayerBeZaneds(PlayerId int(11),PRIMARY KEY (PlayerId))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS PlayerBeZaneds failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='PlayerBeZaneds'", this.m_dbc.m_db_name)
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
	_, hasZaned := columns["Zaned"]
	if !hasZaned {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerBeZaneds ADD COLUMN Zaned int(11)")
		if err != nil {
			log.Error("ADD COLUMN Zaned failed")
			return
		}
	}
	_, hasUpdateTime := columns["UpdateTime"]
	if !hasUpdateTime {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerBeZaneds ADD COLUMN UpdateTime int(11)")
		if err != nil {
			log.Error("ADD COLUMN UpdateTime failed")
			return
		}
	}
	_, hasHistoryTopData := columns["HistoryTopData"]
	if !hasHistoryTopData {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerBeZaneds ADD COLUMN HistoryTopData LONGBLOB")
		if err != nil {
			log.Error("ADD COLUMN HistoryTopData failed")
			return
		}
	}
	return
}
func (this *dbPlayerBeZanedTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT PlayerId,Zaned,UpdateTime,HistoryTopData FROM PlayerBeZaneds")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerBeZanedTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO PlayerBeZaneds (PlayerId,Zaned,UpdateTime,HistoryTopData) VALUES (?,?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerBeZanedTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM PlayerBeZaneds WHERE PlayerId=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerBeZanedTable) Init() (err error) {
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
func (this *dbPlayerBeZanedTable) Preload() (err error) {
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var PlayerId int32
	var dZaned int32
	var dUpdateTime int32
	var dHistoryTopData []byte
		this.m_preload_max_id = 0
	for r.Next() {
		err = r.Scan(&PlayerId,&dZaned,&dUpdateTime,&dHistoryTopData)
		if err != nil {
			log.Error("Scan err[%v]", err.Error())
			return
		}
		if PlayerId>this.m_preload_max_id{
			this.m_preload_max_id =PlayerId
		}
		row := new_dbPlayerBeZanedRow(this,PlayerId)
		row.m_Zaned=dZaned
		row.m_UpdateTime=dUpdateTime
		err = row.HistoryTopData.load(dHistoryTopData)
		if err != nil {
			log.Error("HistoryTopData %v", PlayerId)
			return
		}
		row.m_Zaned_changed=false
		row.m_UpdateTime_changed=false
		row.m_valid = true
		this.m_rows[PlayerId]=row
	}
	return
}
func (this *dbPlayerBeZanedTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbPlayerBeZanedTable) fetch_rows(rows map[int32]*dbPlayerBeZanedRow) (r map[int32]*dbPlayerBeZanedRow) {
	this.m_lock.UnSafeLock("dbPlayerBeZanedTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[int32]*dbPlayerBeZanedRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbPlayerBeZanedTable) fetch_new_rows() (new_rows map[int32]*dbPlayerBeZanedRow) {
	this.m_lock.UnSafeLock("dbPlayerBeZanedTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[int32]*dbPlayerBeZanedRow)
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
func (this *dbPlayerBeZanedTable) save_rows(rows map[int32]*dbPlayerBeZanedRow, quick bool) {
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
func (this *dbPlayerBeZanedTable) Save(quick bool) (err error){
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
	this.m_removed_rows = make(map[int32]*dbPlayerBeZanedRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbPlayerBeZanedTable) AddRow(PlayerId int32) (row *dbPlayerBeZanedRow) {
	this.m_lock.UnSafeLock("dbPlayerBeZanedTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	row = new_dbPlayerBeZanedRow(this,PlayerId)
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
func (this *dbPlayerBeZanedTable) RemoveRow(PlayerId int32) {
	this.m_lock.UnSafeLock("dbPlayerBeZanedTable.RemoveRow")
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
func (this *dbPlayerBeZanedTable) GetRow(PlayerId int32) (row *dbPlayerBeZanedRow) {
	this.m_lock.UnSafeRLock("dbPlayerBeZanedTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[PlayerId]
	if row == nil {
		row = this.m_new_rows[PlayerId]
	}
	return row
}
func (this *dbPlayerBaseInfoRow)GetAccount( )(r string ){
	this.m_lock.UnSafeRLock("dbPlayerBaseInfoRow.GetdbPlayerBaseInfoAccountColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_Account)
}
func (this *dbPlayerBaseInfoRow)SetAccount(v string){
	this.m_lock.UnSafeLock("dbPlayerBaseInfoRow.SetdbPlayerBaseInfoAccountColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Account=string(v)
	this.m_Account_changed=true
	return
}
func (this *dbPlayerBaseInfoRow)GetName( )(r string ){
	this.m_lock.UnSafeRLock("dbPlayerBaseInfoRow.GetdbPlayerBaseInfoNameColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_Name)
}
func (this *dbPlayerBaseInfoRow)SetName(v string){
	this.m_lock.UnSafeLock("dbPlayerBaseInfoRow.SetdbPlayerBaseInfoNameColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Name=string(v)
	this.m_Name_changed=true
	return
}
func (this *dbPlayerBaseInfoRow)GetLevel( )(r int32 ){
	this.m_lock.UnSafeRLock("dbPlayerBaseInfoRow.GetdbPlayerBaseInfoLevelColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_Level)
}
func (this *dbPlayerBaseInfoRow)SetLevel(v int32){
	this.m_lock.UnSafeLock("dbPlayerBaseInfoRow.SetdbPlayerBaseInfoLevelColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Level=int32(v)
	this.m_Level_changed=true
	return
}
func (this *dbPlayerBaseInfoRow)GetHead( )(r int32 ){
	this.m_lock.UnSafeRLock("dbPlayerBaseInfoRow.GetdbPlayerBaseInfoHeadColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_Head)
}
func (this *dbPlayerBaseInfoRow)SetHead(v int32){
	this.m_lock.UnSafeLock("dbPlayerBaseInfoRow.SetdbPlayerBaseInfoHeadColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Head=int32(v)
	this.m_Head_changed=true
	return
}
type dbPlayerBaseInfoRow struct {
	m_table *dbPlayerBaseInfoTable
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
	m_Head int32
}
func new_dbPlayerBaseInfoRow(table *dbPlayerBaseInfoTable, PlayerId int32) (r *dbPlayerBaseInfoRow) {
	this := &dbPlayerBaseInfoRow{}
	this.m_table = table
	this.m_PlayerId = PlayerId
	this.m_lock = NewRWMutex()
	this.m_Account_changed=true
	this.m_Name_changed=true
	this.m_Level_changed=true
	this.m_Head_changed=true
	return this
}
func (this *dbPlayerBaseInfoRow) GetPlayerId() (r int32) {
	return this.m_PlayerId
}
func (this *dbPlayerBaseInfoRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbPlayerBaseInfoRow.save_data")
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
			update_string = "UPDATE PlayerBaseInfos SET "
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
func (this *dbPlayerBaseInfoRow) Save(release bool) (err error, d bool, released bool) {
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
			log.Error("INSERT PlayerBaseInfos exec failed %v ", this.m_PlayerId)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE PlayerBaseInfos exec failed %v", this.m_PlayerId)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbPlayerBaseInfoRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbPlayerBaseInfoRowSort struct {
	rows []*dbPlayerBaseInfoRow
}
func (this *dbPlayerBaseInfoRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbPlayerBaseInfoRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbPlayerBaseInfoRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbPlayerBaseInfoTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[int32]*dbPlayerBaseInfoRow
	m_new_rows map[int32]*dbPlayerBaseInfoRow
	m_removed_rows map[int32]*dbPlayerBaseInfoRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
}
func new_dbPlayerBaseInfoTable(dbc *DBC) (this *dbPlayerBaseInfoTable) {
	this = &dbPlayerBaseInfoTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[int32]*dbPlayerBaseInfoRow)
	this.m_new_rows = make(map[int32]*dbPlayerBaseInfoRow)
	this.m_removed_rows = make(map[int32]*dbPlayerBaseInfoRow)
	return this
}
func (this *dbPlayerBaseInfoTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS PlayerBaseInfos(PlayerId int(11),PRIMARY KEY (PlayerId))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS PlayerBaseInfos failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='PlayerBaseInfos'", this.m_dbc.m_db_name)
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
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerBaseInfos ADD COLUMN Account varchar(256)")
		if err != nil {
			log.Error("ADD COLUMN Account failed")
			return
		}
	}
	_, hasName := columns["Name"]
	if !hasName {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerBaseInfos ADD COLUMN Name varchar(256)")
		if err != nil {
			log.Error("ADD COLUMN Name failed")
			return
		}
	}
	_, hasLevel := columns["Level"]
	if !hasLevel {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerBaseInfos ADD COLUMN Level int(11)")
		if err != nil {
			log.Error("ADD COLUMN Level failed")
			return
		}
	}
	_, hasHead := columns["Head"]
	if !hasHead {
		_, err = this.m_dbc.Exec("ALTER TABLE PlayerBaseInfos ADD COLUMN Head int(11)")
		if err != nil {
			log.Error("ADD COLUMN Head failed")
			return
		}
	}
	return
}
func (this *dbPlayerBaseInfoTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT PlayerId,Account,Name,Level,Head FROM PlayerBaseInfos")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerBaseInfoTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO PlayerBaseInfos (PlayerId,Account,Name,Level,Head) VALUES (?,?,?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerBaseInfoTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM PlayerBaseInfos WHERE PlayerId=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbPlayerBaseInfoTable) Init() (err error) {
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
func (this *dbPlayerBaseInfoTable) Preload() (err error) {
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var PlayerId int32
	var dAccount string
	var dName string
	var dLevel int32
	var dHead int32
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
		row := new_dbPlayerBaseInfoRow(this,PlayerId)
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
func (this *dbPlayerBaseInfoTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbPlayerBaseInfoTable) fetch_rows(rows map[int32]*dbPlayerBaseInfoRow) (r map[int32]*dbPlayerBaseInfoRow) {
	this.m_lock.UnSafeLock("dbPlayerBaseInfoTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[int32]*dbPlayerBaseInfoRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbPlayerBaseInfoTable) fetch_new_rows() (new_rows map[int32]*dbPlayerBaseInfoRow) {
	this.m_lock.UnSafeLock("dbPlayerBaseInfoTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[int32]*dbPlayerBaseInfoRow)
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
func (this *dbPlayerBaseInfoTable) save_rows(rows map[int32]*dbPlayerBaseInfoRow, quick bool) {
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
func (this *dbPlayerBaseInfoTable) Save(quick bool) (err error){
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
	this.m_removed_rows = make(map[int32]*dbPlayerBaseInfoRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbPlayerBaseInfoTable) AddRow(PlayerId int32) (row *dbPlayerBaseInfoRow) {
	this.m_lock.UnSafeLock("dbPlayerBaseInfoTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	row = new_dbPlayerBaseInfoRow(this,PlayerId)
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
func (this *dbPlayerBaseInfoTable) RemoveRow(PlayerId int32) {
	this.m_lock.UnSafeLock("dbPlayerBaseInfoTable.RemoveRow")
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
func (this *dbPlayerBaseInfoTable) GetRow(PlayerId int32) (row *dbPlayerBaseInfoRow) {
	this.m_lock.UnSafeRLock("dbPlayerBaseInfoTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[PlayerId]
	if row == nil {
		row = this.m_new_rows[PlayerId]
	}
	return row
}
func (this *dbApplePayRow)GetBundleId( )(r string ){
	this.m_lock.UnSafeRLock("dbApplePayRow.GetdbApplePayBundleIdColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_BundleId)
}
func (this *dbApplePayRow)SetBundleId(v string){
	this.m_lock.UnSafeLock("dbApplePayRow.SetdbApplePayBundleIdColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_BundleId=string(v)
	this.m_BundleId_changed=true
	return
}
func (this *dbApplePayRow)GetAccount( )(r string ){
	this.m_lock.UnSafeRLock("dbApplePayRow.GetdbApplePayAccountColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_Account)
}
func (this *dbApplePayRow)SetAccount(v string){
	this.m_lock.UnSafeLock("dbApplePayRow.SetdbApplePayAccountColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Account=string(v)
	this.m_Account_changed=true
	return
}
func (this *dbApplePayRow)GetPlayerId( )(r int32 ){
	this.m_lock.UnSafeRLock("dbApplePayRow.GetdbApplePayPlayerIdColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_PlayerId)
}
func (this *dbApplePayRow)SetPlayerId(v int32){
	this.m_lock.UnSafeLock("dbApplePayRow.SetdbApplePayPlayerIdColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_PlayerId=int32(v)
	this.m_PlayerId_changed=true
	return
}
func (this *dbApplePayRow)GetPayTime( )(r int32 ){
	this.m_lock.UnSafeRLock("dbApplePayRow.GetdbApplePayPayTimeColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_PayTime)
}
func (this *dbApplePayRow)SetPayTime(v int32){
	this.m_lock.UnSafeLock("dbApplePayRow.SetdbApplePayPayTimeColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_PayTime=int32(v)
	this.m_PayTime_changed=true
	return
}
func (this *dbApplePayRow)GetPayTimeStr( )(r string ){
	this.m_lock.UnSafeRLock("dbApplePayRow.GetdbApplePayPayTimeStrColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_PayTimeStr)
}
func (this *dbApplePayRow)SetPayTimeStr(v string){
	this.m_lock.UnSafeLock("dbApplePayRow.SetdbApplePayPayTimeStrColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_PayTimeStr=string(v)
	this.m_PayTimeStr_changed=true
	return
}
type dbApplePayRow struct {
	m_table *dbApplePayTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_OrderId        string
	m_BundleId_changed bool
	m_BundleId string
	m_Account_changed bool
	m_Account string
	m_PlayerId_changed bool
	m_PlayerId int32
	m_PayTime_changed bool
	m_PayTime int32
	m_PayTimeStr_changed bool
	m_PayTimeStr string
}
func new_dbApplePayRow(table *dbApplePayTable, OrderId string) (r *dbApplePayRow) {
	this := &dbApplePayRow{}
	this.m_table = table
	this.m_OrderId = OrderId
	this.m_lock = NewRWMutex()
	this.m_BundleId_changed=true
	this.m_Account_changed=true
	this.m_PlayerId_changed=true
	this.m_PayTime_changed=true
	this.m_PayTimeStr_changed=true
	return this
}
func (this *dbApplePayRow) GetOrderId() (r string) {
	return this.m_OrderId
}
func (this *dbApplePayRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbApplePayRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(6)
		db_args.Push(this.m_OrderId)
		db_args.Push(this.m_BundleId)
		db_args.Push(this.m_Account)
		db_args.Push(this.m_PlayerId)
		db_args.Push(this.m_PayTime)
		db_args.Push(this.m_PayTimeStr)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_BundleId_changed||this.m_Account_changed||this.m_PlayerId_changed||this.m_PayTime_changed||this.m_PayTimeStr_changed{
			update_string = "UPDATE ApplePays SET "
			db_args:=new_db_args(6)
			if this.m_BundleId_changed{
				update_string+="BundleId=?,"
				db_args.Push(this.m_BundleId)
			}
			if this.m_Account_changed{
				update_string+="Account=?,"
				db_args.Push(this.m_Account)
			}
			if this.m_PlayerId_changed{
				update_string+="PlayerId=?,"
				db_args.Push(this.m_PlayerId)
			}
			if this.m_PayTime_changed{
				update_string+="PayTime=?,"
				db_args.Push(this.m_PayTime)
			}
			if this.m_PayTimeStr_changed{
				update_string+="PayTimeStr=?,"
				db_args.Push(this.m_PayTimeStr)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE OrderId=?"
			db_args.Push(this.m_OrderId)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_BundleId_changed = false
	this.m_Account_changed = false
	this.m_PlayerId_changed = false
	this.m_PayTime_changed = false
	this.m_PayTimeStr_changed = false
	if release && this.m_loaded {
		atomic.AddInt32(&this.m_table.m_gc_n, -1)
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbApplePayRow) Save(release bool) (err error, d bool, released bool) {
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
			log.Error("INSERT ApplePays exec failed %v ", this.m_OrderId)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE ApplePays exec failed %v", this.m_OrderId)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbApplePayRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbApplePayRowSort struct {
	rows []*dbApplePayRow
}
func (this *dbApplePayRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbApplePayRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbApplePayRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbApplePayTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[string]*dbApplePayRow
	m_new_rows map[string]*dbApplePayRow
	m_removed_rows map[string]*dbApplePayRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
}
func new_dbApplePayTable(dbc *DBC) (this *dbApplePayTable) {
	this = &dbApplePayTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[string]*dbApplePayRow)
	this.m_new_rows = make(map[string]*dbApplePayRow)
	this.m_removed_rows = make(map[string]*dbApplePayRow)
	return this
}
func (this *dbApplePayTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS ApplePays(OrderId varchar(32),PRIMARY KEY (OrderId))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS ApplePays failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='ApplePays'", this.m_dbc.m_db_name)
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
	_, hasBundleId := columns["BundleId"]
	if !hasBundleId {
		_, err = this.m_dbc.Exec("ALTER TABLE ApplePays ADD COLUMN BundleId varchar(256)")
		if err != nil {
			log.Error("ADD COLUMN BundleId failed")
			return
		}
	}
	_, hasAccount := columns["Account"]
	if !hasAccount {
		_, err = this.m_dbc.Exec("ALTER TABLE ApplePays ADD COLUMN Account varchar(256)")
		if err != nil {
			log.Error("ADD COLUMN Account failed")
			return
		}
	}
	_, hasPlayerId := columns["PlayerId"]
	if !hasPlayerId {
		_, err = this.m_dbc.Exec("ALTER TABLE ApplePays ADD COLUMN PlayerId int(11)")
		if err != nil {
			log.Error("ADD COLUMN PlayerId failed")
			return
		}
	}
	_, hasPayTime := columns["PayTime"]
	if !hasPayTime {
		_, err = this.m_dbc.Exec("ALTER TABLE ApplePays ADD COLUMN PayTime int(11)")
		if err != nil {
			log.Error("ADD COLUMN PayTime failed")
			return
		}
	}
	_, hasPayTimeStr := columns["PayTimeStr"]
	if !hasPayTimeStr {
		_, err = this.m_dbc.Exec("ALTER TABLE ApplePays ADD COLUMN PayTimeStr varchar(256)")
		if err != nil {
			log.Error("ADD COLUMN PayTimeStr failed")
			return
		}
	}
	return
}
func (this *dbApplePayTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT OrderId,BundleId,Account,PlayerId,PayTime,PayTimeStr FROM ApplePays")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbApplePayTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO ApplePays (OrderId,BundleId,Account,PlayerId,PayTime,PayTimeStr) VALUES (?,?,?,?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbApplePayTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM ApplePays WHERE OrderId=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbApplePayTable) Init() (err error) {
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
func (this *dbApplePayTable) Preload() (err error) {
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var OrderId string
	var dBundleId string
	var dAccount string
	var dPlayerId int32
	var dPayTime int32
	var dPayTimeStr string
	for r.Next() {
		err = r.Scan(&OrderId,&dBundleId,&dAccount,&dPlayerId,&dPayTime,&dPayTimeStr)
		if err != nil {
			log.Error("Scan err[%v]", err.Error())
			return
		}
		row := new_dbApplePayRow(this,OrderId)
		row.m_BundleId=dBundleId
		row.m_Account=dAccount
		row.m_PlayerId=dPlayerId
		row.m_PayTime=dPayTime
		row.m_PayTimeStr=dPayTimeStr
		row.m_BundleId_changed=false
		row.m_Account_changed=false
		row.m_PlayerId_changed=false
		row.m_PayTime_changed=false
		row.m_PayTimeStr_changed=false
		row.m_valid = true
		this.m_rows[OrderId]=row
	}
	return
}
func (this *dbApplePayTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbApplePayTable) fetch_rows(rows map[string]*dbApplePayRow) (r map[string]*dbApplePayRow) {
	this.m_lock.UnSafeLock("dbApplePayTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[string]*dbApplePayRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbApplePayTable) fetch_new_rows() (new_rows map[string]*dbApplePayRow) {
	this.m_lock.UnSafeLock("dbApplePayTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[string]*dbApplePayRow)
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
func (this *dbApplePayTable) save_rows(rows map[string]*dbApplePayRow, quick bool) {
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
func (this *dbApplePayTable) Save(quick bool) (err error){
	removed_rows := this.fetch_rows(this.m_removed_rows)
	for _, v := range removed_rows {
		_, err := this.m_dbc.StmtExec(this.m_delete_stmt, v.GetOrderId())
		if err != nil {
			log.Error("exec delete stmt failed %v", err)
		}
		v.m_valid = false
		if !quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
	this.m_removed_rows = make(map[string]*dbApplePayRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbApplePayTable) AddRow(OrderId string) (row *dbApplePayRow) {
	this.m_lock.UnSafeLock("dbApplePayTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	row = new_dbApplePayRow(this,OrderId)
	row.m_new = true
	row.m_loaded = true
	row.m_valid = true
	_, has := this.m_new_rows[OrderId]
	if has{
		log.Error("已经存在 %v", OrderId)
		return nil
	}
	this.m_new_rows[OrderId] = row
	atomic.AddInt32(&this.m_gc_n,1)
	return row
}
func (this *dbApplePayTable) RemoveRow(OrderId string) {
	this.m_lock.UnSafeLock("dbApplePayTable.RemoveRow")
	defer this.m_lock.UnSafeUnlock()
	row := this.m_rows[OrderId]
	if row != nil {
		row.m_remove = true
		delete(this.m_rows, OrderId)
		rm_row := this.m_removed_rows[OrderId]
		if rm_row != nil {
			log.Error("rows and removed rows both has %v", OrderId)
		}
		this.m_removed_rows[OrderId] = row
		_, has_new := this.m_new_rows[OrderId]
		if has_new {
			delete(this.m_new_rows, OrderId)
			log.Error("rows and new_rows both has %v", OrderId)
		}
	} else {
		row = this.m_removed_rows[OrderId]
		if row == nil {
			_, has_new := this.m_new_rows[OrderId]
			if has_new {
				delete(this.m_new_rows, OrderId)
			} else {
				log.Error("row not exist %v", OrderId)
			}
		} else {
			log.Error("already removed %v", OrderId)
			_, has_new := this.m_new_rows[OrderId]
			if has_new {
				delete(this.m_new_rows, OrderId)
				log.Error("removed rows and new_rows both has %v", OrderId)
			}
		}
	}
}
func (this *dbApplePayTable) GetRow(OrderId string) (row *dbApplePayRow) {
	this.m_lock.UnSafeRLock("dbApplePayTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[OrderId]
	if row == nil {
		row = this.m_new_rows[OrderId]
	}
	return row
}
func (this *dbGooglePayRow)GetBundleId( )(r string ){
	this.m_lock.UnSafeRLock("dbGooglePayRow.GetdbGooglePayBundleIdColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_BundleId)
}
func (this *dbGooglePayRow)SetBundleId(v string){
	this.m_lock.UnSafeLock("dbGooglePayRow.SetdbGooglePayBundleIdColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_BundleId=string(v)
	this.m_BundleId_changed=true
	return
}
func (this *dbGooglePayRow)GetAccount( )(r string ){
	this.m_lock.UnSafeRLock("dbGooglePayRow.GetdbGooglePayAccountColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_Account)
}
func (this *dbGooglePayRow)SetAccount(v string){
	this.m_lock.UnSafeLock("dbGooglePayRow.SetdbGooglePayAccountColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_Account=string(v)
	this.m_Account_changed=true
	return
}
func (this *dbGooglePayRow)GetPlayerId( )(r int32 ){
	this.m_lock.UnSafeRLock("dbGooglePayRow.GetdbGooglePayPlayerIdColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_PlayerId)
}
func (this *dbGooglePayRow)SetPlayerId(v int32){
	this.m_lock.UnSafeLock("dbGooglePayRow.SetdbGooglePayPlayerIdColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_PlayerId=int32(v)
	this.m_PlayerId_changed=true
	return
}
func (this *dbGooglePayRow)GetPayTime( )(r int32 ){
	this.m_lock.UnSafeRLock("dbGooglePayRow.GetdbGooglePayPayTimeColumn")
	defer this.m_lock.UnSafeRUnlock()
	return int32(this.m_PayTime)
}
func (this *dbGooglePayRow)SetPayTime(v int32){
	this.m_lock.UnSafeLock("dbGooglePayRow.SetdbGooglePayPayTimeColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_PayTime=int32(v)
	this.m_PayTime_changed=true
	return
}
func (this *dbGooglePayRow)GetPayTimeStr( )(r string ){
	this.m_lock.UnSafeRLock("dbGooglePayRow.GetdbGooglePayPayTimeStrColumn")
	defer this.m_lock.UnSafeRUnlock()
	return string(this.m_PayTimeStr)
}
func (this *dbGooglePayRow)SetPayTimeStr(v string){
	this.m_lock.UnSafeLock("dbGooglePayRow.SetdbGooglePayPayTimeStrColumn")
	defer this.m_lock.UnSafeUnlock()
	this.m_PayTimeStr=string(v)
	this.m_PayTimeStr_changed=true
	return
}
type dbGooglePayRow struct {
	m_table *dbGooglePayTable
	m_lock       *RWMutex
	m_loaded  bool
	m_new     bool
	m_remove  bool
	m_touch      int32
	m_releasable bool
	m_valid   bool
	m_OrderId        string
	m_BundleId_changed bool
	m_BundleId string
	m_Account_changed bool
	m_Account string
	m_PlayerId_changed bool
	m_PlayerId int32
	m_PayTime_changed bool
	m_PayTime int32
	m_PayTimeStr_changed bool
	m_PayTimeStr string
}
func new_dbGooglePayRow(table *dbGooglePayTable, OrderId string) (r *dbGooglePayRow) {
	this := &dbGooglePayRow{}
	this.m_table = table
	this.m_OrderId = OrderId
	this.m_lock = NewRWMutex()
	this.m_BundleId_changed=true
	this.m_Account_changed=true
	this.m_PlayerId_changed=true
	this.m_PayTime_changed=true
	this.m_PayTimeStr_changed=true
	return this
}
func (this *dbGooglePayRow) GetOrderId() (r string) {
	return this.m_OrderId
}
func (this *dbGooglePayRow) save_data(release bool) (err error, released bool, state int32, update_string string, args []interface{}) {
	this.m_lock.UnSafeLock("dbGooglePayRow.save_data")
	defer this.m_lock.UnSafeUnlock()
	if this.m_new {
		db_args:=new_db_args(6)
		db_args.Push(this.m_OrderId)
		db_args.Push(this.m_BundleId)
		db_args.Push(this.m_Account)
		db_args.Push(this.m_PlayerId)
		db_args.Push(this.m_PayTime)
		db_args.Push(this.m_PayTimeStr)
		args=db_args.GetArgs()
		state = 1
	} else {
		if this.m_BundleId_changed||this.m_Account_changed||this.m_PlayerId_changed||this.m_PayTime_changed||this.m_PayTimeStr_changed{
			update_string = "UPDATE GooglePays SET "
			db_args:=new_db_args(6)
			if this.m_BundleId_changed{
				update_string+="BundleId=?,"
				db_args.Push(this.m_BundleId)
			}
			if this.m_Account_changed{
				update_string+="Account=?,"
				db_args.Push(this.m_Account)
			}
			if this.m_PlayerId_changed{
				update_string+="PlayerId=?,"
				db_args.Push(this.m_PlayerId)
			}
			if this.m_PayTime_changed{
				update_string+="PayTime=?,"
				db_args.Push(this.m_PayTime)
			}
			if this.m_PayTimeStr_changed{
				update_string+="PayTimeStr=?,"
				db_args.Push(this.m_PayTimeStr)
			}
			update_string = strings.TrimRight(update_string, ", ")
			update_string+=" WHERE OrderId=?"
			db_args.Push(this.m_OrderId)
			args=db_args.GetArgs()
			state = 2
		}
	}
	this.m_new = false
	this.m_BundleId_changed = false
	this.m_Account_changed = false
	this.m_PlayerId_changed = false
	this.m_PayTime_changed = false
	this.m_PayTimeStr_changed = false
	if release && this.m_loaded {
		atomic.AddInt32(&this.m_table.m_gc_n, -1)
		this.m_loaded = false
		released = true
	}
	return nil,released,state,update_string,args
}
func (this *dbGooglePayRow) Save(release bool) (err error, d bool, released bool) {
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
			log.Error("INSERT GooglePays exec failed %v ", this.m_OrderId)
			return err, false, released
		}
		d = true
	} else if state == 2 {
		_, err = this.m_table.m_dbc.Exec(update_string, args...)
		if err != nil {
			log.Error("UPDATE GooglePays exec failed %v", this.m_OrderId)
			return err, false, released
		}
		d = true
	}
	return nil, d, released
}
func (this *dbGooglePayRow) Touch(releasable bool) {
	this.m_touch = int32(time.Now().Unix())
	this.m_releasable = releasable
}
type dbGooglePayRowSort struct {
	rows []*dbGooglePayRow
}
func (this *dbGooglePayRowSort) Len() (length int) {
	return len(this.rows)
}
func (this *dbGooglePayRowSort) Less(i int, j int) (less bool) {
	return this.rows[i].m_touch < this.rows[j].m_touch
}
func (this *dbGooglePayRowSort) Swap(i int, j int) {
	temp := this.rows[i]
	this.rows[i] = this.rows[j]
	this.rows[j] = temp
}
type dbGooglePayTable struct{
	m_dbc *DBC
	m_lock *RWMutex
	m_rows map[string]*dbGooglePayRow
	m_new_rows map[string]*dbGooglePayRow
	m_removed_rows map[string]*dbGooglePayRow
	m_gc_n int32
	m_gcing int32
	m_pool_size int32
	m_preload_select_stmt *sql.Stmt
	m_preload_max_id int32
	m_save_insert_stmt *sql.Stmt
	m_delete_stmt *sql.Stmt
}
func new_dbGooglePayTable(dbc *DBC) (this *dbGooglePayTable) {
	this = &dbGooglePayTable{}
	this.m_dbc = dbc
	this.m_lock = NewRWMutex()
	this.m_rows = make(map[string]*dbGooglePayRow)
	this.m_new_rows = make(map[string]*dbGooglePayRow)
	this.m_removed_rows = make(map[string]*dbGooglePayRow)
	return this
}
func (this *dbGooglePayTable) check_create_table() (err error) {
	_, err = this.m_dbc.Exec("CREATE TABLE IF NOT EXISTS GooglePays(OrderId varchar(32),PRIMARY KEY (OrderId))ENGINE=InnoDB ROW_FORMAT=DYNAMIC")
	if err != nil {
		log.Error("CREATE TABLE IF NOT EXISTS GooglePays failed")
		return
	}
	rows, err := this.m_dbc.Query("SELECT COLUMN_NAME,ORDINAL_POSITION FROM information_schema.`COLUMNS` WHERE TABLE_SCHEMA=? AND TABLE_NAME='GooglePays'", this.m_dbc.m_db_name)
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
	_, hasBundleId := columns["BundleId"]
	if !hasBundleId {
		_, err = this.m_dbc.Exec("ALTER TABLE GooglePays ADD COLUMN BundleId varchar(256)")
		if err != nil {
			log.Error("ADD COLUMN BundleId failed")
			return
		}
	}
	_, hasAccount := columns["Account"]
	if !hasAccount {
		_, err = this.m_dbc.Exec("ALTER TABLE GooglePays ADD COLUMN Account varchar(256)")
		if err != nil {
			log.Error("ADD COLUMN Account failed")
			return
		}
	}
	_, hasPlayerId := columns["PlayerId"]
	if !hasPlayerId {
		_, err = this.m_dbc.Exec("ALTER TABLE GooglePays ADD COLUMN PlayerId int(11)")
		if err != nil {
			log.Error("ADD COLUMN PlayerId failed")
			return
		}
	}
	_, hasPayTime := columns["PayTime"]
	if !hasPayTime {
		_, err = this.m_dbc.Exec("ALTER TABLE GooglePays ADD COLUMN PayTime int(11)")
		if err != nil {
			log.Error("ADD COLUMN PayTime failed")
			return
		}
	}
	_, hasPayTimeStr := columns["PayTimeStr"]
	if !hasPayTimeStr {
		_, err = this.m_dbc.Exec("ALTER TABLE GooglePays ADD COLUMN PayTimeStr varchar(256)")
		if err != nil {
			log.Error("ADD COLUMN PayTimeStr failed")
			return
		}
	}
	return
}
func (this *dbGooglePayTable) prepare_preload_select_stmt() (err error) {
	this.m_preload_select_stmt,err=this.m_dbc.StmtPrepare("SELECT OrderId,BundleId,Account,PlayerId,PayTime,PayTimeStr FROM GooglePays")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbGooglePayTable) prepare_save_insert_stmt()(err error){
	this.m_save_insert_stmt,err=this.m_dbc.StmtPrepare("INSERT INTO GooglePays (OrderId,BundleId,Account,PlayerId,PayTime,PayTimeStr) VALUES (?,?,?,?,?,?)")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbGooglePayTable) prepare_delete_stmt() (err error) {
	this.m_delete_stmt,err=this.m_dbc.StmtPrepare("DELETE FROM GooglePays WHERE OrderId=?")
	if err!=nil{
		log.Error("prepare failed")
		return
	}
	return
}
func (this *dbGooglePayTable) Init() (err error) {
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
func (this *dbGooglePayTable) Preload() (err error) {
	r, err := this.m_dbc.StmtQuery(this.m_preload_select_stmt)
	if err != nil {
		log.Error("SELECT")
		return
	}
	var OrderId string
	var dBundleId string
	var dAccount string
	var dPlayerId int32
	var dPayTime int32
	var dPayTimeStr string
	for r.Next() {
		err = r.Scan(&OrderId,&dBundleId,&dAccount,&dPlayerId,&dPayTime,&dPayTimeStr)
		if err != nil {
			log.Error("Scan err[%v]", err.Error())
			return
		}
		row := new_dbGooglePayRow(this,OrderId)
		row.m_BundleId=dBundleId
		row.m_Account=dAccount
		row.m_PlayerId=dPlayerId
		row.m_PayTime=dPayTime
		row.m_PayTimeStr=dPayTimeStr
		row.m_BundleId_changed=false
		row.m_Account_changed=false
		row.m_PlayerId_changed=false
		row.m_PayTime_changed=false
		row.m_PayTimeStr_changed=false
		row.m_valid = true
		this.m_rows[OrderId]=row
	}
	return
}
func (this *dbGooglePayTable) GetPreloadedMaxId() (max_id int32) {
	return this.m_preload_max_id
}
func (this *dbGooglePayTable) fetch_rows(rows map[string]*dbGooglePayRow) (r map[string]*dbGooglePayRow) {
	this.m_lock.UnSafeLock("dbGooglePayTable.fetch_rows")
	defer this.m_lock.UnSafeUnlock()
	r = make(map[string]*dbGooglePayRow)
	for i, v := range rows {
		r[i] = v
	}
	return r
}
func (this *dbGooglePayTable) fetch_new_rows() (new_rows map[string]*dbGooglePayRow) {
	this.m_lock.UnSafeLock("dbGooglePayTable.fetch_new_rows")
	defer this.m_lock.UnSafeUnlock()
	new_rows = make(map[string]*dbGooglePayRow)
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
func (this *dbGooglePayTable) save_rows(rows map[string]*dbGooglePayRow, quick bool) {
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
func (this *dbGooglePayTable) Save(quick bool) (err error){
	removed_rows := this.fetch_rows(this.m_removed_rows)
	for _, v := range removed_rows {
		_, err := this.m_dbc.StmtExec(this.m_delete_stmt, v.GetOrderId())
		if err != nil {
			log.Error("exec delete stmt failed %v", err)
		}
		v.m_valid = false
		if !quick {
			time.Sleep(time.Millisecond * 5)
		}
	}
	this.m_removed_rows = make(map[string]*dbGooglePayRow)
	rows := this.fetch_rows(this.m_rows)
	this.save_rows(rows, quick)
	new_rows := this.fetch_new_rows()
	this.save_rows(new_rows, quick)
	return
}
func (this *dbGooglePayTable) AddRow(OrderId string) (row *dbGooglePayRow) {
	this.m_lock.UnSafeLock("dbGooglePayTable.AddRow")
	defer this.m_lock.UnSafeUnlock()
	row = new_dbGooglePayRow(this,OrderId)
	row.m_new = true
	row.m_loaded = true
	row.m_valid = true
	_, has := this.m_new_rows[OrderId]
	if has{
		log.Error("已经存在 %v", OrderId)
		return nil
	}
	this.m_new_rows[OrderId] = row
	atomic.AddInt32(&this.m_gc_n,1)
	return row
}
func (this *dbGooglePayTable) RemoveRow(OrderId string) {
	this.m_lock.UnSafeLock("dbGooglePayTable.RemoveRow")
	defer this.m_lock.UnSafeUnlock()
	row := this.m_rows[OrderId]
	if row != nil {
		row.m_remove = true
		delete(this.m_rows, OrderId)
		rm_row := this.m_removed_rows[OrderId]
		if rm_row != nil {
			log.Error("rows and removed rows both has %v", OrderId)
		}
		this.m_removed_rows[OrderId] = row
		_, has_new := this.m_new_rows[OrderId]
		if has_new {
			delete(this.m_new_rows, OrderId)
			log.Error("rows and new_rows both has %v", OrderId)
		}
	} else {
		row = this.m_removed_rows[OrderId]
		if row == nil {
			_, has_new := this.m_new_rows[OrderId]
			if has_new {
				delete(this.m_new_rows, OrderId)
			} else {
				log.Error("row not exist %v", OrderId)
			}
		} else {
			log.Error("already removed %v", OrderId)
			_, has_new := this.m_new_rows[OrderId]
			if has_new {
				delete(this.m_new_rows, OrderId)
				log.Error("removed rows and new_rows both has %v", OrderId)
			}
		}
	}
}
func (this *dbGooglePayTable) GetRow(OrderId string) (row *dbGooglePayRow) {
	this.m_lock.UnSafeRLock("dbGooglePayTable.GetRow")
	defer this.m_lock.UnSafeRUnlock()
	row = this.m_rows[OrderId]
	if row == nil {
		row = this.m_new_rows[OrderId]
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
	PlayerStageTotalScores *dbPlayerStageTotalScoreTable
	PlayerCharms *dbPlayerCharmTable
	PlayerCatOuqis *dbPlayerCatOuqiTable
	PlayerBeZaneds *dbPlayerBeZanedTable
	PlayerBaseInfos *dbPlayerBaseInfoTable
	ApplePays *dbApplePayTable
	GooglePays *dbGooglePayTable
}
func (this *DBC)init_tables()(err error){
	this.PlayerStageTotalScores = new_dbPlayerStageTotalScoreTable(this)
	err = this.PlayerStageTotalScores.Init()
	if err != nil {
		log.Error("init PlayerStageTotalScores table failed")
		return
	}
	this.PlayerCharms = new_dbPlayerCharmTable(this)
	err = this.PlayerCharms.Init()
	if err != nil {
		log.Error("init PlayerCharms table failed")
		return
	}
	this.PlayerCatOuqis = new_dbPlayerCatOuqiTable(this)
	err = this.PlayerCatOuqis.Init()
	if err != nil {
		log.Error("init PlayerCatOuqis table failed")
		return
	}
	this.PlayerBeZaneds = new_dbPlayerBeZanedTable(this)
	err = this.PlayerBeZaneds.Init()
	if err != nil {
		log.Error("init PlayerBeZaneds table failed")
		return
	}
	this.PlayerBaseInfos = new_dbPlayerBaseInfoTable(this)
	err = this.PlayerBaseInfos.Init()
	if err != nil {
		log.Error("init PlayerBaseInfos table failed")
		return
	}
	this.ApplePays = new_dbApplePayTable(this)
	err = this.ApplePays.Init()
	if err != nil {
		log.Error("init ApplePays table failed")
		return
	}
	this.GooglePays = new_dbGooglePayTable(this)
	err = this.GooglePays.Init()
	if err != nil {
		log.Error("init GooglePays table failed")
		return
	}
	return
}
func (this *DBC)Preload()(err error){
	err = this.PlayerStageTotalScores.Preload()
	if err != nil {
		log.Error("preload PlayerStageTotalScores table failed")
		return
	}else{
		log.Info("preload PlayerStageTotalScores table succeed !")
	}
	err = this.PlayerCharms.Preload()
	if err != nil {
		log.Error("preload PlayerCharms table failed")
		return
	}else{
		log.Info("preload PlayerCharms table succeed !")
	}
	err = this.PlayerCatOuqis.Preload()
	if err != nil {
		log.Error("preload PlayerCatOuqis table failed")
		return
	}else{
		log.Info("preload PlayerCatOuqis table succeed !")
	}
	err = this.PlayerBeZaneds.Preload()
	if err != nil {
		log.Error("preload PlayerBeZaneds table failed")
		return
	}else{
		log.Info("preload PlayerBeZaneds table succeed !")
	}
	err = this.PlayerBaseInfos.Preload()
	if err != nil {
		log.Error("preload PlayerBaseInfos table failed")
		return
	}else{
		log.Info("preload PlayerBaseInfos table succeed !")
	}
	err = this.ApplePays.Preload()
	if err != nil {
		log.Error("preload ApplePays table failed")
		return
	}else{
		log.Info("preload ApplePays table succeed !")
	}
	err = this.GooglePays.Preload()
	if err != nil {
		log.Error("preload GooglePays table failed")
		return
	}else{
		log.Info("preload GooglePays table succeed !")
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
	err = this.PlayerStageTotalScores.Save(quick)
	if err != nil {
		log.Error("save PlayerStageTotalScores table failed")
		return
	}
	err = this.PlayerCharms.Save(quick)
	if err != nil {
		log.Error("save PlayerCharms table failed")
		return
	}
	err = this.PlayerCatOuqis.Save(quick)
	if err != nil {
		log.Error("save PlayerCatOuqis table failed")
		return
	}
	err = this.PlayerBeZaneds.Save(quick)
	if err != nil {
		log.Error("save PlayerBeZaneds table failed")
		return
	}
	err = this.PlayerBaseInfos.Save(quick)
	if err != nil {
		log.Error("save PlayerBaseInfos table failed")
		return
	}
	err = this.ApplePays.Save(quick)
	if err != nil {
		log.Error("save ApplePays table failed")
		return
	}
	err = this.GooglePays.Save(quick)
	if err != nil {
		log.Error("save GooglePays table failed")
		return
	}
	return
}
