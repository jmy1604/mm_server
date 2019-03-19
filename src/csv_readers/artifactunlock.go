package csv_readers

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Artifactunlock struct {
	ArtifactID int32
	UnLockLevel int32
	UnLockVIPLevel int32
	UnLockResCost string
	MaxRank int32
}

type ArtifactunlockMgr struct {
	id2items map[int32]*Artifactunlock
	items_array []*Artifactunlock
}

func (this *ArtifactunlockMgr) Read(file_path_name string) bool {
	if file_path_name == "" {
		file_path_name = "../game_csv/artifactunlock.csv"
	}
	cs, err := ioutil.ReadFile(file_path_name)
	if err != nil {
		log.Printf("ArtifactunlockMgr.Read err: %v", err.Error())
		return false
	}

 	r := csv.NewReader(strings.NewReader(string(cs)))
	ss, _ := r.ReadAll()
   	sz := len(ss)
	this.id2items = make(map[int32]*Artifactunlock)
    for i := int32(1); i < int32(sz); i++ {
		//if i < 5 {
		//	continue
		//}
		var v Artifactunlock
		var intv, id int
		// ArtifactID
		intv, err = strconv.Atoi(ss[i][0])
		if err != nil {
			log.Printf("table Artifactunlock convert column ArtifactID value %v with row %v err %v", ss[i][0], 0, err.Error())
			return false
		}
		v.ArtifactID = int32(intv)
		id = intv
		// UnLockLevel
		intv, err = strconv.Atoi(ss[i][1])
		if err != nil {
			log.Printf("table Artifactunlock convert column UnLockLevel value %v with row %v err %v", ss[i][1], 1, err.Error())
			return false
		}
		v.UnLockLevel = int32(intv)
		// UnLockVIPLevel
		intv, err = strconv.Atoi(ss[i][2])
		if err != nil {
			log.Printf("table Artifactunlock convert column UnLockVIPLevel value %v with row %v err %v", ss[i][2], 2, err.Error())
			return false
		}
		v.UnLockVIPLevel = int32(intv)
		// UnLockResCost
		v.UnLockResCost = ss[i][3]
		// MaxRank
		intv, err = strconv.Atoi(ss[i][4])
		if err != nil {
			log.Printf("table Artifactunlock convert column MaxRank value %v with row %v err %v", ss[i][4], 4, err.Error())
			return false
		}
		v.MaxRank = int32(intv)
		if id <= 0 {
			continue
		}
		this.id2items[int32(id)] = &v
		this.items_array = append(this.items_array, &v)
   	}
	return true
}

func (this *ArtifactunlockMgr) Get(id int32) *Artifactunlock {
	return this.id2items[id]
}

func (this *ArtifactunlockMgr) GetByIndex(idx int32) *Artifactunlock {
	if int(idx) >= len(this.items_array) {
		return nil
	}
	return this.items_array[idx]
}

func (this *ArtifactunlockMgr) GetNum() int32 {
	return int32(len(this.items_array))
}

