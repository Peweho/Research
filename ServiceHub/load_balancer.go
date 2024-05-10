package ServiceHub

import (
	"math/rand"
	"sync/atomic"
)

type LoadBalancer interface {
	Take([]string) string
}

// 基于轮询的负载均衡策略，默认策略
type Round struct {
	acc uint64
}

// 实现LoadBalancer接口，通过记录请求数量确定分发请求
func (r *Round) Take(endpoints []string) string {
	if len(endpoints) == 0 {
		return ""
	}
	n := atomic.AddUint64(&r.acc, 1)
	index := n % uint64(len(endpoints))
	return endpoints[index]
}

// 基于随机的负载均衡策略
type Random struct {
}

// 实现LoadBalancer接口，通过随机数确定分发请求
func (r *Random) Take(endpoints []string) string {
	if len(endpoints) == 0 {
		return ""
	}
	intn := rand.Intn(len(endpoints))
	return endpoints[intn]
}
