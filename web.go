package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"

	"poke-maxxer/pokedata"
)

//go:embed templates/*.html
var templateFS embed.FS

var (
	lookupTmpl = template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/lookup.html"))
	teamTmpl   = template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/team.html"))
)

// app holds the data shared by all page handlers.
type app struct {
	dex    *pokedata.Dex
	finder *finder
}

func newApp(dex *pokedata.Dex) *app {
	return &app{dex: dex, finder: newFinder(dex)}
}

// base is the data every page template needs.
type base struct {
	Page    string // "lookup" | "team"; drives nav and page-specific settings
	Options pokedata.Options
}

func render(w http.ResponseWriter, tmpl *template.Template, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("render: %v", err)
	}
}

// optionsFromQuery reads the in-game settings from a form. Missing or
// unrecognized values default to the game's Modern preset.
func optionsFromQuery(q url.Values) pokedata.Options {
	return pokedata.Options{
		ModernTypes:     q.Get("types") != "original",
		FairyTypes:      q.Get("fairy") != "off",
		ModernStats:     q.Get("stats") != "original",
		ModernTypeChart: q.Get("chart") != "gen6",
	}
}

// finder resolves user-typed names to Pokémon.
type finder struct {
	byName map[string]*pokedata.Pokemon
	names  []string // autocomplete suggestions; each resolves via find
}

// normalizeName folds case and the punctuation variants people type:
// "Mr. Mime", "mr mime" and "mr-mime" all become "mr-mime".
func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer(" ", "-", "_", "-", ".", "", "'", "", "’", "").Replace(s)
	return strings.Trim(s, "-")
}

func newFinder(dex *pokedata.Dex) *finder {
	f := &finder{byName: make(map[string]*pokedata.Pokemon, 2*len(dex.Pokemon))}
	for i := range dex.Pokemon {
		p := &dex.Pokemon[i]
		f.byName[normalizeName(p.Key)] = p
	}
	for i := range dex.Pokemon {
		p := &dex.Pokemon[i]
		// Deoxys forms share the name "Deoxys"; the name resolves to the
		// first (base) form and the other forms are suggested by key.
		name := normalizeName(p.Name)
		if other, taken := f.byName[name]; taken && other != p {
			f.names = append(f.names, p.Key)
			continue
		}
		f.byName[name] = p
		f.names = append(f.names, p.Name)
	}
	return f
}

func (f *finder) find(name string) (*pokedata.Pokemon, bool) {
	p, ok := f.byName[normalizeName(name)]
	return p, ok
}
