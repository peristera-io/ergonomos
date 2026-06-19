package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/peristera-io/ergonomos/server/internal/auth"
	"github.com/peristera-io/ergonomos/server/internal/authz"
	"github.com/peristera-io/ergonomos/server/internal/domain"
	"github.com/peristera-io/ergonomos/server/internal/events"
	"github.com/peristera-io/ergonomos/server/internal/task"
)

// testDB opens the database named by DATABASE_URL, applies migrations, and
// truncates every table so each test starts from a clean slate. Tests are
// skipped when DATABASE_URL is unset, so the default `go test ./...` stays
// driver-free (ADR-0008).
func testDB(t *testing.T) *DB {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping Postgres integration test")
	}
	ctx := context.Background()
	db, err := Open(ctx, url)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(db.Close)
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := db.pool.Exec(ctx, `TRUNCATE users, tasks, authz_tuples, outbox`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func TestUserStoreRoundTrip(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	u := auth.User{
		Actor: domain.Actor{
			ID:       domain.NewID(),
			Handle:   "ada",
			Instance: domain.Instance{Domain: "localhost"},
			Local:    true,
		},
		Email:        "ada@example.org",
		PasswordHash: "hash",
	}
	if err := db.CreateUser(ctx, u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	got, err := db.FindByEmail(ctx, "ada@example.org")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if got.Actor.ID != u.Actor.ID || got.Actor.Handle != "ada" || !got.Actor.Local || got.PasswordHash != "hash" {
		t.Fatalf("round-tripped user mismatch: %+v", got)
	}

	if err := db.CreateUser(ctx, u); !errors.Is(err, auth.ErrEmailTaken) {
		t.Fatalf("duplicate email error = %v, want ErrEmailTaken", err)
	}
	if _, err := db.FindByEmail(ctx, "missing@example.org"); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("missing email error = %v, want ErrNotFound", err)
	}
}

func TestTaskStoreCRUD(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	owner := domain.NewID()

	tk := task.Task{ID: domain.NewID(), Title: "draft", Owner: owner, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	if err := db.Create(ctx, tk); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := db.Get(ctx, tk.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "draft" || got.Owner != owner || got.Done {
		t.Fatalf("Get mismatch: %+v", got)
	}

	tk.Title = "final"
	tk.Done = true
	if err := db.Update(ctx, tk); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = db.Get(ctx, tk.ID)
	if got.Title != "final" || !got.Done {
		t.Fatalf("after Update: %+v", got)
	}

	if err := db.Create(ctx, task.Task{ID: domain.NewID(), Title: "other owner", Owner: domain.NewID(), CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("Create other: %v", err)
	}
	owned, err := db.ListByOwner(ctx, owner)
	if err != nil {
		t.Fatalf("ListByOwner: %v", err)
	}
	if len(owned) != 1 || owned[0].ID != tk.ID {
		t.Fatalf("ListByOwner returned %+v", owned)
	}

	if err := db.Delete(ctx, tk.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := db.Get(ctx, tk.ID); !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("Get after delete = %v, want ErrNotFound", err)
	}
	if err := db.Update(ctx, tk); !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("Update absent = %v, want ErrNotFound", err)
	}
	if err := db.Delete(ctx, tk.ID); !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("Delete absent = %v, want ErrNotFound", err)
	}
}

func TestAuthzStore(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	tuple := authz.Tuple{Subject: "user:ada", Relation: "owner", Object: "task:1"}

	ok, err := db.Check(ctx, tuple)
	if err != nil || ok {
		t.Fatalf("Check before write = (%v, %v), want (false, nil)", ok, err)
	}
	if err := db.Write(ctx, []authz.Tuple{tuple}, nil); err != nil {
		t.Fatalf("Write add: %v", err)
	}
	// Adds are idempotent.
	if err := db.Write(ctx, []authz.Tuple{tuple}, nil); err != nil {
		t.Fatalf("Write add (idempotent): %v", err)
	}
	if ok, _ := db.Check(ctx, tuple); !ok {
		t.Fatal("Check after write = false, want true")
	}
	if err := db.Write(ctx, nil, []authz.Tuple{tuple}); err != nil {
		t.Fatalf("Write remove: %v", err)
	}
	if ok, _ := db.Check(ctx, tuple); ok {
		t.Fatal("Check after remove = true, want false")
	}
}

func TestOutboxAppend(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ev := events.Event{
		ID:         string(domain.NewID()),
		Type:       "task.created",
		Subject:    "task:1",
		OccurredAt: time.Now().UTC(),
		Payload:    []byte(`{"id":"1"}`),
	}
	if err := db.Append(ctx, ev); err != nil {
		t.Fatalf("Append: %v", err)
	}
	var (
		count   int
		payload string
	)
	if err := db.pool.QueryRow(ctx, `SELECT count(*), max(payload::text) FROM outbox WHERE id = $1`, ev.ID).Scan(&count, &payload); err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	if count != 1 || payload != `{"id": "1"}` {
		t.Fatalf("outbox row = (count=%d, payload=%q)", count, payload)
	}
}

// TestWithinTxRollback verifies the unit-of-work boundary: when fn fails, every
// write it made through the context-carried transaction is rolled back.
func TestWithinTxRollback(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	tk := task.Task{ID: domain.NewID(), Title: "doomed", Owner: domain.NewID(), CreatedAt: time.Now().UTC()}

	boom := errors.New("boom")
	err := db.WithinTx(ctx, func(ctx context.Context) error {
		if err := db.Create(ctx, tk); err != nil {
			return err
		}
		if err := db.Append(ctx, events.Event{ID: string(domain.NewID()), Type: "task.created", Subject: "task:x", OccurredAt: time.Now().UTC(), Payload: []byte(`{}`)}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("WithinTx error = %v, want boom", err)
	}
	if _, err := db.Get(ctx, tk.ID); !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("task persisted despite rollback: Get = %v, want ErrNotFound", err)
	}
	var count int
	if err := db.pool.QueryRow(ctx, `SELECT count(*) FROM outbox`).Scan(&count); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if count != 0 {
		t.Fatalf("outbox has %d rows after rollback, want 0", count)
	}
}

// TestServiceCreateCommitsAtomically exercises the task service over the
// Postgres adapters end to end: a successful Create commits the task row, the
// owner authorization tuple, and the outbox event together.
func TestServiceCreateCommitsAtomically(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	svc := task.NewService(db, db, db, db)
	ada := domain.Actor{ID: domain.NewID(), Local: true}

	created, err := svc.Create(ctx, ada, "write integration tests")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := svc.Get(ctx, ada, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "write integration tests" {
		t.Fatalf("persisted task = %+v", got)
	}

	var count int
	if err := db.pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE subject = $1`, "task:"+string(created.ID)).Scan(&count); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if count != 1 {
		t.Fatalf("outbox event count = %d, want 1", count)
	}
}
