package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"poke-maxxer/pokedata"
)

const maxTeamSize = 6

type teamMember struct {
	Name  string
	Types []string
	Stats pokedata.BaseStats
}

type coverageCell struct {
	Label string // "", "0×", "¼×", "½×", "2×", "4×"
	Class string // CSS class for the multiplier
}

// coverageRow is one attacking type against every team member.
type coverageRow struct {
	Type   string
	Cells  []coverageCell // one per member, in team order
	Weak   int            // members taking more than 1×
	Resist int            // members taking less than 1× but more than 0×
	Immune int            // members taking 0×
	DefGap float64        // weakness-concentration penalty; see defensiveScore
}

type teamPage struct {
	base
	MaxSize  int
	Input    string
	Errors   []string
	Members  []teamMember
	Coverage []coverageRow
	Score    *defensiveScore
}

// defensiveScore rates a team's defensive type coverage (README.md,
// "Defensive score"). Higher is better; it can be negative, and only the
// ordering between teams is meaningful.
type defensiveScore struct {
	ResistBreadth int      // attack types at least one member resists or is immune to
	TypeCount     int      // attack types considered
	TotalGap      float64  // Σ DefGap over all attack types
	Score         float64  // ResistBreadth − TotalGap
	Uncovered     []string // attack types nobody resists or is immune to
}

// parseTeam splits pasted CSV into names: one per line, commas also
// separate, surrounding quotes and blank entries are ignored.
func parseTeam(input string) []string {
	var names []string
	for _, field := range strings.FieldsFunc(input, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' }) {
		if name := strings.Trim(strings.TrimSpace(field), `"`); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func multiplierCell(m float64) coverageCell {
	switch m {
	case 0:
		return coverageCell{"0×", "m0"}
	case 0.25:
		return coverageCell{"¼×", "m25"}
	case 0.5:
		return coverageCell{"½×", "m50"}
	case 1:
		return coverageCell{}
	case 2:
		return coverageCell{"2×", "m200"}
	case 4:
		return coverageCell{"4×", "m400"}
	}
	return coverageCell{Label: strconv.FormatFloat(m, 'g', -1, 64) + "×"}
}

// defensiveCoverage returns, for every attacking type in game order, the
// multiplier it deals to each member under o's type chart.
func defensiveCoverage(dex *pokedata.Dex, o pokedata.Options, members []teamMember) []coverageRow {
	rows := make([]coverageRow, 0, len(dex.Types))
	for _, attacker := range dex.Types {
		row := coverageRow{Type: attacker, Cells: make([]coverageCell, 0, len(members))}
		for _, m := range members {
			mul := dex.Effectiveness(o, attacker, m.Types...)
			switch {
			case mul == 0:
				row.Immune++
			case mul < 1:
				row.Resist++
			case mul > 1:
				row.Weak++
			}
			row.Cells = append(row.Cells, multiplierCell(mul))
		}
		row.DefGap = float64(row.Weak*row.Weak) / float64(1+row.Resist+2*row.Immune)
		rows = append(rows, row)
	}
	return rows
}

func scoreCoverage(rows []coverageRow) *defensiveScore {
	s := &defensiveScore{TypeCount: len(rows)}
	for _, r := range rows {
		s.TotalGap += r.DefGap
		if r.Resist+r.Immune >= 1 {
			s.ResistBreadth++
		} else {
			s.Uncovered = append(s.Uncovered, r.Type)
		}
	}
	s.Score = float64(s.ResistBreadth) - s.TotalGap
	return s
}

// team shows base stats and defensive type coverage for up to six Pokémon.
func (a *app) team(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pg := teamPage{
		base:    base{Page: "team", Options: optionsFromQuery(q)},
		MaxSize: maxTeamSize,
		Input:   q.Get("team"),
	}
	names := parseTeam(pg.Input)
	if len(names) > maxTeamSize {
		pg.Errors = append(pg.Errors, fmt.Sprintf("A team has at most %d Pokémon; you entered %d.", maxTeamSize, len(names)))
	} else {
		for _, name := range names {
			p, ok := a.finder.find(name)
			if !ok {
				pg.Errors = append(pg.Errors, fmt.Sprintf("No Pokémon named “%s” in Modern Emerald.", name))
				continue
			}
			pg.Members = append(pg.Members, teamMember{Name: p.Name, Types: p.TypesFor(pg.Options), Stats: p.StatsFor(pg.Options)})
		}
		if len(pg.Members) > 0 {
			pg.Coverage = defensiveCoverage(a.dex, pg.Options, pg.Members)
			pg.Score = scoreCoverage(pg.Coverage)
		}
	}
	render(w, teamTmpl, http.StatusOK, pg)
}
