// Command ergonomos is the ergonomos backend server.
//
// At this stage it serves only a liveness probe; feature slices are added
// red→green from the specs in server/features. See CLAUDE.md.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := ":" + port()
	log.Printf("ergonomos listening on %s", addr)
	if err := http.ListenAndServe(addr, routes()); err != nil {
		log.Fatal(err)
	}
}

func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}

// routes builds the HTTP handler. Kept separate from main so tests can
// exercise it without binding a socket.
func routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	return mux
}
