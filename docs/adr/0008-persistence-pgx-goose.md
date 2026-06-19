# ADR-0008: Persistence — pgx, goose, and a context-propagated transaction

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

The first feature slices (auth, identity, task CRUD) ran entirely on in-memory
adapters behind the ports from ADR-0006. That validated the hexagonal seams
cheaply, but two foundational bets remained unproven:

1. that the ports swap cleanly for a real database, and
2. that the transactional-outbox guarantee from ADR-0003 — the domain change
   and its event are written in the **same** transaction — actually holds.

Nothing was durable: a restart dropped every account, task, authorization
tuple, and event. This is the slice that makes the data real and exercises the
unit-of-work the outbox depends on.

The task write is not a single row: creating a task inserts the task, records
the `owner` authorization tuple, and appends a `task.created` event. For
at-least-once delivery to mean anything, those must commit or roll back
together.

## Decision

**Driver — `jackc/pgx/v5` (+ `pgxpool`).** The de-facto standard Postgres
driver for Go, with a native interface (no `database/sql` indirection) and a
pool. The pool and a transaction both satisfy the same small querier interface,
which the adapters depend on.

**Migrations — `pressly/goose/v3`, used as a library.** Schema lives in
`internal/postgres/migrations` as embedded SQL and is applied on boot via
`goose.Up` against an `embed.FS`. Chosen over a hand-rolled migrator (less code
we own, mature up/down + versioning) and over golang-migrate (lighter surface
for a single Postgres).

**Transaction boundary — a context-propagated transaction.** Rather than
threading a `*pgx.Tx` through every port signature (which would leak the driver
into the domain), the `postgres.DB` exposes `WithinTx(ctx, fn)`: it begins a
transaction, stashes it in the returned `context.Context`, runs `fn`, then
commits (or rolls back on error). Every Postgres adapter method resolves its
querier from the context — the ambient transaction if one is present, otherwise
the pool. Port signatures are unchanged (they already take `ctx`).

The application service depends on a tiny consumer-defined `Transactor` port so
it can wrap a multi-step write atomically. The in-memory wiring supplies a
`NopTransactor` that simply calls `fn(ctx)`, so the BDD suite stays in-memory,
fast, and database-free.

**Authorization tuples are written inline, in the same transaction.** ADR-0003
envisions authz-tuple sync as a *consumer* of the event stream (eventually
consistent). For this embedded, single-process phase we instead write the
`owner` tuple inline within the task-create transaction — strongly consistent
and simpler, with no dispatcher yet to run. The tuples live in an
`authz_tuples` table behind a Postgres `Authorizer` that keeps the exact-tuple
semantics of the in-memory one; **this is interim**. Embedding OpenFGA
(ADR-0002) and moving tuple sync onto the event stream (ADR-0003) land with the
sharing slice, when relation rewrites and federation actually need them.

**Sessions stay in-memory.** Bearer tokens remain process-local with no
expiry; a persistent, expiring session store is still deferred (it does not
block durability of accounts/tasks).

**Selection is by environment.** `cmd/ergonomos` uses the Postgres adapters
when `DATABASE_URL` is set (running migrations on boot) and the in-memory
adapters otherwise. Postgres adapter tests are integration tests that skip
unless `DATABASE_URL` is set; CI runs them against a Postgres service.

## Consequences

- The outbox guarantee is now real and testable: an atomicity test forces a
  failure mid-write and asserts the task did not persist.
- Two runtime dependencies (`pgx`, `goose`); both confined to
  `internal/postgres` (and `main` for wiring). The domain and services never
  import them.
- The context-tx pattern keeps the ports clean but is implicit: an adapter that
  forgets to resolve its querier from the context would silently escape the
  transaction. The querier helper is the single chokepoint that prevents this.
- Authz durability arrives now (in `authz_tuples`) but on an interim exact-tuple
  backend, not OpenFGA — a known gap until the sharing slice.
- Local `go test ./...` stays green without Docker; full coverage of the
  Postgres path requires `DATABASE_URL` (compose provides it locally, CI a
  service container).

## Alternatives considered

- **Thread `*pgx.Tx` through port methods** — explicit, but leaks the driver
  into every interface and the domain; rejected for breaking the hexagonal
  boundary.
- **Hand-rolled migrator** — fewer dependencies, but reinvents versioning and
  up/down for no real gain; rejected.
- **Write the owner tuple via an event consumer now (per ADR-0003)** — the
  architecturally "pure" path, but requires building the dispatcher before it
  earns its place and adds eventual-consistency complexity to a single-process
  binary; deferred to the sharing/OpenFGA slice.
- **testcontainers for all tests** — highest fidelity, but makes Docker a
  prerequisite for the whole suite and slows the inner loop; rejected in favour
  of gated integration tests.
