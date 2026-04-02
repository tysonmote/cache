package trace

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestOpenUnknownExtensionClosesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.txt")
	require.NoError(t, os.WriteFile(path, []byte("1\n"), 0o644))

	tr, err := Open(path)
	assert.Nil(t, tr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown trace file type")
}

func TestOpenInvalidGzipClosesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.arc.gz")
	require.NoError(t, os.WriteFile(path, []byte("not gzip"), 0o644))

	tr, err := Open(path)
	assert.Nil(t, tr)
	require.Error(t, err)
}

func TestOpenGzipARC(t *testing.T) {
	dir := t.TempDir()
	plainPath := filepath.Join(dir, "small.arc")
	gzPath := plainPath + ".gz"

	require.NoError(t, os.WriteFile(plainPath, []byte("1 2 0 0\n"), 0o644))

	src, err := os.ReadFile(plainPath)
	require.NoError(t, err)

	gzData, err := gzipCompress(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(gzPath, gzData, 0o644))

	tr, err := Open(gzPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tr.Close() })

	var keys [2]int
	n, err := tr.Read(keys[:])
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []int{1, 2}, keys[:n])
}
