package serve

import (
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStackedFS(t *testing.T) {
	aFS := os.DirFS("testdata/stackedfs/a")
	bFS := os.DirFS("testdata/stackedfs/b")
	stacked := stackedFS{aFS, bFS}

	b, err := fs.ReadFile(stacked, "1.txt")
	require.NoError(t, err)
	require.Equal(t, "1 in a\n", string(b))
	b, err = fs.ReadFile(stacked, "3.txt")
	require.NoError(t, err)
	require.Equal(t, "3 in b\n", string(b))
	b, err = fs.ReadFile(stacked, "2.txt")
	require.NoError(t, err)
	require.Equal(t, "2 in a\n", string(b))

	entries, err := fs.ReadDir(stacked, ".")
	require.NoError(t, err)
	require.Equal(t, 4, len(entries))
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	want := []string{"1.txt", "2.txt", "4.txt", "3.txt"}
	require.Equal(t, want, got)
}
