// Command ergonomos is the ergonomos backend server.
//
// Feature slices are added red→green from the specs in server/features. See
// CLAUDE.md. The HTTP surface lives in internal/rest; this command only wires
// the adapters together and listens.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/peristera-io/ergonomos/server/internal/auth"
	"github.com/peristera-io/ergonomos/server/internal/authz"
	"github.com/peristera-io/ergonomos/server/internal/domain"
	"github.com/peristera-io/ergonomos/server/internal/events"
	"github.com/peristera-io/ergonomos/server/internal/postgres"
	"github.com/peristera-io/ergonomos/server/internal/rest"
	"github.com/peristera-io/ergonomos/server/internal/task"
)

func main() {
	handler, err := routes(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	addr := ":" + port()
	log.Printf("ergonomos listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
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

// routes wires the delivery layer to its ports. When DATABASE_URL is set the
// Postgres adapters back tasks, users, authorization, and the outbox (ADR-0008),
// all sharing one connection pool and transaction boundary; otherwise the
// in-memory adapters are used. Sessions stay in-memory in both cases.
func routes(ctx context.Context) (http.Handler, error) {
	instance := domain.Instance{ID: domain.NewID(), Domain: instanceDomain()}

	if url := os.Getenv("DATABASE_URL"); url != "" {
		db, err := postgres.Open(ctx, url)
		if err != nil {
			return nil, err
		}
		if err := db.Migrate(ctx); err != nil {
			return nil, err
		}
		authsvc := auth.NewService(db, auth.NewMemorySessions(), instance)
		tasksvc := task.NewService(db, db, db, db)
		return rest.New(authsvc, tasksvc), nil
	}

	authsvc := auth.NewService(auth.NewMemoryStore(), auth.NewMemorySessions(), instance)
	tasksvc := task.NewService(task.NewMemoryStore(), authz.NewMemory(), events.NewMemoryOutbox(), task.NopTransactor{})
	return rest.New(authsvc, tasksvc), nil
}
