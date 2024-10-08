package hash

import (
	"errors"
	"github.com/huandu/skiplist"
	"sync"
)

type IHashRing interface {
	// Add 添加一组虚拟节点到哈希环上，对应真实结点为colony，virtualScore为虚拟节点的索引
	Add(virtualScores uint32, colony *Colony) error

	Remove(virtualScore uint32) error
	// 获取虚拟节点对应的真实结点
	GetColony(virtualScore uint32) (*Colony, error)

	// 获取一个虚拟节点的上一个虚拟节点
	GetPrev(virtualScore uint32) (uint32, *Colony, error)

	// GetNext 获取一个虚拟节点的下一个虚拟节点
	GetNext(virtualScore uint32) (uint32, *Colony, error)

	// TransferDataToNext 将虚拟节点数据转移到下一个虚拟节点
	// 找到下一个虚拟节点的真实节点，通过两个真实结点进行转移
	TransferAllDataToNext(colony *Colony) error
}

var (
	ErrVirtualNodeNotExist  = errors.New("virtual node not exist")
	ErrVirtualNodeExist     = errors.New("virtual node has exist")
	ErrNodesNotExist        = errors.New("nodes not exist")
	DefaultVirtualNodeCount = 100
)

type HashRing struct {
	sync.RWMutex
	Nodes    sync.Map           // 真实结点到虚拟节点,map[*service.Colony]*[]uint32
	skipList *skiplist.SkipList // 按序存储虚拟结点，包括对应真实结点
}

func NewHashRing() (hr *HashRing, err error) {
	hr = &HashRing{
		skipList: skiplist.New(skiplist.Int),
		Nodes:    sync.Map{},
	}
	return hr, nil
}

// 向哈希环添加虚拟节点 virtualScore，对应真实结点colony
func (h *HashRing) Add(virtualScore uint32, colony *Colony) error {
	// 添加前检查虚拟节点是否已经存在
	h.RLock()
	elem := h.skipList.Get(virtualScore)
	h.RUnlock()
	if elem != nil {
		return ErrVirtualNodeExist
	}

	// 添加虚拟节点
	h.Lock()
	h.skipList.Set(virtualScore, colony)
	h.Unlock()

	// 加入nodes
	if val, ok := h.Nodes.Load(colony); ok {
		slice := val.(*[]uint32)
		*slice = append(*slice, virtualScore)
	} else {
		slice := make([]uint32, DefaultVirtualNodeCount)
		slice = append(slice, virtualScore)
		h.Nodes.Store(colony, &slice)
	}

	return nil
}

func (h *HashRing) Remove(virtualScore uint32) error {
	h.Lock()
	defer h.Unlock()
	// 在方法TransferAllDataToNext中会维护nodes
	elem := h.skipList.Remove(virtualScore)
	if elem == nil {
		return ErrVirtualNodeNotExist
	}
	return nil
}

func (h *HashRing) GetColony(virtualScore uint32) (*Colony, error) {
	h.Lock()
	defer h.Unlock()

	if elem := h.skipList.FindNext(h.skipList.Front(), virtualScore); elem != nil {
		return elem.Value.(*Colony), nil
	}

	return nil, ErrVirtualNodeNotExist
}

// 返回虚拟节点的前驱以及对应真实结点
func (h *HashRing) GetPrev(virtualScore uint32) (uint32, *Colony, error) {
	h.RLock()
	defer h.RUnlock()

	if h.skipList.Len() <= 1 {
		return 0, nil, ErrVirtualNodeNotExist
	}

	elem := h.skipList.Get(virtualScore)
	if elem == nil {
		return 0, nil, ErrVirtualNodeNotExist
	}

	elem = elem.Prev()
	if elem == nil {
		// 虚拟节点前驱为空，返回最后一个结点
		elem = h.skipList.Back()
	}
	return elem.Key().(uint32), elem.Value.(*Colony), nil
}

func (h *HashRing) GetNext(virtualScore uint32) (uint32, *Colony, error) {
	h.RLock()
	defer h.RUnlock()

	if h.skipList.Len() <= 1 {
		return 0, nil, ErrVirtualNodeNotExist
	}

	elem := h.skipList.Get(virtualScore)
	if elem == nil {
		return 0, nil, ErrVirtualNodeNotExist
	}
	elem = elem.Next()
	if elem == nil {
		// 虚拟节点后驱为空，返回最后一个结点
		elem = h.skipList.Front()
	}
	return elem.Key().(uint32), elem.Value.(*Colony), nil
}

func (h *HashRing) TransferDataToNext(virtualScore uint32) error {
	// 得到虚拟节点的前驱
	left, _, err := h.GetPrev(virtualScore)
	if err != nil {
		return err
	}
	// 确定转移key的范围
	right := virtualScore
	// 得到虚拟节点的真实结点
	c1, err := h.GetColony(virtualScore)
	if err != nil {
		return err
	}
	// 得到后驱的真实结点
	_, c2, err := h.GetNext(virtualScore)
	if err != nil {
		return err
	}

	// 清除虚拟节点
	h.Remove(virtualScore)

	// 转移(left,right]的数据
	err = c1.TransferToColony(c2, left, right)
	if err != nil {
		return err
	}
	return nil
}

func (h *HashRing) TransferAllDataToNext(colony *Colony) error {
	//	遍历虚拟节点进行转移
	value, ok := h.Nodes.Load(colony)
	if !ok {
		return ErrNodesNotExist
	}

	virtualScores := value.(*[]uint32)
	for _, virtualScore := range *virtualScores {
		go func(virtualScore uint32) {
			// todo: 错误处理
			_ = h.TransferDataToNext(virtualScore)
		}(virtualScore)
	}
	h.Nodes.Delete(colony)
	return nil
}
