package disco

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dedelala/disco/color"
)

type Config struct {
	Map   map[string]string
	Link  map[string][]string
	Cue   map[string]Cue
	Chase map[string]Chase
	Sheet []Sheet
}

type Cmdr interface {
	Cmd(cmds []Cmd) ([]Cmd, error)
	Watch(ctx context.Context) (<-chan Cmd, error)
}

func New(c Cmdr, cfg Config) Cmdr {
	return WithCue(WithSplay(WithLink(WithMap(c, cfg.Map), cfg.Link), cfg.Link), cfg.Cue)
}

type Cmd struct {
	Action string
	Target string
	Args   []string
}

func (c Cmd) String() string {
	return strings.Join(append([]string{c.Action, c.Target}, c.Args...), " ")
}

func (c Cmd) Tabbed() string {
	return strings.Join(append([]string{c.Action, c.Target}, c.Args...), "\t")
}

func (c *Cmd) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*c = ParseCmdString(s)
	return nil
}

func ParseCmd(args []string) Cmd {
	var c Cmd
	if len(args) > 0 {
		c.Action = args[0]
	}
	if len(args) > 1 {
		c.Target = args[1]
	}
	if len(args) > 2 {
		c.Args = args[2:]
	}
	return c
}

func ParseCmdString(s string) Cmd {
	return ParseCmd(strings.Fields(s))
}

func ParseCmdPath(s string) Cmd {
	return ParseCmd(strings.Split(strings.Trim(path.Clean(s), "/"), "/"))
}

func newCmd(action, target string, args ...string) Cmd {
	return Cmd{action, target, args}
}

func SwitchCmd(target string, on bool) Cmd {
	return newCmd(
		"switch",
		target,
		map[bool]string{true: "on", false: "off"}[on],
	)
}

func ParseSwitch(s string) (bool, error) {
	switch s {
	case "on":
		return true, nil
	case "off":
		return false, nil
	}
	return false, fmt.Errorf("%s is not a switch value", s)
}

func DimCmd(target string, v float64) Cmd {
	return newCmd(
		"dim",
		target,
		fmt.Sprintf("%.f", v),
	)
}

func ParseDim(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if v > 100 || v < 0 {
		return 0, errors.New("dimming values range from 0 to 100")
	}
	return v, nil
}

func ParseDuration(args []string) (time.Duration, error) {
	if len(args) < 2 {
		return 3 * time.Second, nil
	}
	return time.ParseDuration(args[1])
}

func ColorCmd(target string, c uint32) Cmd {
	return newCmd(
		"color",
		target,
		fmt.Sprintf("%06x", c),
	)
}

func ParseColor(s string) (uint32, error) {
	var c, ok = color.XKCD(s)
	if ok {
		return c, nil
	}
	n, err := fmt.Sscanf(s, "%06x", &c)
	if err != nil {
		return c, err
	}
	if n != 1 {
		return c, fmt.Errorf("unable to parse color %s", s)
	}
	return c, nil
}

type Cmdrs []Cmdr

func (cs Cmdrs) Cmd(cmds []Cmd) ([]Cmd, error) {
	var (
		couts []Cmd
		errs  error
	)
	for _, c := range cs {
		cout, err := c.Cmd(cmds)
		couts = append(couts, cout...)
		errs = errors.Join(errs, err)
	}
	return couts, errs
}

func (cs Cmdrs) Watch(ctx context.Context) (<-chan Cmd, error) {
	cout := make(chan Cmd)
	cin := make([]<-chan Cmd, len(cs))
	ctx, cancel := context.WithCancel(ctx)
	for i := range cs {
		c, err := cs[i].Watch(ctx)
		if err != nil {
			cancel()
			return nil, err
		}
		cin[i] = c
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(cin))
	go func() {
		wg.Wait()
		cancel()
		close(cout)
	}()
	for i := range cin {
		go func(i int) {
			for c := range cin[i] {
				cout <- c
			}
			wg.Done()
		}(i)
	}
	return cout, nil
}

type Prefixer struct {
	Cmdr
	Prefix string
}

func WithPrefix(c Cmdr, prefix string) Prefixer {
	return Prefixer{c, prefix}
}

func (p Prefixer) Cmd(cmds []Cmd) ([]Cmd, error) {
	var cuts []Cmd
	for _, cmd := range cmds {
		target, ok := strings.CutPrefix(cmd.Target, p.Prefix)
		if ok || target == "" {
			cmd.Target = target
			cuts = append(cuts, cmd)
		}
	}
	cout, err := p.Cmdr.Cmd(cuts)
	for i := range cout {
		cout[i].Target = p.Prefix + cout[i].Target
	}
	return cout, err
}

