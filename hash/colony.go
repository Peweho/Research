package hash

import "Research/index_service"

type Colony struct {
	RaftClient []string
	index_service.Sentinel
	GroupId          string
	VirtualNodeCount int
}

// 将哈希值在指定范围的key转移到另一个Colony，范围为(left,right]
func (c *Colony) TransferToColony(cl *Colony, left uint32, right uint32) error {
	// 获取所有虚拟节点
	// 判断key在left和right之间，如果存在，则将key迁移到cl
	return nil
}
