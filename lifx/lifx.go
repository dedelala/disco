package lifx

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"sync"
	"time"
)

type Config struct {
	Timeout int
	Devices int
}

type Client struct {
	Config
	net.PacketConn
	sc <-chan uint32
	rp func() fanRx

	discos chan map[uint64]discovery
	ready  chan struct{}
	done   chan struct{}
}

func New(c Config) (*Client, error) {
	pc, err := net.ListenPacket("udp", ":56700")
	if err != nil {
		return nil, err
	}

	bcaddrs, err := ip4BroadcastAddrs()
	if err != nil {
		return nil, err
	}
	if len(bcaddrs) == 0 {
		return nil, fmt.Errorf("lifx: no ip broadcast address")
	}

	sc := make(chan uint32)
	go func() {
		var s uint32 = 2
		for {
			sc <- s
			s++
			if s == 0 {
				s = 2
			}
		}
	}()

	sp, rp := fan()

	l := &Client{
		Config:     c,
		PacketConn: pc,
		sc:         sc,
		rp:         rp,

		discos: make(chan map[uint64]discovery),
		ready:  make(chan struct{}),
		done:   make(chan struct{}),
	}

	go func() {
		for {
			p, err := l.rx()
			if err != nil {
				slog.Error("lifx rx", "error", err)
			}
			sp(p)
		}
	}()

	go l.discoverTx(bcaddrs)
	rx := rp()
	go l.discoverRx(rx.c)

	return l, nil
}

func (l *Client) End() {
	close(l.done)
}

type State struct {
	Target uint64
	Power  uint16
	Color
	*Product
}

type Color struct {
	H, S, B, K uint16
}

func newState(target uint64, sp *statePayload) State {
	if sp == nil {
		return State{Target: target}
	}
	return State{
		Target: target,
		Power:  sp.power,
		Color: Color{
			H: sp.h,
			S: sp.s,
			B: sp.b,
			K: sp.k,
		},
	}
}

func (l *Client) State(target ...uint64) ([]State, error) {
	<-l.ready

	discos := <-l.discos
	if len(target) == 0 {
		return l.state(discos)
	}
	targetDiscos := map[uint64]discovery{}
	var errs error
	for _, t := range target {
		d, ok := discos[t]
		if !ok {
			errs = errors.Join(errs, fmt.Errorf("%x: light not found or not reachable", t))
			continue
		}
		targetDiscos[t] = d
	}
	ss, err := l.state(targetDiscos)
	for i := range ss {
		ss[i].Product = discos[ss[i].Target].product
	}
	return ss, errors.Join(errs, err)
}

func (l *Client) state(discos map[uint64]discovery) ([]State, error) {
	<-l.ready

	var (
		states = make(chan State)
		errs   = make(chan error)
		wg     = &sync.WaitGroup{}
	)
	for id, d := range discos {
		wg.Add(1)
		go func() {
			s, err := l.get(d.addr)
			if err != nil {
				errs <- err
			}
			if s != nil {
				states <- newState(id, s)
			}
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(states)
		close(errs)
	}()

	var (
		ssout  = make(chan []State)
		errout = make(chan error)
	)
	go func() {
		var ss []State
		for s := range states {
			ss = append(ss, s)
		}
		ssout <- ss
	}()
	go func() {
		var err error
		for e := range errs {
			err = errors.Join(err, e)
		}
		errout <- err
	}()

	return <-ssout, <-errout
}

type SetPower struct {
	Level uint16
}

func (l *Client) SetPower(target uint64, s SetPower) error {
	<-l.ready

	discos := <-l.discos
	d, ok := discos[target]
	if !ok {
		return errors.New("light not found")
	}
	p := &packet{
		header: header{
			ptype: liSetPower,
		},
		addr: d.addr,
		payload: &setPowerPayload{
			level: s.Level,
		},
	}
	if !l.txAck(p) {
		return errors.New("did not ack")
	}
	return nil
}

type SetColor struct {
	Color
	Duration uint32
}

func (l *Client) SetColor(target uint64, s SetColor) error {
	<-l.ready

	discos := <-l.discos
	d, ok := discos[target]
	if !ok {
		return errors.New("light not found")
	}
	p := &packet{
		header: header{
			ptype: liSetColor,
		},
		addr: d.addr,
		payload: &colorPayload{
			h:        s.H,
			s:        s.S,
			b:        s.B,
			k:        s.K,
			duration: s.Duration,
		},
	}
	if !l.txAck(p) {
		return errors.New("did not ack")
	}
	return nil
}

func (l *Client) Watch(ctx context.Context) (<-chan State, error) {
	addrs, err := ip4BroadcastAddrs()
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, errors.New("no ip broadcast address")
	}

	go func() {
		t := time.NewTicker(time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				for _, addr := range addrs {
					l.tx(&packet{
						header: header{
							tagged: true,
							ptype:  liGet,
						},
						addr: addr,
					})
				}
			}
		}
	}()

	sout := make(chan State)
	go func() {
		rx := l.rp()
		sm := map[uint64]State{}
		for {
			select {
			case <-ctx.Done():
				close(rx.done)
				close(sout)
				return
			case p := <-rx.c:
				if p.ptype != liState {
					continue
				}
				sp, ok := p.payload.(*statePayload)
				if !ok || sp == nil {
					continue
				}
				s := newState(p.target, sp)
				if sm[s.Target] != s {
					sm[s.Target] = s
					sout <- s
				}
			}
		}
	}()
	return sout, nil
}

