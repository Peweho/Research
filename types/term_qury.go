package types

import "strings"

// 每个TermQuery只有一个属性有效
type TermQuery struct {
	KeyWord *KeyWord
	Should  []*TermQuery
	Must    []*TermQuery
}

// 创建叶子结点的
func NewTermQuery(field, word string) *TermQuery {
	return &TermQuery{
		KeyWord: &KeyWord{field, word},
	}
}

func (q TermQuery) Empty() bool {
	return q.KeyWord == nil && len(q.Must) == 0 && len(q.Should) == 0
}

func (m *TermQuery) Or(querys ...*TermQuery) *TermQuery {
	if len(querys) == 0 {
		return m
	}
	res := make([]*TermQuery, 0, len(querys)+1)

	//加入自身和参数的Termquery
	if !m.Empty() {
		res = append(res, m)
	}

	for _, val := range querys {
		p := val
		res = append(res, p)
	}

	return &TermQuery{Should: res}
}

func (m *TermQuery) And(querys ...*TermQuery) *TermQuery {
	if len(querys) == 0 {
		return m
	}
	res := make([]*TermQuery, 0, len(querys)+1)

	//加入自身和参数的Termquery
	if !m.Empty() {
		res = append(res, m)
	}

	for _, val := range querys {
		p := val
		res = append(res, p)
	}

	return &TermQuery{Must: res}
}

// 返回TermQuery的条件表达式
func (m *TermQuery) ToString() string {
	//1、判断是否是叶子结点
	if m.KeyWord != nil {
		//1.1、是叶子结点，直接返回keyword的tostring
		return m.KeyWord.ToString()
	}
	//2、判断哪一个属性有效
	if len(m.Must) > 0 {
		return mustOrShould(m.Must, '&')
	} else {
		return mustOrShould(m.Should, '|')
	}
}

func mustOrShould(query []*TermQuery, ch byte) string {
	//1、判断是否只有一个成员
	if len(query) == 1 {
		//1.1、直接调用ToString方法
		return query[0].ToString()
	}
	res := strings.Builder{}
	//2、多个成员需要加符号处理
	res.WriteByte('(')
	for _, val := range query {
		valRes := val.ToString()
		res.WriteString(valRes)
		res.WriteByte(ch)
	}
	str := res.String()
	str = str[:len(str)-1] + ")"
	return str
}
