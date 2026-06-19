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
