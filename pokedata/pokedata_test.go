package pokedata

import (
	"slices"
	"testing"
)

func load(t *testing.T) *Dex {
	t.Helper()
	d, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func lookup(t *testing.T, d *Dex, key string) *Pokemon {
	t.Helper()
	p, ok := d.Lookup(key)
	if !ok {
		t.Fatalf("Lookup(%q) not found", key)
	}
	return p
}

// Typings follow GetTypeBySpecies' rule order: Modern-typing reverts win over
// Fairy reverts, and Snubbull only gets Fairy/Normal with both options on.
func TestTypesFor(t *testing.T) {
	d := load(t)
	tests := []struct {
		key  string
		opts Options
		want []string
	}{
		{"arbok", Options{}, []string{"poison"}},
		{"arbok", Options{ModernTypes: true}, []string{"poison", "dark"}},
		{"clefairy", Options{}, []string{"normal"}},
		{"clefairy", Options{FairyTypes: true}, []string{"fairy"}},
		{"meganium", Options{FairyTypes: true}, []string{"grass"}},
		{"meganium", Options{ModernTypes: true}, []string{"grass", "fairy"}},
		{"snubbull", Options{ModernTypes: true}, []string{"normal"}},
		{"snubbull", Options{FairyTypes: true}, []string{"fairy"}},
		{"snubbull", Options{ModernTypes: true, FairyTypes: true}, []string{"fairy", "normal"}},
	}
	for _, tt := range tests {
		if got := lookup(t, d, tt.key).TypesFor(tt.opts); !slices.Equal(got, tt.want) {
			t.Errorf("%s %+v: TypesFor = %v, want %v", tt.key, tt.opts, got, tt.want)
		}
	}
}

// Original stats fall back to the modern value for stats without an _old override.
func TestStatsFor(t *testing.T) {
	d := load(t)
	arbok := lookup(t, d, "arbok")
	if got := arbok.StatsFor(Options{}); got.Attack != 85 || got.Defense != 69 {
		t.Errorf("arbok original = %+v, want attack 85, defense 69", got)
	}
	if got := arbok.StatsFor(Options{ModernStats: true}); got.Attack != 95 || got.Defense != 69 {
		t.Errorf("arbok modern = %+v, want attack 95, defense 69", got)
	}
}

func TestEffectiveness(t *testing.T) {
	d := load(t)
	gen6, modern := Options{}, Options{ModernTypeChart: true}
	tests := []struct {
		name      string
		opts      Options
		attacker  string
		defenders []string
		want      float64
	}{
		{"immunity", gen6, "normal", []string{"ghost"}, 0},
		{"gen6 resist", gen6, "rock", []string{"ground"}, 0.5},
		{"modern rebalanced", modern, "rock", []string{"ground"}, 1},
		{"modern new resist", modern, "water", []string{"ice"}, 0.5},
		{"dual super", gen6, "electric", []string{"water", "flying"}, 4},
		{"dual cancel", gen6, "fire", []string{"grass", "water"}, 1},
		{"immunity dominates", gen6, "ground", []string{"flying", "electric"}, 0},
	}
	for _, tt := range tests {
		if got := d.Effectiveness(tt.opts, tt.attacker, tt.defenders...); got != tt.want {
			t.Errorf("%s: %s → %v = %v, want %v", tt.name, tt.attacker, tt.defenders, got, tt.want)
		}
	}
}

// Branched evolutions keep every branch with its own condition.
func TestEvolutions(t *testing.T) {
	d := load(t)
	var leafeon *Evolution
	from := d.EvolutionsFrom("eevee")
	for i := range from {
		if from[i].Into == "leafeon" {
			leafeon = &from[i]
		}
	}
	if len(from) != 8 || leafeon == nil || leafeon.Method != "item-hold" || leafeon.Item != "Leaf Stone" {
		t.Errorf("EvolutionsFrom(eevee) = %+v, want 8 branches incl. leafeon via item-hold Leaf Stone", from)
	}
	if into := d.EvolutionsInto("ninjask"); len(into) != 1 || into[0].From != "nincada" || into[0].Level != 20 {
		t.Errorf("EvolutionsInto(ninjask) = %+v, want nincada at level 20", into)
	}
}
