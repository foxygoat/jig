package bones

import (
	"os"
	"path/filepath"
	"testing"

	"foxygo.at/jig/log"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
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

			err := Generate(log.DiscardLogger, getFDS(t, tc.pbFile), dir, false, nil, Formatter{QuoteStyle: tc.quoteStyle})
			require.NoError(t, err)
			err = Generate(log.DiscardLogger, getFDS(t, tc.pbFile), dir, false, nil, Formatter{Lang: JS, QuoteStyle: tc.quoteStyle})
			require.NoError(t, err)
			requireSameContent(t, tc.goldenDir, dir)
		})
	}
}

func getFDS(t *testing.T, filename string) *descriptorpb.FileDescriptorSet {
	t.Helper()
	b, err := os.ReadFile(filename)
	require.NoError(t, err)
	fds := &descriptorpb.FileDescriptorSet{}
	err = proto.Unmarshal(b, fds)
	require.NoError(t, err)
	return fds
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
