package reverse_index

import (
	"Research/types/doc"
	"Research/types/term_query"
	"Research/util"
)

type IReverseIndex interface {
	Add(doc doc.Document)                                                                                       //添加一个doc
	Delete(IntId uint64, keyword *doc.KeyWord)                                                                  //从key上删除对应的doc
	Search(q *term_query.TermQuery, onFlag *util.Bitmap, offFlag *util.Bitmap, orFlags []*util.Bitmap) []string //查找,返回业务侧文档ID
}
