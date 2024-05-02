package types

import "Research/util"

type KeyWord struct {
	Field string
	Word  string
}

type Document struct {
	ID          string
	IntId       uint64
	BitsFeature *util.Bitmap
	keywords    []KeyWord
	Bytes       byte
}

func (kw KeyWord) ToString() string {
	if len(kw.Word) > 0 {
		return kw.Field + "\001" + kw.Word
	} else {
		return ""
	}
}
