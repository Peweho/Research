package index_service

import (
	"Research/util"
	"context"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"strings"
	"sync"
	"time"
)

const (
	SERVICE_ROOT_PATH = "/Research/index" //etcd key的前缀
)

// 服务注册中心
type ServiceHub struct {
	client             *etcdv3.Client
	heartbeatFrequency int64 //server每隔几秒钟不动向中心上报一次心跳（其实就是续一次租约）
	loadBalancer       LoadBalancer
}

var (
	serviceHub *ServiceHub //该全局变量包外不可见，包外想使用时通过GetServiceHub()获得
	hubOnce    sync.Once   //单例模式需要用到一个once
)

// GetServiceHub 单例模式，返回服务注册中心实例
// etcdServers 地址列表
// heartbeatFrequency 心跳频率，使用心跳机制保证连接
func GetServiceHub(etcdServers []string, heartbeatFrequency int64) *ServiceHub {
	// 如果serviceHub被实例化直接返回
	if serviceHub != nil {
		return serviceHub
	}

	hubOnce.Do(func() {
		client, err := etcdv3.New(
			etcdv3.Config{
				Endpoints:   etcdServers,
				DialTimeout: 3 * time.Second,
			},
		)
		if err != nil {
			util.Log.Fatalf("连接不上etcd服务器: %v", err) //发生log.Fatal时go进程会直接退出
		} else {
			serviceHub = &ServiceHub{
				client:             client,
				heartbeatFrequency: heartbeatFrequency, //租约的有效期
				loadBalancer:       &Round{},
			}
		}
	})
	return serviceHub
}

// 更换负载均衡策略
func (hub *ServiceHub) SetLoadBalancer(balancer LoadBalancer) {
	hub.loadBalancer = balancer
}

// Regist 注册服务。 第一次注册向etcd写一个key，后续注册仅仅是在续约
//
// service 微服务的名称
//
// endpoint 微服务server的地址
//
// leaseID 租约ID,第一次注册时置为0即可
func (hub *ServiceHub) Regist(service string, endpoint string, leaseID etcdv3.LeaseID) (etcdv3.LeaseID, error) {
	ctx := context.Background()
	//1、根据租约ID判断是创建租约还是续约
	if leaseID <= 0 {
		//2、创建租约
		if lease, err := hub.client.Grant(ctx, hub.heartbeatFrequency); err != nil {
			util.Log.Printf("创建租约失败：%v", err)
			return 0, err
		} else {
			//2.1、构造服务key
			key := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service + "/" + endpoint
			//2.2、注册服务
			if _, err := hub.client.Put(ctx, key, "", etcdv3.WithLease(leaseID)); err != nil {
				util.Log.Printf("写入服务%s对应的节点%s失败：%v", service, endpoint, err)
				return lease.ID, err
			} else {
				return lease.ID, nil
			}
		}
	} else {
		//3、进行续约
		if _, err := hub.client.KeepAliveOnce(ctx, leaseID); err == rpctypes.ErrLeaseNotFound { //续约一次，到期后还得再续约
			return hub.Regist(service, endpoint, 0) //找不到租约，走注册流程(把leaseID置为0)
		} else if err != nil {
			util.Log.Printf("续约失败:%v", err)
			return 0, err
		} else {
			util.Log.Printf("服务%s对应的节点%s续约成功", service, endpoint)
			return leaseID, nil
		}
	}
}

// 注销服务
func (hub *ServiceHub) UnRegist(service string, endpoint string) error {
	ctx := context.Background()
	key := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service + "/" + endpoint
	if _, err := hub.client.Delete(ctx, key); err != nil {
		util.Log.Printf("注销服务%s对应的节点%s失败: %v", service, endpoint, err)
		return err
	} else {
		util.Log.Printf("注销服务%s对应的节点%s", service, endpoint)
		return nil
	}
}

// 服务发现。client每次进行RPC调用之前都查询etcd，获取server集合，然后采用负载均衡算法选择一台server。或者也可以把负载均衡的功能放到注册中心，即放到getServiceEndpoints函数里，让它只返回一个server
func (hub *ServiceHub) GetServiceEndpoints(service string) []string {
	ctx := context.Background()
	//获取服务
	if resp, err := hub.client.Get(ctx, service, etcdv3.WithPrefix()); err != nil {
		util.Log.Printf("获取服务%s的节点失败: %v", service, err)
		return nil
	} else {
		endpoints := make([]string, 0, len(resp.Kvs))
		for _, val := range resp.Kvs {
			res := strings.Split(string(val.Key), "/")
			endpoints = append(endpoints, res[len(res)-1])
		}
		return endpoints
	}
}

// 先查找服务节点，再通过负载均衡策略从多个节点中选择合适节点
func (hub *ServiceHub) GetServiceEndpoint(service string) string {
	return hub.loadBalancer.Take(hub.GetServiceEndpoints(service))
}

// 关闭etcd client connection
func (hub *ServiceHub) Close() {
	_ = hub.client.Close()
}
