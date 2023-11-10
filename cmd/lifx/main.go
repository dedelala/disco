package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/dedelala/disco/lifx"
)

func main() {
	cfg := lifx.Config{
		Timeout: 3000,
		Devices: 15,
	}
	l, err := lifx.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ss, err := l.Watch(ctx)
	for s := range ss {
		fmt.Println(s)
	}
	fmt.Println("done")
}
