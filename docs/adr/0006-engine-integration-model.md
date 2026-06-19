# ADR-0006: ergonomos as a network-embeddable engine (API + federation)

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

ergonomos should be usable as an embeddable "todo engine," in two concrete
shapes the maintainer described:

- **A.** A third-party SaaS runs its *own* ergonomos server, and the user sees
  those tasks mirrored on their phone.
- **B.** A third-party application pushes tasks and completions into the user's
  *own* ergonomos server.

Both are **network integrations** — federation (A) and API access (B) — not
cases of another program linking ergonomos as a library. We needed to confirm
nothing precludes this and to fix the boundary decisions that keep it open.

## Decision

ergonomos is consumed **over the network**, not as a linked Go library. The
engine code stays under `internal/` (intentionally not importable by other
modules). To support that:

1. **Hexagonal / ports-and-adapters core.** The task/domain engine is decoupled
   from HTTP, authentication, and storage via interfaces (ports); delivery and
   persistence are adapters. This makes "engine" meaningful regardless of
   delivery mechanism.
2. **Task is a first-class aggregate.** A task can exist, be shared, and sync
   independently — it is not mandatorily nested under a Project or Document.
3. **Third-party application authorization is distinct from end-user
   authentication.** The auth work (see `server/features/auth.feature`) must
   not assume the caller is always the interactive human; it must leave room
   for OAuth2 (authorization-code) and scoped API tokens / app passwords, with
   user consent and scopes. (Roadmap; not built yet.)
4. **Outbound webhook / event-subscription surface**, built on the event/outbox
   backbone (ADR-0003), so external systems can receive task/completion events.
   (Roadmap; substrate already exists.)
5. **The client supports multiple accounts / servers**, enabling scenario A's
   mirroring.

## Consequences

- Both target scenarios are supported by decisions already made: REST/OpenAPI
  (ADR-0001), the event/outbox backbone (ADR-0003), and federation-readiness
  (ADR-0004).
- **Licensing is favorable for integrators:** calling an ergonomos server over
  the API is not a derivative work, so API integrators (scenario B) are not
  subject to the AGPL. The AGPL obligation falls only on whoever *runs* a
  (modified) ergonomos server. Code-level embedding, if ever wanted, is handled
  by commercial dual-licensing enabled by the CLA (ADR-0005).
- `internal/` placement is correct and intentional; exposing public library
  packages is explicitly out of scope unless a future ADR revisits it.
- The hexagonal core is now a **standing constraint** for all domain work, not
  a one-off — recorded in `CLAUDE.md`.

## Alternatives considered

- **Publish the engine as an importable Go library** (move out of `internal/`,
  commit to a stable public API) — rejected for now: larger maintenance surface
  and unnecessary for the stated scenarios. Revisitable via a new ADR.
- **Couple the domain directly to HTTP/storage** — rejected; would make the
  engine un-embeddable in any sense and harder to test.
