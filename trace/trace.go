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

// Trace is a trace file that contains a sequence of integers representing a
// sequence of cache accesses.
type Trace struct {
	r       reader
	closers []io.Closer
}

type reader interface {
	Read(k []int) (n int, err error)
}

// Open opens a Trace file at the given path. The file may be gzipped. The file type is determined by the file extension.
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

// Read reads up to len(k) integers from the trace file into k. It returns the
// number of integers read and any error encountered. If the number of integers
// read is less than len(k), err will be io.EOF.
func (t *Trace) Read(k []int) (n int, err error) {
	return t.r.Read(k)
}

// Close releases any resources associated with the Trace.
func (t *Trace) Close() error {
	var errs []error
	for i := len(t.closers) - 1; i >= 0; i-- {
		if err := t.closers[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// arcReader reads ARC trace files: https://scinapse.io/papers/1860107648
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
			line, err := readLine(r.scanner)
			if err != nil {
				return i, err
			}

			k, remain, ok := chompInt(line)
			if !ok {
				return i, fmt.Errorf("invalid line: %q", line)
			}
			r.k = k

			r.n, _, ok = chompInt(remain)
			if !ok {
				return i, fmt.Errorf("invalid line: %q", line)
			}
		}

		keys[i] = r.k
		r.k++
		r.n--
	}

	return len(keys), nil
}

func chompInt(line []byte) (int, []byte, bool) {
	sep := bytes.Index(line, arcSep)
	if sep == -1 {
		return 0, nil, false
	}

	n, err := strconv.Atoi(unsafe.String(&line[0], sep))
	if err != nil {
		return 0, nil, false
	}

	return n, line[sep+1:], true
}

// lirsReader reads LIRS trace files from ben-manes/caffeine:
// https://shorturl.at/esPS3 or http://tinyurl.com/yep9zj57
type lirsReader struct {
	scanner *bufio.Scanner
}

func newLIRSReader(r io.Reader) *lirsReader {
	scanner := bufio.NewScanner(r)
	return &lirsReader{scanner: scanner}
}

func (r *lirsReader) Read(keys []int) (n int, err error) {
	for i := range keys {
		line, err := readLine(r.scanner)
		if err != nil {
			return i, err
		}

		k, err := strconv.Atoi(unsafe.String(&line[0], len(line)))
		if err != nil {
			return i, err
		}

		keys[i] = k
	}

	return len(keys), nil
}

func readLine(s *bufio.Scanner) ([]byte, error) {
	for {
		if !s.Scan() {
			if err := s.Err(); err == nil {
				return nil, io.EOF
			} else {
				return nil, err
			}
		}

		line := s.Bytes()
		if len(line) == 0 {
			continue
		}

		return line, nil
	}
}
