# Operating manual — ergonomos

This file is the entry point for anyone (human or LLM agent) working in this
repository. Read it, plus `docs/worklog.md` (what has been done) and the
relevant ADRs in `docs/adr/`, before starting work.

## What this project is

A hybrid task / project / documentation manager. Multi-user, team-aware,
fine-grained sharing, designed not to preclude federation between ergonomos
servers. See `README.md` for the stack and `docs/adr/` for the reasoning.

## How we work here

We use **specification by example (BDD)** as living documentation, plus fast
tests as the safety net.

1. **Specify.** Express new behavior as a `.feature` file in
   `server/features/` (Gherkin). Keep scenarios at the domain/API level, in
   plain language a non-programmer could read.
2. **Red.** Wire the scenario to a failing `godog` step (or a failing Go test
   for sub-feature logic).
3. **Green.** Implement the smallest change through the layers to pass.
4. **Refactor.** Clean up; keep tests green.

Do **not** force Gherkin onto unit-level logic — use ordinary Go table tests
there. BDD is for behavior; unit tests are for mechanics.

## Document your work (this is the project's memory)

After any meaningful change:

- **Append to `docs/worklog.md`** — one dated entry: what changed and why.
  This is the running memory; keep it current.
- **Add an ADR** (`docs/adr/NNNN-title.md`) when you make a *decision* that is
  non-obvious or would be expensive to reverse. Record the context, the
  decision, and the consequences — not just the outcome.
- **Add to `docs/guidelines/`** when a reusable convention emerges (a pattern
  worth repeating, a pitfall worth avoiding). Guidelines are discovered, not
  decreed — promote them from real experience.

## Conventions (current)

- **Identifiers are opaque and globally unique** (ULID/UUID), never sequential
  integers exposed across the API. Shareable resources also carry a canonical
  URI and a home-instance marker. (ADR-0004)
- **Authorization goes through the `authz.Authorizer` interface only.** Never
  call the authz backend directly from handlers. (ADR-0002)
- **Cross-cutting effects flow through the event/outbox backbone**
  (`internal/events`): authz-tuple sync, realtime fan-out, and future
  federation delivery all consume the same event stream. (ADR-0003)
- **The OpenAPI spec (`api/openapi.yaml`) is the source of truth** for the
  REST surface. Change the spec first, then the code. (ADR-0001)
- **Keep a hexagonal core.** The task/domain engine depends only on ports
  (interfaces); HTTP, auth, and storage are adapters behind them. ergonomos is
  embeddable over the network (API + federation), not as a linked library, so
  the engine stays in `internal/`. (ADR-0006)
- **Task is a first-class aggregate** — it can exist, be shared, and sync on
  its own, not only nested under a Project/Document. (ADR-0006)
- **Don't assume the caller is the interactive user.** Authorization must leave
  room for third-party apps (OAuth2 / scoped tokens), distinct from end-user
  authentication. (ADR-0006)
- **Times are UTC.** Money/quantities, if any, are integers in base units.

## Conventions for working with this repo

- Keep the first commit and CI green: the Go module is currently
  **stdlib-only**. Introduce dependencies deliberately, and run `go mod tidy`.
- Match the style of surrounding code. Comments explain *why*, not *what*.
- Small, reviewable changes over large drops.
