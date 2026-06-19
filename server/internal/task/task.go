// Package task is the task aggregate and its application service (ADR-0006:
// Task is a first-class aggregate, not merely nested under a project).
//
// The service depends only on ports: a Store for persistence, an
// authz.Authorizer for access control (ADR-0002 — never the backend directly),
// and an events.Outbox for the change stream (ADR-0003). The v1 adapters are
// in-memory; Postgres and embedded-OpenFGA implement the same ports later.
package task

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/peristera-io/ergonomos/server/internal/authz"
	"github.com/peristera-io/ergonomos/server/internal/domain"
	"github.com/peristera-io/ergonomos/server/internal/events"
)

// Service-level errors.
var (
	// ErrNotFound is returned when no task is visible to the caller under the
	// given id. Unknown id and "exists but not yours" collapse to the same
	// error so the API cannot be used to probe for tasks owned by others.
	ErrNotFound = errors.New("task: not found")
	// ErrEmptyTitle is returned when a task title is blank.
	ErrEmptyTitle = errors.New("task: title must not be empty")
)

// ownerRelation is the relation a creator holds over their task.
const ownerRelation authz.Relation = "owner"

// Task is a unit of work owned by an actor.
type Task struct {
	ID        domain.ID
	Title     string
	Owner     domain.ID
	Done      bool
	CreatedAt time.Time // UTC
}

// Store persists tasks.
type Store interface {
	// Create stores t.
	Create(ctx context.Context, t Task) error
	// Get returns the task with id, or ErrNotFound if none exists.
	Get(ctx context.Context, id domain.ID) (Task, error)
	// Update replaces the stored task with t, or ErrNotFound if it is absent.
	Update(ctx context.Context, t Task) error
	// ListByOwner returns the tasks owned by owner, in any order.
	ListByOwner(ctx context.Context, owner domain.ID) ([]Task, error)
}

// Service is the task application service.
type Service struct {
	store  Store
	authz  authz.Authorizer
	outbox events.Outbox
}

// NewService builds a Service over its ports.
func NewService(store Store, authorizer authz.Authorizer, outbox events.Outbox) *Service {
	return &Service{store: store, authz: authorizer, outbox: outbox}
}

// Create stores a new task owned by owner, records the owner authorization
// tuple, and appends a task.created event.
func (s *Service) Create(ctx context.Context, owner domain.Actor, title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrEmptyTitle
	}
	t := Task{
		ID:        domain.NewID(),
		Title:     title,
		Owner:     owner.ID,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.Create(ctx, t); err != nil {
		return Task{}, err
	}
	if err := s.authz.Write(ctx, []authz.Tuple{ownerTuple(t.Owner, t.ID)}, nil); err != nil {
		return Task{}, err
	}
	if err := s.outbox.Append(ctx, createdEvent(t)); err != nil {
		return Task{}, err
	}
	return t, nil
}

// Get returns the task with id if actor owns it, else ErrNotFound.
func (s *Service) Get(ctx context.Context, actor domain.Actor, id domain.ID) (Task, error) {
	return s.owned(ctx, actor, id)
}

// UpdateTitle changes the title of actor's task and appends a task.updated
// event. A non-owner (or unknown id) gets ErrNotFound; an empty title is
// rejected only after ownership is confirmed, so a non-owner cannot tell an
// invalid title from a missing task.
func (s *Service) UpdateTitle(ctx context.Context, actor domain.Actor, id domain.ID, title string) (Task, error) {
	t, err := s.owned(ctx, actor, id)
	if err != nil {
		return Task{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrEmptyTitle
	}
	t.Title = title
	if err := s.store.Update(ctx, t); err != nil {
		return Task{}, err
	}
	if err := s.outbox.Append(ctx, changedEvent("task.updated", t)); err != nil {
		return Task{}, err
	}
	return t, nil
}

// Complete marks actor's task done and appends a task.completed event. It is
// idempotent: completing an already-done task leaves it done. A non-owner (or
// unknown id) gets ErrNotFound.
func (s *Service) Complete(ctx context.Context, actor domain.Actor, id domain.ID) (Task, error) {
	t, err := s.owned(ctx, actor, id)
	if err != nil {
		return Task{}, err
	}
	t.Done = true
	if err := s.store.Update(ctx, t); err != nil {
		return Task{}, err
	}
	if err := s.outbox.Append(ctx, changedEvent("task.completed", t)); err != nil {
		return Task{}, err
	}
	return t, nil
}

// owned returns actor's task, or ErrNotFound. The authorization check runs
// before the store read, so a caller cannot distinguish "no such task" from
// "someone else's task".
func (s *Service) owned(ctx context.Context, actor domain.Actor, id domain.ID) (Task, error) {
	ok, err := s.authz.Check(ctx, ownerTuple(actor.ID, id))
	if err != nil {
		return Task{}, err
	}
	if !ok {
		return Task{}, ErrNotFound
	}
	return s.store.Get(ctx, id)
}

// List returns the tasks owned by actor, sorted by id (ULIDs sort by creation
// time, ADR-0004).
func (s *Service) List(ctx context.Context, actor domain.Actor) ([]Task, error) {
	tasks, err := s.store.ListByOwner(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks, nil
}

func ownerTuple(owner, taskID domain.ID) authz.Tuple {
	return authz.Tuple{
		Subject:  authz.Ref("user:" + owner),
		Relation: ownerRelation,
		Object:   authz.Ref("task:" + taskID),
	}
}

func createdEvent(t Task) events.Event {
	e := taskEvent("task.created", t)
	e.OccurredAt = t.CreatedAt
	return e
}

// changedEvent records a mutation that happened now (update, completion).
func changedEvent(eventType string, t Task) events.Event {
	e := taskEvent(eventType, t)
	e.OccurredAt = time.Now().UTC()
	return e
}

func taskEvent(eventType string, t Task) events.Event {
	payload, _ := json.Marshal(struct {
		ID    domain.ID `json:"id"`
		Title string    `json:"title"`
		Owner domain.ID `json:"owner"`
		Done  bool      `json:"done"`
	}{ID: t.ID, Title: t.Title, Owner: t.Owner, Done: t.Done})
	return events.Event{
		ID:      string(domain.NewID()),
		Type:    eventType,
		Subject: "task:" + string(t.ID),
		Payload: payload,
	}
}
