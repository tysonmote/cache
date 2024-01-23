package bench

import (
	"testing"

	"github.com/hashicorp/golang-lru/arc/v2"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/tysonmote/cache/cachetest"
	"github.com/tysonmote/cache/lfu"
)

func BenchmarkTysonmoteLFU(b *testing.B) {
	cachetest.BenchmarkCache(b, func(size int) cachetest.Cache[int, int] {
		return lfu.New[int, int](size)
	})
}

// External cache implementations

type hashiLRU[K comparable, V any] struct {
	*lru.Cache[K, V]
}

func (c *hashiLRU[K, V]) Set(k K, v V) {
	c.Add(k, v)
}

func BenchmarkHashicorpLRU(b *testing.B) {
	cachetest.BenchmarkCache(b, func(size int) cachetest.Cache[int, int] {
		c, err := lru.New[int, int](size)
		if err != nil {
			panic(err)
		}
		return &hashiLRU[int, int]{c}
	})
}

type hashi2Q[K comparable, V any] struct {
	*lru.TwoQueueCache[K, V]
}

func (c *hashi2Q[K, V]) Set(k K, v V) {
	c.Add(k, v)
}

func (c *hashi2Q[K, V]) Remove(k K) bool {
	c.TwoQueueCache.Remove(k)
	return true
}

func BenchmarkHashicorp2Q(b *testing.B) {
	cachetest.BenchmarkCache(b, func(size int) cachetest.Cache[int, int] {
		c, err := lru.New2Q[int, int](size)
		if err != nil {
			panic(err)
		}
		return &hashi2Q[int, int]{c}
	})
}

type hashiARC[K comparable, V any] struct {
	*arc.ARCCache[K, V]
}

func (c *hashiARC[K, V]) Set(k K, v V) {
	c.Add(k, v)
}

func (c *hashiARC[K, V]) Remove(k K) bool {
	c.ARCCache.Remove(k)
	return true
}

func BenchmarkHashicorpARC(b *testing.B) {
	cachetest.BenchmarkCache(b, func(size int) cachetest.Cache[int, int] {
		c, err := arc.NewARC[int, int](size)
		if err != nil {
			panic(err)
		}
		return &hashiARC[int, int]{c}
	})
}
