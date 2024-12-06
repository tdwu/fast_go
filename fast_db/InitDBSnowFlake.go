package fast_db

import (
	"reflect"
	"strconv"
	"sync"
	"time"
)

const (
	workerBits  uint8 = 10
	numberBits  uint8 = 12
	workerMax   int64 = -1 ^ (-1 << workerBits)
	numberMax   int64 = -1 ^ (-1 << numberBits)
	timeShift   uint8 = workerBits + numberBits
	workerShift uint8 = numberBits
	startTime   int64 = 1525705533000 // 如果在程序跑了一段时间修改了epoch这个值 可能会导致生成相同的ID
)

type SnowWorker struct {
	mu        sync.Mutex
	timestamp int64
	workerId  int64
	number    int64
}

func NewSnowWorker(workerId int64) *SnowWorker {
	if workerId < 0 || workerId > workerMax {
		panic("workerId excess of quantity")
	}
	wId = workerId
	// 生成一个新节点
	return &SnowWorker{
		timestamp: 0,
		workerId:  workerId,
		number:    0,
	}
}

var workers = map[string]*SnowWorker{}
var lock sync.Mutex
var wId int64

func GetIdForTable(tableName string) int64 {
	worker := workers[tableName]
	if workers[tableName] == nil {
		lock.Lock()
		if workers[tableName] == nil {
			worker = &SnowWorker{
				timestamp: 0,
				workerId:  wId,
				number:    0,
			}
			workers[tableName] = worker
		}
		lock.Unlock()
	}
	return worker.GetId()
}

func GetIdForStruct(t interface{}) int64 {
	tp := reflect.TypeOf(t)
	if tp.Kind() == reflect.Pointer {
		tp = tp.Elem()
	}
	return GetIdForTable(tp.String())
}

func (w *SnowWorker) GetId() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now().UnixNano() / 1e6
	if w.timestamp == now {
		w.number++
		if w.number > numberMax {
			for now <= w.timestamp {
				now = time.Now().UnixNano() / 1e6
			}
		}
	} else {
		w.number = 0
		w.timestamp = now
	}
	ID := int64((now-startTime)<<timeShift | (w.workerId << workerShift) | (w.number))
	return ID
}

func (w *SnowWorker) GetIdStr() string {
	id := w.GetId()
	return strconv.FormatInt(id, 10)
}

func GetIdStrForTable(tableName string) string {
	id := GetIdForTable(tableName)
	return strconv.FormatInt(int64(id), 10)
}

func GetIdStrForStruct(t interface{}) string {
	id := GetIdForStruct(t)
	return strconv.FormatInt(int64(id), 10)
}
