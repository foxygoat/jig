package bones

import (
	"fmt"
	"strings"
)

type Lang int

const (
	Jsonnet Lang = iota
	JS
)

func (l *Lang) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "jsonnet":
		*l = Jsonnet
	case "js", "javascript":
		*l = JS
	default:
		return fmt.Errorf("invalid language %q", text)
	}
	return nil
}

func (l Lang) String() string {
	switch l {
	case Jsonnet:
		return "jsonnet"
	case JS:
		return "js"
	}
	panic(l)
}

type QuoteStyle int

const (
	Double QuoteStyle = iota
	Single
)

func (qs *QuoteStyle) UnmarshalText(text []byte) error {
	switch string(text) {
	case "double":
		*qs = Double
	case "single":
		*qs = Single
	default:
		return fmt.Errorf("invalid quote style %q", text)
	}
	return nil
}

func (qs QuoteStyle) String() string {
	switch qs {
	case Double:
		return "double"
	case Single:
		return "single"
	}
	panic(qs)
}
