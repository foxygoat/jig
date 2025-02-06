package httprule

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Collect any annotations.Rule entries from a proto method.
func Collect(md protoreflect.MethodDescriptor) (rules []*annotations.HttpRule) {
	md.Options().ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if fd.Kind() == protoreflect.MessageKind {
			if rule, ok := value.Message().Interface().(*annotations.HttpRule); ok {
				rules = append(rules, rule)
			}
		}
		return true
	})
	return
}

// MatchRequest returns a non-nil map of extracted path vars if a http.Request
// matches a rule's request path and method.
func MatchRequest(rule *annotations.HttpRule, req *http.Request) map[string]string {
	method, pattern := extractSelect(rule)
	if req.Method != method {
		return nil
	}
	return matchPath(pattern, req.URL.Path)
}

const (
	ContentTypeBinaryProto = "application/x-protobuf"
	ContentTypeJSON        = "application/json"
)

// DecodeRequest parses a http.Request, using a HttpRule, into a target message.
func DecodeRequest(rule *annotations.HttpRule, pathVars map[string]string, req *http.Request, target proto.Message) error {
	if err := decodeBody(rule, req, target); err != nil {
		return err
	}

	tb := target.ProtoReflect()
	// First set fields from path vars.
	for key, value := range pathVars {
		if err := setField(target, key, value); err != nil {
			return fmt.Errorf("%s: field %s: %w", tb.Descriptor().FullName(), key, err)
		}
	}

	return nil
}

func decodeBody(rule *annotations.HttpRule, req *http.Request, target proto.Message) error {
	if rule.Body != "*" {
		// If body isn't set, the request body is dropped.
		// TODO: Support field paths other than "*".
		return nil
	}
	mediaType := ContentTypeJSON
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		contentType = req.Header.Get("Accept")
	}
	var err error
	if contentType != "" {
		mediaType, _, err = mime.ParseMediaType(contentType)
		if err != nil {
			return err
		}
	}
	var unmarshal func(b []byte, m proto.Message) error
	switch mediaType {
	case ContentTypeBinaryProto:
		unmarshal = proto.Unmarshal
	case ContentTypeJSON:
		unmarshal = protojson.Unmarshal
	default:
		return fmt.Errorf("invalid content type %s", contentType)
	}

	raw, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if err = unmarshal(raw, target); err != nil {
		return err
	}

	return nil
}

var matchSplit = regexp.MustCompile(`/|{[^}]*}|[^/]+`)

func matchPath(pattern, path string) map[string]string {
	vars := map[string]string{}
	patparts := matchSplit.FindAllString(pattern, -1)
	pathparts := matchSplit.FindAllString(path, -1)
	for _, pat := range patparts {
		if len(pathparts) == 0 {
			return nil
		}
		pathpart := pathparts[0]
		pathparts = pathparts[1:]
		if pathpart == pat {
			continue
		}

		if !strings.HasPrefix(pat, "{") {
			return nil
		}

		key := pat[1 : len(pat)-1]

		// Do glob match {name=foo/*}
		keyparts := strings.SplitN(key, "=", 2)
		if len(keyparts) > 1 {
			key = keyparts[0]
			glob := keyparts[1]
			remainder := pathpart + strings.Join(pathparts, "")
			if ok, _ := filepath.Match(glob, remainder); ok {
				pathparts = nil
				vars[key] = remainder
				break
			} else {
				return nil
			}
		}
		vars[key] = pathpart
	}
	if len(pathparts) > 0 {
		return nil
	}
	return vars
}

func extractSelect(rule *annotations.HttpRule) (method, path string) {
	switch pattern := rule.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		return http.MethodGet, pattern.Get
	case *annotations.HttpRule_Put:
		return http.MethodPut, pattern.Put
	case *annotations.HttpRule_Post:
		return http.MethodPost, pattern.Post
	case *annotations.HttpRule_Delete:
		return http.MethodDelete, pattern.Delete
	case *annotations.HttpRule_Patch:
		return http.MethodPatch, pattern.Patch
	case *annotations.HttpRule_Custom:
		return pattern.Custom.Kind, pattern.Custom.Path
	default:
		panic(fmt.Sprintf("%T", pattern))
	}
}

func setField(target proto.Message, name, valstr string) error {
	m := target.ProtoReflect()
	fd := m.Descriptor().Fields().ByTextName(name)
	if fd == nil {
		return fmt.Errorf("no such field")
	}

	var val interface{}
	var err error
	switch fd.Kind() {
	case protoreflect.BoolKind:
		val, err = strconv.ParseBool(valstr)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		var v int64
		v, err = strconv.ParseInt(valstr, 10, 32)
		val = int32(v)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		var v uint64
		v, err = strconv.ParseUint(valstr, 10, 32)
		val = uint32(v)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		var v int64
		v, err = strconv.ParseInt(valstr, 10, 64)
		val = int64(v)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		var v uint64
		v, err = strconv.ParseUint(valstr, 10, 64)
		val = uint64(v)
	case protoreflect.FloatKind:
		var v float64
		v, err = strconv.ParseFloat(valstr, 32)
		val = float32(v)
	case protoreflect.DoubleKind:
		val, err = strconv.ParseFloat(valstr, 64)
	case protoreflect.StringKind:
		val, err = valstr, nil
	case protoreflect.BytesKind:
		val, err = []byte(valstr), nil
	default:
		err = fmt.Errorf("unsupported type %s", fd.Kind())
	}
	if err != nil {
		return err
	}

	value := protoreflect.ValueOf(val)
	if fd.IsList() {
		m.Mutable(fd).List().Append(value)
	} else {
		m.Set(fd, value)
	}
	return nil
}
