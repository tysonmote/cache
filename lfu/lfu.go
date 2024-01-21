package lfu

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	numBuckets     int8    = 4
	maxBucketIndex int8    = numBuckets - 1
	promoteBase    float64 = 0.01
)

// Cache is a thread-safe, fixed-size, in-memory cache with a probabilistic
// least-frequently-used eviction policy. If the cache is full and a new item is
// added, a less-frequently used item is evicted to make room. The item evicted
// is not guaranteed to be the least frequently used item because Cache uses a
// probabilistic approach to tracking item access frequency.
//
// The probabilistic eviction policy is faster and more memory efficient than
// the approach described in the "An O(1) algorithm for implementing the Cache
// cache eviction scheme" paper: https://arxiv.org/pdf/2110.11602.pdf
type Cache[K comparable, V any] struct {
	size int
	rng  *rand.Rand
	mu   sync.Mutex

	// index is a map of keys to bucket indexes.
	index map[K]int8

	// bucket[0] holds items that have been accessed once. bucket[N] holds items
	// that have been accessed ~0.01^N times.
	buckets [numBuckets]map[K]V
}

// New returns a new Cache ready for use with a maximum capacity of size
// items. size of 0 disables caching behavior.
func New[K comparable, V any](size int) *Cache[K, V] {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	buckets := [numBuckets]map[K]V{}
	for i := range buckets {
		buckets[i] = map[K]V{}
	}

	return &Cache[K, V]{
		size:    size,
		rng:     rng,
		index:   map[K]int8{},
		buckets: buckets,
	}
}

// Get returns a value from the cache if it exists and has not expired. If the
// value does not exist or has expired, ok is false.
func (c *Cache[K, V]) Get(key K) (v V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	i, ok := c.index[key]
	if !ok {
		// Cache miss
		return v, false
	}

	// Cache hit
	v = c.buckets[i][key]

	// Probalistically "spill" the item to a more frequently accessed
	// bucket. First bucket is single-access items.
	if i == 0 || (i < maxBucketIndex && c.rng.Float64() < math.Pow(promoteBase, float64(i))) {
		c.promote(i, key)
	}

	return v, true
}

func (c *Cache[K, V]) promote(i int8, key K) {
	c.buckets[i+1][key] = c.buckets[i][key]
	c.index[key] = i + 1
	delete(c.buckets[i], key)
}

// Set adds a value to the cache. If the cache is full, an infrequently used
// item is evicted. The item evicted is not guaranteed to be the least
// frequently used item because LFU uses a probabilistic approach to tracking
// item access frequency.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if i, ok := c.index[key]; ok {
		c.reset(i, key, value)
		return
	}

	if c.size > 0 && len(c.index) == c.size {
		c.evict()
	}

	c.add(key, value)
}

func (c *Cache[K, V]) reset(i int8, key K, value V) {
	c.buckets[0][key] = value
	if i > 0 {
		delete(c.buckets[i], key)
		c.index[key] = 0
	}
}

func (c *Cache[K, V]) add(k K, v V) {
	c.buckets[0][k] = v
	c.index[k] = 0
}

func (c *Cache[K, V]) evict() {
	for _, bucket := range c.buckets {
		for k := range bucket {
			// Map iteration order is undefined, so there are no guarantees as to
			// whether the first item is random, oldest, etc. This is fine for our use
			// case. Guaranteeing a random item or the actual least-frequently-used
			// item would require a more complex data structure, additional work, etc.
			delete(c.index, k)
			delete(bucket, k)
			return
		}
	}
}

func (c *Cache[K, V]) Remove(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if i, ok := c.index[key]; ok {
		delete(c.index, key)
		delete(c.buckets[i], key)
	}
}

func (c *Cache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.index)
}
