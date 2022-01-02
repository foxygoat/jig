package bones

import "google.golang.org/protobuf/reflect/protoreflect"

func services(fd protoreflect.FileDescriptor) []protoreflect.ServiceDescriptor {
	sds := fd.Services()
	result := make([]protoreflect.ServiceDescriptor, sds.Len())
	for i := 0; i < sds.Len(); i++ {
		result[i] = sds.Get(i)
	}
	return result
}

func methods(sd protoreflect.ServiceDescriptor) []protoreflect.MethodDescriptor {
	mds := sd.Methods()
	result := make([]protoreflect.MethodDescriptor, mds.Len())
	for i := 0; i < mds.Len(); i++ {
		result[i] = mds.Get(i)
	}
	return result
}

func fields(md protoreflect.MessageDescriptor) []protoreflect.FieldDescriptor {
	fields := md.Fields()
	result := make([]protoreflect.FieldDescriptor, fields.Len())
	for i := 0; i < fields.Len(); i++ {
		result[i] = fields.Get(i)
	}
	return result
}
