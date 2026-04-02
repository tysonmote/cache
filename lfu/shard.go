package lfu

import (
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"math"
	"math/rand"
	"sync"
	"time"
)

// lfuShard is one stripe of a sharded cache: its own mutex, capacity, and LFU
// bucket maps. Eviction is local to this shard only.
type lfuShard[K comparable, V any] struct {
	cap int
	mu  sync.Mutex
	rng *rand.Rand

	promoteThreshold [numBuckets]float64
	index            map[K]int8
	buckets          [numBuckets]map[K]V
}

func newLFUShard[K comparable, V any](cap int) *lfuShard[K, V] {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	buckets := [numBuckets]map[K]V{}
	for i := range buckets {
		buckets[i] = map[K]V{}
	}
	var th [numBuckets]float64
	for i := int8(1); i < numBuckets; i++ {
		th[i] = math.Pow(promoteBase, float64(i))
	}
	return &lfuShard[K, V]{
		cap:              cap,
		rng:              rng,
		promoteThreshold: th,
		index:            map[K]int8{},
		buckets:          buckets,
	}
}

func (s *lfuShard[K, V]) get(key K) (v V, ok bool) {
	s.mu.Lock()
	i, ok := s.index[key]
	if !ok {
		s.mu.Unlock()
		return v, false
	}
	v = s.buckets[i][key]
	if i == 0 || (i < maxBucketIndex && s.rng.Float64() < s.promoteThreshold[i]) {
		s.promote(i, key)
	}
	s.mu.Unlock()
	return v, true
}

func (s *lfuShard[K, V]) peek(key K) (v V, ok bool) {
	s.mu.Lock()
	i, ok := s.index[key]
	if !ok {
		s.mu.Unlock()
		return v, false
	}
	v = s.buckets[i][key]
	s.mu.Unlock()
	return v, true
}

func (s *lfuShard[K, V]) promote(i int8, key K) {
	s.buckets[i+1][key] = s.buckets[i][key]
	s.index[key] = i + 1
	delete(s.buckets[i], key)
}

func (s *lfuShard[K, V]) set(key K, value V) {
	s.mu.Lock()
	if s.cap == 0 {
		if i, ok := s.index[key]; ok {
			delete(s.index, key)
			delete(s.buckets[i], key)
		}
		s.mu.Unlock()
		return
	}
	if i, ok := s.index[key]; ok {
		s.reset(i, key, value)
		s.mu.Unlock()
		return
	}
	if len(s.index) == s.cap {
		s.evict()
	}
	s.add(key, value)
	s.mu.Unlock()
}

func (s *lfuShard[K, V]) reset(i int8, key K, value V) {
	s.buckets[0][key] = value
	if i > 0 {
		delete(s.buckets[i], key)
		s.index[key] = 0
	}
}

func (s *lfuShard[K, V]) add(k K, v V) {
	s.buckets[0][k] = v
	s.index[k] = 0
}

func (s *lfuShard[K, V]) evict() {
	for _, bucket := range s.buckets {
		for k := range bucket {
			delete(s.index, k)
			delete(bucket, k)
			return
		}
	}
}

func (s *lfuShard[K, V]) remove(key K) bool {
	s.mu.Lock()
	if i, ok := s.index[key]; ok {
		delete(s.index, key)
		delete(s.buckets[i], key)
		s.mu.Unlock()
		return true
	}
	s.mu.Unlock()
	return false
}

func (s *lfuShard[K, V]) clear() {
	s.mu.Lock()
	s.index = map[K]int8{}
	for i := range s.buckets {
		s.buckets[i] = map[K]V{}
	}
	s.mu.Unlock()
}

func (s *lfuShard[K, V]) lenLocked() int {
	return len(s.index)
}

func writeKey[K comparable](h *maphash.Hash, key K) {
	switch k := any(key).(type) {
	case int:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], uint64(int64(k)))
		h.Write(b[:])
	case int8:
		h.WriteByte(byte(k))
	case int16:
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], uint16(k))
		h.Write(b[:])
	case int32:
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], uint32(k))
		h.Write(b[:])
	case int64:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], uint64(k))
		h.Write(b[:])
	case uint:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], uint64(k))
		h.Write(b[:])
	case uint8:
		h.WriteByte(k)
	case uint16:
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], k)
		h.Write(b[:])
	case uint32:
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], k)
		h.Write(b[:])
	case uint64:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], k)
		h.Write(b[:])
	case uintptr:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], uint64(k))
		h.Write(b[:])
	case string:
		h.WriteString(k)
	case bool:
		if k {
			h.WriteByte(1)
		} else {
			h.WriteByte(0)
		}
	default:
		h.WriteString(fmt.Sprintf("%T:%v", key, key))
	}
}
