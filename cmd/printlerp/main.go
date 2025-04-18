package main

import (
	"fmt"
	"log"
	"syscall"

	"github.com/dedelala/disco/color"
	"golang.org/x/term"
)

func seq(first, last float64, num int) []float64 {
	fs := make([]float64, num)
	d := (last - first) / float64(num-1)
	for i := 0; i < num; i++ {
		fs[i] = first + d*float64(i)
	}
	return fs
}

func rgb(first, last color.Color, num int) []color.Color {
	r0, g0, b0 := first.RGBf()
	r1, g1, b1 := last.RGBf()
	rs := seq(r0, r1, num)
	gs := seq(g0, g1, num)
	bs := seq(b0, b1, num)
	cs := make([]color.Color, num)
	for i := 0; i < num; i++ {
		cs[i] = color.RGBf(rs[i], gs[i], bs[i])
	}
	return cs
}

func xy(first, last color.Color, num int) []color.Color {
	x0, y0, b0 := first.XYBfPhilipsWideRGBD65()
	x1, y1, b1 := last.XYBfPhilipsWideRGBD65()
	xs := seq(x0, x1, num)
	ys := seq(y0, y1, num)
	bs := seq(b0, b1, num)
	cs := make([]color.Color, num)
	for i := 0; i < num; i++ {
		cs[i] = color.XYBfPhilipsWideRGBD65(xs[i], ys[i], bs[i])
	}
	return cs
}

func printcs(cs []color.Color) {
	var s string
	for _, c := range cs {
		s += c.TermBG("  ")
	}
	fmt.Println(s)
	fmt.Println(s)
	fmt.Println(s)
}

func main() {
	w, _, err := term.GetSize(syscall.Stdout)
	if err != nil {
		log.Fatal(err)
	}
	num := w / 2

	fmt.Println("=== RGB ===")
	printcs(rgb(0xff0000, 0x0000ff, num))
	fmt.Println()
	fmt.Println()
	printcs(rgb(0x00ffff, 0xff00ff, num))
	fmt.Println()

	fmt.Println("=== HSV ===")
	printcs(color.Seq(0xff0000, 0x0000ff, num))
	fmt.Println()
	fmt.Println()
	printcs(color.Seq(0x00ffff, 0xff00ff, num))
	fmt.Println()

	fmt.Println("=== XY ===")
	printcs(xy(0xff0000, 0x0000ff, num))
	fmt.Println()
	fmt.Println()
	printcs(xy(0x00ffff, 0xff00ff, num))
	fmt.Println()
}