func (l *Client) tx(p *packet) error {
	if p.addr == nil {
		return errors.New("tx: no address")
	}
	if p.source == 0 {
		p.source = <-l.sc
	}
	b, err := p.marshal()
	if err != nil {
		return err
	}
	_, err = l.WriteTo(b, p.addr)
	// slog.Debug("lifx tx", "addr", p.addr, "header", p.header, "payload", p.payload, "error", err)
	if err != nil {
		return err
	}
	return nil
}

func (l *Client) rx() (*packet, error) {
	p := make([]byte, 1024)
	n, addr, err := l.ReadFrom(p)
	if err != nil {
		return nil, err
	}
	var h header
	err = h.unmarshal(p[:n])
	if err != nil {
		return nil, err
	}
	pkt := &packet{
		header: h,
		addr:   addr,
	}
	pld, ok := h.ptype.newPayload()
	if ok {
		err := pld.unmarshal(p[36:n])
		if err != nil {
			return nil, err
		}
		pkt.payload = pld
	}
	// slog.Debug("lifx rx", "header", pkt.header, "payload", pkt.payload)
	return pkt, nil
}

func (l *Client) txAck(p *packet) (ok bool) {
	p.ack = true
	var (
		dly = backoff(1, 100)
		to  = after(l.Timeout)
		tc  = after(0)
		rx  = l.rp()
	)
	defer close(rx.done)
	for {
		select {
		case <-to:
			return false
		case <-tc:
			tc = after(dly())
			l.tx(p)
		case r := <-rx.c:
			if r.ptype != ack {
				continue
			}
			if r.source != p.source {
				continue
			}
			return true
		}
	}
}

func (l *Client) txRes(p *packet) (r *packet, ok bool) {
	p.res = true
	var (
		dly = backoff(1, 100)
		to  = after(l.Timeout)
		tc  = after(0)
		rx  = l.rp()
	)
	defer close(rx.done)
	for {
		select {
		case <-to:
			return nil, false
		case <-tc:
			tc = after(dly())
			l.tx(p)
		case r := <-rx.c:
			if r.source != p.source {
				continue
			}
			if r.ptype == ack {
				continue
			}
			return r, true
		}
	}
}

func (l *Client) discoverTx(addrs []net.Addr) {
	var (
		dly = backoff(1, 60000)
		t   = after(0)
	)
	for {
		select {
		case <-t:
			for _, addr := range addrs {
				l.tx(&packet{
					header: header{
						tagged: true,
						ptype:  devGetService,
					},
					addr: addr,
				})
				l.tx(&packet{
					header: header{
						tagged: true,
						ptype:  devGetVersion,
					},
					addr: addr,
				})
			}
			t = after(dly())
		case <-l.done:
			return
		}
	}
}

type discovery struct {
	addr    *net.UDPAddr
	product *Product
}

func (d discovery) ready() bool {
	return d.addr != nil && d.product != nil
}

func (l *Client) discoverRx(rx <-chan *packet) {
	var (
		discos  = map[uint64]discovery{}
		timeout = after(l.Config.Timeout)
		ready   bool
	)
	for {
		select {
		case p, ok := <-rx:
			if !ok {
				slog.Warn("lifx discover: channel closed!")
				return
			}

			switch p.ptype {
			case devStateService:
			case devStateVersion:
			default:
				continue
			}

			host, _, err := net.SplitHostPort(p.addr.String())
			if err != nil {
				slog.Warn("lifx discover", "error", err)
				continue
			}

			d := discos[p.target]
			switch p.ptype {
			case devStateService:
				pld, ok := p.payload.(*servicePayload)
				if !ok {
					slog.Warn("lifx discover: bad service payload")
					continue
				}
				s := fmt.Sprintf("%s:%d", host, pld.port)
				a, err := net.ResolveUDPAddr("udp", s)
				if err != nil {
					slog.Warn("lifx discover", "error", err)
					continue
				}
				d.addr = a
			case devStateVersion:
				pld, ok := p.payload.(*versionPayload)
				if !ok {
					slog.Warn("lifx discover: bad version payload")
					continue
				}
				d.product = products[pld.product]
			}
			discos[p.target] = d
			if len(discos) >= l.Config.Devices && !ready {
				var finallyReady bool = true
				for _, d := range discos {
					if !d.ready() {
						finallyReady = false
						break
					}
				}
				if finallyReady {
					close(l.ready)
					ready = true
				}
			}
		case <-timeout:
			if !ready {
				slog.Warn("lifx: discovery timeout")
				close(l.ready)
				ready = true
			}
		case l.discos <- discos:
			discos = maps.Clone(discos)
		case <-l.done:
			close(l.discos)
			return
		}
	}
}

