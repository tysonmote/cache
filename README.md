# cache

This repo holds multiple cache algorithm implementations in Go, with an emphasis
on algorithms that make novel trade-offs compared to the popular LRU algorithms
in [`hashicorp/golang-lru`][golang-lru].

This repo also includes tools for benchmarking throughput and hit ratio using
commonly-used test traces.

[golang-lru]: https://github.com/hashicorp/golang-lru

## Benchmarks

TODO

### Running

```
# Throughput
go test ./bench -bench=. -benchmem

# Hit ratios
TODO

# Memory overhead
TODO
```

## Packages

### `lfu`

`lfu` provides a thread-safe, fixed-size, in-memory cache with a probabilistic
least-frequently-used eviction policy. If the cache is full and a new item is
added, a less-frequently used item is evicted to make room. The item evicted is
not guaranteed to be the least frequently used item because Cache uses a
probabilistic approach to tracking item access frequency.

The probabilistic eviction policy is faster and more memory efficient than the
approach described in ["An O(1) algorithm for implementing the Cache cache
eviction scheme"][O1_algo].

[O1_algo]: https://arxiv.org/pdf/2110.11602.pdf
