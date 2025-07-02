package main

import (
	"fmt"

	"github.com/dedelala/disco/color"
)

func main() {
	cs := map[color.Color]struct{}{}
	for c := uint32(0); c <= 0xffffff; c++ {
		cs[color.Color(c).Strip()] = struct{}{}
	}
	fmt.Println(len(cs))
}
