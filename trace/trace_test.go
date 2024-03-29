package trace

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestARCReader(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := newARCReader(strings.NewReader(`
1 1 0 0
10 3 0 1
100 2 0 2
`))

		var k [4]int
		n, err := r.Read(k[:])
		assert.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, []int{1, 10, 11, 12}, k[:n])

		n, err = r.Read(k[:])
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 2, n)
		assert.Equal(t, []int{100, 101}, k[:n])
	})

	t.Run("invalid", func(t *testing.T) {
		r := newARCReader(strings.NewReader("1"))
		n, err := r.Read(make([]int, 1))
		assert.ErrorContains(t, err, `invalid line: "1"`)
		assert.Equal(t, 0, n)

		r = newARCReader(strings.NewReader("1 b 2 3"))
		n, err = r.Read(make([]int, 1))
		assert.ErrorContains(t, err, `invalid line: "1 b 2 3"`)
		assert.Equal(t, 0, n)
	})
}

func TestLIRSReader(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := newLIRSReader(strings.NewReader(`
1
10
100
`))

		var k [2]int
		n, err := r.Read(k[:])
		assert.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.Equal(t, []int{1, 10}, k[:n])

		n, err = r.Read(k[:])
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, []int{100}, k[:n])
	})
}

// loopReader is a Reader that reads from a string forever in a loop.
type loopReader struct {
	s string
	i int
}

func (r *loopReader) Read(p []byte) (n int, err error) {
	n = copy(p, r.s[r.i:])
	r.i += n
	if r.i == len(r.s) {
		r.i = 0
	}
	return n, nil
}

func BenchmarkARCReader(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			r := newARCReader(&loopReader{s: arcTrace})
			keys := make([]int, n)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				r.Read(keys)
				if keys[0] == 0 {
					b.Fatal("keys[0] == 0")
				}
			}

			b.ReportMetric(float64(n*b.N)/float64(b.Elapsed().Milliseconds()), "keys/ms")
		})
	}
}

func BenchmarkLIRSReader(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			r := newLIRSReader(&loopReader{s: lirsTrace})
			keys := make([]int, n)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				r.Read(keys)
				if keys[0] == 0 {
					b.Fatal("keys[0] == 0")
				}
			}

			b.ReportMetric(float64(n*b.N)/float64(b.Elapsed().Milliseconds()), "keys/ms")
		})
	}
}

const arcTrace = `147727 8 0 0
561486 1 0 1
80147 25 0 2
404350 5 0 3
80147 8 0 4
404273 8 0 5
80155 8 0 6
404326 10 0 7
80185 8 0 8
432429 1 0 9
80249 8 0 10
432298 64 0 11
995954 32 0 12
404281 45 0 13
404336 6 0 14
80193 56 0 15
432415 14 0 16
139930 8 0 17
80257 8 0 18
431969 5 0 19
431339 8 0 20
147711 8 0 21
431756 32 0 22
75157 46 0 23
431740 16 0 24
75157 8 0 25
104429 1 0 26
75165 32 0 27
905483 23 0 28
80137 10 0 29
140722 8 0 30
80137 8 0 31
3472 5 0 32
128185 8 0 33
128260 28 0 34
431788 6 0 35
905244 64 0 36
905506 10 0 37
128193 64 0 38
128288 2 0 39
431683 57 0 40
431794 4 0 41
431347 64 0 42
431515 64 0 43
431451 64 0 44
431603 64 0 45
431798 32 0 46
431918 26 0 47
894392 5 0 48
73433 8 0 49
73444 32 0 50
73508 32 0 51
73556 32 0 52
140826 8 0 53
432362 13 0 54
986 5 0 55
432258 40 0 56
141806 8 0 57
428979 5 0 58
141814 8 0 59
1002257 64 0 60
4377 23 0 61
1002321 64 0 62
431411 40 0 63
1002385 64 0 64
1002449 64 0 65
1002513 64 0 66
1002577 64 0 67
1002641 64 0 68
1002705 15 0 69
1002257 8 0 70
1002265 64 0 71
1002329 64 0 72
1002393 64 0 73
1002457 64 0 74
1002521 64 0 75
1002585 64 0 76
1002649 64 0 77
1002719 1 0 78
1002713 6 0 79
141750 8 0 80
141734 8 0 81
4409 34 0 82
675837 62 0 83
675913 2 0 84
775487 40 0 85
1463969 8 0 86
708510 64 0 87
708598 64 0 88
146279 8 0 89
895607 5 0 90
651765 8 0 91
651773 4 0 92
708446 64 0 93
146271 8 0 94
2409 5 0 95
782 2 0 96
20267 9 0 97
438376 5 0 98
438230 8 0 99
`

const lirsTrace = `147727
561486
80147
404350
80147
404273
80155
404326
80185
432429
80249
432298
995954
404281
404336
80193
432415
139930
80257
431969
431339
147711
431756
75157
431740
75157
104429
75165
905483
80137
140722
80137
3472
128185
128260
431788
905244
905506
128193
128288
431683
431794
431347
431515
431451
431603
431798
431918
894392
73433
73444
73508
73556
140826
432362
986
432258
141806
428979
141814
1002257
4377
1002321
431411
1002385
1002449
1002513
1002577
1002641
1002705
1002257
1002265
1002329
1002393
1002457
1002521
1002585
1002649
1002719
1002713
141750
141734
4409
675837
675913
775487
1463969
708510
708598
146279
895607
651765
651773
708446
146271
2409
782
20267
438376
438230
`
