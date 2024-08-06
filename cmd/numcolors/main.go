package main

import (
	"fmt"

	"github.com/dedelala/disco/color"
)

func main() {
	cs := map[uint32]struct{}{}
	for c := uint32(0); c <= 0xffffff; c++ {
		h, s, _ := color.RGBtoHSV(color.CtoRGB(c))
		cs[color.RGBtoC(color.HSVtoRGB(h, s, 1))] = struct{}{}
	}
	fmt.Println(len(cs))
}
