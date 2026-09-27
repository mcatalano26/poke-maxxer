package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"poke-maxxer/pokedata"
)

func loadDex(t *testing.T) *pokedata.Dex {
	t.Helper()
	dex, err := pokedata.Load()
	if err != nil {
		t.Fatal(err)
	}
	return dex
}

func TestFind(t *testing.T) {
	f := newFinder(loadDex(t))
	tests := map[string]string{
		"Mr. Mime":      "mr-mime",
		"mr mime":       "mr-mime",
		"MR-MIME":       "mr-mime",
		"  pikachu ":    "pikachu",
		"Nidoran♀":      "nidoran-f",
		"nidoran-m":     "nidoran-m",
		"Farfetch'd":    "farfetchd",
		"ho oh":         "ho-oh",
		"Deoxys":        "deoxys",
		"deoxys-attack": "deoxys-attack",
	}
	for input, wantKey := range tests {
		p, ok := f.find(input)
		if !ok || p.Key != wantKey {
			t.Errorf("find(%q) = %v, %v; want %s", input, p, ok, wantKey)
		}
	}
	for _, input := range []string{"", "missingno", "mew two"} {
		if p, ok := f.find(input); ok {
			t.Errorf("find(%q) = %s; want not found", input, p.Key)
		}
	}
	// Every autocomplete suggestion must resolve.
	for _, name := range f.names {
		if _, ok := f.find(name); !ok {
			t.Errorf("suggestion %q does not resolve", name)
		}
	}
}

func TestLookupHandler(t *testing.T) {
	h := newApp(loadDex(t)).lookup
	tests := []struct {
		name       string
		query      string
		wantStatus int
		want       []string
		notWant    []string
	}{
		{"defaults to Modern preset", "name=arbok", 200,
			[]string{`class="type type-poison"`, `class="type type-dark"`, `<td>95</td>`}, nil},
		{"original settings", "name=arbok&types=original&fairy=off&stats=original", 200,
			[]string{`class="type type-poison"`, `<td>85</td>`}, []string{`class="type type-dark"`}},
		{"fairy toggle", "name=clefairy&types=original&fairy=on", 200,
			[]string{`class="type type-fairy"`}, []string{`class="type type-normal"`}},
		{"unknown name", "name=missingno", 404,
			[]string{`No Pokémon named`}, []string{`<h2>`}},
		{"empty form", "", 200,
			nil, []string{`<h2>`, `No Pokémon named`}},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil))
		body := rec.Body.String()
		if rec.Code != tt.wantStatus {
			t.Errorf("%s: status %d, want %d", tt.name, rec.Code, tt.wantStatus)
		}
		for _, s := range tt.want {
			if !strings.Contains(body, s) {
				t.Errorf("%s: body missing %q", tt.name, s)
			}
		}
		for _, s := range tt.notWant {
			if strings.Contains(body, s) {
				t.Errorf("%s: body unexpectedly contains %q", tt.name, s)
			}
		}
	}
}
