package doc

//type KeyWord struct {
//	Field string
//	Word  string
//}

//type Document struct {
//	ID          string // 业务id
//	IntId       uint64 // 倒排索引上的id,即跳表上的key
//	BitsFeature *util.Bitmap
//	Keywords    []KeyWord // 记录分词后的结果
//	Bytes       byte      //整个记录序列化
//}

func (kw *KeyWord) ToString() string {
	if len(kw.Word) > 0 {
		return kw.Field + "\001" + kw.Word
	} else {
		return ""
	}
}
