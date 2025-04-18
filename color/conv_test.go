package color

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/spatial/r2"
)

func TestCtoRGBf(t *testing.T) {
	var zs = []struct {
		c       Color
		r, g, b float64
	}{
		{0, 0.0, 0.0, 0.0},
		{0xffffff, 1.0, 1.0, 1.0},
		{0xff0000, 1.0, 0.0, 0.0},
		{0x00ff00, 0.0, 1.0, 0.0},
		{0x0000ff, 0.0, 0.0, 1.0},
	}
	for _, z := range zs {
		r, g, b := z.c.RGBf()
		if r != z.r || g != z.g || b != z.b {
			t.Errorf("%x: expected %f,%f,%f got %f,%f,%f", z.c, z.r, z.g, z.b, r, g, b)
		}
	}
}

func TestRGBftoC(t *testing.T) {
	var zs = []struct {
		r, g, b float64
		c       Color
	}{
		{0.0, 0.0, 0.0, 0},
		{1.0, 1.0, 1.0, 0xffffff},
		{1.0, 0.0, 0.0, 0xff0000},
		{0.0, 1.0, 0.0, 0x00ff00},
		{0.0, 0.0, 1.0, 0x0000ff},
	}
	for _, z := range zs {
		c := RGBf(z.r, z.g, z.b)
		if c != z.c {
			t.Errorf("%f,%f,%f: expected %x got %x", z.r, z.g, z.b, z.c, c)
		}
	}
}

func TestHSVtoRGB(t *testing.T) {
	var zs = []struct {
		h, s, v, r, g, b float64
	}{
		{0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 1.0, 1.0, 1.0},
		{0.0, 1.0, 1.0, 1.0, 0.0, 0.0},
		{1.0 / 3, 1.0, 1.0, 0.0, 1.0, 0.0},
		{2.0 / 3, 1.0, 1.0, 0.0, 0.0, 1.0},
	}
	for _, z := range zs {
		r, g, b := HSVtoRGB(z.h, z.s, z.v)
		if r != z.r || g != z.g || b != z.b {
			t.Errorf("%f,%f,%f: expected %f,%f,%f got %f,%f,%f", z.h, z.s, z.v, z.r, z.g, z.b, r, g, b)
		}
	}
}

