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
		minimal    bool
	}{
		"exemplar-double":         {pbFile: "testdata/exemplar.pb", goldenDir: "testdata/golden/exemplar-double-no-minimal", quoteStyle: Double},
		"greeter-double":          {pbFile: "testdata/greeter.pb", goldenDir: "testdata/golden/greet-double-no-minimal", quoteStyle: Double},
		"exemplar-single":         {pbFile: "testdata/exemplar.pb", goldenDir: "testdata/golden/exemplar-single-no-minimal", quoteStyle: Single},
		"greeter-single":          {pbFile: "testdata/greeter.pb", goldenDir: "testdata/golden/greet-single-no-minimal", quoteStyle: Single},
		"exemplar-double-minimal": {pbFile: "testdata/exemplar.pb", goldenDir: "testdata/golden/exemplar-double-yes-minimal", quoteStyle: Double, minimal: true},
		"greeter-double-minimal":  {pbFile: "testdata/greeter.pb", goldenDir: "testdata/golden/greet-double-yes-minimal", quoteStyle: Double, minimal: true},
		"exemplar-single-minimal": {pbFile: "testdata/exemplar.pb", goldenDir: "testdata/golden/exemplar-single-yes-minimal", quoteStyle: Single, minimal: true},
		"greeter-single-minimal":  {pbFile: "testdata/greeter.pb", goldenDir: "testdata/golden/greet-single-yes-minimal", quoteStyle: Single, minimal: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()

			err := Generate(log.DiscardLogger, getFDS(t, tc.pbFile), dir, false, nil, &FormatterOptions{Lang: Jsonnet, QuoteStyle: tc.quoteStyle, Minimal: tc.minimal})
			require.NoError(t, err)
			err = Generate(log.DiscardLogger, getFDS(t, tc.pbFile), dir, false, nil, &FormatterOptions{Lang: JS, QuoteStyle: tc.quoteStyle, Minimal: tc.minimal})
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
