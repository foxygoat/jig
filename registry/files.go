// Package registry provides a type on top of protoregistry.Files that can be
// used as a protoregistry.ExtensionTypeResolver and a
// protoregistry.MessageTypeResolver. This allows a protoregistry.Files to be
// used as Resolver for protobuf encoding marshaling options.
package registry

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

type Files struct {
	protoregistry.Files
}

func NewFiles(f *protoregistry.Files) *Files {
	return &Files{Files: *f}
}

type extMatchFn func(protoreflect.ExtensionDescriptor) bool

// extensionContainer is implemented by FileDescriptor and MessageDescriptor.
// They are both "namespaces" that contain extensions and have "sub-namespaces".
type extensionContainer interface {
	Messages() protoreflect.MessageDescriptors
	Extensions() protoreflect.ExtensionDescriptors
}

func (f *Files) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	desc, err := f.FindDescriptorByName(field)
	if err != nil {
		return nil, err
	}
	ed, ok := desc.(protoreflect.ExtensionDescriptor)
	if !ok {
		return nil, protoregistry.NotFound
	}
	return dynamicpb.NewExtensionType(ed), nil
}

func (f *Files) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return findExtension(&f.Files, func(ed protoreflect.ExtensionDescriptor) bool {
		return ed.ContainingMessage().FullName() == message && ed.Number() == field
	})
}

func (f *Files) GetExtensionsOfMessage(message protoreflect.FullName) []protoreflect.ExtensionType {
	return walkExtensions(&f.Files, true, func(ed protoreflect.ExtensionDescriptor) bool {
		return ed.ContainingMessage().FullName() == message
	})
}

func findExtension(files *protoregistry.Files, pred extMatchFn) (protoreflect.ExtensionType, error) {
	ets := walkExtensions(files, false, pred)
	if len(ets) == 0 {
		return nil, protoregistry.NotFound
	}
	return ets[0], nil
}

func walkExtensions(files *protoregistry.Files, getAll bool, pred extMatchFn) []protoreflect.ExtensionType {
	var result []protoreflect.ExtensionType

	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		result = append(result, getExtensions(fd, getAll, pred)...)
		// continue if we are getting all extensions or have none so far
		return getAll || len(result) == 0
	})
	return result
}

func getExtensions(ec extensionContainer, getAll bool, pred extMatchFn) []protoreflect.ExtensionType {
	var result []protoreflect.ExtensionType

	eds := ec.Extensions()
	for i := 0; i < eds.Len() && (getAll || len(result) == 0); i++ {
		ed := eds.Get(i)
		if pred(ed) {
			result = append(result, dynamicpb.NewExtensionType(ed))
		}
	}

	mds := ec.Messages()
	for i := 0; i < mds.Len() && (getAll || len(result) == 0); i++ {
		md := mds.Get(i)
		result = append(result, getExtensions(md, getAll, pred)...)
	}

	return result
}

func (f *Files) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}

func (f *Files) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}
