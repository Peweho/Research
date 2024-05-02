package util

import "github.com/huandu/skiplist"

// 求多个SkipList的交集
func IntersectionOfSkipList(lists ...*skiplist.SkipList) *skiplist.SkipList {
	if len(lists) == 0 {
		return nil
	}

	if len(lists) == 1 {
		return lists[0]
	}

	// 1、给每个调表创建指针
	currNode := make([]*skiplist.Element, len(lists))
	for i, list := range lists {
		currNode[i] = list.Front()
	}

	res := skiplist.New(skiplist.Uint64) // 返回值

	for {
		cnt := 0                // 记录当前跳表的最大值数量
		var maxValue uint64 = 0 // 记录当前跳表最大值

		// 2、每次寻找各个跳表中的当前最大值以及数量
		for _, node := range currNode {
			if node.Key().(uint64) > maxValue {
				maxValue = node.Key().(uint64)
				cnt = 1
			} else if node.Key().(uint64) == maxValue {
				cnt++
			}
		}

		if cnt == len(lists) {
			// 3、所有最大值相等则是一个交集元素
			res.Set(currNode[0].Key(), currNode[0].Value)
			// 跳表全部后移一位
			for i := 0; i < len(currNode); i++ {
				// 如果存在一个跳表走到尽头，则结束
				if currNode[i] == nil {
					return res
				}
				currNode[i] = currNode[i].Next()
			}
		} else {
			//3.1、并非所有值相等，小于最大值的后移一位
			for i := 0; i < len(currNode); i++ {
				if currNode[i].Key().(uint64) < maxValue {
					// 如果存在一个跳表走到尽头，则结束
					if currNode[i] == nil {
						return res
					}
					currNode[i] = currNode[i].Next()
				}
			}
		}
	}
}

// 求多个SkipList的并集
func UnionsetOfSkipList(lists ...*skiplist.SkipList) *skiplist.SkipList {
	// 遍历跳表将没有加入过的元素加入到跳表中

	if len(lists) == 0 {
		return nil
	}

	if len(lists) == 1 {
		return lists[0]
	}

	res := skiplist.New(skiplist.Uint64)      // 返回值
	keySet := make(map[uint64]struct{}, 1000) // 记录加入到跳表中的元素

	for _, list := range lists {
		if list == nil {
			continue
		}
		node := list.Front()
		for node != nil {
			if _, ok := keySet[node.Key().(uint64)]; !ok {
				keySet[node.Key().(uint64)] = struct{}{}
				res.Set(node.Key(), node.Value)
			}
			node = node.Next()
		}
	}
	return res
}
