package faux

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/dedelala/disco/color"
)

type Config struct {
	File string
}

type Client struct {
	Config
}

type Data struct {
	Ss map[string]bool
	Ds map[string]float64
	Cs map[string]color.Color
}

func New(c Config) *Client {
	return &Client{c}
}

func (f *Client) Load() (*Data, error) {
	b, err := os.ReadFile(f.File)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			d := &Data{
				Ss: map[string]bool{},
				Ds: map[string]float64{},
				Cs: map[string]color.Color{},
			}
			return d, nil
		}
		return nil, fmt.Errorf("load: %w", err)
	}
	var d = new(Data)
	err = json.Unmarshal(b, d)
	if err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	return d, nil
}

func (f *Client) Save(d *Data) error {
	b, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}
	err = os.WriteFile(f.File, b, 0644)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}
	return nil
}
