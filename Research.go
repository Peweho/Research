package main

// 面向用户使用
import (
	"Research/ServiceHub"
	"Research/etc"
	"Research/util"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"strings"
)
import "Research/index_service"

type Research struct {
	index_service.IIndexer
}

// 创建Research服务
// limiter 限流算法，使用默认令牌桶传输nil
func NewResearch(c *etc.Config, limiter ServiceHub.Limiter) (*Research, error) {
	var rs *Research
	// 判断是单节点还是集群模式
	// 1、分布式分为两个结构，客户端需要注册中心，服务端构造kv数据库
	// 2、单节点，不需要注册中心，直接使用indexer

	// 判断是单节点还是集群模式
	switch c.ConfigType {
	case "cluster":
		//判断是否使用了自定义的限流算法
		if limiter == nil {
			limiter = ServiceHub.NewTokenBucket(c.Limit.Capacity, c.Limit.Rate, c.Limit.Tokens)
		}
		rs.IIndexer = index_service.NewSentinel(NewServiceHub(c, limiter))
	default:
		rs.IIndexer = NewIndexer(c)
	}
	return rs, nil
}

// 生成etcd客户端
func NewEtcdClient(etcdConfig etcdv3.Config) *etcdv3.Client {
	// 检查etcd server格式 地址:端口号
	for _, addr := range etcdConfig.Endpoints {
		split := strings.Split(addr, ":")
		if len(split) != 2 {
			util.Log.Fatalf("etcd地址不正确，格式：\"地址:端口号\"")
			return nil
		}
	}
	// 创建客户端
	client, err := etcdv3.New(etcdConfig)
	if err != nil {
		util.Log.Fatalf("连接不上etcd服务器: %v", err) //发生log.Fatal时go进程会直接退出
	}
	return client
}

// 生产注册中心
func NewServiceHub(c *etc.Config, limiter ServiceHub.Limiter) ServiceHub.IServiceHub {
	switch c.ServiceHub.ServiceHubType {
	case "proxy":
		return ServiceHub.GetServiceHubProxy(
			NewEtcdClient(c.GetEtcdConfig()),
			c.ServiceHub.HeartbeatFrequency,
			limiter,
		)
	default:
		return ServiceHub.GetServiceHub(NewEtcdClient(c.GetEtcdConfig()), c.ServiceHub.HeartbeatFrequency)
	}
}

// 生成单节索引
func NewIndexer(c *etc.Config) *index_service.Indexer {
	indexer := &index_service.Indexer{}
	err := indexer.Init(c.ReverseIndex.DocNumEstimate, c.ForwardIndex.Dbtype, c.GetDateDir())
	if err != nil {
		util.Log.Fatalf("kvdb连接错误")
	}
	// 单节点加载全部key
	indexer.LoadFromIndexFile(1)
	return indexer
}
