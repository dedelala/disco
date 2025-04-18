package color

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Color is a KRGB color value packed in a uint32.
type Color uint32

// Parse converts s to a Color by named color value, RGB 6 digit hex,
// or KRGB 8 digit hex
func Parse(s string) (Color, error) {
	var c, ok = Lookup(s)
	if ok {
		return c, nil
	}
	n, err := fmt.Sscanf(s, "%08x", &c)
	if err != nil {
		return c, err
	}
	if n != 1 {
		return c, fmt.Errorf("unable to parse color %s", s)
	}
	return c, nil
}

// Lookup returns a Color by name from the library if found.
func Lookup(s string) (Color, bool) {
	c, ok := biblio[s]
	return c, ok
}

type ListItem struct {
	Name  string
	Color Color
}

func List(match func(string) bool) []ListItem {
	var matched []ListItem
	for _, s := range liste {
		if match(s) {
			matched = append(matched, ListItem{Name: s, Color: biblio[s]})
		}
	}
	return matched
}

func ListContains(substr string) []ListItem {
	return List(func(s string) bool {
		return strings.Contains(s, substr)
	})
}

func ListPrefix(prefix string) []ListItem {
	return List(func(s string) bool {
		return strings.HasPrefix(s, prefix)
	})
}

func ListMatch(pattern string) ([]ListItem, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return List(func(s string) bool {
		return re.MatchString(s)
	}), nil
}

func (c Color) String() string {
	if c.HasK() {
		return fmt.Sprintf("%08x", uint32(c))
	}
	return fmt.Sprintf("%06x", uint32(c))
}

// TermFG wraps s to set the terminal foreground color
func (c Color) TermFG(s string) string {
	return fmt.Sprintf(
		"\x1b[38;2;%d;%d;%dm%s\x1b[0m",
		uint8(c>>16),
		uint8(c>>8),
		uint8(c),
		s,
	)
}

// TermBG wraps s to set the terminal background
func (c Color) TermBG(s string) string {
	return fmt.Sprintf(
		"\x1b[48;2;%d;%d;%dm%s\x1b[0m",
		uint8(c>>16),
		uint8(c>>8),
		uint8(c),
		s,
	)
}

// HasK returns true if C has a fourth component which may be interpreted as
// color temperature.
func (c Color) HasK() bool {
	return uint8(c>>24) != 0
}

// Kf returns the color temperature component as a float on the range 0.0 to 1.0
func (c Color) Kf() float64 {
	return float64(uint8(c>>24)-1) / (math.MaxUint8 - 1)
}

// RGBf converts red, green, and blue float64 values on the range of 0.0 to 1.0
// to a Color. The inputs are clamped to the range of 0.0 to 1.0
func RGBf(r, g, b float64) (c Color) {
	return Color(max(min(r, 1.0), 0.0)*math.MaxUint8)<<16 |
		Color(max(min(g, 1.0), 0.0)*math.MaxUint8)<<8 |
		Color(max(min(b, 1.0), 0.0)*math.MaxUint8)
}

// RGBf returns the red, green, and blue components as float64 on the range
// 0.0 to 1.0
func (c Color) RGBf() (r, g, b float64) {
	r = float64(uint8(c>>16)) / math.MaxUint8
	g = float64(uint8(c>>8)) / math.MaxUint8
	b = float64(uint8(c)) / math.MaxUint8
	return
}

// HSVf converts hue, saturation and brightness values on the range of 0.0
// to 1.0 to a Color
func HSVf(h, s, v float64) (c Color) {
	return RGBf(HSVtoRGB(h, s, v))
}

// HSVf returns the hue, saturation and brightness components as float64 on the
// range 0.0 to 1.0
func (c Color) HSVf() (h, s, v float64) {
	return RGBtoHSV(c.RGBf())
}

// XYBfPhilipsWideRGBD65 converts x, y, and brightness values on the range 0.0
// to 1.0 to a Color using Philips Wide RGB D65 conversion.
func XYBfPhilipsWideRGBD65(x, y, bri float64) Color {
	return RGBf(XYBtoRGBPhilipsWideRGBD65(x, y, bri))
}

// XYBfPhilipsWideRGBD65 returns x, y, and brightness values on the range 0.0
// to 1.0 using Philips Wide RGB D65 conversion.
func (c Color) XYBfPhilipsWideRGBD65() (x, y, bri float64) {
	return RGBtoXYBPhilipsWideRGBD65(c.RGBf())
}

// Strip returns a new Color with the brightness component maximized.
func (c Color) Strip() Color {
	h, s, _ := c.HSVf()
	return HSVf(h, s, 1.0)
}
