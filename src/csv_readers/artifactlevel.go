package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Artifactlevel struct {
	ClientIndex int32
	ArtifactID int32
	Rank int32
	Level int32
	MaxLevel int32
	SkillID int32
	ArtifactAttr string
	LevelUpResCost string
	RankUpResCost string
	DecomposeRes string
}

type ArtifactlevelMgr struct {
	id2items map[int32]*Artifactlevel
	items_array []*Artifactlevel
}

func (this *ArtifactlevelMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/artifactlevel.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ArtifactlevelMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Artifactlevel)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Artifactlevel
		var intv, id int
		// ClientIndex
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Artifactlevel convert column ClientIndex value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ClientIndex = int32(intv)
		id = intv
		// ArtifactID
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Artifactlevel convert column ArtifactID value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.ArtifactID = int32(intv)
		// Rank
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Artifactlevel convert column Rank value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.Rank = int32(intv)
		// Level
		intv, err = strconv.Atoi(ss[i][3])
		if err != nil {
			log.Printf("table Artifactlevel convert column Level value %v with row %v err %v", ss[i][3], 3, err.Error())
			return false
		}
		v.Level = int32(intv)
		// MaxLevel
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Artifactlevel convert column MaxLevel value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.MaxLevel = int32(intv)
		// SkillID
		intv, err = strconv.Atoi(ss[i][5])
		if err != nil {
			log.Printf("table Artifactlevel convert column SkillID value %v with row %v err %v", ss[i][5], 5, err.Error())
			return false
		}
		v.SkillID = int32(intv)
		// ArtifactAttr
		v.ArtifactAttr = ss[i][6]
		// LevelUpResCost
		v.LevelUpResCost = ss[i][7]
		// RankUpResCost
		v.RankUpResCost = ss[i][8]
		// DecomposeRes
		v.DecomposeRes = ss[i][9]
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ArtifactlevelMgr) Get(id int32) *Artifactlevel {
	return this.id2items[id]
}

func (this *ArtifactlevelMgr) GetByIndex(idx int32) *Artifactlevel {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ArtifactlevelMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

