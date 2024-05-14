package main

import (
	"Research/etc"
	"Research/index_service"
	"Research/types/index"
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
		c.ForwardIndex.DateDir,
		c.Server.NodeIp,
		c.Server.Port)
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
	//监听端口
	lis, err := net.Listen("tcp", strconv.Itoa(c.Server.Port))
	if err != nil {
		util.Log.Fatalf("无法启动服务器: %v", err)
	}

	// 注册服务
	s := &grpc.Server{}
	index.RegisterIndexServiceServer(s, worker)
	// 启动服务器
	if err := s.Serve(lis); err != nil {
		util.Log.Fatalf("服务器启动失败: %v", err)
	}
}
