package main

import (
	"fmt"
	"io"
	"os"

	"foxygo.at/jig/internal/client"
	"github.com/alecthomas/kong"
)

var version = "v0.0.0"

type config struct {
	Address string           `help:"hostname:port" default:"localhost:8080"`
	Stream  string           `short:"s" enum:"none,client,server,bidi" default:"none" help:"Stream requests/responses"`
	Names   []string         `arg:"" help:"Name to greet" default:"🌏"`
	Version kong.VersionFlag `short:"V" help:"Print version information" group:"Other:"`

	out io.Writer
}

func main() {
	cfg := &config{out: os.Stdout}
	kctx := kong.Parse(cfg, kong.Vars{"version": version})
	err := kctx.Run()
	kctx.FatalIfErrorf(err)
}

func (cfg *config) AfterApply() error {
	if (cfg.Stream == "none" || cfg.Stream == "server") && len(cfg.Names) != 1 {
		return fmt.Errorf("exactly 1 name required with stream %s", cfg.Stream)
	}
	return nil
}

func (cfg *config) Run() error {
	c, err := client.New(cfg.Address)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.Call(cfg.out, cfg.Names, cfg.Stream)
}