func (p Prefixer) Watch(ctx context.Context) (<-chan Cmd, error) {
	c, err := p.Cmdr.Watch(ctx)
	if err != nil {
		return c, err
	}
	cout := make(chan Cmd)
	go func() {
		for cmd := range c {
			cmd.Target = p.Prefix + cmd.Target
			cout <- cmd
		}
		close(cout)
	}()
	return cout, nil
}

type Linker struct {
	Cmdr
	L map[string][]string
}

func WithLink(c Cmdr, l map[string][]string) Linker {
	return Linker{c, l}
}

func (l Linker) Cmd(cmds []Cmd) ([]Cmd, error) {
	var (
		links []Cmd
		again = true
	)
	for again {
		links = []Cmd{}
		again = false
		for _, cmd := range cmds {
			targets, ok := l.L[cmd.Target]
			if ok {
				again = true
				for _, target := range targets {
					cmd.Target = target
					links = append(links, cmd)
				}
			} else {
				links = append(links, cmd)
			}
		}
		cmds = links
	}
	return l.Cmdr.Cmd(cmds)
}

type Splay struct {
	Cmdr
	L map[string][]string
}

func WithSplay(c Cmdr, l map[string][]string) Splay {
	return Splay{c, l}
}

func (s Splay) Cmd(cmds []Cmd) ([]Cmd, error) {
	var splays []Cmd
	for _, cmd := range cmds {
		if cmd.Action != "splay" && cmd.Action != "shuffle" {
			continue
		}
		targets := s.L[cmd.Target]
		first, err := ParseColor(cmd.Args[0])
		if err != nil {
			return nil, err
		}
		last, err := ParseColor(cmd.Args[1])
		if err != nil {
			return nil, err
		}
		colors := color.Seq(first, last, len(targets))
		if cmd.Action == "shuffle" {
			rand.Shuffle(len(colors), func(i, j int) {
				colors[i], colors[j] = colors[j], colors[i]
			})
		}
		for i := 0; i < len(targets); i++ {
			c := ColorCmd(targets[i], colors[i])
			if len(cmd.Args) > 2 {
				c.Args = append(c.Args, cmd.Args[2:]...)
			}
			splays = append(splays, c)
		}
	}
	cmds = slices.DeleteFunc(cmds, func(cmd Cmd) bool {
		return cmd.Action == "splay" || cmd.Action == "shuffle"
	})
	cmds = append(cmds, splays...)
	return s.Cmdr.Cmd(cmds)
}

type Mapper struct {
	Cmdr
	M map[string]string
	m map[string]string
}

func WithMap(c Cmdr, m map[string]string) Mapper {
	var p = Mapper{
		Cmdr: c,
		M:    m,
		m:    map[string]string{},
	}
	for k, v := range m {
		p.m[v] = k
	}
	return p
}

func (m Mapper) Cmd(cmds []Cmd) ([]Cmd, error) {
	for i := range cmds {
		if target, ok := m.m[cmds[i].Target]; ok {
			cmds[i].Target = target
		}
	}
	cmds, err := m.Cmdr.Cmd(cmds)
	for i := range cmds {
		if target, ok := m.M[cmds[i].Target]; ok {
			cmds[i].Target = target
		}
	}
	return cmds, err
}

func (m Mapper) Watch(ctx context.Context) (<-chan Cmd, error) {
	c, err := m.Cmdr.Watch(ctx)
	if err != nil {
		return c, err
	}
	cout := make(chan Cmd)
	go func() {
		for cmd := range c {
			if target, ok := m.M[cmd.Target]; ok {
				cmd.Target = target
			}
			cout <- cmd
		}
		close(cout)
	}()
	return cout, nil
}

type MultiWatcher struct {
	Cmdr

	mu     *sync.Mutex
	cs     map[context.Context]chan Cmd
	cancel func()
}

func Multi(c Cmdr) *MultiWatcher {
	return &MultiWatcher{
		Cmdr:   c,
		mu:     &sync.Mutex{},
		cs:     map[context.Context]chan Cmd{},
		cancel: func() {},
	}
}

