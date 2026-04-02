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

func TestZeroCapacityDoesNotRetain(t *testing.T) {
	c := New[int, int](0)
	for i := 0; i < 1000; i++ {
		c.Set(i, i)
	}
	if c.Len() != 0 {
		t.Fatalf("expected Len 0 with size 0, got %d", c.Len())
	}
	if _, ok := c.Get(0); ok {
		t.Fatal("expected miss after Set with size 0")
	}
	c.Set(1, 1)
	if _, ok := c.Peek(1); ok {
		t.Fatal("expected Peek miss with size 0")
	}
}

func TestClearAndLen(t *testing.T) {
	c := New[int, int](10)
	c.Set(1, 1)
	c.Set(2, 2)
	if c.Len() != 2 {
		t.Fatalf("expected Len 2, got %d", c.Len())
	}
	c.Clear()
	if c.Len() != 0 {
		t.Fatalf("expected Len 0 after Clear, got %d", c.Len())
	}
	if _, ok := c.Get(1); ok {
		t.Fatal("expected miss after Clear")
	}
}
