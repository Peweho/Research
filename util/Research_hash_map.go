package util

import (
	farmhash "github.com/leemcloughlin/gofarmhash"
	"sync"
	//"golang.org/x/exp/maps"
)

type ResearchMap struct {
	mps   []map[string]any // 存放文档
	seg   int              // 分片个数
	locks []sync.RWMutex   // 每个分片的map对应一把锁
	seed  uint32           // 好哈希算法种子
}

// 构造函数
// seg 分片个数，cap总容量
func NewResearchMap(seg, cap int) *ResearchMap {
	mps := make([]map[string]any, seg)
	locks := make([]sync.RWMutex, seg)

	for i := 0; i < seg; i++ {
		mps[i] = make(map[string]any, cap/seg)
	}

	return &ResearchMap{
		mps:   mps,
		seg:   seg,
		locks: locks,
		seed:  0,
	}
}

// 给key分配对应分片
func (m *ResearchMap) getSegIndex(key string) int {
	hashVal := int(farmhash.Hash32WithSeed([]byte(key), m.seed))
	return hashVal % m.seg
}

// 设置KV
func (m *ResearchMap) Set(key string, value any) {
	segId := m.getSegIndex(key)
	m.locks[segId].Lock()
	defer m.locks[segId].Unlock()
	m.mps[segId][key] = value
}

// 获取KV
func (m *ResearchMap) Get(key string) (any, bool) {
	segId := m.getSegIndex(key)
	m.locks[segId].RLock()
	defer m.locks[segId].RUnlock()
	res, ok := m.mps[segId][key]
	return res, ok
}
