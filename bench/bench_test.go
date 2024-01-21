package bench

import (
	"testing"

	"github.com/tysonmote/cache/cachetest"
	"github.com/tysonmote/cache/lfu"
)

func BenchmarkLFU(b *testing.B) {
	cachetest.BenchmarkCache(b, func(size int) cachetest.Cache[int, int] {
		return lfu.New[int, int](size)
	})
}
