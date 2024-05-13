package index_service

import (
	ServiceHub2 "Research/ServiceHub"
	"Research/types/doc"
	"Research/types/index"
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

const (
	INDEX_SERVICE = "index_service"
)

type IndexServiceWorker struct {
	Indexer  *Indexer
	hub      ServiceHub2.IServiceHub
	selfAddr string
}

// 创建IndexServiceWorker，
// DocNumEstimate 对文档数量的预估
// dbtype 正排索引类型
// DataDir 正排索引地址
// localIp 本机内网Ip
// servicePort 服务暴露端口号，应大于1024
func NewIndexServiceWorker(DocNumEstimate int, dbtype int, DataDir string, localIp string, servicePort int) (*IndexServiceWorker, error) {
	service := &IndexServiceWorker{}
	// 1、初始化索引
	service.Indexer = new(Indexer)
	err := service.Indexer.Init(DocNumEstimate, dbtype, DataDir)
	if err != nil {
		return nil, err
	}
	// 2、设置本机地址
	if servicePort <= 1024 {
		return nil, fmt.Errorf("invalid listen port %d, should more than 1024", servicePort)
	}
	service.selfAddr = fmt.Sprintf("%s:%s", localIp, servicePort)
	return service, nil
}

// 进行服务注册
func (service *IndexServiceWorker) Regist(hub ServiceHub2.IServiceHub, heartBeat int64) error {
	//1、设置注册中心
	service.hub = hub
	//2、服务注册
	leaseID, err := service.hub.Regist(INDEX_SERVICE, service.selfAddr, 0)
	if err != nil {
		panic(err)
	}
	//3、异步进行续约
	go func(Id clientv3.LeaseID) {
		for {
			Id, err = service.hub.Regist(INDEX_SERVICE, service.selfAddr, Id)
			if err != nil {
				return
			}
			time.Sleep(time.Duration(heartBeat)*time.Second - 100*time.Millisecond)
		}
	}(leaseID)
	return nil
}

// 关闭索引
func (service *IndexServiceWorker) Close() error {
	if service.hub != nil {
		service.hub.UnRegist(INDEX_SERVICE, service.selfAddr)
	}
	return service.Indexer.Close()
}

// 从索引上删除文档
func (service *IndexServiceWorker) DeleteDoc(ctx context.Context, docId *index.DocId) (*index.AffectedCount, error) {
	return &index.AffectedCount{Count: int32(service.Indexer.DeleteDoc(docId.DocId))}, nil
}

// 向索引中添加文档(如果已存在，会先删除)
func (service *IndexServiceWorker) AddDoc(ctx context.Context, doc *doc.Document) (*index.AffectedCount, error) {
	n, err := service.Indexer.AddDoc(doc)
	return &index.AffectedCount{Count: int32(n)}, err
}

// 检索，返回文档列表
func (service *IndexServiceWorker) Search(ctx context.Context, request *index.SearchRequest) (*index.SearchResult, error) {
	result := service.Indexer.Search(request.Query, request.OnFlag, request.OffFlag, request.OrFlags)
	return &index.SearchResult{Results: result}, nil
}

// 索引里有几个文档
func (service *IndexServiceWorker) Count(ctx context.Context, request *index.CountRequest) (*index.AffectedCount, error) {
	return &index.AffectedCount{Count: int32(service.Indexer.Count())}, nil
}
