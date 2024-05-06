package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/dedelala/disco"
	"github.com/dedelala/disco/hue"
	"github.com/dedelala/disco/huecmd"
	"github.com/dedelala/disco/lifx"
	"github.com/dedelala/disco/lifxcmd"
)

type page struct {
	config *disco.Config
	chaser disco.Chaser
}

func (p page) Cue(s string) string {
	return p.config.Cue[s].Text
}

func (p page) Chase(s string) string {
	return p.config.Chase[s].Text
}

func (p page) Chasing() []string {
	return p.chaser.Chasing()
}

func (p page) Sheet() []disco.Sheet {
	return p.config.Sheet
}

type pageHandler struct {
	*template.Template
	page
}

func (h pageHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var bb bytes.Buffer
	err := h.Execute(&bb, h.page)
	if err != nil {
		Error(w, err, http.StatusInternalServerError)
		return
	}
	io.Copy(w, &bb)
}

type cueHandler struct {
	disco.Cmdr
}

func (h cueHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	cmd := disco.Cmd{
		Action: "cue",
		Target: req.URL.Path,
	}
	_, err := h.Cmd([]disco.Cmd{cmd})
	if err != nil {
		Error(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type chaseHandler struct {
	disco.Chaser
}

func (h chaseHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	chase, stop := strings.CutSuffix(req.URL.Path, "/stop")
	if stop {
		h.Stop(chase)
	} else {
		h.Chase(chase)
	}
	http.Redirect(w, req, "/", http.StatusFound)
}

type logHandler struct {
	http.Handler
}

func (h logHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	lw := &statusLogger{
		ResponseWriter: w,
		ok:             true,
		log: func(v any) {
			log.Printf("%s %v %s %s", req.RemoteAddr, v, req.Method, req.URL)
		},
	}
	h.Handler.ServeHTTP(lw, req)
	if lw.ok {
		lw.log(http.StatusOK)
	}
}

type statusLogger struct {
	http.ResponseWriter
	ok  bool
	log func(any)
}

func (w *statusLogger) WriteHeader(code int) {
	w.ok = false
	w.ResponseWriter.WriteHeader(code)
	w.log(code)
}

func Error(w http.ResponseWriter, err error, code int) {
	s := fmt.Sprintf("Error: %s", err)
	log.Print(s)
	http.Error(w, s, code)
}

//go:embed *.html *.css *.ttf *.png *.ico *.json
var files embed.FS

type flags struct {
	config string
	listen string
}

func main() {
	var f flags
	flag.StringVar(&f.config, "c", "/etc/disco.yml", "path to config `file`")
	flag.StringVar(&f.listen, "l", ":80", "listen `addr`ess")
	flag.Parse()

	cfg, err := disco.Load(f.config)
	if err != nil {
		log.Fatal(err)
	}

	b, err := files.ReadFile("disco.html")
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.New("disco").Parse(string(b))
	if err != nil {
		log.Fatal(err)
	}

	h := huecmd.Cmdr{Client: hue.New(cfg.Hue)}
	lc, err := lifx.New(cfg.Lifx)
	if err != nil {
		log.Fatal(err)
	}
	l := lifxcmd.Cmdr{Client: lc}

	cmdr := disco.WithCue(disco.WithSplay(disco.WithLink(disco.WithMap(disco.Cmdrs{
		disco.WithPrefix(h, "hue/"),
		disco.WithPrefix(l, "lifx/"),
	}, cfg.Map), cfg.Link), cfg.Link), cfg.Cue)

	fs := logHandler{http.FileServer(http.FS(files))}
	http.Handle("/NotoSansMono-VariableFont_wdth,wght.ttf", fs)
	http.Handle("/android-chrome-192x192.png", fs)
	http.Handle("/android-chrome-512x512.png", fs)
	http.Handle("/apple-touch-icon.png", fs)
	http.Handle("/disco.css", fs)
	http.Handle("/favicon-16x16.png", fs)
	http.Handle("/favicon-32x32.png", fs)
	http.Handle("/favicon.ico", fs)
	http.Handle("/manifest.json", fs)

	ch := logHandler{http.StripPrefix("/cue/", cueHandler{cmdr})}
	http.Handle("/cue/", ch)

	chsr, errs := disco.NewChaser(cmdr, cfg.Chase)
	go func() {
		for err := range errs {
			log.Printf("Error: %s", err)
		}
	}()
	sh := logHandler{http.StripPrefix("/chase/", chaseHandler{chsr})}
	http.Handle("/chase/", sh)

	ph := logHandler{pageHandler{t, page{cfg, chsr}}}
	http.Handle("/", ph)
	log.Fatal(http.ListenAndServe(f.listen, nil))
}
