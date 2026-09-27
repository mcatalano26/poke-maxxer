package main

import (
	"net/http"
	"strings"

	"poke-maxxer/pokedata"
)

type lookupResult struct {
	Name  string
	Types []string
	Stats pokedata.BaseStats
}

type lookupPage struct {
	base
	Query    string
	Names    []string
	Result   *lookupResult
	NotFound bool
}

// lookup shows one Pokémon's types and base stats.
func (a *app) lookup(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pg := lookupPage{
		base:  base{Page: "lookup", Options: optionsFromQuery(q)},
		Query: strings.TrimSpace(q.Get("name")),
		Names: a.finder.names,
	}
	status := http.StatusOK
	if pg.Query != "" {
		if p, ok := a.finder.find(pg.Query); ok {
			pg.Result = &lookupResult{Name: p.Name, Types: p.TypesFor(pg.Options), Stats: p.StatsFor(pg.Options)}
		} else {
			pg.NotFound = true
			status = http.StatusNotFound
		}
	}
	render(w, lookupTmpl, status, pg)
}
