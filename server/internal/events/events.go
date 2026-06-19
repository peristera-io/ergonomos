// Package events defines the domain-event / transactional-outbox backbone.
//
// Per ADR-0003 a single event stream is the substrate for three concerns:
// keeping Postgres and the embedded authz store consistent, driving realtime
// websocket fan-out, and (later) federation delivery. The dispatcher and
// consumers land in a later slice; this file fixes the shape.
package events

import (
	"context"
	"time"
)

// Event is an immutable record of something that happened in the domain.
type Event struct {
	ID         string    // opaque event id
	Type       string    // e.g. "task.created", "project.shared"
	Subject    string    // "type:id" of the affected object
	OccurredAt time.Time // UTC
	Payload    []byte    // JSON body
}

// Outbox persists events atomically with the domain write that produced them
// (same database transaction), giving at-least-once delivery. Consumers must
// therefore be idempotent.
type Outbox interface {
	// Append stores events as part of an ongoing transaction.
	Append(ctx context.Context, events ...Event) error
}
