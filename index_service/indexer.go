package index_service

import (
	"Research/internal/kvdb"
	"Research/types"
	"Research/util"
	"bytes"
	"encoding/gob"
	"strings"
	"sync/atomic"
)
import reverseindex "Research/internal/reverse_index"

// 外观Facade模式。把正排和倒排2个子系统封装到了一起
type Indexer struct {
	forwardIndex kvdb.IKeyValueDB
	reverseIndex reverseindex.IReverseIndex
	maxIntId     uint64
}

// 初始化索引
func (indexer *Indexer) Init(DocNumEstimate int, dbtype int, DataDir string) error {
	db, err := kvdb.GetKvdb(dbtype, DataDir) //调用工厂方法，打开本地的KV数据库
	if err != nil {
		return err
	}
	indexer.forwardIndex = db
	// todo: 这里指定使用跳表，可以采用策略模式更换不同数据结构
	indexer.reverseIndex = reverseindex.NewSkipListReverseIndex(DocNumEstimate)
	return nil
}

// 从正排索引加载文件到倒排索引
func (indexer *Indexer) LoadFromIndexFile() int {
	reader := bytes.NewReader([]byte{})
	n := indexer.forwardIndex.IterDB(func(k, v []byte) error {
		reader.Reset(v)
		gobDecode := gob.NewDecoder(reader) // 构造gob反序列化器
		// 反序列化
		var doc types.Document
		err := gobDecode.Decode(&doc)
		if err != nil {
			util.Log.Printf("gob decode document failed：%s", err)
			return nil
		}
		indexer.reverseIndex.Add(doc)
		return err
	})
	util.Log.Printf("load %d data from forward index %s", n, indexer.forwardIndex.GetDbPath())
	indexer.maxIntId = uint64(n)
	return int(n)
}

// 关闭索引
func (indexer *Indexer) Close() error {
	return indexer.forwardIndex.Close()
}

// 向索引中添加(亦是更新)文档(如果已存在，会先删除)
func (indexer *Indexer) AddDoc(doc types.Document) (int, error) {
	docId := strings.TrimSpace(doc.ID)
	if len(docId) == 0 {
		return 0, nil
	}
	//先从正排和倒排索引上将docId删除
	indexer.DeleteDoc(docId)

	doc.IntId = atomic.AddUint64(&indexer.maxIntId, 1) //写入索引时自动为文档生成IntId
	//写入正排索引
	var value bytes.Buffer
	encoder := gob.NewEncoder(&value) // 构造编码器，传输到缓冲区

	if err := encoder.Encode(doc); err == nil {
		_ = indexer.forwardIndex.Set([]byte(docId), value.Bytes())
		// 写入正排索引
	} else {
		return 0, err
	}

	//写入倒排索引
	indexer.reverseIndex.Add(doc)
	return 1, nil
}

// 删除文档
func (indexer *Indexer) DeleteDoc(docId string) int {
	// 读取正排索引
	docByte, err := indexer.forwardIndex.Get([]byte(docId))
	if err != nil {
		return 0
	}
	//反序列化为文档
	reader := bytes.NewReader([]byte{})
	reader.Reset(docByte)
	decoder := gob.NewDecoder(reader)
	var doc types.Document
	err = decoder.Decode(&doc)
	if err != nil {
		return 0
	}
	//读取文档关键字，删除倒排索引
	for _, keyWord := range doc.Keywords {
		indexer.reverseIndex.Delete(doc.IntId, &keyWord)
	}
	// 删除正排索引
	_ = indexer.forwardIndex.Delete([]byte(docId))
	return 1
}

// 检索文档
// 从倒排索引中查询文档业务id，再从正排索引查询完整文档
func (indexer *Indexer) Search(query *types.TermQuery, onFlag *util.Bitmap, offFlag *util.Bitmap, orFlags []*util.Bitmap) []*types.Document {
	// 1、从倒排索引中查询文档业务id
	searchResult := indexer.reverseIndex.Search(query, onFlag, offFlag, orFlags)
	if len(searchResult) == 0 {
		return nil
	}
	// 2、从正排索引查询完整文档
	keys := make([][]byte, 0, len(searchResult))
	for _, id := range searchResult {
		keys = append(keys, []byte(id))
	}
	docs, err := indexer.forwardIndex.BatchGet(keys)
	if err != nil {
		return nil
	}
	//3、序列化文档
	reader := bytes.NewReader([]byte{})
	result := make([]*types.Document, 0, len(searchResult))
	for _, docByte := range docs {
		if len(docByte) == 0 {
			continue
		}
		//反序列化
		reader.Reset(docByte)
		decoder := gob.NewDecoder(reader)
		var doc types.Document
		err = decoder.Decode(&doc)
		if err != nil {
			return nil
		}
		// 添加反序列化结果
		result = append(result, &doc)
	}
	return result
}
