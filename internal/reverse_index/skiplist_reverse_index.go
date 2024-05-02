package reverse_index

import (
	"Research/types"
	"Research/util"
	"github.com/huandu/skiplist"
	farmhash "github.com/leemcloughlin/gofarmhash"
	"runtime"
	"sync"
)

// 倒排索引整体上是个map，map的value是一个List
type SkipListReverseIndex struct {
	table *util.ResearchMap //分段map，并发安全
	locks []sync.RWMutex    //修改倒排索引时，相同的key需要去竞争同一把锁
}

type SkipListValue struct {
	Id          string
	BitsFeature *util.Bitmap
}

// DocNumEstimate是预估的doc数量
func NewSkipListReverseIndex(DocNumEstimate int) *SkipListReverseIndex {
	indexer := new(SkipListReverseIndex)
	indexer.table = util.NewResearchMap(runtime.NumCPU(), DocNumEstimate) // 分片数量为cpu数量
	indexer.locks = make([]sync.RWMutex, 1000)
	return indexer
}

// 根据key获取对应锁
func (indexer SkipListReverseIndex) getLock(key string) *sync.RWMutex {
	n := int(farmhash.Hash32WithSeed([]byte(key), 0))
	return &indexer.locks[n%len(indexer.locks)]
}

// 添加文档
func (m *SkipListReverseIndex) Add(doc types.Document) {
	for _, keyWord := range doc.Keywords {
		key := keyWord.ToString()
		//对可能相同的key加锁
		lock := m.getLock(key)
		lock.Lock()

		skipListValue := &SkipListValue{Id: doc.ID, BitsFeature: doc.BitsFeature}
		if val, exist := m.table.Get(key); !exist {
			//不存在，加入key，创建跳表
			skipList := skiplist.New(skiplist.Uint64)
			skipList.Set(doc.IntId, skipListValue)
			m.table.Set(key, skipList)
		} else {
			//存在，获得跳表加入doc
			skipList := val.(*skiplist.SkipList)
			skipList.Set(doc.IntId, skipListValue)
		}
		lock.Unlock()
	}
}

//删除doc
func (m *SkipListReverseIndex) Delete(intId uint64, keyWord *types.KeyWord) {
	key := keyWord.ToString()
	lock := m.getLock(key)
	lock.Lock()

	if val, exist := m.table.Get(key); exist {
		skipList := val.(*skiplist.SkipList)
		skipList.Remove(intId)
	}

	lock.Unlock()
}
