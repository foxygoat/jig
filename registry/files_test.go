package registry

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ensure Files implments ExtensionTypeResolver
var _ protoregistry.ExtensionTypeResolver = (*Files)(nil)

// ensure Files implments MessageTypeResolver
var _ protoregistry.MessageTypeResolver = (*Files)(nil)

func TestFindExtensionByName(t *testing.T) {
	tests := map[string]struct {
		extName string
		err     error
	}{
		"top-level extension":      {"regtest.ef1", nil},
		"nested extension":         {"regtest.ExtensionMessage.ef2", nil},
		"deeply nested extension":  {"regtest.ExtensionMessage.NestedExtension.ef3", nil},
		"other package extension":  {"regtest.base", nil},
		"imported extension":       {"google.api.http", nil},
		"unknown extension":        {"unknown.extension", protoregistry.NotFound},
		"non-extension descriptor": {"regtest.BaseMessage", protoregistry.NotFound},
	}

	f := newFiles(t)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			extName := protoreflect.FullName(tc.extName)
			et, err := f.FindExtensionByName(extName)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err, tc.extName)
				require.Equal(t, extName, et.TypeDescriptor().FullName())
			}
		})
	}
}

func TestFindExtensionByNumber(t *testing.T) {
	tests := map[string]struct {
		message     string
		fieldNumber int32
		extName     string
		err         error
	}{
		"top-level extension":     {"regtest.BaseMessage", 1000, "regtest.ef1", nil},
		"nested extension":        {"regtest.BaseMessage", 1001, "regtest.ExtensionMessage.ef2", nil},
		"deeply nested extension": {"regtest.BaseMessage", 1002, "regtest.ExtensionMessage.NestedExtension.ef3", nil},
		"other package extension": {"google.protobuf.MethodOptions", 56789, "regtest.base", nil},
		"imported extension":      {"google.protobuf.MethodOptions", 72295728, "google.api.http", nil},
		"unknown message":         {"regtest.Foo", 999, "unknown.message", protoregistry.NotFound},
		"unknown extension":       {"regtest.BaseMessage", 999, "unknown.extension", protoregistry.NotFound},
	}

	f := newFiles(t)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			messageName := protoreflect.FullName(tc.message)
			fieldNumber := protoreflect.FieldNumber(tc.fieldNumber)
			et, err := f.FindExtensionByNumber(messageName, fieldNumber)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err, tc.extName)
				extName := protoreflect.FullName(tc.extName)
				require.Equal(t, extName, et.TypeDescriptor().FullName())
			}
		})
	}
}

func TestGetExtensionsOfMessage(t *testing.T) {
	tests := map[string]struct {
		message string
		fields  []int32
	}{
		"package message":  {"regtest.BaseMessage", []int32{1000, 1001, 1002}},
		"imported message": {"google.protobuf.MethodOptions", []int32{56789, 72295728}},
		"unknown message":  {"regtest.Foo", nil},
	}

	f := newFiles(t)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			messageName := protoreflect.FullName(tc.message)
			ets := f.GetExtensionsOfMessage(messageName)
			var fields []int32
			for _, et := range ets {
				fields = append(fields, int32(et.TypeDescriptor().Number()))
			}
			require.ElementsMatch(t, tc.fields, fields)
		})
	}
}

func TestFindMessageByName(t *testing.T) {
	tests := map[string]struct {
		name string
		err  error
	}{
		"top-level message":      {"regtest.BaseMessage", nil},
		"nested message":         {"regtest.ExtensionMessage.NestedExtension", nil},
		"unknown message":        {"regtest.Foo", protoregistry.NotFound},
		"non-message descriptor": {"regtest.ef1", protoregistry.NotFound},
	}

	f := newFiles(t)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			messageName := protoreflect.FullName(tc.name)
			mt, err := f.FindMessageByName(messageName)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err, tc.name)
				require.Equal(t, messageName, mt.Descriptor().FullName())
			}
		})
	}
}

func TestFindMessageByURL(t *testing.T) {
	tests := map[string]struct {
		url string
		err error
	}{
		"simple url":       {"regtest.BaseMessage", nil},
		"hostname url":     {"example.com/regtest.BaseMessage", nil},
		"multiple slashes": {"example.com/foo/bar/regtest.BaseMessage", nil},
		"unknown message":  {"example.com/regtest.Foo", protoregistry.NotFound},
	}

	f := newFiles(t)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mt, err := f.FindMessageByURL(tc.url)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err, tc.url)
				expected := protoreflect.FullName("regtest.BaseMessage")
				require.Equal(t, expected, mt.Descriptor().FullName())
			}
		})
	}
}

func newFiles(t *testing.T) *Files {
	t.Helper()
	b, err := os.ReadFile("testdata/regtest.pb")
	require.NoError(t, err)
	fds := descriptorpb.FileDescriptorSet{}
	err = proto.Unmarshal(b, &fds)
	require.NoError(t, err)
	files, err := protodesc.NewFiles(&fds)
	require.NoError(t, err)
	return NewFiles(files)
}
