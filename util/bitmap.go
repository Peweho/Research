package util

type Bitmap struct {
	bits []uint64 // 小端存储
	cap  int      // bit数
	code int      // 编码
}

func NewBitmap(cap int) *Bitmap {
	// 获取编码方式

	return &Bitmap{
		bits: make([]uint64, cap/64+1),
		cap:  cap,
		code: 64,
	}
}

// 下标从1开始
func (m *Bitmap) SetBit(index int) bool {
	if index <= 0 {
		return false
	}

	pos := index / m.code    // 获取bits第几个数字
	offset := index % m.code // 获取uint内的偏移量

	if pos >= len(m.bits) {
		cnt := pos - len(m.bits) + 1 // 需要增加的Uint64个数
		add := make([]uint64, cnt)
		m.bits = append(m.bits, add...)
		m.cap += m.code * cnt
	}

	m.bits[pos] |= 1 << (m.code - 1 - offset)
	return true
}

func (m *Bitmap) GetBit(index int) (int, bool) {
	if index >= m.cap {
		return 0, false
	}

	pos := index / m.code    // 获取bits第几个数字
	offset := index % m.code // 获取uint内的偏移量

	return int(m.bits[pos] >> (m.code - 1 - offset) & 1), true
}

// 多个Bitmap求交集
func IntersectionOfBitmaps(bitmaps ...*Bitmap) *Bitmap {
	if len(bitmaps) < 2 {
		return nil
	}

	//获取最小的cap
	newCap := bitmaps[0].cap
	for _, bitmap := range bitmaps {
		if newCap > bitmap.cap {
			newCap = bitmap.cap
		}
	}

	res := NewBitmap(newCap)

	for i := 0; i < newCap/res.code+1; i++ {
		var r uint64
		r = ^r

		for _, bitmap := range bitmaps {
			r &= bitmap.bits[i]
		}

		res.bits[i] = r
	}

	return res
}

// 判断bitmap是否全为0
func (m *Bitmap) IsZero() bool {
	for _, val := range m.bits {
		if val != 0 {
			return false
		}
	}
	return true
}

// 判断两个bitmap是否相同
func (m *Bitmap) IsEqual(b *Bitmap) bool {
	//1、先判断容量
	if m.cap != b.cap {
		return false
	}
	//2、再比较bits
	for i := 0; i < len(m.bits); i++ {
		if m.bits[i] != b.bits[i] {
			return false
		}
	}
	return true
}
