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
)

// maxScanToken is the maximum length of a single line in a trace file
// (bufio.Scanner token). Large traces use one integer per line well under this.
const maxScanToken = 16 * 1024 * 1024

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
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	closers := []io.Closer{f}
	closeAll := func() {
		for i := len(closers) - 1; i >= 0; i-- {
			_ = closers[i].Close()
		}
	}

	var r io.Reader = f
	if filepath.Ext(path) == ".gz" {
		gr, err := gzip.NewReader(f)
		if err != nil {
			closeAll()
			return nil, err
		}
		closers = append(closers, gr)
		r = gr
		path = path[:len(path)-3]
	}

	switch filepath.Ext(path) {
	case ".arc":
		return &Trace{r: newARCReader(r), closers: closers}, nil
	case ".lirs":
		return &Trace{r: newLIRSReader(r), closers: closers}, nil
	default:
		closeAll()
		return nil, fmt.Errorf("unknown trace file type: %s", filepath.Ext(path))
	}
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

func newScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	buf := make([]byte, 0, bufio.MaxScanTokenSize)
	s.Buffer(buf, maxScanToken)
	return s
}

// arcReader reads ARC trace files: https://scinapse.io/papers/1860107648
type arcReader struct {
	scanner *bufio.Scanner
	k       int
	n       int
}

func newARCReader(r io.Reader) *arcReader {
	return &arcReader{scanner: newScanner(r)}
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

	n, err := strconv.Atoi(string(line[:sep]))
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
	return &lirsReader{scanner: newScanner(r)}
}

func (r *lirsReader) Read(keys []int) (n int, err error) {
	for i := range keys {
		line, err := readLine(r.scanner)
		if err != nil {
			return i, err
		}

		k, err := strconv.Atoi(string(line))
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
			scanErr := s.Err()
			if scanErr == nil {
				return nil, io.EOF
			}
			return nil, scanErr
		}

		line := s.Bytes()
		if len(line) == 0 {
			continue
		}

		return line, nil
	}
}
