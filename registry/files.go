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

func (f *Files) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return findExtension(&f.Files, func(ed protoreflect.ExtensionDescriptor) bool {
		return ed.FullName() == field
	})
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
	var eds []protoreflect.ExtensionDescriptor

	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		eds = append(eds, rangeExtensions(fd.Extensions(), getAll, pred)...)
		if len(eds) > 0 && !getAll {
			return false // stop after the first found
		}
		eds = append(eds, rangeMessages(fd.Messages(), getAll, pred)...)
		if len(eds) > 0 && !getAll {
			return false // stop after the first found
		}
		return true
	})

	result := make([]protoreflect.ExtensionType, len(eds))
	for i, ed := range eds {
		result[i] = dynamicpb.NewExtensionType(ed)
	}
	return result
}

func rangeExtensions(eds protoreflect.ExtensionDescriptors, getAll bool, pred extMatchFn) []protoreflect.ExtensionDescriptor {
	var result []protoreflect.ExtensionDescriptor

	for i := 0; i < eds.Len(); i++ {
		ed := eds.Get(i)
		if pred(ed) {
			result = append(result, ed)
			if !getAll {
				break
			}
		}
	}
	return result
}

func rangeMessages(mds protoreflect.MessageDescriptors, getAll bool, pred extMatchFn) []protoreflect.ExtensionDescriptor {
	var result []protoreflect.ExtensionDescriptor

	for i := 0; i < mds.Len(); i++ {
		md := mds.Get(i)
		result = append(result, rangeExtensions(md.Extensions(), getAll, pred)...)
		if len(result) > 0 && !getAll {
			break
		}
		result = append(result, rangeMessages(md.Messages(), getAll, pred)...)
		if len(result) > 0 && !getAll {
			break
		}
	}
	return result
}

func (f *Files) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}

func (f *Files) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}
