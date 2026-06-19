# Worklog

The running memory of ergonomos: what was done, when, and why. Newest entries
at the top. One entry per meaningful unit of work. Keep it honest — record
what was skipped or left unfinished too.

---

## 2026-06-19 — Project scaffolding and foundational decisions

Set up the repository skeleton and recorded the founding decisions.

- Established the stack and method: Go backend, Flutter (fvm-pinned) clients,
  PostgreSQL, REST/OpenAPI contract, BDD via godog, AGPL-3.0 + App Store
  exception. Recorded in **ADR-0001**.
- Decided to **embed OpenFGA in-process behind an `authz.Authorizer`
  interface** (single binary, shared Postgres) — **ADR-0002**.
- Decided on a **domain-event / transactional-outbox backbone** as the shared
  substrate for authz consistency, realtime, and future federation —
  **ADR-0003**.
- Fixed **federation-readiness constraints** (global opaque IDs, actor/instance
  model, home-instance owns grants, changes-are-events) without building
  federation — **ADR-0004**.
- Created the tree: `LICENSE` (+ exception), `CONTRIBUTING.md`/`DCO`,
  `Taskfile.yml`, `docker-compose.yml` (Postgres only — authz is embedded),
  `.github/workflows/ci.yml`, `api/openapi.yaml`, `CLAUDE.md`, and the
  `docs/` memory system (ADRs, this worklog, guidelines).
- Go skeleton is **stdlib-only** so the first commit builds and CI is green:
  `cmd/ergonomos` serves `/healthz`; `internal/authz`, `internal/events`, and
  `internal/domain` fix the key interfaces and types.
- Seeded the BDD loop with `server/features/auth.feature` (registration &
  authentication). Not yet wired to godog.

**Open / next:**
- Module path `github.com/ergonomos/ergonomos/server` is a placeholder until
  the GitHub org/repo is decided.
- Next slice: wire `auth.feature` to godog and implement registration & login
  red→green (this pulls in the first dependencies, Postgres access, and the
  first real `Authorizer`/event usage).
