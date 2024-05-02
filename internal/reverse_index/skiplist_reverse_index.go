package reverse_index

import (
	"Research/util"
	"sync"
)

// 倒排索引整体上是个map，map的value是一个List
type SkipListReverseIndex struct {
	table *util.ResearchMap //分段map，并发安全
	locks []sync.RWMutex    //修改倒排索引时，相同的key需要去竞争同一把锁
}
