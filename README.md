# poke-maxxer

Go HTTP app that helps pick the "best" 6-Pokémon team out of a caught roster.

## Defensive score

Scores a candidate team purely on **defensive type coverage**: does the team
have safe answers to every attack type, and does it avoid stacking shared
weaknesses? This does not consider base stats, movepools, or offensive
coverage — those are separate, not-yet-specified scoring components.

Higher score is always better. The score can be negative; that's fine and
expected for teams with real defensive holes — only relative ordering between
candidate teams matters, never the absolute value or its sign.

### Inputs

- `Dex.Types` — the ordered list of all 18 attack types. It does not depend
  on `Options`: Fairy exists in both type charts even when the FAIRY TYPE
  setting is Off (that setting only changes Pokémon typings). Loop over this
  for every "for each attack type `a`" step below, not a hardcoded 18.
- `Dex.Effectiveness(o, a, defenderTypes...)` — the damage multiplier when
  attack type `a` hits a Pokémon with the given defending type(s). Already
  multiplies dual-type matchups together (e.g. Electric vs Water/Flying →
  2 × 2 = 4). Use this directly per team member per attack type; do not
  reimplement the multiplication.
- `Pokemon.TypesFor(o)` — a team member's one or two defending types.

### Step 1 — per-member multiplier, per attack type

For each attack type `a` and each team member `p`:

```go
mult := dex.Effectiveness(o, a, p.TypesFor(o)...)
```

Bucket `p` for that `a` (buckets are disjoint — a member lands in exactly
one, `mult == 1` counts as neutral and lands in none):

- `mult > 1` → weak
- `0 < mult < 1` → resist
- `mult == 0` → immune

### Step 2 — per-type counts

For each attack type `a`, across the team's 6 members:

```
weak(a)   = count of members bucketed weak
resist(a) = count of members bucketed resist (partial resist only, i.e.
            0 < mult < 1 — does NOT include immune members)
immune(a) = count of members bucketed immune (mult == 0)
```

### Step 3 — DefGap: weakness-concentration penalty, per attack type

```
DefGap(a) = weak(a)^2 / (1 + resist(a) + 2*immune(a))
```

- **Numerator squares `weak(a)`**: danger grows faster than linearly as more
  teammates share a weakness, because an opponent who finds that weakness can
  potentially knock out several team members in a row with one move type.
  Going from 5/6 to 6/6 weak costs far more than going from 0/6 to 1/6.
- **Denominator discounts by safe answers**: each resister lets you switch
  away from the danger, with diminishing returns — the first resister roughly
  halves the risk, the fourth barely moves it further.
- **Immune counts double** (`2*immune` vs `1*resist`): a hard 0× stop is
  categorically safer than a partial reduction, so it discounts the penalty
  more per unit.
- **`+1`** only avoids division by zero when a type has no weak/resist/immune
  members at all; it does not affect ranking.

### Step 4 — sum DefGap across all types

```
totalGap = Σ DefGap(a) for every a in dex.Types
```

### Step 5 — ResistBreadth: versatility count

For each attack type `a`, ask a strict yes/no question — does the team have
*at least one* safe answer? Cap each type's contribution at 1 regardless of
how many members qualify (5 resisters and 1 resister both count as 1):

```
ResistBreadth = count of types a where (resist(a) + immune(a)) >= 1
```

Max value equals `len(dex.Types)`. This measures breadth of coverage
("do I have *any* plan for this type"), which is a different question from
`DefGap` ("how exposed am I where I don't"). A type can be "covered" by
`ResistBreadth` (one immune member) while still being the team's worst
`DefGap` outlier (four other members weak to it) — both terms are needed,
neither alone tells the full story.

Immunity does **not** get extra weight here beyond the same +1 a plain
resist gets — the checkbox is already ticked once any safe answer exists,
covered is covered. Immunity's extra value is already priced into `DefGap`
via the `2*immune` term; giving it double credit here too would double-count
the same fact under two different terms.

### Step 6 — final score

```
DefensiveScore = ResistBreadth - totalGap
```

Maximize over candidate 6-member teams drawn from the caught roster.

### Worked example

Team: Charizard (Fire/Flying), Blastoise (Water), Venusaur (Grass/Poison),
Pikachu (Electric), Gengar (Ghost/Poison), Onix (Rock/Ground), with
`ModernTypes`, `FairyTypes`, `ModernStats` true and `ModernTypeChart` **false**
(the Gen VI chart). Under the app's default Modern preset (Modern chart) the
same team scores **-3.00**: Ice becomes uncovered and Ground drops to 2 weak
members. Both results were verified by running this formula against the actual embedded
`pokedata.Dex`, not hand-computed from general type-chart knowledge. This
repo's chart is sourced from a Modern Emerald decompile and can differ from
the mainline games' chart in specific matchups (e.g. this chart's `ground`
entry has no explicit `rock` multiplier under some presets); always trust
`Dex.Effectiveness`'s actual output over memorized type-chart trivia.

