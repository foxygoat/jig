package httprule

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"foxygo.at/jig/pb/httpgreet"
)

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern, path string
		vars          map[string]string
	}{
		{"/api/hello/{name}", "/api/hello/nobody/true", nil},
		{"/api/hello/{name}", "/api/hello/nobody", map[string]string{"name": "nobody"}},
		{"/v1/{name=messages/*}", "/v1/messages/12345", map[string]string{"name": "messages/12345"}},
		{"/v1/{name=messages/*}", "/v1/messages/12345", map[string]string{"name": "messages/12345"}},
		// TODO: Support **
		//{"/v1/{name=**}", "/v1/messages/12345", map[string]string{"name": "messages/12345"}},
	}
	for _, test := range tests {
		require.Equalf(t,
			test.vars,
			matchPath(test.pattern, test.path),
			"pattern=%q, path=%q", test.pattern, test.path)
	}
}

func TestDecodeRequest(t *testing.T) {
	tests := []struct {
		name        string
		rule        *annotations.HttpRule
		path        string
		contentType string
		body        func() ([]byte, error)
		expected    *httpgreet.HelloRequest
	}{
		{
			name: "get basic path param",
			rule: &annotations.HttpRule{
				Pattern: &annotations.HttpRule_Get{Get: "/api/hello/{first_name}"},
			},
			path:     "/api/hello/Rob",
			expected: &httpgreet.HelloRequest{FirstName: "Rob"},
		},
		{
			name: "post json",
			rule: &annotations.HttpRule{
				Pattern: &annotations.HttpRule_Post{Post: "/api/hello"},
				Body:    "*",
			},
			contentType: "application/json; charset=utf-8",
			path:        "/api/hello",
			body: func() ([]byte, error) {
				return protojson.Marshal(&httpgreet.HelloRequest{FirstName: "Rob", LastName: "Robinson"})
			},
			expected: &httpgreet.HelloRequest{FirstName: "Rob", LastName: "Robinson"},
		},
		{
			name: "post json missing content type",
			rule: &annotations.HttpRule{
				Pattern: &annotations.HttpRule_Post{Post: "/api/hello"},
				Body:    "*",
			},
			path: "/api/hello",
			body: func() ([]byte, error) {
				return protojson.Marshal(&httpgreet.HelloRequest{FirstName: "Rob", LastName: "Robinson"})
			},
			expected: &httpgreet.HelloRequest{FirstName: "Rob", LastName: "Robinson"},
		},
		{
			name: "post binary proto missing content type",
			rule: &annotations.HttpRule{
				Pattern: &annotations.HttpRule_Post{Post: "/api/hello"},
				Body:    "*",
			},
			path:        "/api/hello",
			contentType: ContentTypeBinaryProto,
			body: func() ([]byte, error) {
				return proto.Marshal(&httpgreet.HelloRequest{FirstName: "Rob", LastName: "Robinson"})
			},
			expected: &httpgreet.HelloRequest{FirstName: "Rob", LastName: "Robinson"},
		},
		{
			name: "post ignore body",
			rule: &annotations.HttpRule{
				Pattern: &annotations.HttpRule_Post{Post: "/api/hello"},
			},
			path:        "/api/hello",
			contentType: ContentTypeBinaryProto,
			body: func() ([]byte, error) {
				return proto.Marshal(&httpgreet.HelloRequest{FirstName: "Rob", LastName: "Robinson"})
			},
			expected: &httpgreet.HelloRequest{},
		},
		{
			name: "post merge path param and body",
			rule: &annotations.HttpRule{
				Pattern: &annotations.HttpRule_Post{Post: "/api/hello/{first_name}"},
				Body:    "*",
			},
			path:        "/api/hello/Rob",
			contentType: ContentTypeBinaryProto,
			body: func() ([]byte, error) {
				return proto.Marshal(&httpgreet.HelloRequest{LastName: "Robinson"})
			},
			expected: &httpgreet.HelloRequest{FirstName: "Rob", LastName: "Robinson"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body io.Reader

			method := "GET"
			if test.body != nil {
				raw, err := test.body()
				require.NoError(t, err)
				body = bytes.NewReader(raw)
				method = "POST"
			}
			req := httptest.NewRequest(method, test.path, body)
			if test.contentType != "" {
				req.Header.Set("Content-Type", test.contentType)
			}
			vars := MatchRequest(test.rule, req)
			require.NotNil(t, vars)

			actual := &httpgreet.HelloRequest{}
			require.NoError(t, DecodeRequest(test.rule, vars, req, actual))

			require.Truef(t, proto.Equal(test.expected, actual), "expected: %s,\nactual: %s", test.expected, actual)
		})
	}
}
