package raft

import (
	"Research/index_service"
	"Research/types/doc"
	"Research/types/index"
	"Research/types/raft"
	"Research/types/term_query"
	"Research/util"
	"bytes"
	"context"
	"encoding/gob"
	"errors"
)

type RaftRequest struct {
	raft.UnimplementedResearchClientServiceServer
	*index_service.IndexServiceWorker
}

func NewRaftRequest(worker *index_service.IndexServiceWorker) *RaftRequest {
	return &RaftRequest{
		IndexServiceWorker: worker,
	}
}

var (
	errserialize = errors.New("serialize failed")
	errArgs      = errors.New("args error")
)

func (r *RaftRequest) Research(ctx context.Context, req *raft.ResearchRequest) (resp *raft.ResearchResponse, err error) {
	*resp.Success = false
	// 参数校验
	if len(req.Args) == 0 || *req.ReqType == 3 && len(req.Args) != 4 {
		*resp.ErrorMsg = errArgs.Error()
		return
	}

	// 1: add, 2: delete, 3: search
	resp = &raft.ResearchResponse{}
	switch *req.ReqType {
	case 1:
		docs, err := Deserialize[*doc.Document](req.Args[0])
		if err != nil {
			*resp.ErrorMsg = err.Error()
			return
		}
		addDoc, err := r.AddDoc(ctx, docs)
		resp.Values = append(resp.Values, addDoc.String())
	case 2:
		docId, err := Deserialize[*index.DocId](req.Args[0])
		if err != nil {
			*resp.ErrorMsg = err.Error()
			return
		}
		addDoc, err := r.DeleteDoc(ctx, docId)
		resp.Values = append(resp.Values, addDoc.String())
	case 3:

		query, err := Deserialize[*term_query.TermQuery](req.Args[0])
		if err != nil {
			*resp.ErrorMsg = err.Error()
			return
		}
		onFlag, err := Deserialize[*util.Bitmap](req.Args[1])
		if err != nil {
			*resp.ErrorMsg = err.Error()
			return
		}
		offFlag, err := Deserialize[*util.Bitmap](req.Args[2])
		if err != nil {
			*resp.ErrorMsg = err.Error()
			return
		}
		OrFlags, err := Deserialize[[]*util.Bitmap](req.Args[3])
		if err != nil {
			*resp.ErrorMsg = err.Error()
			return
		}
		isr := &index.SearchRequest{
			Query:   query,
			OnFlag:  onFlag,
			OffFlag: offFlag,
			OrFlags: OrFlags,
		}
		searchRes, err := r.Search(ctx, isr)
		for _, v := range searchRes.Results {
			str, err := Serialize[*doc.Document](v)
			if err != nil {
				*resp.ErrorMsg = err.Error()
				return
			}
			resp.Values = append(resp.Values, str)
		}
	}
	*resp.Success = true
	return
}

//type SerializeType[T interface{*doc.Document} | *index.DocId | *term_query.TermQuery | *util.Bitmap | []*util.Bitmap] interface {}

// 将序列化后的字符串转为相应类型
func Deserialize[T *doc.Document | *index.DocId | *term_query.TermQuery | *util.Bitmap | []*util.Bitmap](docStr string) (target T, err error) {
	reader := bytes.NewReader([]byte{})
	reader.Reset([]byte(docStr))
	decoder := gob.NewDecoder(reader)
	err = decoder.Decode(target)
	return
}

// 反序列化
func Serialize[T *doc.Document | *index.DocId | *term_query.TermQuery | *util.Bitmap | []*util.Bitmap](input T) (str string, err error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err = encoder.Encode(input)
	if err != nil {
		return
	}
	str = buffer.String()
	return
}
