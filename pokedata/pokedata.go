// Package pokedata serves static Pokémon Modern Emerald data: per-species
// typings and base stats, evolutions, and the type effectiveness charts.
//
// The JSON files in this directory were extracted once from the Modern Emerald
// decompilation (https://github.com/resetes12/pokeemerald); each file records
// the source commit. Values that depend on an in-game option are stored for
// every setting of that option and selected through Options.
package pokedata

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed pokemon.json evolutions.json type_chart.json
var files embed.FS

// Options mirrors the in-game settings that change typings, stats, or type
// effectiveness. The zero value is the game's "Classic" preset.
type Options struct {
	ModernTypes     bool // POKéMON TYPES: Modern (false = Original)
	FairyTypes      bool // FAIRY TYPE: On
	ModernStats     bool // POKéMON STATS: Modern (false = Original Gen III)
	ModernTypeChart bool // EFFECTIVENESS: Modern (false = Gen VI)
}

// Modern is the game's "Modern" preset.
var Modern = Options{ModernTypes: true, FairyTypes: true, ModernStats: true, ModernTypeChart: true}

type FairyTypes struct {
	Off []string `json:"fairyOff"`
	On  []string `json:"fairyOn"`
}

type Typings struct {
	Original FairyTypes `json:"original"`
	Modern   FairyTypes `json:"modern"`
}

type BaseStats struct {
	HP        int `json:"hp"`
	Attack    int `json:"attack"`
	Defense   int `json:"defense"`
	SpAttack  int `json:"spAttack"`
	SpDefense int `json:"spDefense"`
	Speed     int `json:"speed"`
}

// Total is the base stat total (sum of all six base stats).
func (s BaseStats) Total() int {
	return s.HP + s.Attack + s.Defense + s.SpAttack + s.SpDefense + s.Speed
}

type StatSets struct {
	Original BaseStats `json:"original"`
	Modern   BaseStats `json:"modern"`
}

type Pokemon struct {
	Key   string   `json:"key"` // e.g. "mr-mime", "deoxys-attack"
	Name  string   `json:"name"`
	Types Typings  `json:"types"`
	Stats StatSets `json:"stats"`
}

// TypesFor returns the Pokémon's one or two types under o.
func (p *Pokemon) TypesFor(o Options) []string {
	t := p.Types.Original
	if o.ModernTypes {
		t = p.Types.Modern
	}
	if o.FairyTypes {
		return t.On
	}
	return t.Off
}

// StatsFor returns the Pokémon's base stats under o.
func (p *Pokemon) StatsFor(o Options) BaseStats {
	if o.ModernStats {
		return p.Stats.Modern
	}
	return p.Stats.Original
}

// Evolution is one evolution from From into Into. Method is the game's EVO_*
// constant in kebab case; which parameter field is set depends on it:
//
//	level, level-atk-gt-def, level-atk-eq-def, level-atk-lt-def,
//	level-silcoon, level-cascoon, level-ninjask, level-shedinja,
//	level-male, level-female, level-day, level-night,
//	level-male-morning, level-female-morning      → Level
//	beauty                                        → Beauty
//	item (use), item-hold, item-hold-day,
//	item-hold-night, trade-item                   → Item
//	move (knows move)                             → Move
//	move-type (knows a move of type)              → MoveType
//	friendship, friendship-day, friendship-night,
//	trade                                         → no parameter
//
// Friendship evolutions need friendship ≥ 220. Day is 07:00–19:59, night is
// 20:00–06:59, morning is 06:00–08:59.
type Evolution struct {
	From     string `json:"from"`
	Into     string `json:"into"`
	Method   string `json:"method"`
	Level    int    `json:"level,omitempty"`
	Beauty   int    `json:"beauty,omitempty"`
	Item     string `json:"item,omitempty"`
	Move     string `json:"move,omitempty"`
	MoveType string `json:"moveType,omitempty"`
}

type matchup struct {
	Attacker   string  `json:"attacker"`
	Defender   string  `json:"defender"`
	Multiplier float64 `json:"multiplier"`
}

type typePair struct{ attacker, defender string }

