package lifx

import (
	"fmt"
	"net"
)

type marshaler interface {
	marshal() ([]byte, error)
}

type unmarshaler interface {
	unmarshal(b []byte) error
}

type packet struct {
	header
	payload
	addr net.Addr
}

type payload interface {
	marshaler
	unmarshaler
}

func (p packet) marshal() ([]byte, error) {
	hb, err := p.header.marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal packet header: %s", err)
	}
	if p.payload == nil {
		return hb, nil
	}
	pb, err := p.payload.marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal packet payload: %s", err)
	}
	return append(hb, pb...), nil
}

type header struct {
	// frame
	size uint16
	// protocol 12b = 1024
	// addressable bool = true
	tagged bool
	// origin 2b = 0
	source uint32

	// frame address
	target uint64
	// reserved [6]uint8
	res bool
	ack bool
	// reserved 6b
	sequence uint8

	// protocol header
	// reserved 64b
	ptype ptype
	// reserved 16b
}

func (h *header) marshal() ([]byte, error) {
	var (
		pato uint16 = 1024 | (1 << 12)
		rar  uint8
	)
	if h.tagged {
		pato |= 1 << 13
	}
	if h.res {
		rar |= 1
	}
	if h.ack {
		rar |= 1 << 1
	}
	var vs = []interface{}{
		h.size,
		pato,
		h.source,
		h.target,
		[6]uint8{},
		rar,
		h.sequence,
		[8]uint8{},
		h.ptype,
		[2]uint8{},
	}
	return binwrite(vs)
}

func (h *header) unmarshal(b []byte) error {
	if len(b) < 36 {
		return fmt.Errorf("cannot unmarshal %d bytes into packet header", len(b))
	}
	var (
		pato uint16
		rar  uint8
	)
	var vs = []interface{}{
		&h.size,
		&pato,
		&h.source,
		&h.target,
		new([6]uint8),
		&rar,
		&h.sequence,
		new([8]uint8),
		&h.ptype,
		new([2]uint8),
	}
	if err := binread(b, vs); err != nil {
		return err
	}
	if pato&0xfff != 1024 {
		return fmt.Errorf("invalid protocol: %d", pato&0xfff)
	}
	h.tagged = pato&(1<<13) == 1<<13
	h.res = rar&1 == 1
	h.ack = rar&(1<<1) == 1<<1
	return nil
}

type ptype uint16

const (
	devGetService             ptype = 2
	devStateService           ptype = 3
	devGetPower               ptype = 20
	devSetPower               ptype = 21
	devStatePower             ptype = 22
	devGetVersion             ptype = 32
	devStateVersion           ptype = 33
	ack                       ptype = 45
	liGet                     ptype = 101
	liSetColor                ptype = 102
	liState                   ptype = 107
	liGetPower                ptype = 116
	liSetPower                ptype = 117
	liStatePower              ptype = 118
	liSetExtendedColorZones   ptype = 510
	liGetExtendedColorZones   ptype = 511
	liStateExtendedColorZones ptype = 512
)

func (t ptype) String() string {
	s, ok := map[ptype]string{
		devGetService:             "devGetService",
		devStateService:           "devStateService",
		devGetPower:               "devGetPower",
		devSetPower:               "devSetPower",
		devStatePower:             "devStatePower",
		devGetVersion:             "devGetVersion",
		devStateVersion:           "devStateVersion",
		ack:                       "ack",
		liGet:                     "liGet",
		liSetColor:                "liSetColor",
		liState:                   "liState",
		liGetPower:                "liGetPower",
		liSetPower:                "liSetPower",
		liStatePower:              "liStatePower",
		liSetExtendedColorZones:   "liSetExtendedColorZones",
		liGetExtendedColorZones:   "liGetExtendedColorZones",
		liStateExtendedColorZones: "liStateExtendedColorZones",
	}[t]
	if !ok {
		return fmt.Sprintf("not supported: %d", t)
	}
	return s
}

func (t ptype) newPayload() (payload, bool) {
	switch t {
	case devStateService:
		return &servicePayload{}, true
	case devStatePower:
		return &powerPayload{}, true
	case devStateVersion:
		return &versionPayload{}, true
	case liSetColor:
		return &colorPayload{}, true
	case liState:
		return &statePayload{}, true
	case liSetPower:
		return &setPowerPayload{}, true
	case liStatePower:
		return &powerPayload{}, true
	case liSetExtendedColorZones:
		return &setExtendedColorZonesPayload{}, true
	case liStateExtendedColorZones:
		return &stateExtendedColorZonesPayload{}, true
	}
	return nil, false
}

type servicePayload struct {
	port uint32
}

func (p *servicePayload) marshal() ([]byte, error) {
	var vs = []interface{}{
		[1]uint8{},
		p.port,
	}
	return binwrite(vs)
}

