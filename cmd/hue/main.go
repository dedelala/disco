package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"

	"github.com/ghodss/yaml"
	"github.com/dedelala/disco/hue"
)

func main() {
	var c hue.Config
	b, err := ioutil.ReadFile("hue.yml")
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		log.Fatal(err)
	}
	h := hue.New(c)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	e, err := h.Watch(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for s := range e {
		fmt.Println(s)
	}
}