| Attack type | weak | resist | immune | DefGap | covered? |
|---|---|---|---|---|---|
| normal | — | Onix | Gengar | 0.00 | yes |
| fighting | Onix | Charizard, Venusaur | Gengar | 0.20 | yes |
| flying | Venusaur | Pikachu, Onix | — | 0.33 | yes |
| poison | — | Gengar, Onix | — | 0.00 | yes |
| ground | Pikachu, Gengar, Onix | — | Charizard | 3.00 | yes |
| rock | Charizard | Onix | — | 0.50 | yes |
| bug | — | Charizard, Gengar | — | 0.00 | yes |
| **ghost** | Gengar | — | — | 1.00 | **no** |
| steel | Onix | Charizard, Blastoise, Pikachu | — | 0.25 | yes |
| fire | Venusaur | Charizard, Blastoise, Onix | — | 0.25 | yes |
| water | Charizard, Onix | Blastoise, Venusaur | — | 1.33 | yes |
| grass | Blastoise, Onix | Charizard, Venusaur, Gengar | — | 1.00 | yes |
| electric | Charizard, Blastoise | Venusaur, Pikachu | Onix | 0.80 | yes |
| **psychic** | Venusaur, Gengar | — | — | **4.00** | **no** |
| ice | Venusaur, Onix | Blastoise | — | 2.00 | yes |
| **dragon** | — | — | — | 0.00 | **no** |
| **dark** | Gengar | — | — | 1.00 | **no** |
| fairy | — | Charizard, Venusaur, Gengar | — | 0.00 | yes |

```
ResistBreadth = 14      (4 uncovered: ghost, psychic, dragon, dark)
totalGap      = 15.67
DefensiveScore = 14 - 15.67 = -1.67
```

Three distinct "uncovered" shapes show up here, worth telling apart:

- **psychic** — 2 members weak, 0 resist/immune. Uncovered *and* the
  team's single largest `DefGap` (4.00, the worst row in the table): this is
  the real, dangerous hole.
- **ghost** / **dark** — only 1 member weak each, but still uncovered
  (nobody resists or is immune), so `DefGap` for these is a modest 1.00 —
  mildly annoying, not dangerous, but flagged anyway since no teammate has
  an edge there.
- **dragon** — 0 members weak, 0 resist/immune, `DefGap` = 0.00 — a pure
  "blind spot": nobody on the team interacts with this type at all. Not
  dangerous today, but also not a strength; ResistBreadth is what makes
  this visible, since DefGap alone would call it a non-issue.

Contrast with **ground**: 3 of 6 members weak (a real concentration,
`DefGap` = 3.00, second-worst row) yet still "covered" by `ResistBreadth`
purely because Charizard is immune. This is exactly why both terms are
summed rather than either replacing the other — `ResistBreadth` alone would
call ground fine; `DefGap` alone (without the covered/uncovered breakdown)
wouldn't tell you psychic is worse than its `weak` count alone suggests.

This sample team's `DefensiveScore` of **-1.67** is unremarkable/mediocre —
not disastrous, but not tight either. A well-covered 6-member team assembled
deliberately for type synergy should score positive; use this team's numbers
as a regression fixture (same `Options`, same six species) when implementing.

### Implementation notes

- Build nothing per-team beyond looping `dex.Types` × team members — all
  data comes from the already-embedded `pokedata` package, no network calls
  and no additional PokéAPI or pokeemerald fetches needed.
- Selecting the best 6-of-N from a caught roster is `C(N,6)`; brute-force is
  fine for the roster sizes this app expects (no per-combination API calls,
  pure in-memory arithmetic over cached type data). Fall back to a greedy or
  hill-climbing search only if roster sizes make brute force impractical.
- Optional display-only adjustment: if negative scores look wrong to show a
  user, add a fixed constant offset (`DefensiveScore + C`) — this preserves
  ranking exactly since it shifts every team's score equally. Do **not**
  take `abs(DefensiveScore)`: that discards the sign, which is the part of
  the score that encodes good vs. bad, and would rank very bad teams
  (large negative) above merely mediocre ones (small positive) once both
  are made positive by the absolute value.
- Offensive coverage (does the team have super-effective answers against
  every type, weighted against being "walled") is a planned separate
  scoring component, not covered by this formula.