func (m *MultiWatcher) Watch(ctx context.Context) (<-chan Cmd, error) {
	m.mu.Lock()
	if len(m.cs) == 0 {
		ctx, cancel := context.WithCancel(context.Background())
		c, err := m.Cmdr.Watch(ctx)
		if err != nil {
			cancel()
			return c, err
		}
		go func(c <-chan Cmd) {
			for cmd := range c {
				m.mu.Lock()
				for _, c := range m.cs {
					c <- cmd
				}
				m.mu.Unlock()
			}
		}(c)
		m.cancel = cancel
	}
	c := make(chan Cmd)
	m.cs[ctx] = c
	m.mu.Unlock()

	go func(ctx context.Context, c chan Cmd) {
		<-ctx.Done()
		m.mu.Lock()
		delete(m.cs, ctx)
		if len(m.cs) == 0 {
			m.cancel()
		}
		m.mu.Unlock()
		close(c)
	}(ctx, c)
	return c, nil
}

type Cue struct {
	Text string
	Cmds []Cmd
}

type Cuer struct {
	Cmdr
	Cues map[string]Cue
}

func WithCue(c Cmdr, q map[string]Cue) Cuer {
	return Cuer{c, q}
}

func (c Cuer) Cmd(cmds []Cmd) ([]Cmd, error) {
	var (
		again = true
	)
	for again {
		again = false
		for i, cmd := range cmds {
			if cmd.Action != "cue" {
				continue
			}
			cue, ok := c.Cues[cmd.Target]
			if !ok {
				return nil, fmt.Errorf("cue not found: %q", cmd.Target)
			}
			cmds = slices.Replace(cmds, i, i+1, cue.Cmds...)
			again = true
			break
		}
	}
	return c.Cmdr.Cmd(cmds)
}

type Chase struct {
	Text  string
	Steps [][]Cmd
}

type Chaser struct {
	Cmdr
	Chases map[string]Chase
	stop   map[string]func()
	errs   chan<- error
	mu     *sync.Mutex
}

func NewChaser(cmdr Cmdr, chases map[string]Chase) (Chaser, <-chan error) {
	var (
		stop = map[string]func(){}
		errs = make(chan error)
		lock = &sync.Mutex{}
	)
	return Chaser{cmdr, chases, stop, errs, lock}, errs
}

func (c Chaser) Chase(s string) {
	chase, ok := c.Chases[s]
	if !ok {
		return
	}

	c.mu.Lock()
	_, ok = c.stop[s]
	c.mu.Unlock()
	if ok {
		return
	}

	done := make(chan struct{})
	c.mu.Lock()
	c.stop[s] = func() {
		close(done)
	}
	c.mu.Unlock()

	go func() {
		var (
			step int           = 0
			wait time.Duration = 0
			run                = time.After(wait)
		)
		for {
			select {
			case <-run:
			case <-done:
				return
			}

			wait = 0
			steps := chase.Steps[step]
			for _, cmd := range steps {
				if cmd.Action == "wait" {
					d, err := time.ParseDuration(cmd.Target)
					if err != nil {
						c.errs <- fmt.Errorf("chase %s step %d wait: %w", s, step, err)
					}
					wait = d
				}
			}

			_, err := c.Cmd(steps)
			if err != nil {
				c.errs <- fmt.Errorf("chase %s step %d wait: %w", s, step, err)
			}

			step++
			step %= len(chase.Steps)
			run = time.After(wait)
		}
	}()
}

func (c Chaser) Chasing() []string {
	var ss []string
	c.mu.Lock()
	for s := range c.stop {
		ss = append(ss, s)
	}
	c.mu.Unlock()
	return ss
}

func (c Chaser) Stop(s string) {
	c.mu.Lock()
	stop, ok := c.stop[s]
	if ok {
		stop()
	}
	delete(c.stop, s)
	c.mu.Unlock()
}

func (c Chaser) StopAll() {
	c.mu.Lock()
	for _, stop := range c.stop {
		stop()
	}
	clear(c.stop)
	c.mu.Unlock()
}

type Sheet struct {
	Text  string
	Group [][]Call
}

type Call struct {
	Cue   string
	Chase string
}

const Banner = `
+---------------------------------------------------------------------+
|          ===========   =====  ==========   ========     ========    |
|          ===     ===   ===  ===      ==  ===    ===   ===    ===    |
|         ===      ===  ===  ====        ===          ===      ===    |
|        ===      ===  ===   ====       ===          ===      ===     |
|       ===      ===  ===     =====    ===          ===      ===      |
|      ===      ===  ===        ====  ===          ===      ===       |
|     ===      ===  ===         ==== ===          ===      ===        |
|    ===     ===   ===  ==      ===  ===    ===   ===    ===          |
|  ===========   ===== ==========    ========     ========            |
+---------------------------------------------------------------------+
|          Domestic  Illumination  System  Control  Operator          |
+---------------------------------------------------------------------+
`
