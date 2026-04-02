# cache

This repo holds multiple cache algorithm implementations in Go, with an emphasis
on algorithms that make novel trade-offs compared to the popular LRU algorithms
in [`hashicorp/golang-lru`][golang-lru].

This repo also includes tools for benchmarking throughput and hit ratio using
commonly-used test traces.

## Benchmarks

The `bench` package compares this repository’s `lfu` implementation with
HashiCorp’s LRU, 2Q, and ARC caches using the shared scenarios in `cachetest`.

### Running

```
# Throughput (and allocation stats via -benchmem)
go test ./bench -bench=. -benchmem
```

Sample throughput numbers for `lfu` (including `NewSharded`) versus HashiCorp’s
implementations are in the `lfu` package section below.

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
probabilistic approach to tracking item access frequency. A maximum size of
zero disables caching: `Set` does not retain entries and `Get` always misses.

The probabilistic eviction policy is faster and more memory efficient than the
approach described in ["An O(1) algorithm for implementing the Cache cache
eviction scheme"][o1_algo].

Use `New` for a single mutex over the whole map, or `NewSharded` for striped
locks (`hash/maphash` key routing) when concurrent access spreads across many
keys. A sharded cache trades global eviction semantics for throughput under
contention and may hold slightly more than the nominal size (see package docs).

#### Sample benchmarks

The following figures were collected using `cachetest` on an Apple M5 MacBook
Pro with default parallelism and 100,000 entry capacity. Values are ns/op.

| Workload | `lfu` 1 shard | `lfu` 16 shards | `lfu` 64 shards | `lfu` 256 shards | HashiCorp LRU | HashiCorp 2Q | HashiCorp ARC |
|----------|----------------:|------------------:|----:|-----:|--------------:|---:|----:|
| get_miss | 97.91 | 23.45 | 13.62 | 12.57 | 95.57 | 103.3 | 105.5 |
| get_hit | 162.7 | 51.47 | 34.21 | 22.05 | 117.1 | 124.0 | 141.2 |
| set_miss | 169.7 | 57.87 | 42.28 | 29.79 | 144.7 | 172.9 | 212.4 |
| set_hit | 83.19 | 99.87 | 97.67 | 98.40 | 77.41 | 61.59 | 83.26 |
| zipf | 140.6 | 46.23 | 31.08 | 21.95 | 159.6 | 164.5 | 169.6 |

Your particular machine and `-cpu` settings will, of course, move the absolute
numbers; the pattern — contention on one lock vs many stripes, and hot
single-key set_hit — tends to hold.

[golang-lru]: https://github.com/hashicorp/golang-lru
[o1_algo]: https://arxiv.org/pdf/2110.11602.pdf
