# Worklog

The running memory of ergonomos: what was done, when, and why. Newest entries
at the top. One entry per meaningful unit of work. Keep it honest — record
what was skipped or left unfinished too.

---

## 2026-06-19 — Third slice: task management + first authz/outbox use

The first slice with a domain aggregate. Tasks can be created, retrieved, and
listed — and, crucially, this is where `authz.Authorizer` (ADR-0002) and the
event outbox (ADR-0003) finally do real work instead of just existing as ports.

- **Strict BDD harness.** Flipped `godog.Options.Strict = true`. Until now a new
  `.feature` with no step definitions passed silently (undefined steps are
  non-fatal by default), so the spec couldn't go red. Strict makes undefined or
  pending steps fail the run — the red in red→green is now real.
- **Spec first (ADR-0001):** rewrote `task.feature` as "Task management" with
  cross-user isolation as the centerpiece (another user gets 404 for my task;
  my list shows only my tasks with a second user's task present), plus
  empty-title 400, unknown-id 404, and the unauthenticated 401 cases.
  `api/openapi.yaml` gains `Task`/`TaskInput` schemas and `POST /tasks`,
  `GET /tasks`, `GET /tasks/{id}` (all bearer-protected).
- **`internal/task` aggregate (ADR-0006).** `Task{ID,Title,Owner,CreatedAt}`
  with a `Store` port (in-memory adapter) and a `Service` over three ports:
  `Store`, `authz.Authorizer`, `events.Outbox`. `Create` validates → stores →
  writes the `user:<id> owner task:<id>` tuple → appends a `task.created`
  event. `Get` runs `authz.Check` **first**, so unknown id and "someone else's
  task" both collapse to `ErrNotFound` — no existence leak. `List` is
  owner-scoped, sorted by ULID.
- **`events.MemoryOutbox`** added (slice+mutex, `Events()` accessor) — first
  concrete outbox. The transactional Postgres adapter replaces it later behind
  the same port.
- **REST:** `rest.New` now takes the task service; task handlers live in the
  existing `requireAuth` chi group. `main.go` wires `authz.NewMemory()` and
  `events.NewMemoryOutbox()`.
- **Acceptance:** 16 scenarios / 76 steps green (7 prior + 9 new). Unit tests on
  `task.Service` cover the tuple write, the emitted event, ownership
  enforcement, and owner-scoped sorted listing.

**Deferred (unchanged + new):**
- The authz model is exact-tuple match only (`authz.Memory`); relation rewrites
  (`editor` implies `viewer`, sharing) arrive with embedded OpenFGA. `Get`
  therefore checks `owner` directly — viewers/sharing land with that slice.
- The outbox is in-memory and nothing consumes it yet; the dispatcher
  (authz-sync, realtime, federation) is still roadmap (ADR-0003).
- No update/delete/complete on tasks yet; this slice is create/read/list.

---

## 2026-06-19 — Second slice: authenticated identity (GET /me)

Closed the "tokens minted but never validated" gap from the previous slice.
The server now recognises a caller from their bearer token.

- **Spec first:** `identity.feature` (fetch own profile with a valid token;
  reject missing token; reject unknown token). `api/openapi.yaml` gains a
  `bearerAuth` security scheme and `GET /me` (200 `Actor` / 401 `Problem`).
- **Sessions are now a port (ADR-0006).** `auth.Sessions` (`Issue`/`Resolve`)
  with an in-memory adapter; `Service.Register`/`Authenticate` mint tokens
  through it, and `Service.ActorFromToken` resolves a token back to its actor.
  Token generation moved out of the service into the adapter. Added
  `ErrInvalidToken`.
- **Adopted chi (ADR-0007).** `internal/rest` now uses `go-chi/chi/v5`: public
  routes plus a protected group behind a `requireAuth` middleware that resolves
  the token and stashes the actor in the request context. This realises the
  router decision deferred in the previous entry — first middleware, so chi
  earns its place. Bearer parsing is case-insensitive (RFC 7235).
- **Dependency:** `github.com/go-chi/chi/v5` (v5.3.0) — runtime, zero-dep core,
  confined to the `rest` adapter.
- **Acceptance:** 7 scenarios / 28 steps green (4 prior + 3 new); unit test
  added for `ActorFromToken` incl. the invalid-token path.

**Still deferred (unchanged):**
- Sessions are in-memory and never expire/revoke; a persistent adapter with
  expiry lands with the Postgres slice. No outbox event on login yet.
- Third-party / OAuth2 scoped tokens (ADR-0006) remain roadmap; today's tokens
  are interactive-user session tokens.

---

## 2026-06-19 — First feature slice: registration & sign-in (red→green)

Wired `auth.feature` to godog and implemented the four scenarios end to end
through the real HTTP surface. This is the first slice with behavior, the first
dependency, and the first proof of the BDD loop.

- **Spec first (ADR-0001):** added `POST /auth/register` and
  `POST /auth/sessions` to `api/openapi.yaml`, with `Credentials`, `Actor`, and
  `Session` schemas and RFC 9457 problem responses (201/409 for register,
  200/401 for sign-in).
- **Hexagonal core (ADR-0006):** new `internal/auth` holds the `Service` plus a
  `Store` port; `internal/rest` is the HTTP adapter. `cmd/ergonomos` wires them.
- **In-memory first, deliberately.** The auth `Store` uses an in-memory adapter;
  Postgres and embedded-OpenFGA are *not* pulled in yet — they implement the
  same ports in a later slice. Cheap and reversible because it's behind a port.
  (Revises the earlier "auth slice pulls in Postgres" note; it doesn't have to.)
- **Passwords** are salted PBKDF2-SHA256 via the stdlib `crypto/pbkdf2` (Go 1.24+),
  encoded self-describingly (`pbkdf2-sha256$iter$salt$hash`); verify is
  constant-time.
- **Actor IDs are ULIDs** via `domain.NewID`, using `oklog/ulid` with a
  monotonic entropy source seeded from `crypto/rand` (guarded by a mutex — the
  monotonic reader isn't concurrency-safe). Chose ULID over hand-rolled UUIDv4
  for time-ordered sortability (DB index locality) and to use a tested library;
  this is the concrete pick within ADR-0004's "ULID/UUID".
- **Dependencies:** `github.com/cucumber/godog` (v0.15.1, **test-only**) and
  `github.com/oklog/ulid/v2` (v2.1.1, **runtime** — first runtime dep, taken
  deliberately for IDs). Ran `go mod tidy`; `go.sum` now exists.
- **Acceptance:** 4 scenarios / 17 steps green; each scenario runs a fresh
  instance via `httptest` against `rest.New`. CI updated to cache on
  `server/go.sum` (the "enable caching once deps land" TODO is done).

**Deferred (not blocking, documented so it isn't lost):**
- No event emitted on registration yet (ADR-0003 outbox); wire it with the
  first shareable-resource slice where authz tuples also start mattering.
- Tokens are opaque and minted but not persisted/validated — there are no
  protected routes yet. Session storage + auth middleware come with the first
  authenticated endpoint; OAuth2 / scoped tokens (ADR-0006) layer on later.
- **Router:** evaluated chi vs. stdlib `ServeMux`. Go 1.22+ `ServeMux` already
  does method + path-param routing, so chi adds no routing value yet; its real
  edge is middleware composition / route groups. Decision: stay on `ServeMux`
  and adopt chi *in the slice that introduces the first middleware* (token
  auth, logging, recovery, CORS). The router sits behind `rest.New`, so the
  switch is localized — record it then (worklog/ADR).

---

## 2026-06-19 — Finalize org/repo and publish to GitHub

Repository published at **github.com/peristera-io/ergonomos** (public).

- Renamed the Go module to `github.com/peristera-io/ergonomos/server`.
- Filled the `cla.yml` placeholders: real `CLA.md` URL and added `jlspielmann`
  to the allowlist. (Action version still pinned to v2.3.0 — bump when handy.)
- Removed the module-path "placeholder" note from `README.md`.

---

## 2026-06-19 — Decide engine integration model (embeddable over the network)

Confirmed ergonomos can serve as an embeddable todo engine for two scenarios —
a third-party SaaS running its own instance with tasks mirrored to the user's
phone (federation), and a third-party app pushing tasks/completions into the
user's own server (API). Recorded **ADR-0006**.

- Chose **network embedding (API + federation)**, not a linked Go library;
  engine stays in `internal/`.
- Standing constraints added to `CLAUDE.md`: hexagonal core (ports/adapters),
  Task as a first-class aggregate, and "don't assume the caller is the
  interactive user."
- Roadmap items (not built, substrate ready): third-party app authorization
  (OAuth2 / scoped tokens) layered into the auth slice, and an outbound
  webhook / event-subscription surface on the outbox (ADR-0003).
- Licensing confirmed favorable: API integration is not a derivative work, so
  integrators aren't bound by the AGPL; code-embedding stays available via
  commercial dual-licensing (ADR-0005).

---

## 2026-06-19 — Add CONTRIBUTORS.md stub

Added a curated `CONTRIBUTORS.md` acknowledgements list (seeded with the
maintainer), explicitly distinct from the legal CLA-signature record
(`signatures/cla.json`) and from git history. Linked it from `CONTRIBUTING.md`.

---

## 2026-06-19 — Switch contribution model from DCO to CLA

Replaced the DCO with a **Contributor License Agreement that includes a
relicensing grant** (**ADR-0005**, amending ADR-0001), to preserve the option
to relicense/dual-license in future without chasing down past contributors.

- Added `CLA.md` (individual CLA; contributors keep copyright, grant a broad
  license including the right to relicense; moral-rights non-assertion for
  EU/Luxembourg). Template — needs legal review and a finalized "Us" party +
  governing jurisdiction.
- Added `.github/workflows/cla.yml` (CLA Assistant lite); signatures stored in
  `signatures/cla.json`, committed by `GITHUB_TOKEN`.
- Removed the `DCO` file and the `git commit -s` requirement.
- Updated `CONTRIBUTING.md`, `README.md`, `LICENSE-EXCEPTION.md`, and the ADR
  index to point at the CLA.

**Publish-time TODO (in `cla.yml`):** set `path-to-document` to the real
`CLA.md` URL, add the repo owner to `allowlist`, and pin the action to its
latest release.

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
