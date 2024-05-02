package types

type TermQuery struct {
	KeyWord KeyWord
	Should  []TermQuery
	Must    []TermQuery
}
