package main

import (
	"os"

	"foxygo.at/jig/bones"
	"foxygo.at/jig/log"
	"foxygo.at/jig/serve"
	"foxygo.at/jig/serve/httprule"
	"github.com/alecthomas/kong"
)

var version = "v0.0.0"

type config struct {
	Version kong.VersionFlag `short:"V" help:"Print version information" group:"Other"`
	Serve   cmdServe         `cmd:"" help:"Serve GRPC services"`
	Bones   cmdBones         `cmd:"" help:"Generate skeleton jsonnet methods"`
}

type cmdServe struct {
	ProtoSet []string     `short:"p" help:"Protoset .pb files containing service and deps"`
	LogLevel log.LogLevel `help:"Server logging level" default:"error"`
	Listen   string       `short:"l" default:"localhost:8080" help:"TCP listen address"`
	HTTP     bool         `short:"h" help:"Serve on HTTP too, using HttpRule annotations"`

	Dirs []string `arg:"" help:"Directory containing method definitions and protoset .pb file"`
}

type cmdBones struct {
	ProtoSet  string   `short:"p" help:"Protoset .pb file containing service and deps" required:""`
	MethodDir string   `short:"m" help:"Directory to write method definitions to"`
	Force     bool     `short:"f" help:"Overwrite existing bones files"`
	Targets   []string `arg:"" optional:"" help:"Target pkg/service/method to generate"`

	Language   bones.Lang       `help:"Target language" default:"jsonnet"`
	QuoteStyle bones.QuoteStyle `help:"Print single or double quotes" default:"double"`
}

func main() {
	cli := &config{}
	kctx := kong.Parse(cli, kong.Vars{"version": version})
	err := kctx.Run()
	kctx.FatalIfErrorf(err)
}

func (cs *cmdServe) Run() error {
	withLogger := serve.WithLogger(log.NewLogger(os.Stderr, cs.LogLevel))
	withProtosets := serve.WithProtosets(cs.ProtoSet...)
	dirs := serve.NewFSFromDirs(cs.Dirs...)
	s, err := serve.NewServer(serve.JsonnetEvaluator(), dirs, withLogger, withProtosets)
	if err != nil {
		return err
	}

	if cs.HTTP {
		h := httprule.NewServer(s.Files, s.UnknownHandler)
		s.SetHTTPHandler(h)
	}

	return s.ListenAndServe(cs.Listen)
}

func (cb *cmdBones) Run() error {
	opts := bones.FormatOptions{
		Lang:       cb.Language,
		QuoteStyle: cb.QuoteStyle,
	}
	return bones.Generate(cb.ProtoSet, cb.MethodDir, cb.Force, cb.Targets, opts)
}