func (l *Client) addr(dev string) (*net.UDPAddr, error) {
	var id uint64
	n, err := fmt.Sscanf(dev, "%x", &id)
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, fmt.Errorf("invalid dev %s", dev)
	}

	<-l.ready

	discos := <-l.discos
	d, ok := discos[id]
	if !ok {
		return nil, fmt.Errorf("dev %s not found", dev)
	}

	return d.addr, nil
}

func (l *Client) get(addr net.Addr) (*statePayload, error) {
	p := &packet{
		header: header{
			ptype: liGet,
		},
		addr: addr,
	}

	r, ok := l.txRes(p)
	if !ok {
		return nil, fmt.Errorf("lifx get %s: no response", addr)
	}
	if r.ptype != liState {
		return nil, fmt.Errorf("lifx get %s: response is not state", addr)
	}
	s, ok := r.payload.(*statePayload)
	if !ok || s == nil {
		return nil, fmt.Errorf("lifx get %s: payload is not state", addr)
	}
	return s, nil
}

func fan() (func(*packet), func() fanRx) {
	pc := make(chan *packet)
	rc := make(chan fanRx)
	nc := make(chan struct{})

	go func() {
		cs := map[chan *packet]struct{}{}
		xc := make(chan chan *packet)
		for {
			select {
			case <-nc:
				rc <- newFanRx(cs, xc)
			case x := <-xc:
				close(x)
				delete(cs, x)
			case p := <-pc:
				for c := range cs {
					fanSend(cs, xc, c, p)
				}
			}
		}
	}()

	return func(p *packet) {
			pc <- p
		},
		func() fanRx {
			nc <- struct{}{}
			return <-rc
		}
}

type fanRx struct {
	c    <-chan *packet
	done chan<- struct{}
}

func newFanRx(cs map[chan *packet]struct{}, xc chan chan *packet) fanRx {
	c := make(chan *packet)
	done := make(chan struct{})
	cs[c] = struct{}{}
	go func() {
		for range done {
		}
		xc <- c
	}()
	return fanRx{c, done}
}

func fanSend(cs map[chan *packet]struct{}, xc chan chan *packet, c chan *packet, p *packet) {
	for {
		select {
		case c <- p:
			return
		case x := <-xc:
			close(x)
			delete(cs, x)
			if c == x {
				return
			}
		}
	}
}

func binread(b []byte, vs []interface{}) error {
	br := bytes.NewReader(b)
	for i, v := range vs {
		err := binary.Read(br, binary.LittleEndian, v)
		if err != nil {
			return fmt.Errorf("part %d: %s", i, err)
		}
	}
	return nil
}

func binwrite(vs []interface{}) ([]byte, error) {
	var buf bytes.Buffer
	for i, v := range vs {
		err := binary.Write(&buf, binary.LittleEndian, v)
		if err != nil {
			return nil, fmt.Errorf("part %d: %s", i, err)
		}
	}
	return buf.Bytes(), nil
}

func ip4BroadcastAddrs() ([]net.Addr, error) {
	as, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	var addrs []net.Addr
	for _, a := range as {
		n, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		if n.IP.IsLoopback() {
			continue
		}
		ip := n.IP.To4()
		if ip == nil {
			continue
		}
		var bc = make([]byte, len(n.Mask))
		for i := range bc {
			bc[i] = (0xff &^ n.Mask[i]) | ip[i]
		}
		addr, err := net.ResolveUDPAddr("udp", net.IP(bc).String()+":56700")
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}

func backoff(t0, t1 int) func() int {
	return func() int {
		if t0 > t1 {
			return t1
		}
		t := t0
		t0 *= 2
		return t
	}
}

func after(ms int) <-chan time.Time {
	return time.After(time.Duration(ms) * time.Millisecond)
}
