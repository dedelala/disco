package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"text/tabwriter"

	"github.com/dedelala/disco"
	"github.com/dedelala/disco/color"
	"github.com/dedelala/disco/hue"
	"github.com/dedelala/disco/huecmd"
	"github.com/dedelala/disco/lifx"
	"github.com/dedelala/disco/lifxcmd"
	"golang.org/x/crypto/ssh/terminal"
)

func colorStdout(cmd disco.Cmd) string {
	if cmd.Action != "color" {
		return ""
	}
	if len(cmd.Args) == 0 {
		return ""
	}
	c, _ := disco.ParseColor(cmd.Args[0])
	return " " + color.CtoTermBG(c, "  ")
}

type flags struct {
	config string
	watch  bool
}

func main() {
	var f flags

	configDir := os.Getenv("HOME")
	if configDir != "" {
		configDir += "/.config/"
	}

	flag.StringVar(&f.config, "c", configDir+"disco.yml", "path to config `file`")
	flag.BoolVar(&f.watch, "w", false, "watch for changes")
	flag.Parse()

	cfg, err := disco.Load(f.config)
	if err != nil {
		log.Fatal(err)
	}

	h := huecmd.Cmdr{Client: hue.New(cfg.Hue)}
	lc, err := lifx.New(cfg.Lifx)
	if err != nil {
		log.Fatal(err)
	}
	defer lc.End()
	l := lifxcmd.Cmdr{Client: lc}

	cmdr := disco.WithCue(disco.WithSplay(disco.WithLink(disco.WithMap(disco.Cmdrs{
		disco.WithPrefix(h, "hue/"),
		disco.WithPrefix(l, "lifx/"),
	}, cfg.Map), cfg.Link), cfg.Link), cfg.Cue)

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
		log.Fatal(err)
	}
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].String() < cmds[j].String()
	})

	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
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
