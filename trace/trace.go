package trace

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
)

type Trace interface {
	Read(k []Key) (n int, err error)
}

type Key struct {
	Key int
	N   int
}

const (
	DS1Trace  = "ds1.arc.gz"
	LoopTrace = "loop.lirs.gz"
	OLTPTrace = "oltp.arc.gz"
	P3Trace   = "p3.arc.gz"
	P8Trace   = "p8.arc.gz"
	S3Trace   = "s3.arc.gz"
)

// TODO: https://scinapse.io/papers/1860107648
//
// ParseARC takes a single line of input from an ARC trace file as described in
// "ARC: a self-tuning, low overhead replacement cache" [1] by Nimrod Megiddo
// and Dharmendra S. Modha [1] and returns a sequence of numbers generated from
// the line and any error. For use with NewReader.
//
// [1]: https://scinapse.io/papers/1860107648
type arcReader struct {
	scanner *bufio.Scanner
}

func newARCReader(r io.Reader) *arcReader {
	scanner := bufio.NewScanner(r)
	return &arcReader{scanner: scanner}
}

var arcSep = []byte(" ")

func (r *arcReader) Read(k []Key) (n int, err error) {
	for i := range k {
		if !r.scanner.Scan() {
			return i, r.scanner.Err()
		}

		line := r.scanner.Bytes()
		key, line := chompInt(line)
		n, line := chompInt(line)

		k[i] = Key{
			Key: key,
			N:   n,
		}
	}

	return len(k), nil
}

func chompInt(line []byte) (int, []byte) {
	sep := bytes.Index(line, arcSep)
	n, _ := strconv.Atoi(string(line[:sep]))
	return n, line[sep+1:]
}

// ParseLIRS takes a single line of input from a LIRS trace file as described in
// multiple papers [1] and returns a slice containing one number. A nice
// collection of LIRS trace files can be found in Ben Manes' repo [2].
//
// [1]: https://en.wikipedia.org/wiki/LIRS_caching_algorithm
// [2]: https://git.io/fj9gU

type lirsReader struct {
	scanner *bufio.Scanner
}

func newLIRSReader(r io.Reader) *lirsReader {
	scanner := bufio.NewScanner(r)
	return &lirsReader{scanner: scanner}
}

func (r *lirsReader) Read(k []Key) (n int, err error) {
	for i := range k {
		if !r.scanner.Scan() {
			return i, r.scanner.Err()
		}

		line := r.scanner.Bytes()
		key, _ := strconv.Atoi(string(line))

		k[i] = Key{
			Key: key,
			N:   1,
		}
	}

	return len(k), nil
}
