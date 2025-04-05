package lifx

import (
	_ "embed"
	"encoding/json"
	"log/slog"
)

//go:embed products.json
var productsJson []byte

type Registry []struct {
	Vid      uint32 `json:"vid"`
	Name     string `json:"name"`
	Defaults struct {
		Buttons           bool     `json:"buttons"`
		Chain             bool     `json:"chain"`
		Color             bool     `json:"color"`
		ExtendedMultizone bool     `json:"extended_multizone"`
		Hev               bool     `json:"hev"`
		Infrared          bool     `json:"infrared"`
		Matrix            bool     `json:"matrix"`
		Multizone         bool     `json:"multizone"`
		Relays            bool     `json:"relays"`
		TemperatureRange  []uint16 `json:"temperature_range"`
	} `json:"defaults"`
	Products []Product `json:"products"`
}

type Product struct {
	Pid      uint32 `json:"pid"`
	Name     string `json:"name"`
	Features struct {
		Buttons                    bool     `json:"buttons"`
		Chain                      bool     `json:"chain"`
		Color                      bool     `json:"color"`
		ExtendedMultizone          bool     `json:"extended_multizone"`
		Hev                        bool     `json:"hev"`
		Infrared                   bool     `json:"infrared"`
		Matrix                     bool     `json:"matrix"`
		MinExtMzFirmware           int      `json:"min_ext_mz_firmware"`
		MinExtMzFirmwareComponents []int    `json:"min_ext_mz_firmware_components"`
		Multizone                  bool     `json:"multizone"`
		Relays                     bool     `json:"relays"`
		TemperatureRange           []uint16 `json:"temperature_range"`
	} `json:"features"`
	Upgrades []struct {
		Features struct {
			ExtendedMultizone bool     `json:"extended_multizone"`
			TemperatureRange  []uint16 `json:"temperature_range"`
		} `json:"features"`
		Major float64 `json:"major"`
		Minor float64 `json:"minor"`
	} `json:"upgrades"`
}

var (
	registry Registry
	products map[uint32]*Product
)

func init() {
	err := json.Unmarshal(productsJson, &registry)
	if err != nil {
		slog.Error("lifx init", "error", err)
		return
	}
	if len(registry) == 0 {
		slog.Error("lifx init", "error", "no products")
		return
	}
	if registry[0].Vid != 1 {
		slog.Error("lifx init", "error", "wrong vid")
		return
	}
	products = map[uint32]*Product{}
	for i := range registry[0].Products {
		products[registry[0].Products[i].Pid] = &registry[0].Products[i]
	}
}
