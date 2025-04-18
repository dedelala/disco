package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"text/tabwriter"

	"github.com/dedelala/disco"
	"github.com/dedelala/disco/backend"
	"github.com/dedelala/disco/color"
	"golang.org/x/term"
)

func colorStdout(cmd disco.Cmd) string {
	if cmd.Action != "color" {
		return ""
	}
	if len(cmd.Args) == 0 {
		return ""
	}
	c, _ := color.Parse(cmd.Args[0])
	return " " + c.TermBG("  ")
}

type flags struct {
	config   string
	watch    bool
	logLevel slog.Level
}

func main() {
	var f flags

	configDir := os.Getenv("HOME")
	if configDir != "" {
		configDir += "/.config/"
	}

	flag.StringVar(&f.config, "c", configDir+"disco.yml", "path to config `file`")
	flag.BoolVar(&f.watch, "w", false, "watch for changes")
	flag.TextVar(&f.logLevel, "v", f.logLevel, "log `level`")
	flag.Parse()

	lh := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: f.logLevel})
	slog.SetDefault(slog.New(lh))

	cfg, err := backend.Load(f.config)
	if err != nil {
		log.Fatal(err)
	}
	cmdrs, err := backend.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Shutdown()

	cmdr := disco.New(cmdrs, cfg.Config)

	if f.watch {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		c, err := cmdr.Watch(ctx)
		if err != nil {
			log.Fatal(err)
		}
		for cmd := range c {
			fmt.Printf("%s\n", cmd)
		}
		return
	}

	cmd := disco.ParseCmd(flag.Args())
	cmds, err := cmdr.Cmd([]disco.Cmd{cmd})
	if err != nil {
		slog.Error(err.Error())
	}
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].String() < cmds[j].String()
	})

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		for _, cmd := range cmds {
			fmt.Printf("%s\n", cmd)
		}
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, cmd := range cmds {
		fmt.Fprintln(w, cmd.Tabbed()+colorStdout(cmd))
	}
	w.Flush()
}
