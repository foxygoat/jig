package bones

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// Formatter is a configurable target language exemplar generator.
type formatter struct {
	messagesSeen map[protoreflect.FullName]bool
	opts         *FormatterOptions
}

type FormatterOptions struct {
	Lang       Lang
	QuoteStyle QuoteStyle
	Minimal    bool
}

// NewFormatter creates a new Formatter for given language, quote style,
// and weather to use just input/output type names or flash out all
// data structures on their first occurrence.
func newFormatter(opts *FormatterOptions) *formatter {
	return &formatter{
		opts:         opts,
		messagesSeen: map[protoreflect.FullName]bool{},
	}
}

// Return the file extension for the given language.
func (f *formatter) Extension() string {
	return "." + f.opts.Lang.String()
}

// Return the file extension for the given language.
func (f *formatter) quote(s string) string {
	if f.opts.QuoteStyle == Double {
		return `"` + s + `"`
	}
	return `'` + s + `'`
}

func (f *formatter) reset() {
	f.messagesSeen = map[protoreflect.FullName]bool{}
}

// MethodExemplar returns an exemplar for a method, with an exemplar for the
// input message as a comment and a function returning an exemplar of the
// output message as the method implementation.
func (f *formatter) MethodExemplar(md protoreflect.MethodDescriptor) exemplar {
	f.reset()
	// Format the input message exemplar as a comment
	var ime exemplar
	if f.opts.Minimal {
		ime = f.minimalMessageExemplar(md.Input())
	} else {
		ime = f.MessageExemplar(md.Input(), "  // "+string(md.Input().Name()))
	}
	ime.append(",")
	if md.IsStreamingClient() && !md.IsStreamingServer() {
		ime.nest("stream: [", "],")
	} else {
		ime.prepend("request: ")
	}
	ime.nest("{", "}")
	ime.prefix("// ")

	// Format the output message exemplar
	f.reset()
	var ome exemplar
	if f.opts.Minimal {
		ome = f.minimalMessageExemplar(md.Output())
	} else {
		ome = f.MessageExemplar(md.Output(), "  // "+string(md.Output().Name()))
	}
	ome.append(",")
	if md.IsStreamingServer() {
		ome.nest("stream: [", "],")
	} else {
		ome.prepend("response: ")
	}
	if f.opts.Lang == Jsonnet {
		ome.nest("function(input) {", "}")
	} else {
		ome.nest("return {", "}")
		ome.nest("function "+string(md.Name())+"(input) {", "}")
	}

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

func (f *formatter) minimalMessageExemplar(md protoreflect.MessageDescriptor) exemplar {
	var e exemplar
	e.line("{  // " + string(md.FullName()))
	e.line("}")
	return e
}

// MessageExemplar returns an exemplar of a protobuf message as a JSON object
// with a field for every message field. Each field value is an exemplar of the
// type of the field. Oneof fields are emitted as comments as a message should
// not have more than one oneof specified.
func (f *formatter) MessageExemplar(md protoreflect.MessageDescriptor, headerPostfix string) exemplar {
	var e exemplar

	if strings.HasPrefix(string(md.FullName()), "google.protobuf.") {
		if e = f.WellKnownExemplar(md); len(e.lines) > 0 {
			return e
		}
	}

	if f.messagesSeen[md.FullName()] {
		e.line("{}")
		return e
	}

	f.messagesSeen[md.FullName()] = true
	for _, fd := range fields(md) {
		fe := f.FieldExemplar(fd)
		if fd.ContainingOneof() != nil {
			// Comment out one-of fields since they should not all be present.
			fe.prefix("// ")
		}
		e.extend(fe)
	}

	e.nest("{"+headerPostfix, "}")
	return e
}

// WellKnownExemplar returns an exemplar for a well-known type (those in the
// google.protobuf package). These are typically messages that are rendered in
// JSON as a single field, rather than as an object.
//
// https://developers.google.com/protocol-buffers/docs/reference/google.protobuf
func (f *formatter) WellKnownExemplar(md protoreflect.MessageDescriptor) exemplar {
	var e exemplar
	switch string(md.Name()) {
	case "Api", "Enum", "EnumValue", "Field", "Method", "Mixin", "Option", "SourceContext", "Type":
		return e // empty exemplar. will be formatted as a message
	case "Any":
		// Emit an Any that can be read back in without modification
		// Duration chosen at random, almost. Also for its simplicity.
		e.line(f.quote("@type") + ": " + f.quote("type.googleapis.com/google.protobuf.Duration") + ",")
		e.line("value: " + f.quote("0s") + ",")
		e.nest("{", "}")
	case "BoolValue", "BytesValue", "DoubleValue", "FloatValue",
		"Int32Value", "Int64Value", "StringValue", "UInt32Value", "UInt64Value":
		return f.FieldValueExemplar(md.Fields().ByName("value"))
	case "Duration":
		e.line(f.quote("0s"))
	case "Empty":
		e.line("{}")
	case "FieldMask":
		e.line(f.quote("field1.field2,field3"))
	case "ListValue":
		return f.FieldValueExemplar(md.Fields().ByName("values"))
	case "Struct":
		e = f.FieldValueExemplar(md.Fields().Get(0).MapValue())
		e.prepend("structField: ")
		e.append(",")
		e.nest("{", "}")
	case "Timestamp":
		e.line(f.quote("2006-01-02T15:04:05.999999999Z"))
	case "Value":
		e.line(f.quote("https://developers.google.com/protocol-buffers/docs/reference/google.protobuf#value"))
	}
	return e
}

// FieldExemplar returns an exemplar for a message field. It has the JSON name
// for the field prefixed and a comment appended to the first line describing
// the type of the field.
func (f *formatter) FieldExemplar(fd protoreflect.FieldDescriptor) exemplar {
	desc := f.typeDescription(fd)
	seen := false
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		seen = f.messagesSeen[fd.Message().FullName()]
	}

	e := f.FieldValueExemplar(fd)
	e.prepend(fd.JSONName() + ": ")
	e.append(",")

	// Add a description of the type to the end of the first line of the
	// exemplar as a comment. If part of a oneof, name the oneof too.
	if desc != "" {
		seenComment := "see example above"
		if od := fd.ContainingOneof(); od != nil && desc != "" {
			desc += " (one-of " + string(od.Name())
			if seen {
				desc += ", " + seenComment
			}
			desc += ")"
		} else if seen {
			desc += " (" + seenComment + ")"
		}

		e.lines[0] += "  // " + desc
	}

	return e
}

