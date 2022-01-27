package bones

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateGolden(t *testing.T) {
	tests := map[string]struct {
		pbFile     string
		goldenDir  string
		quoteStyle QuoteStyle
	}{
		"exemplar-double": {pbFile: "testdata/exemplar.pb", goldenDir: "testdata/golden/exemplar-double", quoteStyle: Double},
		"greeter-double":  {pbFile: "testdata/greeter.pb", goldenDir: "testdata/golden/greet-double", quoteStyle: Double},
		"exemplar-single": {pbFile: "testdata/exemplar.pb", goldenDir: "testdata/golden/exemplar-single", quoteStyle: Single},
		"greeter-single":  {pbFile: "testdata/greeter.pb", goldenDir: "testdata/golden/greet-single", quoteStyle: Single},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			err := Generate(tc.pbFile, dir, false, nil, FormatOptions{QuoteStyle: tc.quoteStyle})
			require.NoError(t, err)
			err = Generate(tc.pbFile, dir, false, nil, FormatOptions{Lang: JS, QuoteStyle: tc.quoteStyle})
			require.NoError(t, err)
			requireSameContent(t, tc.goldenDir, dir)
		})
	}
}

func requireSameContent(t *testing.T, wantDir, gotDir string) {
	t.Helper()
	wantFiles, err := os.ReadDir(wantDir)
	require.NoError(t, err)
	gotFiles, err := os.ReadDir(gotDir)
	require.NoError(t, err)
	require.Equal(t, len(wantFiles), len(gotFiles))

	for i, wantFile := range wantFiles {
		gotFile := gotFiles[i]
		require.Equal(t, wantFile.Name(), gotFile.Name())
		wantFileName := filepath.Join(wantDir, wantFile.Name())
		wantBytes, err := os.ReadFile(wantFileName)
		require.NoError(t, err)
		gotFileName := filepath.Join(gotDir, gotFile.Name())
		gotBytes, err := os.ReadFile(gotFileName)
		require.NoError(t, err)
		require.Equalf(t, string(wantBytes), string(gotBytes), "file contents are not the same for %s", wantFile.Name())
	}
}
