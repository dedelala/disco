package huecmd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/dedelala/disco"
	"github.com/dedelala/disco/color"
	"github.com/dedelala/disco/hue"
)

type Cmdr struct {
	*hue.Client
}

func (c Cmdr) Cmd(cmds []disco.Cmd) ([]disco.Cmd, error) {
	var (
		cout   []disco.Cmd
		errs   error
		sreqs  = map[string]hue.LightPutRequest{}
		dcreqs = map[string]hue.LightPutRequest{}
	)

	ls, err := c.Lights()
	if err != nil {
		return nil, fmt.Errorf("hue: %w", err)
	}
	lm := map[string]hue.Light{}
	for _, l := range ls {
		lm[l.Id] = l
	}

	for _, cmd := range cmds {
		var (
			cs  []disco.Cmd
			err error
		)
		switch cmd.Action {
		case "switch":
			cs, err = cmdSwitch(cmd, lm, sreqs)
		case "dim":
			cs, err = cmdDim(cmd, lm, dcreqs)
		case "color":
			cs, err = cmdColor(cmd, lm, dcreqs)
		}
		cout = append(cout, cs...)
		errs = errors.Join(errs, err)
	}

	for id, req := range sreqs {
		err := c.LightPut(id, req)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	for id, req := range dcreqs {
		err := c.LightPut(id, req)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return cout, errs
}

func cmdSwitch(cmd disco.Cmd, ls map[string]hue.Light, reqs map[string]hue.LightPutRequest) ([]disco.Cmd, error) {
	if cmd.Target == "" {
		var cout []disco.Cmd
		for _, l := range ls {
			cout = append(cout, disco.SwitchCmd(l.Id, l.On.On))
		}
		return cout, nil
	}
	l, ok := ls[cmd.Target]
	if !ok {
		return nil, fmt.Errorf("hue: has no target %s", cmd.Target)
	}
	if len(cmd.Args) == 0 {
		return []disco.Cmd{disco.SwitchCmd(l.Id, l.On.On)}, nil
	}
	on, err := disco.ParseSwitch(cmd.Args[0])
	if err != nil {
		return nil, fmt.Errorf("hue: %s: %w", cmd.Target, err)
	}
	reqs[cmd.Target] = hue.LightPutRequest{
		On: &hue.LightPutOn{On: on},
	}
	return nil, nil
}

func cmdDim(cmd disco.Cmd, ls map[string]hue.Light, reqs map[string]hue.LightPutRequest) ([]disco.Cmd, error) {
	if cmd.Target == "" {
		var cout []disco.Cmd
		for _, l := range ls {
			if l.Dimming == nil {
				continue
			}
			cout = append(cout, disco.DimCmd(l.Id, l.Dimming.Brightness))
		}
		return cout, nil
	}

	id, _, _ := strings.Cut(cmd.Target, "/")
	l, ok := ls[id]
	if !ok {
		return nil, fmt.Errorf("hue: has no target %s", id)
	}
	if l.Dimming == nil {
		return nil, fmt.Errorf("hue: has no dimming %s", id)
	}
	if len(cmd.Args) == 0 {
		return []disco.Cmd{disco.DimCmd(l.Id, l.Dimming.Brightness)}, nil
	}

	v, err := disco.ParseDim(cmd.Args[0])
	if err != nil {
		return nil, fmt.Errorf("hue: %s: %w", id, err)
	}
	req := reqs[id]
	req.Dimming = &hue.LightPutDimming{Brightness: v}

	d, err := disco.ParseDuration(cmd.Args)
	if err != nil {
		return nil, fmt.Errorf("hue: %s: %w", id, err)
	}
	if req.Dynamics != nil && req.Dynamics.Duration != d.Milliseconds() {
		return nil, fmt.Errorf("hue: %s: commands have conflicting durations", id)
	}
	if req.Dynamics == nil {
		req.Dynamics = &hue.LightPutDynamics{Duration: d.Milliseconds()}
	}

	reqs[id] = req
	return nil, nil
}

func cmdColor(cmd disco.Cmd, ls map[string]hue.Light, reqs map[string]hue.LightPutRequest) ([]disco.Cmd, error) {
	if cmd.Target == "" {
		var cout []disco.Cmd
		for _, l := range ls {
			cout = append(cout, cmdColorGet(l)...)
		}
		return cout, nil
	}

	id, index, isPoint := strings.Cut(cmd.Target, "/")
	l, ok := ls[id]
	if !ok {
		return nil, fmt.Errorf("hue: has no target %s", cmd.Target)
	}
	if l.Color == nil {
		return nil, fmt.Errorf("hue: has no color %s", cmd.Target)
	}

	if len(cmd.Args) == 0 {
		cout := cmdColorGet(l)
		if !isPoint {
			return cout, nil
		}
		i, err := strconv.Atoi(index)
		if err != nil || i < 0 || i >= len(cout) {
			return nil, fmt.Errorf("hue: has no target %s", cmd.Target)
		}
		return cout[i : i+1], nil
	}

	req := reqs[id]
	d, err := disco.ParseDuration(cmd.Args)
	if err != nil {
		return nil, fmt.Errorf("hue: %s: %w", cmd.Target, err)
	}
	if req.Dynamics != nil && req.Dynamics.Duration != d.Milliseconds() {
		return nil, fmt.Errorf("hue: %s: commands have conflicting durations", cmd.Target)
	}
	if req.Dynamics == nil {
		req.Dynamics = &hue.LightPutDynamics{Duration: d.Milliseconds()}
	}

	clr, err := color.Parse(cmd.Args[0])
	if err != nil {
		return nil, fmt.Errorf("hue: %s: %w", cmd.Target, err)
	}

	if clr.HasK() {
		if l.ColorTemperature == nil {
			return nil, fmt.Errorf("hue: %s has no mirek", cmd.Target)
		}
		k := clr.Kf()
		v := k * float64(l.ColorTemperature.MirekSchema.MirekMaximum-l.ColorTemperature.MirekSchema.MirekMinimum)
		m := l.ColorTemperature.MirekSchema.MirekMinimum + int(v)
		req.ColorTemperature = &hue.LightPutColorTemperature{Mirek: m}
		reqs[id] = req
		return nil, nil
	}

	x, y, _ := clr.XYBfPhilipsWideRGBD65()
	x, y = color.BoundToGamutXY(
		x, y,
		l.Color.Gamut.Red.X, l.Color.Gamut.Red.Y,
		l.Color.Gamut.Green.X, l.Color.Gamut.Green.Y,
		l.Color.Gamut.Blue.X, l.Color.Gamut.Blue.Y,
	)

	if !isPoint {
		req.Color = hue.NewLightPutColor(x, y)
		reqs[id] = req
		return nil, nil
	}

	points := l.Gradient.Points
	for len(points) < l.Gradient.PointsCapable {
		points = append(points, hue.NewPoint(l.Color.XY.X, l.Color.XY.Y))
	}
	i, err := strconv.Atoi(index)
	if err != nil || i < 0 || i >= l.Gradient.PointsCapable {
		return nil, fmt.Errorf("hue: has no target %s", cmd.Target)
	}
	points[i] = hue.NewPoint(x, y)
	req.Gradient = &hue.LightPutGradient{Points: points}
	reqs[id] = req
	return nil, nil
}

func cmdColorGet(l hue.Light) []disco.Cmd {
	if l.Color == nil {
		return nil
	}

	clr := color.XYBfPhilipsWideRGBD65(l.Color.XY.X, l.Color.XY.Y, 1.0)

	if l.Gradient == nil {
		return []disco.Cmd{disco.ColorCmd(l.Id, clr)}
	}

	var cout []disco.Cmd
	for i := 0; i < l.Gradient.PointsCapable; i++ {
		id := fmt.Sprintf("%s/%d", l.Id, i)
		if i >= len(l.Gradient.Points) {
			cout = append(cout, disco.ColorCmd(id, clr))
			continue
		}
		clr = color.XYBfPhilipsWideRGBD65(
			l.Gradient.Points[i].Color.XY.X,
			l.Gradient.Points[i].Color.XY.Y,
			1.0,
		)
		cout = append(cout, disco.ColorCmd(id, clr))
	}
	return cout
}

func (c Cmdr) Watch(ctx context.Context) (<-chan disco.Cmd, error) {
	events, err := c.Client.Watch(ctx)
	if err != nil {
		return nil, err
	}

	cout := make(chan disco.Cmd)
	go func() {
		for e := range events {
			if e.Type != "update" {
				continue
			}
			for _, d := range e.Data {
				watchEventData(cout, d)
			}
		}
		close(cout)
	}()
	return cout, nil
}

func watchEventData(cout chan<- disco.Cmd, d hue.EventData) {
	if d.Type != "light" {
		return
	}
	if d.On != nil {
		cout <- disco.SwitchCmd(d.Id, d.On.On)
	}
	if d.Dimming != nil {
		cout <- disco.DimCmd(d.Id, d.Dimming.Brightness)
	}
	if d.Color != nil {
		c := color.XYBfPhilipsWideRGBD65(d.Color.XY.X, d.Color.XY.Y, 1.0)
		cout <- disco.ColorCmd(d.Id, c)
	}
	if d.Gradient != nil {
		for i, p := range d.Gradient.Points {
			c := color.XYBfPhilipsWideRGBD65(p.Color.XY.X, p.Color.XY.Y, 1.0)
			id := fmt.Sprintf("%s/%d", d.Id, i)
			cout <- disco.ColorCmd(id, c)
		}
	}
}
