# ADR-0003: Domain-event / transactional-outbox backbone

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

Three otherwise-separate requirements share an underlying need to react to
"something changed in the domain":

1. **Authz consistency** — domain data lives in Postgres, authorization tuples
   live in the embedded OpenFGA store (ADR-0002). They have no shared
   transaction, so a naive "write row, then write tuple" can drift on failure.
2. **Realtime** — clients need to be notified of changes over websockets
   (ADR-0001).
3. **Federation** — a future server-to-server protocol must deliver changes to
   other instances (ADR-0004).

## Decision

Adopt a **domain-event model with a transactional outbox**. Every domain
mutation, in the same database transaction, appends one or more immutable
`events.Event` records to an outbox. A dispatcher then delivers events to
consumers:

- the authz synchronizer (translates events into tuple writes),
- the realtime fan-out (websocket hub),
- (later) the federation delivery worker.

The shape is fixed now in `server/internal/events`; the dispatcher and
consumers are implemented in later slices.

## Consequences

- **One mechanism serves all three concerns**, instead of three bespoke
  integrations. This is the highest-leverage structural decision in the early
  design.
- Event production must be **atomic with the domain write** (same transaction)
  to get at-least-once delivery; consumers must therefore be **idempotent**.
- Introduces eventual consistency between Postgres and the authz store; reads
  that must be authoritative should check authz directly rather than relying
  on a cache.
- Events become a de-facto API surface; their `Type` and payload schema need
  care and versioning over time.

## Alternatives considered

- **Direct, synchronous dual writes** (write row + tuple inline) — simplest,
  but no recovery path when the second write fails, and nothing to reuse for
  realtime or federation.
- **External broker (Kafka/NATS) from day one** — more operational weight than
  warranted now; the Postgres-backed outbox can be fronted by a broker later
  without changing producers.
