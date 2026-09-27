package main

import (
	"log"
	"net/http"
	"os"

	"poke-maxxer/pokedata"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := "localhost:" + port

	dex, err := pokedata.Load()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("loaded %d Pokémon, %d evolutions, %d types", len(dex.Pokemon), len(dex.Evolutions), len(dex.Types))

	mux := http.NewServeMux()
	a := newApp(dex)
	mux.HandleFunc("GET /{$}", a.lookup)
	mux.HandleFunc("GET /team", a.team)

	log.Printf("poke-maxxer listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
