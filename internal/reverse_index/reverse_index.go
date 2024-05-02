package reverse_index

import (
	"Research/types"
	"Research/util"
)

type IReverseIndex interface {
	Add(doc types.Document)                                                                                //添加一个doc
	Delete(IntId uint64, keyword *types.KeyWord)                                                           //从key上删除对应的doc
	Search(q *types.TermQuery, onFlag *util.Bitmap, offFlag *util.Bitmap, orFlags []*util.Bitmap) []string //查找,返回业务侧文档ID
}
