package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/dedelala/disco"
	"github.com/dedelala/disco/backend"
)

type page struct {
	config disco.Config
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

func (p page) Sheet() []disco.Page {
	return p.config.Sheet
}

type pageIndex struct {
	page
	N int
}

func (p pageIndex) Page() disco.Page {
	return p.config.Sheet[p.N]
}

func (p pageIndex) IsPage(n int) bool {
	return p.N == n
}

func pageIndexNumber(n int) string {
	return []string{
		"",
		"un", "deux", "trois", "quatre",
		"cinq", "six", "sept", "huit",
		"neuf", "dix", "onze", "douze",
		"treize", "quatorze", "quinze", "seize",
	}[n]
}

type pageHandler struct {
	*template.Template
	page
}

func (h pageHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	n, ok := map[string]int{
		"chasseur": -1,
		"":         0,
		"un":       1, "deux": 2, "trois": 3, "quatre": 4,
		"cinq": 5, "six": 6, "sept": 7, "huit": 8,
		"neuf": 9, "dix": 10, "onze": 11, "douze": 12,
		"treize": 13, "quatorze": 14, "quinze": 15, "seize": 16,
	}[strings.TrimPrefix(req.URL.Path, "/")]
	if !ok || n > len(h.page.config.Sheet)-1 {
		http.Error(w, "Cette page n'existe pas !", http.StatusNotFound)
		return
	}

	var bb bytes.Buffer
	err := h.Execute(&bb, pageIndex{h.page, n})
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, fmt.Sprintf("Error: %s", err), http.StatusInternalServerError)
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
		slog.Error(err.Error())
	}
	w.WriteHeader(http.StatusNoContent)
}

type chaseHandler struct {
	disco.Chaser
}

func (h chaseHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	chase, stop := strings.CutSuffix(req.URL.Path, "/stop")
	switch {
	case chase == "stop":
		stop = true
		h.StopAll()
	case stop:
		h.Stop(chase)
	default:
		h.Chase(chase)
	}
	switch {
	case stop && len(h.Chasing()) == 0:
		http.Redirect(w, req, "/", http.StatusFound)
	case stop:
		http.Redirect(w, req, "/chasseur", http.StatusFound)
	default:
		page := "/" + url.PathEscape(req.PostFormValue("page"))
		http.Redirect(w, req, page, http.StatusFound)
	}
}

type logHandler struct {
	http.Handler
}

func (h logHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	lw := &statusLogger{
		ResponseWriter: w,
		ok:             true,
		log: func(status int) {
			slog.Info("handled",
				"addr", req.RemoteAddr,
				"status", status,
				"method", req.Method,
				"url", req.URL,
			)
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
	log func(int)
}

func (w *statusLogger) WriteHeader(status int) {
	w.ok = false
	w.ResponseWriter.WriteHeader(status)
	w.log(status)
}

type logTailHandler struct {
	*template.Template
	logs  []byte
	lines int
	mu    *sync.RWMutex
}

func (h *logTailHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.mu.RLock()
	logs := bytes.Clone(h.logs)
	h.mu.RUnlock()
	var bb bytes.Buffer
	err := h.Execute(&bb, string(logs))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %s", err), http.StatusInternalServerError)
		return
	}
	io.Copy(w, &bb)
}

func (h *logTailHandler) Write(p []byte) (n int, err error) {
	h.mu.Lock()
	h.lines += bytes.Count(p, []byte{'\n'})
	h.logs = append(h.logs, p...)
	for h.lines > 1000 {
		_, h.logs, _ = bytes.Cut(h.logs, []byte{'\n'})
		h.lines--
	}
	h.mu.Unlock()
	return len(p), nil
}

//go:embed *.html *.css *.ttf *.png *.ico *.json
var files embed.FS

var logLevel = new(slog.LevelVar)

type flags struct {
	config string
	listen string
}

func main() {
	var f flags
	flag.StringVar(&f.config, "c", "/etc/disco.yml", "path to config `file`")
	flag.StringVar(&f.listen, "l", ":80", "listen `address`")
	flag.TextVar(logLevel, "v", logLevel, "log `level`")
	flag.Parse()

	b, err := files.ReadFile("log.html")
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.New("log").Parse(string(b))
	if err != nil {
		log.Fatal(err)
	}
	lth := &logTailHandler{
		Template: t,
		mu:       &sync.RWMutex{},
	}
	lh := slog.NewTextHandler(
		io.MultiWriter(os.Stderr, lth),
		&slog.HandlerOptions{Level: logLevel},
	)
	slog.SetDefault(slog.New(lh))
	http.Handle("/log", lth)

	cfg, err := backend.Load(f.config)
	if err != nil {
		log.Fatal(err)
	}
	cmdrs, err := backend.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Shutdown()

	cmdr := disco.New(cmdrs, cfg.Config)

	b, err = files.ReadFile("disco.html")
	if err != nil {
		log.Fatal(err)
	}
	t, err = template.New("disco").Funcs(template.FuncMap{
		"pageIndexNumber": pageIndexNumber,
	}).Parse(string(b))
	if err != nil {
		log.Fatal(err)
	}

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

	ph := logHandler{pageHandler{t, page{cfg.Config, chsr}}}
	http.Handle("/", ph)
	log.Fatal(http.ListenAndServe(f.listen, nil))
}
