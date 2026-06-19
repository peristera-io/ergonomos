# ADR-0002: Authorization — embed OpenFGA behind an interface

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

ergonomos needs fine-grained sharing: users grant read-only or editable access
to parts of their data to other users, teams, or the public. This is
relationship-based access control (ReBAC), the Google Zanzibar model. We chose
OpenFGA as the engine. The open question was deployment: a separate service or
embedded in the Go binary.

## Decision

Embed OpenFGA **in-process** as a library (`github.com/openfga/openfga`),
backed by the same PostgreSQL instance (its own schema). All application code
depends only on our own **`authz.Authorizer` interface** (`Check`, `Write`,
…); the embedded OpenFGA is one implementation behind it.

## Consequences

- **Single binary + Postgres** to deploy. This is the simplest possible story
  for third parties self-hosting ergonomos — which matters given the AGPL and
  the federation ambition (ADR-0004).
- **No network hop** on `Check`, which is on the hot path of nearly every
  request.
- We **couple to OpenFGA's Go API**, which is less stable than its gRPC API;
  version upgrades will cost some maintenance. The `Authorizer` interface
  contains the blast radius and lets us switch to a remote OpenFGA or to
  SpiceDB later without touching call sites.
- We lose the process/security boundary of a separate service and the ability
  to scale authz independently; acceptable at current scale.
- License is compatible: OpenFGA is Apache-2.0, which can be combined into an
  AGPL-3.0 work.
- Keeping Postgres and the authz store consistent (two stores, no shared
  transaction) is handled by the event/outbox backbone — see ADR-0003.

## Alternatives considered

- **OpenFGA as a separate service** — cleaner boundary and independent
  scaling, at the cost of more operational surface for every self-hoster.
- **SpiceDB** — more powerful at scale (Watch API, caveats) but heavier;
  reachable later via the `Authorizer` interface if needed.
- **Hand-rolled permissions tables** — rejected; the sharing model is exactly
  what ReBAC engines exist to handle.
