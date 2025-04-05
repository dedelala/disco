package hue

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Config struct {
	Host string
	Key  string
}

type Client struct {
	Config
	client *http.Client
}

func New(c Config) *Client {
	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			// TODO
			InsecureSkipVerify: true,
		},
	}
	cl := &http.Client{
		Transport: t,
	}
	return &Client{c, cl}
}

func (h *Client) do(meth, path string, v any) (*http.Response, error) {
	s, err := url.JoinPath("https://", h.Host, "clip/v2", path)
	if err != nil {
		return nil, err
	}

	var bb bytes.Buffer
	if v != nil {
		err := json.NewEncoder(&bb).Encode(v)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(meth, s, &bb)
	if err != nil {
		return nil, err
	}
	req.Header.Add("hue-application-key", h.Key)
	rsp, err := h.client.Do(req)
	if err != nil {
		return rsp, err
	}
	if rsp.Body == nil {
		return rsp, fmt.Errorf("%s %s: no body in response", meth, path)
	}
	return rsp, nil
}

func (h *Client) Lights() ([]Light, error) {
	rsp, err := h.do(http.MethodGet, "resource/light", nil)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	var lr LightResponse
	if err := json.NewDecoder(rsp.Body).Decode(&lr); err != nil {
		return nil, err
	}

	return lr.Lights, joinErrs(lr.Errors)
}

func (h *Client) Light(id string) (Light, error) {
	if !idReg.MatchString(id) {
		return Light{}, fmt.Errorf("invalid resource id %q", id)
	}

	rsp, err := h.do(http.MethodGet, "resource/light/"+id, nil)
	if err != nil {
		return Light{}, err
	}
	defer rsp.Body.Close()

	var lr LightResponse
	if err := json.NewDecoder(rsp.Body).Decode(&lr); err != nil {
		return Light{}, err
	}
	if len(lr.Lights) == 0 {
		return Light{}, joinErrs(lr.Errors)
	}
	return lr.Lights[0], joinErrs(lr.Errors)
}

func (h *Client) LightPut(id string, req LightPutRequest) error {
	rsp, err := h.do(http.MethodPut, "resource/light/"+id, req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	return checkPutResponse(rsp.Body)
}

func (h *Client) Watch(ctx context.Context) (<-chan Event, error) {
	s, err := url.JoinPath("https://", h.Host, "eventstream/clip/v2")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("hue-application-key", h.Key)
	req.Header.Set("Accept", "text/event-stream")
	rsp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	if rsp.Body == nil {
		return nil, errors.New("no body in response")
	}

	events := make(chan Event)
	go func() {
		scn := bufio.NewScanner(rsp.Body)
		for scn.Scan() {
			line := scn.Text()
			line, ok := strings.CutPrefix(line, "data: ")
			if !ok {
				continue
			}
			var es []Event
			err := json.Unmarshal([]byte(line), &es)
			if err != nil {
				slog.Error("hue decoding events", "error", err)
			}
			for _, e := range es {
				events <- e
			}
		}
		rsp.Body.Close()
		close(events)
	}()
	return events, nil
}

type LightPutRequest struct {
	On       *LightPutOn       `json:"on,omitempty"`
	Dimming  *LightPutDimming  `json:"dimming,omitempty"`
	Color    *LightPutColor    `json:"color,omitempty"`
	Gradient *LightPutGradient `json:"gradient,omitempty"`
	Dynamics *LightPutDynamics `json:"dynamics,omitempty"`
}

type LightPutOn struct {
	On bool `json:"on"`
}

type LightPutDynamics struct {
	Duration int64 `json:"duration"`
}

type LightPutDimming struct {
	Brightness float64 `json:"brightness"`
}

type LightPutColor struct {
	XY XY `json:"xy"`
}

func NewLightPutColor(x, y float64) *LightPutColor {
	return &LightPutColor{XY{x, y}}
}

type LightPutGradient struct {
	Points []Point `json:"points"`
}

type Point struct {
	Color Color `json:"color"`
}

func NewPoint(x, y float64) Point {
	return Point{Color{XY{x, y}}}
}

type Color struct {
	XY XY `json:"xy"`
}

type XY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type PutResponse struct {
	Data []struct {
		Rid   string `json:"rid"`
		Rtype string `json:"rtype"`
	} `json:"data"`
	Errors []Error `json:"errors"`
}

func checkPutResponse(r io.Reader) error {
	var pr PutResponse
	if err := json.NewDecoder(r).Decode(&pr); err != nil {
		return err
	}
	return joinErrs(pr.Errors)
}

type LightResponse struct {
	Lights []Light `json:"data"`
	Errors []Error `json:"errors"`
}

type Light struct {
	Alert struct {
		ActionValues []string `json:"action_values"`
	} `json:"alert"`
	Color *struct {
		Gamut struct {
			Blue  XY `json:"blue"`
			Green XY `json:"green"`
			Red   XY `json:"red"`
		} `json:"gamut"`
		GamutType string `json:"gamut_type"`
		XY        XY     `json:"xy"`
	} `json:"color"`
	ColorTemperature *struct {
		Mirek       int `json:"mirek"`
		MirekSchema struct {
			MirekMaximum int `json:"mirek_maximum"`
			MirekMinimum int `json:"mirek_minimum"`
		} `json:"mirek_schema"`
		MirekValid bool `json:"mirek_valid"`
	} `json:"color_temperature"`
	ColorTemperatureDelta struct {
	} `json:"color_temperature_delta"`
	Dimming *struct {
		Brightness  float64 `json:"brightness"`
		MinDimLevel float64 `json:"min_dim_level"`
	} `json:"dimming"`
	DimmingDelta struct {
	} `json:"dimming_delta"`
	Dynamics struct {
		Speed        float64  `json:"speed"`
		SpeedValid   bool     `json:"speed_valid"`
		Status       string   `json:"status"`
		StatusValues []string `json:"status_values"`
	} `json:"dynamics"`
	Effects struct {
		EffectValues []string `json:"effect_values"`
		Status       string   `json:"status"`
		StatusValues []string `json:"status_values"`
	} `json:"effects"`
	Gradient *struct {
		Mode          string   `json:"mode"`
		ModeValues    []string `json:"mode_values"`
		PixelCount    int      `json:"pixel_count"`
		Points        []Point  `json:"points"`
		PointsCapable int      `json:"points_capable"`
	} `json:"gradient"`
	Id       string `json:"id"`
	IdV1     string `json:"id_v1"`
	Metadata struct {
		Archetype string `json:"archetype"`
		Name      string `json:"name"`
	} `json:"metadata"`
	Mode string `json:"mode"`
	On   struct {
		On bool `json:"on"`
	} `json:"on"`
	Owner struct {
		Rid   string `json:"rid"`
		Rtype string `json:"rtype"`
	} `json:"owner"`
	Powerup struct {
		Color struct {
			Mode string `json:"mode"`
		} `json:"color"`
		Configured bool `json:"configured"`
		Dimming    struct {
			Mode string `json:"mode"`
		} `json:"dimming"`
		On struct {
			Mode string `json:"mode"`
			On   struct {
				On bool `json:"on"`
			} `json:"on"`
		} `json:"on"`
		Preset string `json:"preset"`
	} `json:"powerup"`
	Signaling struct {
		SignalValues []string `json:"signal_values"`
	} `json:"signaling"`
	TimedEffects struct {
		EffectValues []string `json:"effect_values"`
		Status       string   `json:"status"`
		StatusValues []string `json:"status_values"`
	} `json:"timed_effects"`
	Type string `json:"type"`
}

type Error struct {
	Description string `json:"description"`
}

func (e Error) Error() string {
	return e.Description
}

func joinErrs(errs []Error) error {
	e := make([]error, len(errs))
	for i := range errs {
		e[i] = errs[i]
	}
	return errors.Join(e...)
}

type Event struct {
	CreationTime string      `json:"creationtime"`
	Data         []EventData `json:"data"`
	Id           string      `json:"id"`
	Type         string      `json:"type"`
}

type EventData struct {
	Color   *Color `json:"color"`
	Dimming *struct {
		Brightness float64 `json:"brightness"`
	} `json:"dimming"`
	Gradient *struct {
		Points        []Point `json:"points"`
		PointsCapable float64 `json:"points_capable"`
	} `json:"gradient"`
	Id   string `json:"id"`
	IdV1 string `json:"id_v1"`
	On   *struct {
		On bool `json:"on"`
	} `json:"on"`
	Owner struct {
		Rid   string `json:"rid"`
		Rtype string `json:"rtype"`
	} `json:"owner"`
	Type string `json:"type"`
}

var idReg = regexp.MustCompile(`^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$`)
