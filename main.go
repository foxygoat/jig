package main

import (
	"fmt"
	"os"
	"strings"

	"foxygo.at/jig/bones"
	"foxygo.at/jig/log"
	"foxygo.at/jig/serve"
	"foxygo.at/jig/serve/httprule"
	"github.com/alecthomas/kong"
	"github.com/alecthomas/protobuf/compiler"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

var version = "v0.0.0"

type config struct {
	Version  kong.VersionFlag `short:"V" help:"Print version information" group:"Other"`
	LogLevel log.LogLevel     `short:"L" help:"Log level" default:"error"`
	Serve    cmdServe         `cmd:"" help:"Serve GRPC services"`
	Bones    cmdBones         `cmd:"" help:"Generate skeleton jsonnet methods"`
}

type cmdServe struct {
	ProtoSet []string `short:"p" help:"Protoset .pb files containing service and deps"`

	Proto     []string `short:"P" help:"Proto source .proto files containing service"`
	ProtoPath []string `short:"I" help:"Import paths for --proto files' dependencies"`

	Listen string `short:"l" default:"localhost:8080" help:"TCP listen address"`
	HTTP   bool   `short:"h" help:"Serve on HTTP too, using HttpRule annotations"`

	Dirs []string `arg:"" help:"Directory containing method definitions and optionally protoset .pb file"`
}

type cmdBones struct {
	ProtoSet string `short:"p" help:"Protoset .pb file containing service and deps" xor:"proto"`

	Proto     string   `short:"P" help:"Proto source .proto file containing service" xor:"proto"`
	ProtoPath []string `short:"I" help:"Import paths for --proto files' dependencies"`

	MethodDir string   `short:"m" help:"Directory to write method definitions to"`
	Force     bool     `short:"f" help:"Overwrite existing bones files"`
	Targets   []string `arg:"" optional:"" help:"Target pkg/service/method to generate"`

	Language   bones.Lang       `help:"Target language" default:"jsonnet"`
	QuoteStyle bones.QuoteStyle `help:"Print single or double quotes" default:"double"`
}

func main() {
	cli := &config{}
	kctx := kong.Parse(cli, kong.Vars{"version": version})
	err := kctx.Run(cli.LogLevel)
	kctx.FatalIfErrorf(err)
}

func (cli *config) AfterApply() error {
	debug := strings.ToLower(os.Getenv("DEBUG"))
	if debug == "1" || debug == "yes" || debug == "true" {
		cli.LogLevel = log.LogLevelDebug
	}
	return nil
}

func (cs *cmdServe) Run(logLevel log.LogLevel) error {
	logger := log.NewLogger(os.Stderr, logLevel)
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
		h := httprule.NewServer(s.Files, s.UnknownHandler, logger, nil)
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

func (cb *cmdBones) Run(logLevel log.LogLevel) error {
	logger := log.NewLogger(os.Stderr, logLevel)
	fds := &descriptorpb.FileDescriptorSet{}
	if cb.ProtoSet != "" {
		logger.Debugf("read and unmarshal FileDescriptSet file %q", cb.ProtoSet)
		b, err := os.ReadFile(cb.ProtoSet)
		if err != nil {
			return err
		}
		if err := proto.Unmarshal(b, fds); err != nil {
			return err
		}
	} else {
		logger.Debugf("compiling FileDescriptorSet")
		var err error
		includeImports := true
		fds, err = compiler.Compile([]string{cb.Proto}, cb.ProtoPath, includeImports)
		if err != nil {
			return fmt.Errorf("cannot compile protos %v with import paths %v: %w", cb.Proto, cb.ProtoPath, err)
		}
	}

	opts := bones.NewFormatter(cb.Language, cb.QuoteStyle)
	return bones.Generate(logger, fds, cb.MethodDir, cb.Force, cb.Targets, opts)
}
