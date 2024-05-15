package ServiceHub

import (
	"math/rand"
	"sync/atomic"
)

type LoadBalancer interface {
	Take([]*EndPoint) *EndPoint
}

// 基于轮询的负载均衡策略，默认策略
type Round struct {
	acc uint64
}

// 实现LoadBalancer接口，通过记录请求数量确定分发请求
func (r *Round) Take(endpoints []*EndPoint) *EndPoint {
	if len(endpoints) == 0 {
		return nil
	}
	n := atomic.AddUint64(&r.acc, 1)
	index := n % uint64(len(endpoints))
	return endpoints[index]
}

// 基于随机的负载均衡策略
type Random struct {
}

// 实现LoadBalancer接口，通过随机数确定分发请求
func (r *Random) Take(endpoints []*EndPoint) *EndPoint {
	if len(endpoints) == 0 {
		return nil
	}
	intn := rand.Intn(len(endpoints))
	return endpoints[intn]
}

// 基于权重的负载均衡策略
type Weight struct{}

// 基于权重的负载均衡策略,
// 随机取0-1中的浮点数，判断落入哪个权重区间
func (w *Weight) Take(endpoints []*EndPoint) *EndPoint {
	var dfs func(base float64, i int, value float64) int
	dfs = func(base float64, i int, value float64) int {
		if i >= len(endpoints) {
			return len(endpoints) - 1
		}
		if value <= base+endpoints[i].Weight {
			return i
		}
		return dfs(base+endpoints[i].Weight, i+1, value)
	}
	pos := dfs(0, 0, rand.Float64())
	return endpoints[pos]
}
