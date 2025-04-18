package fauxcmd

import (
	"context"
	"fmt"

	"github.com/dedelala/disco"
	"github.com/dedelala/disco/color"
	"github.com/dedelala/disco/faux"
)

type Cmdr struct {
	*faux.Client
}

func (c Cmdr) Cmd(cmds []disco.Cmd) ([]disco.Cmd, error) {
	var (
		cout []disco.Cmd
	)

	d, err := c.Load()
	if err != nil {
		return nil, fmt.Errorf("faux: %w", err)
	}

	for _, cmd := range cmds {
		switch cmd.Action {
		case "switch":
			cs, err := cmdSwitch(cmd, d.Ss)
			if err != nil {
				return nil, err
			}
			cout = append(cout, cs...)
		case "dim":
			cs, err := cmdDim(cmd, d.Ds)
			if err != nil {
				return nil, err
			}
			cout = append(cout, cs...)
		case "color":
			cs, err := cmdColor(cmd, d.Cs)
			if err != nil {
				return nil, err
			}
			cout = append(cout, cs...)
		}
	}

	err = c.Save(d)
	if err != nil {
		return cout, fmt.Errorf("faux: %w", err)
	}

	return cout, nil
}

func cmdSwitch(cmd disco.Cmd, ss map[string]bool) ([]disco.Cmd, error) {
	if cmd.Target == "" {
		var cout []disco.Cmd
		for t, on := range ss {
			cout = append(cout, disco.SwitchCmd(t, on))
		}
		return cout, nil
	}
	on, ok := ss[cmd.Target]
	if len(cmd.Args) == 0 {
		if !ok {
			return nil, fmt.Errorf("faux: has no target %s", cmd.Target)
		}
		return []disco.Cmd{disco.SwitchCmd(cmd.Target, on)}, nil
	}
	on, err := disco.ParseSwitch(cmd.Args[0])
	if err != nil {
		return nil, fmt.Errorf("faux: %s: %w", cmd.Target, err)
	}
	ss[cmd.Target] = on
	return nil, nil
}

func cmdDim(cmd disco.Cmd, ds map[string]float64) ([]disco.Cmd, error) {
	if cmd.Target == "" {
		var cout []disco.Cmd
		for t, v := range ds {
			cout = append(cout, disco.DimCmd(t, v))
		}
		return cout, nil
	}
	v, ok := ds[cmd.Target]
	if len(cmd.Args) == 0 {
		if !ok {
			return nil, fmt.Errorf("faux: has no target %s", cmd.Target)
		}
		return []disco.Cmd{disco.DimCmd(cmd.Target, v)}, nil
	}
	v, err := disco.ParseDim(cmd.Args[0])
	if err != nil {
		return nil, fmt.Errorf("faux: %s: %w", cmd.Target, err)
	}
	ds[cmd.Target] = v
	return nil, nil
}

func cmdColor(cmd disco.Cmd, cs map[string]color.Color) ([]disco.Cmd, error) {
	if cmd.Target == "" {
		var cout []disco.Cmd
		for t, c := range cs {
			cout = append(cout, disco.ColorCmd(t, c))
		}
		return cout, nil
	}
	c, ok := cs[cmd.Target]
	if len(cmd.Args) == 0 {
		if !ok {
			return nil, fmt.Errorf("faux: has no target %s", cmd.Target)
		}
		return []disco.Cmd{disco.ColorCmd(cmd.Target, c)}, nil
	}
	c, err := color.Parse(cmd.Args[0])
	if err != nil {
		return nil, fmt.Errorf("faux: %s: %w", cmd.Target, err)
	}

	h, s, _ := c.HSVf()
	c = color.HSVf(h, s, 1.0)

	cs[cmd.Target] = c
	return nil, nil
}

func (c Cmdr) Watch(ctx context.Context) (<-chan disco.Cmd, error) {
	return nil, nil
}
