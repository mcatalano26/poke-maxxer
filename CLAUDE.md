# poke-maxxer

Go HTTP app (`main.go`) for the Pokémon Modern Emerald ROM hack. Pages: `/` looks up one Pokémon (`lookup.go`); `/team` analyzes up to 6 pasted Pokémon, showing base stats, defensive type coverage, and the DefensiveScore specified in `README.md` (`team.go`). Shared handler plumbing lives in `web.go`, templates in `templates/`.

## Game data (`pokedata/`)

- Typings, base stats, evolutions, and type charts come ONLY from the static JSON in `pokedata/`, embedded into the binary via `go:embed` and read through `pokedata.Load()`. Never fetch this data at runtime, and never take it from PokéAPI: its values differ from Modern Emerald's.
- Source: extracted once from the Modern Emerald decompilation, https://github.com/resetes12/pokeemerald. Each JSON file records the exact source `commit`. Don't hand-edit values. To pick up a hack update, re-extract from a newer commit and update the `commit` fields.
- Scope is only typings, base stats, evolutions, and type charts. Don't add other species data.
- Option-dependent values are stored for every setting and selected with `pokedata.Options`, which mirrors the in-game settings:
  - `POKéMON TYPES` (Original/Modern) and `FAIRY TYPE` (Off/On) → `pokemon.json` `types.{original,modern}.{fairyOff,fairyOn}`. Resolution follows `GetTypeBySpecies` in `src/pokemon.c`, including its rule-order quirks (e.g. Meganium stays Grass/Fairy with Modern typings even when Fairy is Off).
  - `POKéMON STATS` (Original Gen III/Modern) → `pokemon.json` `stats.{original,modern}`.
  - `EFFECTIVENESS` (Gen VI/Modern) → `type_chart.json` `charts.{gen6,modern}`. Only non-1× matchups are listed; multipliers combine multiplicatively across a defender's types.
- Evolutions (`evolutions.json`) don't depend on any setting. Method names and their parameters are documented on `pokedata.Evolution`.
- Excluded from the data: `SPECIES_NONE`, the `OLD_UNOWN` placeholders, and the debug-only `SPECIES_TEST`.

PokéAPI may be used only for data outside that scope, and must follow the rules below.

## PokéAPI requirements

Source: https://pokeapi.co/docs/v2 — "Information", "Fair Use Policy", "Resource Lists/Pagination".
Violating the fair use policy results in a **permanent IP ban**. Every change touching PokéAPI MUST comply.

### Fair use policy (mandatory)

- **Locally cache resources whenever you request them.** Every PokéAPI response MUST be cached locally; the same resource MUST NOT be fetched twice when a cached copy exists. Never proxy user requests straight through to PokéAPI.
- **Limit request frequency.** Rate limiting was removed in November 2018, but the maintainers ask clients to limit request frequency to limit their hosting costs. No tight loops, no unbounded concurrent fetches, no polling.
- **No denial-of-service behavior.** PokéAPI is primarily an educational tool; traffic that degrades it for others is not tolerated.
- **Be nice and friendly** to fellow PokéAPI developers.
- **Report security vulnerabilities responsibly** via https://github.com/PokeAPI/pokeapi/blob/master/SECURITY.md#reporting-a-vulnerability.

### API usage

- **Use v2 only**: base URL `https://pokeapi.co/api/v2/`. v1 is retired.
- **Read-only**: only HTTP `GET` exists on resources.
- **No authentication**: send no API keys or auth headers.
- **Resource URLs**: `GET https://pokeapi.co/api/v2/{endpoint}/{id or name}/`.

### Pagination

- Calling an endpoint without an ID or name returns a paginated list.
- Default page size is 20. Use `?limit=N` to change it and `?offset=N` to page, e.g. `?limit=60&offset=60`.
- Follow `next` until it is `null`; `count` is the total.
- Named endpoints return `NamedAPIResourceList` (`results[]` has `name` + `url`).
- Unnamed endpoints return `APIResourceList` (`results[]` has `url` only): `characteristic`, `contest-effect`, `evolution-chain`, `machine`, `super-contest-effect`. These are fetched by numeric ID only.
