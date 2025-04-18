package main

import (
	"fmt"
	"log"
	"syscall"

	"github.com/dedelala/disco/color"
	"golang.org/x/term"
)

func main() {
	w, h, err := term.GetSize(syscall.Stdout)
	if err != nil {
		log.Fatal(err)
	}
	step := 1.0 / float64(min(w/2, h))
	for y := 1.0; y >= 0.0; y -= step * 1.25 {
		for x := 0.0; x <= 1.0; x += step {
			xb, yb := color.BoundToGamutXY(x, y, 0.6915, 0.3083, 0.17, 0.7, 0.1532, 0.0475)
			c := color.XYBfPhilipsWideRGBD65(xb, yb, 1.0)
			fmt.Print(c.TermBG("  "))
		}
		fmt.Println()
	}
}
