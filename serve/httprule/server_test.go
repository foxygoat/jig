package httprule

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern, path string
		vars          map[string]string
	}{
		{"/api/hello/{name}", "/api/hello/nobody/true", nil},
		{"/api/hello/{name}", "/api/hello/nobody", map[string]string{"name": "nobody"}},
		{"/v1/{name=messages/*}", "/v1/messages/12345", map[string]string{"name": "messages/12345"}},
	}
	for _, test := range tests {
		require.Equalf(t,
			test.vars,
			matchPath(test.pattern, test.path),
			"pattern=%q, path=%q", test.pattern, test.path)
	}
}
