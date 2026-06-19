// Command ergonomos is the ergonomos backend server.
//
// Feature slices are added red→green from the specs in server/features. See
// CLAUDE.md. The HTTP surface lives in internal/rest; this command only wires
// the adapters together and listens.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/peristera-io/ergonomos/server/internal/auth"
	"github.com/peristera-io/ergonomos/server/internal/domain"
	"github.com/peristera-io/ergonomos/server/internal/rest"
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

func instanceDomain() string {
	if d := os.Getenv("ERGONOMOS_DOMAIN"); d != "" {
		return d
	}
	return "localhost"
}

// routes wires the in-memory adapters and the HTTP delivery layer. Postgres
// and embedded-OpenFGA adapters replace the in-memory ones in later slices,
// behind the same ports.
func routes() http.Handler {
	instance := domain.Instance{ID: domain.NewID(), Domain: instanceDomain()}
	authsvc := auth.NewService(auth.NewMemoryStore(), auth.NewMemorySessions(), instance)
	return rest.New(authsvc)
}
