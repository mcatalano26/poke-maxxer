package main

import (
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"poke-maxxer/pokedata"
)

func TestParseTeam(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"pikachu\ngyarados\nskarmory", []string{"pikachu", "gyarados", "skarmory"}},
		{"pikachu\r\ngyarados\r\n", []string{"pikachu", "gyarados"}},
		{"pikachu, gyarados,skarmory", []string{"pikachu", "gyarados", "skarmory"}},
		{"\"Mr. Mime\"\n\n  \nabra ", []string{"Mr. Mime", "abra"}},
		{"", nil},
	}
	for _, tt := range tests {
		if got := parseTeam(tt.input); !slices.Equal(got, tt.want) {
			t.Errorf("parseTeam(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDefensiveCoverage(t *testing.T) {
	dex := loadDex(t)
	f := newFinder(dex)
	member := func(o pokedata.Options, name string) teamMember {
		p, ok := f.find(name)
		if !ok {
			t.Fatalf("find(%q) failed", name)
		}
		return teamMember{Name: p.Name, Types: p.TypesFor(o)}
	}
	row := func(rows []coverageRow, attacker string) coverageRow {
		for _, r := range rows {
			if r.Type == attacker {
				return r
			}
		}
		t.Fatalf("no coverage row for %s", attacker)
		return coverageRow{}
	}

	for _, o := range []pokedata.Options{{}, pokedata.Modern} {
		team := []teamMember{member(o, "gyarados"), member(o, "skarmory")}
		rows := defensiveCoverage(dex, o, team)
		if len(rows) != len(dex.Types) {
			t.Fatalf("%+v: %d rows, want one per type (%d)", o, len(rows), len(dex.Types))
		}
		checks := []struct {
			attacker             string
			labels               []string
			weak, resist, immune int
		}{
			{"electric", []string{"4×", "2×"}, 2, 0, 0},
			{"ground", []string{"0×", "0×"}, 0, 0, 2},
			{"fire", []string{"½×", "2×"}, 1, 1, 0},
			{"normal", []string{"", "½×"}, 0, 1, 0},
		}
		for _, c := range checks {
			r := row(rows, c.attacker)
			var labels []string
			for _, cell := range r.Cells {
				labels = append(labels, cell.Label)
			}
			if !slices.Equal(labels, c.labels) || r.Weak != c.weak || r.Resist != c.resist || r.Immune != c.immune {
				t.Errorf("%+v %s: labels %q weak/resist/immune %d/%d/%d, want %q %d/%d/%d",
					o, c.attacker, labels, r.Weak, r.Resist, r.Immune, c.labels, c.weak, c.resist, c.immune)
			}
		}
	}

	// The EFFECTIVENESS setting changes coverage: Modern makes Ice neutral
	// against Water, so Gyarados goes from 1× to 2× against Ice.
	gyarados := []teamMember{member(pokedata.Modern, "gyarados")}
	if got := row(defensiveCoverage(dex, pokedata.Options{}, gyarados), "ice").Cells[0].Label; got != "" {
		t.Errorf("gen6 ice vs gyarados = %q, want neutral", got)
	}
	if got := row(defensiveCoverage(dex, pokedata.Modern, gyarados), "ice").Cells[0].Label; got != "2×" {
		t.Errorf("modern ice vs gyarados = %q, want 2×", got)
	}
}

// The worked example from README.md ("Defensive score"). It was computed with
// the Gen VI chart; the app's default Modern chart scores the same team lower.
func TestDefensiveScore(t *testing.T) {
	dex := loadDex(t)
	f := newFinder(dex)
	build := func(o pokedata.Options) ([]coverageRow, *defensiveScore) {
		var team []teamMember
		for _, name := range []string{"charizard", "blastoise", "venusaur", "pikachu", "gengar", "onix"} {
			p, ok := f.find(name)
			if !ok {
				t.Fatalf("find(%q) failed", name)
			}
			team = append(team, teamMember{Name: p.Name, Types: p.TypesFor(o)})
		}
		rows := defensiveCoverage(dex, o, team)
		return rows, scoreCoverage(rows)
	}
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

	gen6 := pokedata.Options{ModernTypes: true, FairyTypes: true, ModernStats: true}
	rows, s := build(gen6)
	if s.ResistBreadth != 14 || s.TypeCount != 18 || !near(s.TotalGap, 47.0/3) || !near(s.Score, -5.0/3) ||
		!slices.Equal(s.Uncovered, []string{"ghost", "psychic", "dragon", "dark"}) {
		t.Errorf("gen6 chart: got %+v, want breadth 14/18, gap 15.67, score -1.67, uncovered ghost/psychic/dragon/dark", *s)
	}
	wantGap := map[string]float64{"psychic": 4, "ground": 3, "water": 4.0 / 3, "fighting": 0.2, "electric": 0.8, "dragon": 0}
	for _, r := range rows {
		if want, ok := wantGap[r.Type]; ok && !near(r.DefGap, want) {
			t.Errorf("gen6 chart %s: DefGap %v, want %v", r.Type, r.DefGap, want)
		}
	}

	if _, s := build(pokedata.Modern); s.ResistBreadth != 13 || !near(s.Score, -3) || !slices.Contains(s.Uncovered, "ice") {
		t.Errorf("modern chart: got %+v, want breadth 13, score -3, ice uncovered", *s)
	}
}

func TestTeamHandler(t *testing.T) {
	h := newApp(loadDex(t)).team
	get := func(team string) string {
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/team?team="+url.QueryEscape(team), nil))
		if rec.Code != http.StatusOK {
			t.Errorf("team %q: status %d", team, rec.Code)
		}
		return rec.Body.String()
	}

	body := get("pikachu\nmissingno\narbok")
	for _, want := range []string{`<th>Pikachu</th>`, `<th>Arbok</th>`, `No Pokémon named “missingno”`, `Defensive coverage`, `Defensive score: `} {
		if !strings.Contains(body, want) {
			t.Errorf("partial team: body missing %q", want)
		}
	}

	body = get("a\nb\nc\nd\ne\nf\ng")
	if !strings.Contains(body, "at most 6 Pokémon; you entered 7") || strings.Contains(body, "Defensive coverage") || strings.Contains(body, "Defensive score") {
		t.Error("7 names: want size error and no results")
	}
}
