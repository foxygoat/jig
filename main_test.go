package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"foxygo.at/jig/log"
	"foxygo.at/jig/pb/exemplar"
	"foxygo.at/jig/serve"
	"foxygo.at/jig/serve/httprule"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestHTTPRuleServer(t *testing.T) {
	c := cmdServe{
		Proto:     []string{"httpgreet/httpgreet.proto"},
		ProtoPath: []string{"proto"},
	}
	opts, err := c.getServerOptions(log.DiscardLogger)
	require.NoError(t, err)

	ts := serve.NewUnstartedTestServer(serve.JsonnetEvaluator(), os.DirFS("serve/testdata/httpgreet"), opts...)
	ts.SetHTTPHandler(httprule.NewServer(ts.Files, ts.UnknownHandler, log.DiscardLogger, nil))
	ts.Start()
	defer ts.Stop()

	baseURL := "http://" + ts.Addr()
	resp, err := http.Get(baseURL + "/api/greet/hello/Dolly")
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"greeting":"httpgreet: Hello, Dolly"}`, string(b))

	resp, err = http.Post(baseURL+"/api/greet/hello", "application/json", strings.NewReader(`{"firstName": "Kitty"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"greeting":"Thanks for the post, Kitty"}`, string(b))

	resp, err = http.Post(baseURL+"/api/greet/world", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"greeting":"Thanks for the post and the path, world"}`, string(b))
}

func TestExemplar(t *testing.T) {
	c := cmdServe{
		ProtoSet: []string{"pb/exemplar/exemplar.pb"},
	}
	opts, err := c.getServerOptions(log.DiscardLogger)
	require.NoError(t, err)

	ts := serve.NewUnstartedTestServer(serve.JsonnetEvaluator(), os.DirFS("bones/testdata/golden/exemplar-single-no-minimal"), opts...)
	ts.SetHTTPHandler(httprule.NewServer(ts.Files, ts.UnknownHandler, log.DiscardLogger, nil))
	ts.Start()
	defer ts.Stop()

	cc, err := grpc.Dial(ts.Addr(), grpc.WithInsecure())
	require.NoError(t, err)
	defer cc.Connect()
	client := exemplar.NewExemplarClient(cc)

	req := &exemplar.SampleRequest{Name: "Grace"}
	resp, err := client.Sample(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.GetABool())
	require.Equal(t, map[int32]bool{0: false}, resp.GetAMap())

	want := &exemplar.SampleResponse{
		ABool:     false,
		AInt32:    0,
		ASint32:   0,
		ASfixed32: 0,
		AUint32:   0,
		AFixed32:  0,
		AInt64:    0,
		ASint64:   0,
		ASfixed64: 0,
		AUint64:   0,
		AFixed64:  0,
		AFloat:    0,
		ADouble:   0,
		AString:   "",
		ABytes:    nil,
		AEnum:     exemplar.SampleResponse_SAMPLE_ENUM_FIRST,
		AMessage: &exemplar.SampleResponse_SampleMessage1{
			Repeat: []int32{0},
		},
		AMap: map[int32]bool{0: false},
		ADeepMap: map[string]*exemplar.SampleResponse_SampleMessage2{
			"": {
				Weird_FieldName_1_: "",
				AStringList:        []string{""},
				AMsgList:           []*exemplar.SampleResponse_SampleMessage1{{}},
			},
		},
		AIntList:     []int32{0},
		AEnumList:    []exemplar.SampleResponse_SampleEnum{exemplar.SampleResponse_SAMPLE_ENUM_FIRST},
		AMessageList: []*exemplar.SampleResponse_SampleMessage1{{}},
		Recursive:    &exemplar.SampleResponse{},
	}
	diff := cmp.Diff(want, resp, protocmp.Transform())
	require.Empty(t, diff)
}
