package ServiceHub

import "strconv"

type EndPoint struct {
	SelfAddr string  // 地址
	Weight   float64 // 权重
}

func NewEndPoint(ip string, port int, Weight float64) *EndPoint {
	return &EndPoint{
		SelfAddr: ip + ":" + strconv.Itoa(port),
		Weight:   Weight,
	}
}
