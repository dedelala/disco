package lifxcmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/dedelala/disco"
	"github.com/dedelala/disco/color"
	"github.com/dedelala/disco/lifx"
)

type Cmdr struct {
	*lifx.Client
}

func (c Cmdr) Cmd(cmds []disco.Cmd) ([]disco.Cmd, error) {
	var (
		cout  []disco.Cmd
		errs  error
		preqs = map[string]lifx.SetPower{}
		creqs = map[string]lifx.SetColor{}
	)

	states, err := c.states(cmds)
	if len(states) == 0 {
		return nil, err
	}
	errs = errors.Join(errs, err)

	for _, cmd := range cmds {
		var (
			cs  []disco.Cmd
			err error
		)
		switch cmd.Action {
		case "switch":
			cs, err = cmdSwitch(cmd, states, preqs)
		case "dim":
			cs, err = cmdDim(cmd, states, creqs)
		case "color":
			cs, err = cmdColor(cmd, states, creqs)
		}
		cout = append(cout, cs...)
		errs = errors.Join(errs, err)
	}

	wg := &sync.WaitGroup{}
	for t, r := range preqs {
		wg.Add(1)
		go func() {
			err := c.SetPower(states[t].Target, r)
			if err != nil {
				slog.Warn("lifx did not ack", "target", t)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	wg = &sync.WaitGroup{}
	for t, r := range creqs {
		wg.Add(1)
		go func() {
			err := c.SetColor(states[t].Target, r)
			if err != nil {
				slog.Warn("lifx did not ack", "target", t)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	return cout, nil
}

func (c Cmdr) states(cmds []disco.Cmd) (map[string]lifx.State, error) {
	var (
		targets []uint64
		errs    error
	)
	for _, cmd := range cmds {
		if cmd.Target == "" {
			targets = []uint64{}
			break
		}
		t, err := parseTarget(cmd.Target)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("lifx: %s: %w", cmd.Target, err))
			continue
		}
		targets = append(targets, t)
	}
	if targets == nil {
		return nil, errs
	}

	ss, err := c.State(targets...)
	states := map[string]lifx.State{}
	for _, s := range ss {
		states[fmt.Sprintf("%x", s.Target)] = s
	}
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("lifx: %w", err))
	}
	return states, errs
}

func cmdSwitch(cmd disco.Cmd, states map[string]lifx.State, preqs map[string]lifx.SetPower) ([]disco.Cmd, error) {
	if cmd.Target == "" {
		var cout []disco.Cmd
		for t, s := range states {
			cout = append(cout, disco.SwitchCmd(t, s.Power != 0))
		}
		return cout, nil
	}
	s, ok := states[cmd.Target]
	if !ok {
		return nil, fmt.Errorf("lifx: has no target %s", cmd.Target)
	}
	if len(cmd.Args) == 0 {
		return []disco.Cmd{disco.SwitchCmd(cmd.Target, s.Power != 0)}, nil
	}
	on, err := disco.ParseSwitch(cmd.Args[0])
	if err != nil {
		return nil, fmt.Errorf("lifx: %s: %w", cmd.Target, err)
	}
	preqs[cmd.Target] = lifx.SetPower{
		Level: map[bool]uint16{true: math.MaxUint16}[on],
	}
	return nil, nil
}

func cmdDim(cmd disco.Cmd, states map[string]lifx.State, creqs map[string]lifx.SetColor) ([]disco.Cmd, error) {
	if cmd.Target == "" {
		var cout []disco.Cmd
		for t, s := range states {
			cout = append(cout, disco.DimCmd(t, 100*float64(s.B)/math.MaxUint16))
		}
		return cout, nil
	}
	s, ok := states[cmd.Target]
	if !ok {
		return nil, fmt.Errorf("lifx: has no target %s", cmd.Target)
	}
	if len(cmd.Args) == 0 {
		return []disco.Cmd{disco.DimCmd(cmd.Target, 100*float64(s.B)/math.MaxUint16)}, nil
	}

	v, err := disco.ParseDim(cmd.Args[0])
	if err != nil {
		return nil, fmt.Errorf("lifx: %s: %w", cmd.Target, err)
	}

	d, err := disco.ParseDuration(cmd.Args)
	if err != nil {
		return nil, fmt.Errorf("hue: %s: %w", cmd.Target, err)
	}
	dms := uint32(min(max(0, d.Milliseconds()), math.MaxUint32))

	r, ok := creqs[cmd.Target]
	if ok && r.Duration != dms {
		return nil, fmt.Errorf("lifx: %s: commands have conflicting durations", cmd.Target)
	}
	if !ok {
		r = lifx.SetColor{
			Color:    s.Color,
			Duration: dms,
		}
	}
	r.B = uint16(v / 100.0 * math.MaxUint16)
	creqs[cmd.Target] = r
	return nil, nil
}

func cmdColor(cmd disco.Cmd, states map[string]lifx.State, creqs map[string]lifx.SetColor) ([]disco.Cmd, error) {
	if cmd.Target == "" {
		var cout []disco.Cmd
		for t, s := range states {
			clr := color.RGBtoC(color.HSVtoRGB(
				float64(s.H)/math.MaxUint16,
				float64(s.S)/math.MaxUint16,
				1.0,
			))
			cout = append(cout, disco.ColorCmd(t, clr))
		}
		return cout, nil
	}
	s, ok := states[cmd.Target]
	if !ok {
		return nil, fmt.Errorf("lifx: has no target %s", cmd.Target)
	}
	if len(cmd.Args) == 0 {
		clr := color.RGBtoC(color.HSVtoRGB(
			float64(s.H)/math.MaxUint16,
			float64(s.S)/math.MaxUint16,
			1.0,
		))
		return []disco.Cmd{disco.ColorCmd(cmd.Target, clr)}, nil
	}

	clr, err := disco.ParseColor(cmd.Args[0])
	if err != nil {
		return nil, fmt.Errorf("lifx: %s: %w", cmd.Target, err)
	}

	d, err := disco.ParseDuration(cmd.Args)
	if err != nil {
		return nil, fmt.Errorf("hue: %s: %w", cmd.Target, err)
	}
	dms := uint32(min(max(0, d.Milliseconds()), math.MaxUint32))

	r, ok := creqs[cmd.Target]
	if ok && r.Duration != dms {
		return nil, fmt.Errorf("lifx: %s: commands have conflicting durations", cmd.Target)
	}
	if !ok {
		r = lifx.SetColor{
			Color:    s.Color,
			Duration: dms,
		}
	}
	hue, sat, _ := color.RGBtoHSV(color.CtoRGB(clr))
	r.H = uint16(hue * math.MaxUint16)
	r.S = uint16(sat * math.MaxUint16)
	creqs[cmd.Target] = r
	return nil, nil
}

func parseTarget(s string) (target uint64, err error) {
	n, err := fmt.Sscanf(s, "%x", &target)
	if err != nil {
		return
	}
	if n != 1 {
		err = errors.New("invalid target")
	}
	return
}

func (c Cmdr) Watch(ctx context.Context) (<-chan disco.Cmd, error) {
	states, err := c.Client.Watch(ctx)
	if err != nil {
		return nil, err
	}

	cout := make(chan disco.Cmd)
	go func() {
		sm := map[uint64]lifx.State{}
		for n := range states {
			p, ok := sm[n.Target]
			sm[n.Target] = n
			if !ok {
				continue
			}
			target := fmt.Sprintf("%x", n.Target)
			if n.Power != p.Power {
				cout <- disco.SwitchCmd(target, n.Power != 0)
			}
			if n.B != p.B {
				cout <- disco.DimCmd(target, 100*float64(n.B)/math.MaxUint16)
			}
			if n.H != p.H || n.S != p.S {
				c := color.RGBtoC(color.HSVtoRGB(
					float64(n.H)/math.MaxUint16,
					float64(n.S)/math.MaxUint16,
					1.0,
				))
				cout <- disco.ColorCmd(target, c)
			}
		}
		close(cout)
	}()
	return cout, nil
}
