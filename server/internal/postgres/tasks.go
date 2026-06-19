package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/peristera-io/ergonomos/server/internal/domain"
	"github.com/peristera-io/ergonomos/server/internal/task"
)

// Create implements task.Store.
func (db *DB) Create(ctx context.Context, t task.Task) error {
	_, err := db.q(ctx).Exec(ctx,
		`INSERT INTO tasks (id, title, owner_id, done, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		string(t.ID), t.Title, string(t.Owner), t.Done, t.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres: create task: %w", err)
	}
	return nil
}

// Get implements task.Store, returning task.ErrNotFound when no row matches.
func (db *DB) Get(ctx context.Context, id domain.ID) (task.Task, error) {
	var (
		t              task.Task
		gotID, ownerID string
	)
	err := db.q(ctx).QueryRow(ctx,
		`SELECT id, title, owner_id, done, created_at FROM tasks WHERE id = $1`, string(id),
	).Scan(&gotID, &t.Title, &ownerID, &t.Done, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return task.Task{}, task.ErrNotFound
		}
		return task.Task{}, fmt.Errorf("postgres: get task: %w", err)
	}
	t.ID = domain.ID(gotID)
	t.Owner = domain.ID(ownerID)
	return t, nil
}

// Update implements task.Store, returning task.ErrNotFound when the task is absent.
func (db *DB) Update(ctx context.Context, t task.Task) error {
	tag, err := db.q(ctx).Exec(ctx,
		`UPDATE tasks SET title = $2, owner_id = $3, done = $4 WHERE id = $1`,
		string(t.ID), t.Title, string(t.Owner), t.Done,
	)
	if err != nil {
		return fmt.Errorf("postgres: update task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return task.ErrNotFound
	}
	return nil
}

// Delete implements task.Store, returning task.ErrNotFound when the task is absent.
func (db *DB) Delete(ctx context.Context, id domain.ID) error {
	tag, err := db.q(ctx).Exec(ctx, `DELETE FROM tasks WHERE id = $1`, string(id))
	if err != nil {
		return fmt.Errorf("postgres: delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return task.ErrNotFound
	}
	return nil
}

// ListByOwner implements task.Store.
func (db *DB) ListByOwner(ctx context.Context, owner domain.ID) ([]task.Task, error) {
	rows, err := db.q(ctx).Query(ctx,
		`SELECT id, title, owner_id, done, created_at FROM tasks WHERE owner_id = $1`, string(owner),
	)
	if err != nil {
		return nil, fmt.Errorf("postgres: list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		var (
			t              task.Task
			gotID, ownerID string
		)
		if err := rows.Scan(&gotID, &t.Title, &ownerID, &t.Done, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan task: %w", err)
		}
		t.ID = domain.ID(gotID)
		t.Owner = domain.ID(ownerID)
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list tasks: %w", err)
	}
	return tasks, nil
}
