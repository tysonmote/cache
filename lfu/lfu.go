package lfu

import (
	"hash/maphash"
)

const (
	numBuckets     int8    = 4
	maxBucketIndex int8    = numBuckets - 1
	promoteBase    float64 = 0.01
)

// minKeysPerShard avoids splitting a tiny total capacity across many stripes:
// with one key per shard, distinct keys can collide on the same shard and
// evict each other immediately. Larger caches still use up to numShards stripes.
const minKeysPerShard = 64

// Cache is a thread-safe, fixed-size, in-memory cache with a probabilistic
// least-frequently-used eviction policy. If the cache is full and a new item is
// added, a less-frequently used item is evicted to make room. The item evicted
// is not guaranteed to be the least frequently used item because Cache uses a
// probabilistic approach to tracking item access frequency.
//
// Overwriting an existing key with Set resets that key's frequency to one
// access; the key is moved to the lowest-frequency bucket as if it were newly
// added.
//
// The probabilistic eviction policy is faster and more memory efficient than
// the approach described in the "An O(1) algorithm for implementing the Cache
// cache eviction scheme" paper: https://arxiv.org/pdf/2110.11602.pdf
//
// A Cache is implemented as one or more internal shards (stripes), each with
// its own mutex. New uses a single shard. NewSharded uses multiple shards so
// concurrent operations on different keys can proceed in parallel; eviction is
// local to each shard. The effective stripe count is at most numShards and is
// reduced when size is small so each shard holds at least minKeysPerShard keys
// on average (except a single zero-capacity shard when size is 0).
type Cache[K comparable, V any] struct {
	shards []*lfuShard[K, V]
	seed   maphash.Seed
}

// New returns a new Cache ready for use with a maximum capacity of size
// items. A size of 0 disables caching: Set does not retain entries and Get
// always misses.
func New[K comparable, V any](size int) *Cache[K, V] {
	return NewSharded[K, V](size, 1)
}

// NewSharded returns a cache with up to numShards stripes. The sum of per-shard
// limits is at least size (often slightly more per shard to absorb hash skew),
// so the cache may hold more than size entries under a striped layout. When
// size is small relative to numShards, fewer stripes are used so each holds at
// least about minKeysPerShard items on average.
//
// Key-to-shard routing uses hash/maphash with a per-cache random seed. int and
// other common primitive keys use fast encodings; other comparable types fall
// back to a string representation for hashing.
func NewSharded[K comparable, V any](size, numShards int) *Cache[K, V] {
	if numShards < 1 {
		panic("lfu: numShards must be at least 1")
	}
	effective := numShards
	if size == 0 {
		effective = 1
	} else {
		maxStripes := size / minKeysPerShard
		if maxStripes < 1 {
			maxStripes = 1
		}
		if effective > maxStripes {
			effective = maxStripes
		}
	}
	caps := distributeCapacity(size, effective)
	shards := make([]*lfuShard[K, V], effective)
	for i := range shards {
		shards[i] = newLFUShard[K, V](caps[i])
	}
	return &Cache[K, V]{
		shards: shards,
		seed:   maphash.MakeSeed(),
	}
}

func distributeCapacity(size, n int) []int {
	caps := make([]int, n)
	if n == 0 || size == 0 {
		return caps
	}
	if n == 1 {
		caps[0] = size
		return caps
	}
	// Ceil(size/n) per shard plus slack: hashing is not perfectly uniform, so a
	// tight sum==size split can overflow a shard during load and evict keys that
	// other stripes still have room for. Total max entries can exceed size.
	base := (size + n - 1) / n
	slack := max(512, min(2048, max(1, base/4)))
	for i := range caps {
		caps[i] = base + slack
	}
	return caps
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Cache[K, V]) shardIndex(key K) int {
	if len(c.shards) == 1 {
		return 0
	}
	var h maphash.Hash
	h.SetSeed(c.seed)
	writeKey(&h, key)
	return int(h.Sum64() % uint64(len(c.shards)))
}

// Get returns a value from the cache if it exists. If the value does not
// exist, ok is false.
func (c *Cache[K, V]) Get(key K) (v V, ok bool) {
	return c.shards[c.shardIndex(key)].get(key)
}

// Peek returns a value from the cache if it exists, without updating its
// access frequency. If the value does not exist, ok is false.
func (c *Cache[K, V]) Peek(key K) (v V, ok bool) {
	return c.shards[c.shardIndex(key)].peek(key)
}

// Set adds or updates a value in the cache. If the key already exists, its
// value is updated and its access frequency is reset to one (the key is treated
// as newly added for eviction purposes). If the cache is full and the key is
// new, an infrequently used item is evicted. The item evicted is not guaranteed
// to be the least frequently used item because LFU uses a probabilistic
// approach to tracking item access frequency.
func (c *Cache[K, V]) Set(key K, value V) {
	c.shards[c.shardIndex(key)].set(key, value)
}

// Remove deletes a key from the cache. It returns true if the key was present.
func (c *Cache[K, V]) Remove(key K) bool {
	return c.shards[c.shardIndex(key)].remove(key)
}

// Clear removes all entries from the cache.
func (c *Cache[K, V]) Clear() {
	for _, s := range c.shards {
		s.clear()
	}
}

// Len returns the number of entries in the cache.
func (c *Cache[K, V]) Len() int {
	n := 0
	for _, s := range c.shards {
		s.mu.Lock()
		n += s.lenLocked()
		s.mu.Unlock()
	}
	return n
}
