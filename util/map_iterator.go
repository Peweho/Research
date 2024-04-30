package util

type MapIterator interface {
	Next() *MapEntry
}

type MapEntry struct {
	key   string
	value any
}

// 哈希表迭代器
type ReMapIterator struct {
	remap    *ResearchMap
	keys     [][]string
	rowIndex int
	colIndex int
}

func (m *ReMapIterator) Next() *MapEntry {
	//1、 判断rowindex是否到末尾
	if m.rowIndex >= len(m.keys) {
		return nil
	}
	//2、判断本行是否为空
	if len(m.keys[m.rowIndex]) == 0 {
		//2.1、为空，行数加一递归到下一行
		m.rowIndex++
		return m.Next()
	}

	//3、获取目标元素
	key := m.keys[m.rowIndex][m.colIndex]
	value, _ := m.remap.Get(key) // 元素一定存在

	//4、更新坐标
	if m.colIndex >= len(m.keys[m.rowIndex]) {
		m.colIndex = 0
		m.rowIndex++
	}

	return &MapEntry{
		key:   key,
		value: value,
	}
}
