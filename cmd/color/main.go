package main

import (
	"fmt"

	"github.com/dedelala/disco/color"
)

func main() {

	// var r, g, b float64
	// var c uint32
	// r, g, b = color.XYBtoRGB(0.6915, 0.3083, 1.0)
	// c = color.RGBtoC(r, g, b)
	// fmt.Printf("%f %f %f %06x %s\n", r, g, b, c, color.CtoTermBG(c, "  "))
	// r, g, b = color.XYBtoRGB(0.17, 0.7, 1.0)
	// c = color.RGBtoC(r, g, b)
	// fmt.Printf("%f %f %f %06x %s\n", r, g, b, c, color.CtoTermBG(c, "  "))
	// r, g, b = color.XYBtoRGB(0.1532, 0.0475, 1.0)
	// c = color.RGBtoC(r, g, b)
	// fmt.Printf("%f %f %f %06x %s\n", r, g, b, c, color.CtoTermBG(c, "  "))

	// fmt.Println()

	// var x, y float64
	// x, y, _ = color.RGBtoXYB(color.CtoRGB(0xff0000))
	// fmt.Printf("%f %f\n", x, y)
	// x, y = color.BoundToGamutXY(x, y, 0.6915, 0.3083, 0.17, 0.7, 0.1532, 0.0475)
	// fmt.Printf("%f %f\n", x, y)
	// fmt.Println()
	// x, y, _ = color.RGBtoXYB(color.CtoRGB(0x00ff00))
	// fmt.Printf("%f %f\n", x, y)
	// x, y = color.BoundToGamutXY(x, y, 0.6915, 0.3083, 0.17, 0.7, 0.1532, 0.0475)
	// fmt.Printf("%f %f\n", x, y)
	// fmt.Println()
	// x, y, _ = color.RGBtoXYB(color.CtoRGB(0x0000ff))
	// fmt.Printf("%f %f\n", x, y)
	// x, y = color.BoundToGamutXY(x, y, 0.6915, 0.3083, 0.17, 0.7, 0.1532, 0.0475)
	// fmt.Printf("%f %f\n", x, y)
	// fmt.Println()

	// x, y, _ = color.RGBtoXYB(color.CtoRGB(0xffffff))
	// fmt.Printf("%f %f\n", x, y)
	// x, y = color.BoundToGamutXY(x, y, 0.6915, 0.3083, 0.17, 0.7, 0.1532, 0.0475)
	// fmt.Printf("%f %f\n", x, y)
	// fmt.Println()

	for y := 1.0; y >= 0.0; y -= 1.0 / 50 {
		for x := 0.0; x <= 1.0; x += 1.0 / 50 {
			var s = " "
			switch {
			case x < 0.6915 && x+1.0/50 > 0.6915 && y < 0.3083 && y+1.0/50 > 0.3083:
				s = "R"
			case x < 0.17 && x+1.0/50 > 0.17 && y < 0.7 && y+1.0/50 > 0.7:
				s = "G"
			case x < 0.1532 && x+1.0/50 > 0.1532 && y < 0.0475 && y+1.0/50 > 0.0475:
				s = "B"
			}

			xb, yb := color.BoundToGamutXY(x, y, 0.6915, 0.3083, 0.17, 0.7, 0.1532, 0.0475)
			cb := color.RGBtoC(color.XYBtoRGB(xb, yb, 1.0))
			c := color.RGBtoC(color.XYBtoRGB(x, y, 1.0))
			if s == " " && c == cb {
				s = "X"
			}
			// fmt.Print(color.CtoTermBG(c, s) + color.CtoTermBG(cb, s))
			fmt.Print(color.CtoTermBG(cb, s) + color.CtoTermBG(cb, s))
		}
		fmt.Println()
	}

}
