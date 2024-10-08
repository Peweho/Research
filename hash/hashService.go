package hash

import (
	"Research/types/doc"
	"Research/types/term_query"
	"Research/util"
	"github.com/huandu/skiplist"
	"hash/crc32"
	"strconv"
)

type HashService struct {
	*HashRing
	DelId map[string]struct{}
}

func NewHashService(colonys []*Colony, virtualNodeNum []int) (hs *HashService, err error) {
	hs.HashRing, err = NewHashRing()
	// 填充虚拟节点到哈希环上
	for i := 0; i < len(colonys); i++ {
		start := hs.HashStingTo32Bit(colonys[i].GroupId)
		gap := 1<<32 - 1/virtualNodeNum[i]
		for j := 0; j < virtualNodeNum[i]; j++ {
			_ = hs.Add(start+uint32(j*gap), colonys[i])
		}
	}
	err = nil
	return
}

func (h *HashService) HashStingTo32Bit(s string) uint32 {
	return crc32.Checksum([]byte(s), crc32.IEEETable)
}

// 有文档中的关键词存储到多个Colony中
func (h *HashService) AddDoc(document *doc.Document) (int, error) {
	var res int
	for _, v := range document.Keywords {
		// 哈希到虚拟节点
		virtualNode := h.HashStingTo32Bit(v.ToString())
		// 获取虚拟节点对应的真实节点
		colony, err := h.GetColony(virtualNode)
		if err != nil {
			break
		}
		// 添加文档
		addDoc, err := colony.AddDoc(document)
		if err != nil {
			break
		}
		res += addDoc
	}
	return res, nil
}

// 懒删除，推迟到查询到文档，比较业务id，判断是否删除
func (h *HashService) DeleteDoc(Id string) int {
	h.DelId[Id] = struct{}{}
	return 0
}

func (h *HashService) Search(query *term_query.TermQuery, onFlag *util.Bitmap, offFlag *util.Bitmap, orFlags []*util.Bitmap) []*doc.Document {
	search := h.search(query, onFlag, offFlag, orFlags)
	res := make([]*doc.Document, search.Len())
	font := search.Front()
	for font != nil {
		res = append(res, font.Value.(*doc.Document))
		font = font.Next()
	}
	return res
}

func (h *HashService) Count() int {
	res := 0
	h.Nodes.Range(func(key, value any) bool {
		colony := key.(*Colony)
		res += colony.Count()
		return true
	})
	return res
}

func (h *HashService) Close() error {
	h.Nodes.Range(func(key, value any) bool {
		colony := key.(*Colony)
		err := colony.Close()
		if err != nil {
			return false
		}
		return true
	})
	return nil
}

func (h *HashService) search(q *term_query.TermQuery, onFlag *util.Bitmap, offFlag *util.Bitmap, orFlags []*util.Bitmap) *skiplist.SkipList {
	//根据查询表达式分三种情况
	//1、存在关键字
	if q.Keyword != nil {
		res := skiplist.New(skiplist.Uint64)
		key := q.Keyword.ToString()
		//得到关键字的跳表
		colony, err := h.GetColony(h.HashStingTo32Bit(key))
		if err != nil {
			return nil
		}
		search := colony.Sentinel.Search(q, onFlag, offFlag, orFlags)
		for _, document := range search {
			if _, ok := h.DelId[document.Id]; ok {
				// 懒删除
				colony.DeleteDoc(strconv.FormatUint(document.IntId, 10))
				continue
			}
			res.Set(document.Id, document)
		}
		return res
	}

	//2、must关系
	if len(q.Must) > 0 {
		res := make([]*skiplist.SkipList, 0, len(q.Must))
		for _, val := range q.Must {
			resSkl := h.search(val, onFlag, offFlag, orFlags)
			res = append(res, resSkl)
		}
		//must求交集
		return util.IntersectionOfSkipList(res...)
	}

	//3、should关系
	if len(q.Should) > 0 {
		res := make([]*skiplist.SkipList, 0, len(q.Should))
		for _, val := range q.Should {
			resSkl := h.search(val, onFlag, offFlag, orFlags)
			res = append(res, resSkl)
		}
		//should求并集
		return util.UnionsetOfSkipList(res...)
	}
	return nil
}
