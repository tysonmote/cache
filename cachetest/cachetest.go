package cachetest

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type Cache[K comparable, V any] interface {
	Get(key K) (v V, ok bool)
	Set(key K, value V)
	Remove(key K) bool
}

func TestCache(create func(size int) Cache[int, int]) error {
	if err := testCacheBasic(create(1)); err != nil {
		return err
	}

	if err := testCacheEviction(create(1)); err != nil {
		return err
	}

	return nil
}

func testCacheBasic(c Cache[int, int]) error {
	if _, ok := c.Get(1); ok {
		return fmt.Errorf("expected cache miss after initialization")
	}

	c.Set(1, 1)
	v, ok := c.Get(1)
	if !ok {
		return fmt.Errorf("expected cache hit after set")
	}
	if v != 1 {
		return fmt.Errorf("expected 1, got %d", v)
	}

	c.Set(1, 2)
	v, ok = c.Get(1)
	if !ok {
		return fmt.Errorf("expected cache hit after overwrite")
	}
	if v != 2 {
		return fmt.Errorf("expected 2, got %d", v)
	}

	c.Remove(1)
	if _, ok := c.Get(1); ok {
		return fmt.Errorf("expected cache miss after removal")
	}

	return nil
}

func testCacheEviction(c Cache[int, int]) error {
	c.Set(1, 1)
	c.Set(2, 2)

	if _, ok := c.Get(1); ok {
		return fmt.Errorf("expected cache miss")
	}

	if _, ok := c.Get(2); !ok {
		return fmt.Errorf("expected cache hit")
	}

	return nil
}

func BenchmarkCache(b *testing.B, create func(size int) Cache[int, int]) {
	size := 100_000

	b.Run("get miss", func(b *testing.B) {
		c := create(size)

		// Fill cache
		for i := -1; i > -size; i-- {
			c.Set(i, i)
		}

		b.ResetTimer()

		b.RunParallel(func(p *testing.PB) {
			counter := 0
			for p.Next() {
				if _, ok := c.Get(counter % size); ok {
					b.Fatal("expected cache miss")
				}
				counter++
			}
		})
	})

	b.Run("get hit", func(b *testing.B) {
		c := create(size)

		// Fill cache
		for i := 0; i < size; i++ {
			c.Set(i, i)
		}

		b.ResetTimer()

		b.RunParallel(func(p *testing.PB) {
			counter := 0
			for p.Next() {
				if _, ok := c.Get(counter % size); !ok {
					b.Fatal("expected cache hit")
				}
				counter++
			}
		})
	})

	b.Run("set miss", func(b *testing.B) {
		c := create(size)

		b.RunParallel(func(p *testing.PB) {
			counter := 0
			for p.Next() {
				c.Set(counter, counter)
				counter++
			}
		})
	})

	b.Run("set hit", func(b *testing.B) {
		c := create(size)

		// Fill cache
		for i := 0; i < size; i++ {
			c.Set(i, i)
		}

		b.ResetTimer()

		b.RunParallel(func(p *testing.PB) {
			for p.Next() {
				c.Set(0, 0)
			}
		})
	})

	b.Run("zipf", func(b *testing.B) {
		c := create(size)

		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		z := rand.NewZipf(rng, 1.0001, 10, uint64(size*2))
		keys := make([]int, uint64(size*2))
		for i := range keys {
			keys[i] = int(z.Uint64())
		}

		b.ResetTimer()

		b.RunParallel(func(p *testing.PB) {
			counter := 0
			for p.Next() {
				if _, ok := c.Get(keys[counter%len(keys)]); !ok {
					c.Set(keys[counter%len(keys)], 1)
				}
				counter++
			}
		})
	})
}
