package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/dedelala/disco/color"
	"golang.org/x/term"
)

func main() {
	c := flag.String("c", "", "contains substr")
	p := flag.String("p", "", "has prefix")
	m := flag.String("m", "", "match regexp")
	flag.Parse()

	var re *regexp.Regexp
	if m != nil {
		r, err := regexp.Compile(*m)
		if err != nil {
			log.Fatal(err)
		}
		re = r
	}

	clrs := color.List(func(s string) bool {
		if c != nil && !strings.Contains(s, *c) {
			return false
		}
		if p != nil && !strings.HasPrefix(s, *p) {
			return false
		}
		if re != nil && !re.MatchString(s) {
			return false
		}
		return true
	})

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		for _, clr := range clrs {
			fmt.Printf("%s %s\n", clr.Name, clr.Color)
		}
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, clr := range clrs {
		fmt.Fprintf(w, "%s\t%s\t%s %s\n", clr.Name, clr.Color,
			clr.Color.TermBG("  "), clr.Color.Strip().TermBG("  "))
	}
	w.Flush()
}
