package main

import (
	"Research/etc"
	"Research/util"
	"fmt"
)

// 集群模式下，提供文档增删查方法
//var server index.UnimplementedIndexServiceServer

func main() {
	// 得到配置文件
	util.Log.Println("开始读取配置文件")
	c := etc.GetConfig("etc/etc.yaml")
	fmt.Println(c)
	util.Log.Println("读取配置文件结束")
	////构造索引服务
	//worker, err := index_service.NewIndexServiceWorker(c.ReverseIndex.DocNumEstimate, c.ForwardIndex.Dbtype, c.ForwardIndex.DateDir, c.Server.NodeIp, c.Server.Port)
	//if err != nil {
	//	return
	//}
	////监听端口
	//lis, err := net.Listen("tcp", strconv.Itoa(c.Server.Port))
	//if err != nil {
	//	util.Log.Fatalf("无法启动服务器: %v", err)
	//}
	//
	////注册服务
	//s := &grpc.Server{}
	//index.RegisterIndexServiceServer(s, worker)
	//
	//// 启动服务器
	//if err := s.Serve(lis); err != nil {
	//	util.Log.Fatalf("服务器启动失败: %v", err)
	//}
}
