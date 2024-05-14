package ServiceHub

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Limiter interface {
	Allow(ctx context.Context) error // 等待请求达到通过条件
}

// 令牌桶算法
type TokenBucket struct {
	capacity  int64      // 桶的容量
	rate      float64    // 令牌放入速率
	tokens    float64    // 当前令牌数量
	lastToken time.Time  // 上一次放令牌的时间
	mtx       sync.Mutex // 互斥锁
}

var (
	tokenBucket *TokenBucket
	tbOnce      sync.Once
)

func NewTokenBucket(capacity int64, rate float64, tokens float64) *TokenBucket {
	if tokenBucket != nil {
		return tokenBucket
	}
	tbOnce.Do(func() {
		tokenBucket = &TokenBucket{
			capacity: capacity,
			rate:     rate,
			tokens:   tokens,
		}
	})
	return tokenBucket
}

func (tb *TokenBucket) Allow(ctx context.Context) error {
	tb.mtx.Lock()
	defer tb.mtx.Unlock()

	for {
		if tb.tokens > 1 {
			// 发令牌
			tb.tokens--
			return nil
		}
		time.Sleep(100 * time.Millisecond)

		// 超时检查
		select {
		case <-ctx.Done():
			return errors.New("获取令牌超时")
		default:
		}

		now := time.Now()
		// 计算需要放的令牌数量
		tb.tokens = tb.tokens + tb.rate*now.Sub(tb.lastToken).Seconds()
		tb.lastToken = now
		if tb.tokens > float64(tb.capacity) {
			tb.tokens = float64(tb.capacity)
		}
	}
}