func (p *servicePayload) unmarshal(b []byte) error {
	var vs = []interface{}{
		new(uint8),
		&p.port,
	}
	return binread(b, vs)
}

type powerPayload struct {
	level uint16
}

func (p *powerPayload) marshal() ([]byte, error) {
	var vs = []interface{}{
		p.level,
	}
	return binwrite(vs)
}

func (p *powerPayload) unmarshal(b []byte) error {
	var vs = []interface{}{
		&p.level,
	}
	return binread(b, vs)
}

type versionPayload struct {
	vendor  uint32
	product uint32
	// reserved 4 bytes
}

func (p *versionPayload) marshal() ([]byte, error) {
	var vs = []interface{}{
		p.vendor,
		p.product,
		// [4]byte{},
	}
	return binwrite(vs)
}

func (p *versionPayload) unmarshal(b []byte) error {
	var vs = []interface{}{
		&p.vendor,
		&p.product,
		// [4]byte{},
	}
	return binread(b, vs)
}

type colorPayload struct {
	// reserved 8
	h, s, b, k uint16
	duration   uint32
}

func (p *colorPayload) marshal() ([]byte, error) {
	var vs = []interface{}{
		[1]byte{},
		p.h, p.s, p.b, p.k,
		p.duration,
	}
	return binwrite(vs)
}

func (p *colorPayload) unmarshal(b []byte) error {
	var vs = []interface{}{
		new([1]byte),
		&p.h, &p.s, &p.b, &p.k,
		&p.duration,
	}
	return binread(b, vs)
}

type statePayload struct {
	h, s, b, k uint16
	// reserved 16
	power uint16
	label [32]byte
	// reserved 64
}

func (p *statePayload) marshal() ([]byte, error) {
	var vs = []interface{}{
		p.h, p.s, p.b, p.k,
		[2]byte{},
		p.power,
		p.label,
		[8]byte{},
	}
	return binwrite(vs)
}

func (p *statePayload) unmarshal(b []byte) error {
	var vs = []interface{}{
		&p.h, &p.s, &p.b, &p.k,
		new([2]byte),
		&p.power,
		&p.label,
		new([8]byte),
	}
	return binread(b, vs)
}

type setPowerPayload struct {
	level    uint16
	duration uint32
}

func (p *setPowerPayload) marshal() ([]byte, error) {
	var vs = []interface{}{
		p.level,
	}
	return binwrite(vs)
}

func (p *setPowerPayload) unmarshal(b []byte) error {
	var vs = []interface{}{
		&p.level,
	}
	return binread(b, vs)
}

type multiZoneApplicationRequest uint8

const (
	multiZoneApplicationRequestNoApply multiZoneApplicationRequest = iota
	multiZoneApplicationRequestApply
	multiZoneApplicationRequestApplyOnly
)

type color struct {
	h, s, b, k uint16
}

type setExtendedColorZonesPayload struct {
	// The time it takes to transition to the new values in milliseconds.
	duration uint32

	// Whether to make this change now
	apply multiZoneApplicationRequest

	// The first zone to apply colors from. If the light has more
	// than 82 zones, then send multiple messages with different
	// indices to update the whole device.
	zoneIndex uint16

	// The number of colors in the colors field to apply to the strip
	colorsCount uint8

	colors [82]color
}

func (p *setExtendedColorZonesPayload) marshal() ([]byte, error) {
	var vs = []interface{}{
		p.duration,
		p.apply,
		p.zoneIndex,
		p.colorsCount,
		p.colors,
	}
	return binwrite(vs)
}

func (p *setExtendedColorZonesPayload) unmarshal(b []byte) error {
	var vs = []interface{}{
		&p.duration,
		&p.apply,
		&p.zoneIndex,
		&p.colorsCount,
		&p.colors,
	}
	return binread(b, vs)
}

type stateExtendedColorZonesPayload struct {
	// The number of zones on your strip
	zonesCount uint16

	// The first zone represented in the packet. If the light has more
	// than 82 zones, then this property indicates the relative
	// positioning of the colors contained in a given message.
	zoneIndex uint16

	// The number of HSBK values in the colors array that map to zones.
	colorsCount uint8

	colors [82]color
}

func (p *stateExtendedColorZonesPayload) marshal() ([]byte, error) {
	var vs = []interface{}{
		p.zonesCount,
		p.zoneIndex,
		p.colorsCount,
		p.colors,
	}
	return binwrite(vs)
}

func (p *stateExtendedColorZonesPayload) unmarshal(b []byte) error {
	var vs = []interface{}{
		&p.zonesCount,
		&p.zoneIndex,
		&p.colorsCount,
		&p.colors,
	}
	return binread(b, vs)
}
