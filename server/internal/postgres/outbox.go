package postgres

import (
	"context"
	"fmt"

	"github.com/peristera-io/ergonomos/server/internal/events"
)

// Append implements events.Outbox. It runs through db.q(ctx), so when called
// inside WithinTx the events commit atomically with the domain write that
// produced them (ADR-0003). published_at is left NULL for the dispatcher.
func (db *DB) Append(ctx context.Context, evs ...events.Event) error {
	q := db.q(ctx)
	for _, e := range evs {
		if _, err := q.Exec(ctx,
			`INSERT INTO outbox (id, type, subject, occurred_at, payload)
			 VALUES ($1, $2, $3, $4, $5)`,
			e.ID, e.Type, e.Subject, e.OccurredAt, string(e.Payload),
		); err != nil {
			return fmt.Errorf("postgres: outbox append: %w", err)
		}
	}
	return nil
}
