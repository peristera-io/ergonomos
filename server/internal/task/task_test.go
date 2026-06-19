package task

import (
	"context"
	"testing"

	"github.com/peristera-io/ergonomos/server/internal/authz"
	"github.com/peristera-io/ergonomos/server/internal/domain"
	"github.com/peristera-io/ergonomos/server/internal/events"
)

func newActor() domain.Actor { return domain.Actor{ID: domain.NewID(), Local: true} }

type fixture struct {
	svc    *Service
	authz  *authz.Memory
	outbox *events.MemoryOutbox
}

func newFixture() fixture {
	a := authz.NewMemory()
	o := events.NewMemoryOutbox()
	return fixture{svc: NewService(NewMemoryStore(), a, o), authz: a, outbox: o}
}

func TestCreateWritesOwnerTupleAndEmitsEvent(t *testing.T) {
	f := newFixture()
	ada := newActor()

	got, err := f.svc.Create(context.Background(), ada, "Write acceptance tests")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID == "" || got.Owner != ada.ID {
		t.Fatalf("unexpected task %+v", got)
	}

	ok, err := f.authz.Check(context.Background(), ownerTuple(ada.ID, got.ID))
	if err != nil || !ok {
		t.Fatalf("owner tuple not written (ok=%v err=%v)", ok, err)
	}

	evs := f.outbox.Events()
	if len(evs) != 1 || evs[0].Type != "task.created" || evs[0].Subject != "task:"+string(got.ID) {
		t.Fatalf("unexpected outbox events: %+v", evs)
	}
}

func TestCreateRejectsEmptyTitle(t *testing.T) {
	f := newFixture()
	for _, title := range []string{"", "   "} {
		if _, err := f.svc.Create(context.Background(), newActor(), title); err != ErrEmptyTitle {
			t.Fatalf("Create(%q) error = %v, want ErrEmptyTitle", title, err)
		}
	}
	if evs := f.outbox.Events(); len(evs) != 0 {
		t.Fatalf("expected no events for rejected creates, got %+v", evs)
	}
}

func TestGetEnforcesOwnership(t *testing.T) {
	f := newFixture()
	ada, grace := newActor(), newActor()

	created, err := f.svc.Create(context.Background(), ada, "ada's task")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := f.svc.Get(context.Background(), ada, created.ID); err != nil {
		t.Fatalf("owner Get: %v", err)
	}
	if _, err := f.svc.Get(context.Background(), grace, created.ID); err != ErrNotFound {
		t.Fatalf("non-owner Get error = %v, want ErrNotFound", err)
	}
}

func TestGetUnknownIDIsNotFound(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Get(context.Background(), newActor(), domain.NewID()); err != ErrNotFound {
		t.Fatalf("Get unknown error = %v, want ErrNotFound", err)
	}
}

func TestUpdateTitle(t *testing.T) {
	f := newFixture()
	ada, grace := newActor(), newActor()
	created, _ := f.svc.Create(context.Background(), ada, "Draft")

	updated, err := f.svc.UpdateTitle(context.Background(), ada, created.ID, "  Final draft  ")
	if err != nil {
		t.Fatalf("UpdateTitle: %v", err)
	}
	if updated.Title != "Final draft" {
		t.Fatalf("title = %q, want %q", updated.Title, "Final draft")
	}

	if _, err := f.svc.UpdateTitle(context.Background(), ada, created.ID, "   "); err != ErrEmptyTitle {
		t.Fatalf("empty title error = %v, want ErrEmptyTitle", err)
	}
	if _, err := f.svc.UpdateTitle(context.Background(), grace, created.ID, "Hijacked"); err != ErrNotFound {
		t.Fatalf("non-owner update error = %v, want ErrNotFound", err)
	}

	got, _ := f.svc.Get(context.Background(), ada, created.ID)
	if got.Title != "Final draft" {
		t.Fatalf("persisted title = %q, want unchanged %q", got.Title, "Final draft")
	}
	if !hasEventType(f.outbox, "task.updated") {
		t.Fatalf("expected a task.updated event, got %+v", f.outbox.Events())
	}
}

func TestComplete(t *testing.T) {
	f := newFixture()
	ada, grace := newActor(), newActor()
	created, _ := f.svc.Create(context.Background(), ada, "Write acceptance tests")
	if created.Done {
		t.Fatal("new task should not be done")
	}

	done, err := f.svc.Complete(context.Background(), ada, created.ID)
	if err != nil || !done.Done {
		t.Fatalf("Complete: task=%+v err=%v", done, err)
	}
	// Idempotent.
	if again, err := f.svc.Complete(context.Background(), ada, created.ID); err != nil || !again.Done {
		t.Fatalf("second Complete: task=%+v err=%v", again, err)
	}
	if _, err := f.svc.Complete(context.Background(), grace, created.ID); err != ErrNotFound {
		t.Fatalf("non-owner complete error = %v, want ErrNotFound", err)
	}
	if !hasEventType(f.outbox, "task.completed") {
		t.Fatalf("expected a task.completed event, got %+v", f.outbox.Events())
	}
}

func TestDelete(t *testing.T) {
	f := newFixture()
	ada, grace := newActor(), newActor()
	created, _ := f.svc.Create(context.Background(), ada, "Write acceptance tests")

	if err := f.svc.Delete(context.Background(), grace, created.ID); err != ErrNotFound {
		t.Fatalf("non-owner delete error = %v, want ErrNotFound", err)
	}
	if err := f.svc.Delete(context.Background(), ada, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := f.svc.Get(context.Background(), ada, created.ID); err != ErrNotFound {
		t.Fatalf("Get after delete error = %v, want ErrNotFound", err)
	}
	// Owner tuple retracted, so a second delete is also not found.
	if err := f.svc.Delete(context.Background(), ada, created.ID); err != ErrNotFound {
		t.Fatalf("second delete error = %v, want ErrNotFound", err)
	}
	if !hasEventType(f.outbox, "task.deleted") {
		t.Fatalf("expected a task.deleted event, got %+v", f.outbox.Events())
	}
}

func hasEventType(o *events.MemoryOutbox, typ string) bool {
	for _, e := range o.Events() {
		if e.Type == typ {
			return true
		}
	}
	return false
}

func TestListReturnsOnlyOwnedTasksSorted(t *testing.T) {
	f := newFixture()
	ada, grace := newActor(), newActor()

	first, _ := f.svc.Create(context.Background(), ada, "first")
	second, _ := f.svc.Create(context.Background(), ada, "second")
	if _, err := f.svc.Create(context.Background(), grace, "grace's"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tasks, err := f.svc.List(context.Background(), ada)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("List returned %d tasks, want 2: %+v", len(tasks), tasks)
	}
	if tasks[0].ID != first.ID || tasks[1].ID != second.ID {
		t.Fatalf("List not sorted by id: got %s,%s want %s,%s", tasks[0].ID, tasks[1].ID, first.ID, second.ID)
	}
}