// FieldValueExemplar returns an exemplar for the value of a FieldDescriptor.
// The value is as per the exemplar form for the fields type. See the other
// *Exemplar functions for details of those.
//
// Map fields are emitted as objects with non-string keys converted to strings.
//
// Repeated fields are emitted with a single element exemplar of the repeated
// type.
func (f *formatter) FieldValueExemplar(fd protoreflect.FieldDescriptor) exemplar {
	var e exemplar

	switch kind := fd.Kind(); {
	case fd.IsMap():
		e = f.MapExemplar(fd)
	case kind == protoreflect.EnumKind:
		e = f.EnumExemplar(fd)
	case kind == protoreflect.MessageKind || kind == protoreflect.GroupKind:
		e = f.MessageExemplar(fd.Message(), "")
	default:
		e = f.ScalarExemplar(fd.Kind())
	}

	// Maps appear as repeated in proto representation but are not rendered
	// that way in JSON.
	if fd.Cardinality() == protoreflect.Repeated && !fd.IsMap() {
		e.nestCompact("[", "]")
	}

	return e
}

// MapExemplar returns an exemplar for a sample value for a map. An empty
// exemplar is returned if the field is not a map.
//
// Maps exemplars are emitted as an object with a single entry with the key
// type as the key and the value type as the value. Non-string keys are
// converted to strings.
//
// Reference: https://protobuf.dev/programming-guides/editions/#maps
func (f *formatter) MapExemplar(fd protoreflect.FieldDescriptor) exemplar {
	var e exemplar
	if !fd.IsMap() {
		return e
	}

	e = f.FieldValueExemplar(fd.MapValue())

	var key string
	switch fd.MapKey().Kind() {
	case protoreflect.BoolKind:
		key = `"false"`
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Uint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind,
		protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		key = `"0"`
	case protoreflect.StringKind:
		key = `"key"`
	default:
		key = `"invalid_type"`
	}

	e.prepend(key + ": ")
	e.append(",")
	e.nest("{", "}")

	return e
}

// EnumExemplar returns an exemplar with a sample value for the enum. An empty
// exemplar is returned if the field is not an enum.
//
// Enum exemplars are emitted as a string with the name of the second enum if
// there is more than one enum value, otherwise the first enum. The second enum
// is preferred as often the first enum is the "invalid" value for that enum.
func (f *formatter) EnumExemplar(fd protoreflect.FieldDescriptor) exemplar {
	var e exemplar
	if fd.Kind() != protoreflect.EnumKind {
		return e
	}

	// The well-known google.protobuf.NullValue enum renders as "null"
	if fd.Enum().FullName() == "google.protobuf.NullValue" {
		e.line("null")
		return e
	}

	ev := fd.Enum().Values()
	name := ev.Get(0).Name()
	if ev.Len() > 1 {
		name = ev.Get(1).Name()
	}
	e.line(f.quote(string(name)))
	return e
}

// ScalarExemplar returns an exemplar with a value for basic kinds that have a
// single value (scalars). An empty exemplar is returned for other kinds.
func (f *formatter) ScalarExemplar(kind protoreflect.Kind) exemplar {
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
		e.line(f.quote(""))
	}
	return e
}

// typeDescription returns a string description of a field's type.
func (f *formatter) typeDescription(fd protoreflect.FieldDescriptor) string {
	if fd.IsMap() {
		return fmt.Sprintf("map<%s, %s>", f.typeDescription(fd.MapKey()), f.typeDescription(fd.MapValue()))
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