// Dex is the loaded data set. It is read-only after Load.
type Dex struct {
	Pokemon    []Pokemon   // in game species order
	Evolutions []Evolution // in game species order
	Types      []string    // in game type order

	byKey       map[string]int
	gen6Chart   map[typePair]float64
	modernChart map[typePair]float64
}

// Load decodes and validates the embedded data.
func Load() (*Dex, error) {
	var mons struct {
		Pokemon []Pokemon `json:"pokemon"`
	}
	var evos struct {
		Evolutions []Evolution `json:"evolutions"`
	}
	var chart struct {
		Types  []string             `json:"types"`
		Charts map[string][]matchup `json:"charts"`
	}
	for name, dst := range map[string]any{"pokemon.json": &mons, "evolutions.json": &evos, "type_chart.json": &chart} {
		b, err := files.ReadFile(name)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, dst); err != nil {
			return nil, fmt.Errorf("pokedata: %s: %w", name, err)
		}
	}

	d := &Dex{Pokemon: mons.Pokemon, Evolutions: evos.Evolutions, Types: chart.Types, byKey: make(map[string]int, len(mons.Pokemon))}
	types := make(map[string]bool, len(chart.Types))
	for _, t := range chart.Types {
		types[t] = true
	}
	for i, p := range d.Pokemon {
		if _, dup := d.byKey[p.Key]; dup {
			return nil, fmt.Errorf("pokedata: duplicate Pokémon %q", p.Key)
		}
		d.byKey[p.Key] = i
		for _, set := range [][]string{p.Types.Original.Off, p.Types.Original.On, p.Types.Modern.Off, p.Types.Modern.On} {
			if len(set) == 0 || len(set) > 2 {
				return nil, fmt.Errorf("pokedata: %s has %d types", p.Key, len(set))
			}
			for _, t := range set {
				if !types[t] {
					return nil, fmt.Errorf("pokedata: %s has unknown type %q", p.Key, t)
				}
			}
		}
	}
	for _, e := range d.Evolutions {
		if _, ok := d.byKey[e.From]; !ok {
			return nil, fmt.Errorf("pokedata: evolution from unknown Pokémon %q", e.From)
		}
		if _, ok := d.byKey[e.Into]; !ok {
			return nil, fmt.Errorf("pokedata: evolution into unknown Pokémon %q", e.Into)
		}
	}
	for name, dst := range map[string]*map[typePair]float64{"gen6": &d.gen6Chart, "modern": &d.modernChart} {
		rows, ok := chart.Charts[name]
		if !ok {
			return nil, fmt.Errorf("pokedata: missing %q type chart", name)
		}
		*dst = make(map[typePair]float64, len(rows))
		for _, r := range rows {
			if !types[r.Attacker] || !types[r.Defender] {
				return nil, fmt.Errorf("pokedata: %s chart: unknown type in %s → %s", name, r.Attacker, r.Defender)
			}
			(*dst)[typePair{r.Attacker, r.Defender}] = r.Multiplier
		}
	}
	return d, nil
}

// Lookup returns the Pokémon with the given key.
func (d *Dex) Lookup(key string) (*Pokemon, bool) {
	i, ok := d.byKey[key]
	if !ok {
		return nil, false
	}
	return &d.Pokemon[i], true
}

// EvolutionsFrom returns the evolutions the Pokémon can undergo.
func (d *Dex) EvolutionsFrom(key string) []Evolution {
	var out []Evolution
	for _, e := range d.Evolutions {
		if e.From == key {
			out = append(out, e)
		}
	}
	return out
}

// EvolutionsInto returns the evolutions that produce the Pokémon.
func (d *Dex) EvolutionsInto(key string) []Evolution {
	var out []Evolution
	for _, e := range d.Evolutions {
		if e.Into == key {
			out = append(out, e)
		}
	}
	return out
}

// Effectiveness returns the damage multiplier of an attacking type against a
// defender with the given types under o's type chart: 0, 0.25, 0.5, 1, 2 or 4.
// Unlisted pairs are neutral (1×).
func (d *Dex) Effectiveness(o Options, attacker string, defenders ...string) float64 {
	chart := d.gen6Chart
	if o.ModernTypeChart {
		chart = d.modernChart
	}
	m := 1.0
	for _, def := range defenders {
		if v, ok := chart[typePair{attacker, def}]; ok {
			m *= v
		}
	}
	return m
}
