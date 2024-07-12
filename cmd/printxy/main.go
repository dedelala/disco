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
			c := color.RGBtoC(color.XYBtoRGB(x, y, 1.0))
			fmt.Print(color.SprintcTermBG(c, "  "))
		}
		fmt.Println()
	}
}
