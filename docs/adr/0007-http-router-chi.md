# ADR-0007: HTTP router and middleware — adopt go-chi/chi

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

The first slice (registration & sign-in, ADR-0001 BDD loop) used the stdlib
`net/http.ServeMux`. Since Go 1.22 the stdlib mux does method-based routing
(`POST /auth/register`) and path wildcards (`/tasks/{id}` + `r.PathValue`), so
routing alone gave no reason to add a dependency, and we deliberately stayed on
the stdlib.

The authenticated-identity slice introduces the first **middleware** (bearer
token → actor) and the first **protected route group** (`GET /me` and, soon,
all per-actor endpoints). The stdlib offers no built-in middleware chaining or
route grouping; you compose `http.Handler` wrappers and sub-muxes by hand. We
flagged at the time that the first middleware would be the moment to revisit
this.

## Decision

Adopt **`github.com/go-chi/chi/v5`** as the router for the `internal/rest`
adapter. Public routes (`/healthz`, `/auth/*`) are registered directly;
protected routes live in a `chi` group behind a `requireAuth` middleware that
resolves the token via `auth.Service.ActorFromToken` and stashes the actor in
the request context.

chi is `http.Handler`-compatible and its core has no third-party dependencies.
It stays confined to the `internal/rest` adapter — the domain and application
services never import it.

## Consequences

- Clean middleware composition and route grouping for the auth-gated surface,
  which is most of the API to come.
- One more runtime dependency (after `oklog/ulid`); justified by the middleware
  ergonomics and kept behind the `rest.New` adapter boundary, so it is cheap to
  reverse.
- A consistent home for future cross-cutting middleware: request logging, panic
  recovery, request IDs (useful for event/federation tracing, ADR-0003), and
  CORS for the Flutter web client.

## Alternatives considered

- **Stay on stdlib `ServeMux` with a hand-rolled `chain()` helper** — zero
  router dependency, but reimplements grouping/middleware ergonomics that chi
  already provides well; rejected once a real middleware-bearing surface
  arrived.
- **Heavier frameworks (Gin, Echo)** — more batteries, but more opinionated and
  a larger surface than a hexagonal adapter needs; rejected.
