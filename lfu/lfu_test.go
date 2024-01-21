package lfu

import (
	"testing"

	"github.com/tysonmote/cache/cachetest"
)

func TestCache(t *testing.T) {
	err := cachetest.TestCache(func(size int) cachetest.Cache[int, int] {
		return New[int, int](size)
	})
	if err != nil {
		t.Fatal(err)
	}
}
