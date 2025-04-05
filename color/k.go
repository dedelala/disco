package color

func K(s string) (uint32, bool) {
	c, ok := ck[s]
	return c, ok
}

var ck = map[string]uint32{
	"frigid":      0x01ffffff,
	"nippy":       0x1affffff,
	"chilly":      0x33ffffff,
	"brisk":       0x4cffffff,
	"cool":        0x65ffffff,
	"mild":        0x7effffff,
	"comfortable": 0x97ffffff,
	"warm":        0xb0ffffff,
	"toasty":      0xc9ffffff,
	"hot":         0xe2ffffff,
	"roasting":    0xffffffff,
}
