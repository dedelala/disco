package color

import (
	"math"
	"sort"

	"gonum.org/v1/gonum/spatial/r2"
)

// HSVtoRGB converts hue, saturation and brightness values on the range of 0.0
// to 1.0 to RGB floating point values on the range of 0.0 to 1.0
func HSVtoRGB(h, s, v float64) (r, g, b float64) {
	var (
		c = s * v
		x = c * (1 - math.Abs(math.Mod(h*6, 2)-1))
		m = v - c
	)

	switch {
	case h >= 0 && h <= 1.0/6:
		r, g, b = c, x, 0
	case h > 1.0/6 && h <= 2.0/6:
		r, g, b = x, c, 0
	case h > 2.0/6 && h <= 3.0/6:
		r, g, b = 0, c, x
	case h > 3.0/6 && h <= 4.0/6:
		r, g, b = 0, x, c
	case h > 4.0/6 && h <= 5.0/6:
		r, g, b = x, 0, c
	case h > 5.0/6 && h <= 1.0:
		r, g, b = c, 0, x
	}

	r, g, b = r+m, g+m, b+m
	return
}

// RGBtoHSV converts red, green, and blue floating point values on the range
// 0.0 to 1.0 to hue, saturation and brightness values on the range 0.0 to 1.0
func RGBtoHSV(r, g, b float64) (h, s, v float64) {
	var (
		xmax = max(r, g, b)
		xmin = min(r, g, b)
		c    = xmax - xmin
	)
	v = xmax
	switch {
	case c == 0:
		h = 0
	case v == r:
		h = (g - b) / (c * 6)
	case v == g:
		h = 1.0/3 + (b-r)/(c*6)
	case v == b:
		h = 2.0/3 + (r-g)/(c*6)
	}
	if h < 0 {
		h += 1
	}
	if xmax > 0 {
		s = c / xmax
	}
	return
}

// RGBtoXYBPhilipsWideRGBD65 converts red, green, and blue floating point values on the range
// 0.0 to 1.0 to CIE colorspace x, y, and brightness values on the range 0.0
// to 1.0 using Philips Wide RGB D65 conversion.
func RGBtoXYBPhilipsWideRGBD65(r, g, b float64) (x, y, bri float64) {
	r = gammaCorrect(r)
	g = gammaCorrect(g)
	b = gammaCorrect(b)

	var black bool
	if r == 0 && g == 0 && b == 0 {
		black = true
		r, g, b = 1.0, 1.0, 1.0
	}

	// Philips wide gamut conversion D65
	X := r*0.664511 + g*0.154324 + b*0.162028
	Y := r*0.283881 + g*0.668433 + b*0.047685
	Z := r*0.000088 + g*0.072310 + b*0.986039

	x = X / (X + Y + Z)
	y = Y / (X + Y + Z)
	if !black {
		bri = Y
	}
	return
}

func gammaCorrect(f float64) float64 {
	if f > 0.04045 {
		return math.Pow((f+0.055)/(1.055), 2.4)
	}
	return f / 12.92
}

// XYBtoRGBPhilipsWideRGBD65 converts CIE colorspace x, y, and brightness values on the range 0.0
// to 1.0 to red, green, and blue floating point values on the range 0.0 to 1.0
// using Philips Wide RGB D65 conversion.
func XYBtoRGBPhilipsWideRGBD65(x, y, bri float64) (r, g, b float64) {
	var X, Y, Z float64
	if bri != 0 {
		X = x * bri / y
		Y = bri
		Z = (1 - x - y) * bri / y
	}

	// Philips Wide RGB D65
	r = X*1.656492 - Y*0.354851 - Z*0.255038
	g = -X*0.707196 + Y*1.655397 + Z*0.036152
	b = X*0.051713 - Y*0.121364 + Z*1.011530

	if m := max(r, g, b); m > 1.0 {
		r, g, b = r/m, g/m, b/m
	}

	r = gammaReverse(r)
	g = gammaReverse(g)
	b = gammaReverse(b)

	if m := max(r, g, b); m > 1.0 {
		r, g, b = r/m, g/m, b/m
	}
	return
}

func gammaReverse(f float64) float64 {
	if f <= 0.0031308 {
		return 12.92 * f
	}
	return (1.055)*math.Pow(f, (1.0/2.4)) - 0.055
}

// BoundToGamutXY compares the point x,y to the triangle formed by rx,ry, gx,gy,
// bx,by. If the point falls within the triangle, x and y are returned. If the
// point falls outside the triangle, the x and y values of the nearest point
// on the triangle are returned.
func BoundToGamutXY(x, y, rx, ry, gx, gy, bx, by float64) (cx, cy float64) {
	c := boundRangeXY(
		r2.Vec{X: x, Y: y},
		r2.Triangle{
			r2.Vec{X: rx, Y: ry},
			r2.Vec{X: gx, Y: gy},
			r2.Vec{X: bx, Y: by},
		},
	)
	return c.X, c.Y
}

func inRangeXY(c r2.Vec, g r2.Triangle) bool {
	v0 := g[0]
	v1 := r2.Sub(g[1], g[0])
	v2 := r2.Sub(g[2], g[0])

	a := (r2.Cross(c, v2) - r2.Cross(v0, v2)) / r2.Cross(v1, v2)
	b := -(r2.Cross(c, v1) - r2.Cross(v0, v1)) / r2.Cross(v1, v2)

	return a >= 0 && b >= 0 && a+b <= 1
}

func nearestPoint(p, a, b r2.Vec) (r2.Vec, float64) {
	ab := r2.Sub(b, a)
	ap := r2.Sub(p, a)

	f := r2.Dot(ap, ab) / r2.Dot(ab, ab)
	an := r2.Scale(f, ab)
	n := r2.Add(a, an)

	if r2.Dot(an, ab) < 0 {
		return a, r2.Norm(r2.Sub(a, p))
	}
	if r2.Norm(an) < r2.Norm(ab) {
		return n, r2.Norm(r2.Sub(n, p))
	}
	return b, r2.Norm(r2.Sub(b, p))
}

func boundRangeXY(c r2.Vec, g r2.Triangle) r2.Vec {
	if inRangeXY(c, g) {
		return c
	}
	var ns = make([]struct {
		n    r2.Vec
		dist float64
	}, 3)
	ns[0].n, ns[0].dist = nearestPoint(c, g[0], g[1])
	ns[1].n, ns[1].dist = nearestPoint(c, g[1], g[2])
	ns[2].n, ns[2].dist = nearestPoint(c, g[2], g[0])
	sort.Slice(ns, func(i, j int) bool {
		return ns[i].dist < ns[j].dist
	})
	return ns[0].n
}
