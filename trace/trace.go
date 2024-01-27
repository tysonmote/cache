package trace

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"unsafe"
)

type Trace struct {
	r       reader
	closers []io.Closer
}

type reader interface {
	Read(k []int) (n int, err error)
}

// Open opens a trace file at the given path and returns a Trace. The file may
// be gzipped. The file type is determined by the file extension.
func Open(path string) (*Trace, error) {
	trace := &Trace{}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	trace.closers = append(trace.closers, f)

	var r io.Reader = f

	if filepath.Ext(path) == ".gz" {
		r, err = gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		trace.closers = append(trace.closers, r.(io.Closer))
		path = path[:len(path)-3]
	}

	switch filepath.Ext(path) {
	case ".arc":
		trace.r = newARCReader(r)
	case ".lirs":
		trace.r = newLIRSReader(r)
	default:
		return nil, fmt.Errorf("unknown trace file type: %s" + filepath.Ext(path))
	}

	return trace, nil
}

func (t *Trace) Read(k []int) (n int, err error) {
	return t.r.Read(k)
}

func (t *Trace) Close() error {
	var errs []error
	for i := len(t.closers) - 1; i >= 0; i-- {
		if err := t.closers[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

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
	k       int
	n       int
}

func newARCReader(r io.Reader) *arcReader {
	scanner := bufio.NewScanner(r)
	return &arcReader{scanner: scanner}
}

var arcSep = []byte(" ")

func (r *arcReader) Read(keys []int) (n int, err error) {
	for i := range keys {
		for r.n == 0 {
			if !r.scanner.Scan() {
				return i, r.scanner.Err()
			}

			line := r.scanner.Bytes()
			r.k, line = chompInt(line)
			r.n, _ = chompInt(line)
		}

		keys[i] = r.k
		r.k++
		r.n--
	}

	return len(keys), nil
}

func chompInt(line []byte) (int, []byte) {
	sep := bytes.Index(line, arcSep)
	n, _ := strconv.Atoi(unsafe.String(&line[0], sep))
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

func (r *lirsReader) Read(keys []int) (n int, err error) {
	for i := range keys {
		if !r.scanner.Scan() {
			return i, r.scanner.Err()
		}

		line := r.scanner.Bytes()
		keys[i], _ = strconv.Atoi(unsafe.String(&line[0], len(line)))
	}

	return len(keys), nil
}
