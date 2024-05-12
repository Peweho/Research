package index_service

import (
	"Research/types/doc"
	"Research/types/term_query"
	"Research/util"
)

type IIndexer interface {
	AddDoc(*doc.Document) (int, error)
	DeleteDoc(docId string) int
	Search(query *term_query.TermQuery, onFlag *util.Bitmap, offFlag *util.Bitmap, orFlags []*util.Bitmap) []*doc.Document
	Count() int
	Close() error
}
