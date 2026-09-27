package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := "localhost:" + port

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello from poke-maxxer")
	})

	log.Printf("poke-maxxer listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
