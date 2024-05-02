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

// 删除doc
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

// 根据查询表达式查找结果,返回业务id集合
func (m *SkipListReverseIndex) Search(q *types.TermQuery, onFlag *util.Bitmap, offFlag *util.Bitmap, orFlags []*util.Bitmap) []string {
	//获取查询结果
	search := m.search(q, onFlag, offFlag, orFlags)
	if search == nil {
		return nil
	}
	res := make([]string, 0, search.Len())
	//遍历跳表找出业务id
	node := search.Front()
	for node != nil {
		skv := node.Value.(*SkipListValue)
		res = append(res, skv.Id)
		node = node.Next()
	}
	return res
}

// 根据查询表达式查找结果，保存在跳表中
func (m *SkipListReverseIndex) search(q *types.TermQuery, onFlag *util.Bitmap, offFlag *util.Bitmap, orFlags []*util.Bitmap) *skiplist.SkipList {
	//根据查询表达式分三种情况
	//1、存在关键字
	if q.KeyWord != nil {
		res := skiplist.New(skiplist.Uint64)
		key := q.KeyWord.ToString()
		//得到关键字的跳表
		value, exist := m.table.Get(key)

		if !exist {
			return nil
		}
		list := value.(*skiplist.SkipList)
		node := list.Front()
		//遍历跳表
		for node != nil {
			intId := node.Key().(uint64)
			skv, _ := node.Value.(SkipListValue)
			//跳表上的BitsFeature保存了文档信息，判断文档是否满足要求
			if intId > 0 && filter(skv.BitsFeature, onFlag, offFlag, orFlags) {
				res.Set(node.Key(), node.Value)
			}
			node = node.Next()
		}
		return res
	}

	//2、must关系
	if len(q.Must) > 0 {
		res := make([]*skiplist.SkipList, 0, len(q.Must))
		for _, val := range q.Must {
			resSkl := m.search(val, onFlag, offFlag, orFlags)
			res = append(res, resSkl)
		}
		//must求交集
		return util.IntersectionOfSkipList(res...)
	}

	//3、should关系
	if len(q.Should) > 0 {
		res := make([]*skiplist.SkipList, 0, len(q.Should))
		for _, val := range q.Should {
			resSkl := m.search(val, onFlag, offFlag, orFlags)
			res = append(res, resSkl)
		}
		//should求并集
		return util.UnionsetOfSkipList(res...)
	}
	return nil
}

// 判断bitmap是否满足条件
func filter(q *util.Bitmap, onFlag *util.Bitmap, offFlag *util.Bitmap, orFlags []*util.Bitmap) bool {
	// onFalg全部满足，求交集后判断是否等与onFlag
	r1 := util.IntersectionOfBitmaps(q, onFlag)
	if !onFlag.IsEqual(r1) {
		return false
	}

	// offFalg全部满足，求交集后判断是否为0
	r2 := util.IntersectionOfBitmaps(q, offFlag)
	if !r2.IsZero() {
		return false
	}

	// orFlag 表示至少满足一个
	for _, val := range orFlags {
		if val.IsZero() {
			continue
		}
		res := util.IntersectionOfBitmaps(q, val)
		//为0表示没有条件成立
		if res.IsZero() {
			return false
		}
	}

	return true
}
