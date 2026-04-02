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

Sample throughput numbers for `lfu` (including `NewSharded`) versus HashiCorp’s
implementations are in the **`lfu`** package section below.

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
eviction scheme"][O1_algo].

Use `New` for a single mutex over the whole map, or `NewSharded` for striped
locks (`hash/maphash` key routing) when concurrent access spreads across many
keys. A sharded cache trades global eviction semantics for throughput under
contention and may hold slightly more than the nominal size (see package docs).

#### Sample throughput (one run)

The following figures are from **darwin / arm64** (**Apple M5**), default
parallelism (`-10` in the benchmark name is `GOMAXPROCS`), **100,000** entry
capacity, and the shared `cachetest` workloads (`get_miss`, `get_hit`,
`set_miss`, `set_hit`, `zipf`). Values are **ns/op**; **B/op** and **allocs/op**
come from `-benchmem`.

| Workload | `lfu` (1 shard) | `lfu` sharded ×16 | ×64 | ×256 | HashiCorp LRU | 2Q | ARC |
|----------|----------------:|------------------:|----:|-----:|--------------:|---:|----:|
| get_miss | 102.4 | 26.27 | 14.49 | 15.86 | 97.36 | 104.6 | 101.6 |
| get_hit | 160.0 | 49.10 | 34.10 | 25.42 | 116.6 | 136.2 | 132.0 |
| set_miss | 169.8 | 56.83 | 41.15 | 31.92 | 146.3 | 177.5 | 201.1 |
| set_hit | 88.15 | 100.6 | 99.53 | 99.62 | 78.90 | 62.50 | 76.13 |
| zipf | 140.6 | 44.95 | 30.27 | 21.50 | 161.9 | 164.6 | 172.8 |

**How to read this.** The single-sharded `lfu` cache uses one mutex; under parallel
load, **get_hit** is slower than HashiCorp’s LRU here because every goroutine
contends on the same lock, even though the underlying map work is cheap.
**Striped `lfu`** (`NewSharded`) spreads unrelated keys across locks, so **get**,
**set_miss**, and **zipf** improve sharply as stripe count rises (up to a point:
routing work still has a fixed cost). **set_hit** only updates **one** key, so
almost all traffic hits a **single** stripe; throughput stays near **~100 ns/op**
and does not beat LRU’s **~79 ns/op** on this scenario. On **zipf**, striped `lfu`
ends up **well below** LRU/2Q/ARC for these settings.

Allocation-wise, **set_miss** on LRU reports **9 B/op**; single-sharded `lfu`
reports **2 B/op**; sharded **×256** reports **1 B/op** (others **0 B/op** in this
run). 2Q and ARC report **17 B/op** and **22 B/op** on **set_miss** here.

Your machine and `-cpu` settings will move the absolute numbers; the pattern
— contention on one lock vs many stripes, and hot single-key **set_hit** — tends
to hold.

[O1_algo]: https://arxiv.org/pdf/2110.11602.pdf
