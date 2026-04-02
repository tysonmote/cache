# cache

This repo holds multiple cache algorithm implementations in Go, with an emphasis
on algorithms that make novel trade-offs compared to the popular LRU algorithms
in [`hashicorp/golang-lru`][golang-lru].

This repo also includes tools for benchmarking throughput and hit ratio using
commonly-used test traces.

[golang-lru]: https://github.com/hashicorp/golang-lru

## Benchmarks

The `bench` package compares this repository’s `lfu` implementation with
HashiCorp’s LRU, 2Q, and ARC caches using the shared scenarios in `cachetest`.

### Running

```
# Throughput (and allocation stats via -benchmem)
go test ./bench -bench=. -benchmem
```

Hit ratio depends on your workload and trace. Use the `trace` package to read
standard `.arc` or `.lirs` traces, drive your cache with the decoded keys, and
compare hits to total accesses for the metric you care about.

`go test -benchmem` reports bytes allocated per operation and allocs per run,
which is a practical way to compare memory overhead between implementations in
these benchmarks.

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
