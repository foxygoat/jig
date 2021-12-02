package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.T) {
	b := &bytes.Buffer{}
	out = b
	main()
	require.Equal(t, "Hello 🌏\n", b.String())
}
