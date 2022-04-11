package main

import (
	"fmt"
	"os"

	"foxygo.at/jig/bones"
	"foxygo.at/jig/log"
	"foxygo.at/jig/serve"
	"foxygo.at/jig/serve/httprule"
	"github.com/alecthomas/kong"
	"github.com/alecthomas/protobuf/compiler"
	"google.golang.org/protobuf/proto"
)

var version = "v0.0.0"

type config struct {
	Version kong.VersionFlag `short:"V" help:"Print version information" group:"Other"`
	Serve   cmdServe         `cmd:"" help:"Serve GRPC services"`
	Bones   cmdBones         `cmd:"" help:"Generate skeleton jsonnet methods"`
}

type cmdServe struct {
	ProtoSet []string `short:"p" help:"Protoset .pb files containing service and deps"`

	Proto     []string `short:"P" help:"Proto source .proto files containing service"`
	ProtoPath []string `short:"I" help:"Import paths for --proto files' dependencies"`

	LogLevel log.LogLevel `help:"Server logging level" default:"error"`
	Listen   string       `short:"l" default:"localhost:8080" help:"TCP listen address"`
	HTTP     bool         `short:"h" help:"Serve on HTTP too, using HttpRule annotations"`

	Dirs []string `arg:"" help:"Directory containing method definitions and optionally protoset .pb file"`
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
	logger := log.NewLogger(os.Stderr, cs.LogLevel)
	opts, err := cs.getServerOptions(logger)
	if err != nil {
		return err
	}
	dirs := serve.NewFSFromDirs(cs.Dirs...)
	s, err := serve.NewServer(serve.JsonnetEvaluator(), dirs, opts...)
	if err != nil {
		return err
	}

	if cs.HTTP {
		h := httprule.NewServer(s.Files, s.UnknownHandler, logger)
		s.SetHTTPHandler(h)
	}

	return s.ListenAndServe(cs.Listen)
}

func (cs *cmdServe) getServerOptions(logger log.Logger) ([]serve.Option, error) {
	opts := []serve.Option{serve.WithLogger(logger), serve.WithProtosets(cs.ProtoSet...)}
	if len(cs.Proto) != 0 {
		includeImports := true
		fds, err := compiler.Compile(cs.Proto, cs.ProtoPath, includeImports)
		if err != nil {
			return nil, fmt.Errorf("cannot compile protos %v with import paths %v: %w", cs.Proto, cs.ProtoPath, err)
		}
		opts = append(opts, serve.WithFileDescriptorSets(fds))
	}
	return opts, nil
}

func (cb *cmdBones) Run() error {
	opts := bones.FormatOptions{
		Lang:       cb.Language,
		QuoteStyle: cb.QuoteStyle,
	}
	return bones.Generate(cb.ProtoSet, cb.MethodDir, cb.Force, cb.Targets, opts)
}
