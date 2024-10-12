package main

import (
	"Research/ServiceHub"
	"Research/etc"
	"Research/index_service"
	raftreq "Research/raft"
	"Research/types/index"
	"Research/types/raft"
	"Research/util"
	"google.golang.org/grpc"
	"net"
	"strconv"
)

func main() {
	// 得到配置文件
	c := etc.GetConfig("etc/etc.yaml")
	//构造索引服务
	worker, err := index_service.NewIndexServiceWorker(
		c.ReverseIndex.DocNumEstimate,
		c.ForwardIndex.Dbtype,
		c.GetDateDir(),
		ServiceHub.NewEndPoint(c.Server.NodeIp, c.Server.Port, c.Server.Weight))
	if err != nil {
		return
	}
	// 创建服务注册中心，不需要限流算法
	serviceHub := NewServiceHub(c, nil)
	// 向etcd注册服务节点
	if err := worker.Regist(serviceHub, c.ServiceHub.HeartbeatFrequency); err != nil {
		util.Log.Fatalf("服务注册失败: %v", err)
		return
	}
	// 根据权重加载数据
	worker.Indexer.LoadFromIndexFile(worker.Endpoint.Weight)
	//监听端口
	lis, err := net.Listen("tcp", strconv.Itoa(c.Server.Port))
	if err != nil {
		util.Log.Fatalf("无法启动服务器: %v", err)
	}

	// 注册服务
	s := &grpc.Server{}
	// 非事务请求
	index.RegisterIndexServiceServer(s, worker)
	//事务请求走raft算法协商
	raft.RegisterResearchClientServiceServer(s, raftreq.NewRaftRequest(worker))
	// 启动服务器
	if err := s.Serve(lis); err != nil {
		util.Log.Fatalf("服务器启动失败: %v", err)
	}
}
