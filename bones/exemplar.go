package bones

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// MethodExemplar returns an exemplar for a method, with an exemplar for the
// input message as a comment and a function returning an exemplar of the
// output message as the method implementation.
func MethodExemplar(md protoreflect.MethodDescriptor) exemplar {
	// Format the input message exemplar as a comment
	ime := MessageExemplar(md.Input())
	ime.append(",")
	if md.IsStreamingClient() && !md.IsStreamingServer() {
		ime.nest("stream: [", "],")
	} else {
		ime.prepend("request: ")
	}
	ime.nest("{", "}")
	ime.prefix("// ")

	// Format the output message exemplar
	ome := MessageExemplar(md.Output())
	ome.append(",")
	if md.IsStreamingServer() {
		ome.nest("stream: [", "],")
	} else {
		ome.prepend("response: ")
	}
	ome.nest("function(input) {", "}")

	var methodType string
	switch {
	case md.IsStreamingClient() && md.IsStreamingServer():
		methodType = " (Bidirectional streaming)"
	case md.IsStreamingClient():
		methodType = " (Client streaming)"
	case md.IsStreamingServer():
		methodType = " (Server streaming)"
	default:
		methodType = " (Unary)"
	}

	// Format the method exemplar
	var e exemplar
	e.line("// ", string(md.FullName()), methodType)
	e.line()
	e.line("// Input:")
	e.extend(ime)
	e.line()
	e.extend(ome)
	return e
}

// MessageExemplar returns an exemplar of a protobuf message as a JSON object
// with a field for every message field. Each field value is an exemplar of the
// type of the field. Oneof fields are emitted as comments as a message should
// not have more than one oneof specified.
func MessageExemplar(md protoreflect.MessageDescriptor) exemplar {
	var e exemplar

	for _, fd := range fields(md) {
		fe := FieldExemplar(fd)
		if fd.ContainingOneof() != nil {
			// Comment out one-of fields since they should not all be present.
			fe.prefix("// ")
		}
		e.extend(fe)
	}

	e.nest("{", "}")
	return e
}

// FieldExemplar returns an exemplar for a message field. It has the JSON name
// for the field prefixed and a comment appended to the first line describing
// the type of the field.
func FieldExemplar(fd protoreflect.FieldDescriptor) exemplar {
	e := FieldValueExemplar(fd)
	e.prepend(fd.JSONName() + ": ")
	e.append(",")

	// Add a description of the type to the end of the first line of the
	// exemplar as a comment. If part of a oneof, name the oneof too.
	if desc := typeDescription(fd); desc != "" {
		if od := fd.ContainingOneof(); od != nil {
			desc += " (one-of " + string(od.Name()) + ")"
		}
		e.lines[0] += "  // " + desc
	}

	return e
}

// FieldValueExemplar returns an exemplar for the value of a FieldDescriptor.
// The value is as per the exemplar form for the fields type. See the other
// *Exemplar functions for details of those.
//
// Map fields are emitted as repeated key/value message pairs in the expanded
// backward-compatible form, as opposed to {key: value, ...} objects.
//
// Repeated fields are emitted with a single element exemplar of the repeated
// type.
func FieldValueExemplar(fd protoreflect.FieldDescriptor) exemplar {
	var e exemplar
	switch fd.Kind() {
	case protoreflect.EnumKind:
		e = EnumExemplar(fd)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		e = MessageExemplar(fd.Message())
	default:
		e = ScalarExemplar(fd.Kind())
	}

	if fd.Cardinality() == protoreflect.Repeated {
		e.nestCompact("[", "]")
	}

	return e
}

// EnumExemplar returns an exemplar with a sample value for the enum. An empty
// exemplar is returned if the field is not an enum.
//
// Enum exemplars are emitted as a string with the name of the second enum if
// there is more than one enum value, otherwise the first enum. The second enum
// is preferred as often the first enum is the "invalid" value for that enum.
func EnumExemplar(fd protoreflect.FieldDescriptor) exemplar {
	var e exemplar
	if fd.Kind() != protoreflect.EnumKind {
		return e
	}
	ev := fd.Enum().Values()
	name := ev.Get(0).Name()
	if ev.Len() > 1 {
		name = ev.Get(1).Name()
	}
	e.line(`"`, string(name), `"`)
	return e
}

// ScalarExemplar returns an exemplar with a value for basic kinds that have a
// single value (scalars). An empty exemplar is returned for other kinds.
func ScalarExemplar(kind protoreflect.Kind) exemplar {
	var e exemplar
	switch kind {
	case protoreflect.BoolKind:
		e.line("false")
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Uint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind,
		protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		e.line("0")
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		e.line("0.0")
	case protoreflect.StringKind, protoreflect.BytesKind:
		e.line(`""`)
	}
	return e
}

// typeDescription returns a string description of a field's type.
func typeDescription(fd protoreflect.FieldDescriptor) string {
	if fd.IsMap() {
		return fmt.Sprintf("map<%s, %s>", typeDescription(fd.MapKey()), typeDescription(fd.MapValue()))
	}
	result := ""
	switch fd.Kind() {
	case protoreflect.EnumKind:
		result = string(fd.Enum().Name())
	case protoreflect.MessageKind, protoreflect.GroupKind:
		result = string(fd.Message().Name())
	case protoreflect.BoolKind,
		protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Uint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind,
		protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind,
		protoreflect.FloatKind, protoreflect.DoubleKind,
		protoreflect.StringKind, protoreflect.BytesKind:
		result = fd.Kind().String()
	}

	if fd.IsList() && result != "" {
		result = "repeated " + result
	}

	return result
}

// exemplar is type of builder for constucting exemplars for protobuf messages.
// It provides operations for combining exemplars with strings and other
// exemplars, maintaining the line-by-line nature of the exemplar.
type exemplar struct {
	lines []string
}

// line adds a line to the exemplar made up of the arguments combined as per
// fmt.Print.
func (e *exemplar) line(a ...interface{}) {
	e.lines = append(e.lines, fmt.Sprint(a...))
}

// extend adds another exemplar to the end of this one.
func (e *exemplar) extend(other exemplar) {
	e.lines = append(e.lines, other.lines...)
}

// prefix prepends a string prefix to every line of the exemplar.
func (e *exemplar) prefix(prefix string) {
	for i := range e.lines {
		e.lines[i] = prefix + e.lines[i]
	}
}

// nest indents the exemplar and places a prefix and suffix line before and after
// the indented exemplar.
func (e *exemplar) nest(prefix, suffix string) {
	// indent exemplar before nesting
	e.prefix("  ")
	e.lines = append([]string{prefix}, e.lines...)
	e.lines = append(e.lines, suffix)
}

// nestCompact places the prefix and suffix on the same line for single line
// exemplars, or calls nest() for multi-line exemplars.
func (e *exemplar) nestCompact(prefix, suffix string) {
	if len(e.lines) == 1 {
		e.lines[0] = prefix + e.lines[0] + suffix
	} else {
		e.nest(prefix, suffix)
	}
}

// prepend inserts a string at the start of the first line of the exemplar.
func (e *exemplar) prepend(prefix string) {
	if len(e.lines) == 0 {
		e.line()
	}
	e.lines[0] = prefix + e.lines[0]
}

// append adds a string to the end of the last line of the exemplar.
func (e *exemplar) append(suffix string) {
	if len(e.lines) == 0 {
		e.line()
	}
	e.lines[len(e.lines)-1] += suffix
}

// String returns the exemplar as a single string with newline separators.
// It implements the fmt.Stringer interface.
func (e exemplar) String() string {
	return strings.Join(e.lines, "\n")
}
