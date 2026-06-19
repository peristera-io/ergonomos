package postgres

import (
	"context"
	"fmt"

	"github.com/peristera-io/ergonomos/server/internal/authz"
)

// Check implements authz.Authorizer with an exact-tuple lookup (ADR-0008
// interim; OpenFGA with relation rewrites replaces it in the sharing slice).
func (db *DB) Check(ctx context.Context, t authz.Tuple) (bool, error) {
	var exists bool
	err := db.q(ctx).QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM authz_tuples
			WHERE subject = $1 AND relation = $2 AND object = $3
		)`,
		string(t.Subject), string(t.Relation), string(t.Object),
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("postgres: authz check: %w", err)
	}
	return exists, nil
}

// Write implements authz.Authorizer, adding then removing tuples. Adds are
// idempotent; removes ignore absent tuples.
func (db *DB) Write(ctx context.Context, add, remove []authz.Tuple) error {
	q := db.q(ctx)
	for _, t := range add {
		if _, err := q.Exec(ctx,
			`INSERT INTO authz_tuples (subject, relation, object)
			 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
			string(t.Subject), string(t.Relation), string(t.Object),
		); err != nil {
			return fmt.Errorf("postgres: authz write add: %w", err)
		}
	}
	for _, t := range remove {
		if _, err := q.Exec(ctx,
			`DELETE FROM authz_tuples WHERE subject = $1 AND relation = $2 AND object = $3`,
			string(t.Subject), string(t.Relation), string(t.Object),
		); err != nil {
			return fmt.Errorf("postgres: authz write remove: %w", err)
		}
	}
	return nil
}
