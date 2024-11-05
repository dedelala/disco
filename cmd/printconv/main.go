package main

import (
	"fmt"

	"github.com/dedelala/disco/color"
)

func main() {
	const arg = "royal-blue"
	c, _ := color.XKCD(arg)

	fmt.Println("=== XKCD ===")
	fmt.Printf("arg=%s\n", arg)
	fmt.Printf("c=%06x %s\n", c, color.SprintcTermBG(c, "  "))
	fmt.Println()

	fmt.Println("=== CtoRGB ===")
	r, g, b := color.CtoRGB(c)
	fmt.Printf("r=%f g=%f b=%f\n", r, g, b)
	fmt.Println()

	fmt.Println("=== RGBtoHSV ===")
	h, s, v := color.RGBtoHSV(color.CtoRGB(c))
	fmt.Printf("h=%f s=%f v=%f\n", h, s, v)
	fmt.Println()

	fmt.Println("=== RGBtoXYB ===")
	x, y, br := color.RGBtoXYB(color.CtoRGB(c))
	fmt.Printf("x=%f y=%f b=%f\n", x, y, br)
	fmt.Println()

	fmt.Println("=== HSVtoRGB (Strip Brightness)  ===")
	fmt.Printf("h=%f s=%f v=%f\n", h, s, 1.0)
	c = color.RGBtoC(color.HSVtoRGB(h, s, 1.0))
	fmt.Printf("c=%06x %s\n", c, color.SprintcTermBG(c, "  "))
	fmt.Println()

	fmt.Println("=== XYBtoRGB (Strip Brightness) ===")
	fmt.Printf("x=%f y=%f b=%f\n", x, y, 1.0)
	c = color.RGBtoC(color.XYBtoRGB(x, y, 1.0))
	fmt.Printf("c=%06x %s\n", c, color.SprintcTermBG(c, "  "))
	fmt.Println()
}
