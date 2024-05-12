package ServiceHub

import (
	"Research/util"
	"context"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"log"
	"strings"
	"sync"
	"time"
)
import "golang.org/x/time/rate"

type IServiceHub interface {
	Regist(service string, endpoint string, leaseID etcdv3.LeaseID) (etcdv3.LeaseID, error) // 注册服务
	UnRegist(service string, endpoint string) error                                         // 注销服务
	GetServiceEndpoints(service string) []string                                            //服务发现
	GetServiceEndpoint(service string) string                                               //选择服务的一台endpoint
	Close()                                                                                 //关闭etcd client connection
}

// 代理模式。对ServiceHub做一层代理，想访问endpoints时需要通过代理，代理提供了2个功能：缓存和限流保护
// 继承ServiceHub
type HubProxy struct {
	*ServiceHub
	endpointCache sync.Map //维护每一个service下的所有servers
	limiter       *rate.Limiter
	watched       sync.Map // 记录监听的service
}

var (
	proxy     *HubProxy
	proxyOnce sync.Once
)

// proxy也是单例模式
func GetServiceHubProxy(client *etcdv3.Client, heartbeatFrequency int64, qps int) *HubProxy {
	if proxy != nil {
		return proxy
	}

	proxyOnce.Do(func() {
		proxy = &HubProxy{
			ServiceHub:    GetServiceHub(client, heartbeatFrequency),
			endpointCache: sync.Map{},
			limiter:       rate.NewLimiter(rate.Every(time.Duration(1e9/qps)*time.Nanosecond), qps), //每隔1E9/qps纳秒产生一个令牌，即一秒钟之内产生qps个令牌。令牌桶的容量为qps
			watched:       sync.Map{},
		}
	})
	return proxy
}

// 监听指定的service，将其endpoints缓存到本地，并保持一致性
func (proxy *HubProxy) watchEndpointsOfService(service string) {
	// 1、判断是否已经监听
	if _, ok := proxy.watched.Load(service); ok {
		return
	}
	//2、构造服务名称前缀
	key := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service + "/"
	//3、获取数据流管道
	ctx := context.Background()
	watch := proxy.client.Watch(ctx, key, etcdv3.WithPrefix())
	util.Log.Printf("监听服务%s的节点变化", service)
	//4、开始监听
	go func(service string) {
		// 一定存在对应的serviceMap, 详情看 GetServiceEndpoints
		serviceEndPoints, _ := proxy.endpointCache.Load(service)
		serviceEndPointsMap := serviceEndPoints.(*map[string]bool)
		//遍历管道，取出一个事件集合
		for resp := range watch {
			//遍历事件集合
			for _, event := range resp.Events {
				path := strings.Split(string(event.Kv.Key), "/")
				endpoint := path[len(path)-1]
				log.Printf("event service is %s ,type is %d (0:PUT, 1:DELETE)", service, event.Type)
				switch event.Type {
				// 添加endpoint到本地缓存
				case etcdv3.EventTypePut:
					(*serviceEndPointsMap)[endpoint] = true
				// 从本地缓存删除endpoint
				case etcdv3.EventTypeDelete:
					delete(*serviceEndPointsMap, endpoint)
				}
			}
		}
	}(service)
}

// 获取指定service下的point
func (proxy *HubProxy) GetServiceEndpoints(service string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	//阻塞，直到桶中有1个令牌或超时。
	err := proxy.limiter.Wait(ctx)
	if err != nil {
		return nil
	}
	// 获取令牌
	if allow := proxy.limiter.Allow(); !allow {
		return nil
	}
	// 本地缓存存在，直接返回结果
	if value, ok := proxy.endpointCache.Load(service); ok {
		serviceEndpointsMap := value.(*map[string]bool)
		res := make([]string, 0, len((*serviceEndpointsMap)))
		for key, _ := range *serviceEndpointsMap {
			endpoint := key
			res = append(res, endpoint)
		}
		return res
	}

	// 本地缓存不存在，从etcd中进行全量同步
	endpoints := proxy.GetServiceEndpoints(service) // 查询etcd
	serviceMap := make(map[string]bool, len(endpoints))
	for i := 0; i < len(endpoints); i++ {
		serviceMap[endpoints[i]] = true
	}
	proxy.endpointCache.Store(service, &serviceMap) // 向本地缓存中存储
	// 监听该service，进行增量同步
	proxy.watchEndpointsOfService(service)

	return endpoints
}
