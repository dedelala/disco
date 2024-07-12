package backend

import (
	"errors"
	"os"

	"github.com/dedelala/disco"
	"github.com/dedelala/disco/faux"
	"github.com/dedelala/disco/fauxcmd"
	"github.com/dedelala/disco/hue"
	"github.com/dedelala/disco/huecmd"
	"github.com/dedelala/disco/lifx"
	"github.com/dedelala/disco/lifxcmd"
	"github.com/ghodss/yaml"
)

type Config struct {
	disco.Config
	Hue  *hue.Config
	Lifx *lifx.Config
	Faux *faux.Config
}

func Load(file string) (*Config, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var c Config
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

var onShutdown []func()

func New(cfg *Config) (disco.Cmdrs, error) {
	var cmdrs disco.Cmdrs
	if cfg.Hue != nil {
		h := huecmd.Cmdr{Client: hue.New(*cfg.Hue)}
		cmdrs = append(cmdrs, disco.WithPrefix(h, "hue/"))
	}
	if cfg.Lifx != nil {
		lc, err := lifx.New(*cfg.Lifx)
		if err != nil {
			return nil, err
		}
		onShutdown = append(onShutdown, lc.End)
		l := lifxcmd.Cmdr{Client: lc}
		cmdrs = append(cmdrs, disco.WithPrefix(l, "lifx/"))
	}
	if cfg.Faux != nil {
		x := fauxcmd.Cmdr{Client: faux.New(*cfg.Faux)}
		cmdrs = append(cmdrs, disco.WithPrefix(x, "faux/"))
	}

	if len(cmdrs) == 0 {
		return nil, errors.New("no backend was configured in disco.yml")
	}

	return cmdrs, nil
}

func Shutdown() {
	for _, f := range onShutdown {
		f()
	}
}
