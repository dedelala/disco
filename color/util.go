package color

import "math"

// Seq returns num colors from first to last by interpolating in HSV colorspace
func Seq(first, last uint32, num int) []uint32 {
	h0, s0, v0 := RGBtoHSV(CtoRGB(first))
	h1, s1, v1 := RGBtoHSV(CtoRGB(last))

	switch {
	case h1-h0 > 0.5:
		h0 += 1.0
	case h1-h0 < -0.5:
		h1 += 1.0
	}

	hs := seq(h0, h1, num)
	ss := seq(s0, s1, num)
	vs := seq(v0, v1, num)

	cs := make([]uint32, num)
	for i := 0; i < num; i++ {
		cs[i] = RGBtoC(HSVtoRGB(wrap(hs[i]), ss[i], vs[i]))
	}

	return cs
}

func seq(first, last float64, num int) []float64 {
	fs := make([]float64, num)
	d := (last - first) / float64(num-1)
	for i := 0; i < num; i++ {
		fs[i] = first + d*float64(i)
	}
	return fs
}

func wrap(h float64) float64 {
	_, h = math.Modf(h)
	if h < 0 {
		h += 1.0
	}
	return h
}