func TestRGBtoHSV(t *testing.T) {
	var zs = []struct {
		r, g, b, h, s, v float64
	}{
		{0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
		{1.0, 1.0, 1.0, 0.0, 0.0, 1.0},
		{1.0, 0.0, 0.0, 0.0, 1.0, 1.0},
		{0.0, 1.0, 0.0, 1.0 / 3, 1.0, 1.0},
		{0.0, 0.0, 1.0, 2.0 / 3, 1.0, 1.0},
	}
	for _, z := range zs {
		h, s, v := RGBtoHSV(z.r, z.g, z.b)
		if h != z.h || s != z.s || v != z.v {
			t.Errorf("%f,%f,%f: expected %f,%f,%f got %f,%f,%f", z.r, z.g, z.b, z.h, z.s, z.v, h, s, v)
		}
	}
}

// lacking reference values for xyb let's just test the round trip?
func TestRGBtoXYBtoRGB(t *testing.T) {
	var zs = []struct {
		r, g, b float64
	}{
		{0.0, 0.0, 0.0},
		{1.0, 1.0, 1.0},
		{1.0, 0.0, 0.0},
		{0.0, 1.0, 0.0},
		{0.0, 0.0, 1.0},
	}
	for _, z := range zs {
		r, g, b := XYBtoRGBPhilipsWideRGBD65(RGBtoXYBPhilipsWideRGBD65(z.r, z.g, z.b))
		if r4(r) != z.r || r4(g) != z.g || r4(b) != z.b {
			t.Errorf("expected %f,%f,%f got %f,%f,%f", z.r, z.g, z.b, r, g, b)
		}
	}
}

// conversions aren't perfect, 4 points precision is all we get
// we just want to make the light red we're not sending people to the moon
func r4(f float64) float64 { return math.Round(f*1e4) / 1e4 }

func TestBoundToGamutXY(t *testing.T) {
	var zs = []struct {
		x, y, rx, ry, gx, gy, bx, by, cx, cy float64
	}{
		{0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.0, 0.0},
		{1.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 1.0, 1.0, 0.0},
		{0.0, 1.0, 0.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.0, 1.0},
		{0.25, 0.25, 0.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.25, 0.25},
		{0.0, 0.0, 1.0, 1.0, 2.0, 1.0, 1.0, 2.0, 1.0, 1.0},
		{-1.0, 0.5, 0.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.0, 0.5},
	}

	for i, z := range zs {
		x, y := BoundToGamutXY(z.x, z.y, z.rx, z.ry, z.gx, z.gy, z.bx, z.by)
		if x != z.cx || y != z.cy {
			t.Errorf("[%d] expected %f %f got %f %f", i, z.cx, z.cy, x, y)
		}
	}
}

func TestInRangeXY2(t *testing.T) {
	var zs = []struct {
		c  r2.Vec
		g  r2.Triangle
		in bool
	}{
		{
			c: r2.Vec{X: 0.5, Y: 0.5},
			g: r2.Triangle{
				r2.Vec{X: 0.0, Y: 0.0},
				r2.Vec{X: 1.0, Y: 0.0},
				r2.Vec{X: 0.0, Y: 1.0},
			},
			in: true,
		},
		{
			c: r2.Vec{X: 0.0, Y: 0.0},
			g: r2.Triangle{
				r2.Vec{X: 0.0, Y: 0.0},
				r2.Vec{X: 1.0, Y: 0.0},
				r2.Vec{X: 0.0, Y: 1.0},
			},
			in: true,
		},
		{
			c: r2.Vec{X: 1.0, Y: 0.0},
			g: r2.Triangle{
				r2.Vec{X: 0.0, Y: 0.0},
				r2.Vec{X: 1.0, Y: 0.0},
				r2.Vec{X: 0.0, Y: 1.0},
			},
			in: true,
		},
		{
			c: r2.Vec{X: 0.0, Y: 1.0},
			g: r2.Triangle{
				r2.Vec{X: 0.0, Y: 0.0},
				r2.Vec{X: 1.0, Y: 0.0},
				r2.Vec{X: 0.0, Y: 1.0},
			},
			in: true,
		},
		{
			c: r2.Vec{X: 0.0, Y: 0.0},
			g: r2.Triangle{
				r2.Vec{X: 1.0, Y: 1.0},
				r2.Vec{X: 2.0, Y: 1.0},
				r2.Vec{X: 1.0, Y: 2.0},
			},
			in: false,
		},
		{
			c: r2.Vec{X: 0.75, Y: 0.75},
			g: r2.Triangle{
				r2.Vec{X: 0.0, Y: 0.0},
				r2.Vec{X: 1.0, Y: 0.0},
				r2.Vec{X: 0.0, Y: 1.0},
			},
			in: false,
		},
		{
			c: r2.Vec{X: -1.0, Y: 0.5},
			g: r2.Triangle{
				r2.Vec{X: 0.0, Y: 0.0},
				r2.Vec{X: 1.0, Y: 0.0},
				r2.Vec{X: 0.0, Y: 1.0},
			},
			in: false,
		},
	}

	for i, z := range zs {
		in := inRangeXY(z.c, z.g)
		if in != z.in {
			t.Errorf("[%d] expected %t got %t", i, z.in, in)
		}
	}
}

func TestNearestPoint(t *testing.T) {
	var zs = []struct {
		p, a, b, n r2.Vec
		dist       float64
	}{
		{
			p:    r2.Vec{X: 1.0, Y: 0.5},
			a:    r2.Vec{X: 0.0, Y: 0.0},
			b:    r2.Vec{X: 0.0, Y: 1.0},
			n:    r2.Vec{X: 0.0, Y: 0.5},
			dist: 1.0,
		},
		{
			p:    r2.Vec{X: 0.0, Y: 2.0},
			a:    r2.Vec{X: 0.0, Y: 0.0},
			b:    r2.Vec{X: 0.0, Y: 1.0},
			n:    r2.Vec{X: 0.0, Y: 1.0},
			dist: 1.0,
		},
		{
			p:    r2.Vec{X: 0.0, Y: -1.0},
			a:    r2.Vec{X: 0.0, Y: 0.0},
			b:    r2.Vec{X: 0.0, Y: 1.0},
			n:    r2.Vec{X: 0.0, Y: 0.0},
			dist: 1.0,
		},
		{
			p:    r2.Vec{X: 0.0, Y: 0.0},
			a:    r2.Vec{X: 1.0, Y: 1.0},
			b:    r2.Vec{X: 2.0, Y: 1.0},
			n:    r2.Vec{X: 1.0, Y: 1.0},
			dist: math.Sqrt(2),
		},
		{
			p:    r2.Vec{X: -1.0, Y: 0.5},
			a:    r2.Vec{X: 0.0, Y: 0.0},
			b:    r2.Vec{X: 0.0, Y: 1.0},
			n:    r2.Vec{X: 0.0, Y: 0.5},
			dist: 1.0,
		},
		{
			p:    r2.Vec{X: -1.0, Y: 0.5},
			a:    r2.Vec{X: 0.0, Y: 1.0},
			b:    r2.Vec{X: 0.0, Y: 0.0},
			n:    r2.Vec{X: 0.0, Y: 0.5},
			dist: 1.0,
		},
		{
			p:    r2.Vec{X: -1.0, Y: 0.5},
			a:    r2.Vec{X: 1.0, Y: 0.0},
			b:    r2.Vec{X: 0.0, Y: 1.0},
			n:    r2.Vec{X: 0.0, Y: 1.0},
			dist: math.Sqrt(1.25),
		},
		{
			p:    r2.Vec{X: 0.5, Y: 0.5},
			a:    r2.Vec{X: 0.0, Y: 1.0},
			b:    r2.Vec{X: 1.0, Y: 0.0},
			n:    r2.Vec{X: 0.5, Y: 0.5},
			dist: 0.0,
		},
		{
			p:    r2.Vec{X: 0.5, Y: 0.5},
			a:    r2.Vec{X: 1.0, Y: 0.0},
			b:    r2.Vec{X: 0.0, Y: 1.0},
			n:    r2.Vec{X: 0.5, Y: 0.5},
			dist: 0.0,
		},
		{
			p:    r2.Vec{X: 1.0, Y: 1.0},
			a:    r2.Vec{X: 0.0, Y: 1.0},
			b:    r2.Vec{X: 1.0, Y: 0.0},
			n:    r2.Vec{X: 0.5, Y: 0.5},
			dist: math.Sqrt(0.5),
		},
		{
			p:    r2.Vec{X: 1.0, Y: 1.0},
			a:    r2.Vec{X: 1.0, Y: 0.0},
			b:    r2.Vec{X: 0.0, Y: 1.0},
			n:    r2.Vec{X: 0.5, Y: 0.5},
			dist: math.Sqrt(0.5),
		},
		{
			p:    r2.Vec{X: 0.2532, Y: 0.0475},
			a:    r2.Vec{X: 0.1532, Y: 0.0475},
			b:    r2.Vec{X: 0.6915, Y: 0.3083},
			n:    r2.Vec{X: 0.23418944353307708, Y: 0.08673843000822312},
			dist: 0.04360109685194042,
		},
	}

	for i, z := range zs {
		n, dist := nearestPoint(z.p, z.a, z.b)
		if n != z.n || dist != z.dist {
			t.Errorf("[%d] expected %v %v got %v %v", i, z.n, z.dist, n, dist)
		}
	}
}
