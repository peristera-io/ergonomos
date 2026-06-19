# ADR-0004: Federation-readiness constraints

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

We want the option, later, to federate independent ergonomos servers so that a
user on instance A can be assigned tasks by, or share projects with, a user on
instance B — the model email and the fediverse use. We are **not** building
federation now, but several data-model decisions would be expensive or
impossible to retrofit, so we fix the constraints up front.

## Decision

We will **not preclude** server-to-server federation. The likely eventual
protocol is **ActivityPub** (actors, inbox/outbox, HTTP Signatures), so we keep
the model compatible with that shape. Binding constraints from day one:

1. **Globally-unique, opaque identifiers** (ULID/UUID) for all entities; never
   expose sequential integer keys across the API. Every shareable resource
   also has a **canonical URI** and a **home-instance** marker.
2. **An actor abstraction that can be local or remote**, plus a first-class
   **instance** entity. Remote actors are representable principals that do not
   authenticate locally. Reserve a place for **per-actor keypairs** (for HTTP
   Signatures) even though they are unused in v1.
3. **The object's home instance is authoritative for that object's grants.**
   Authz tuple subjects are strings, so a federated subject
   (`user:alice@other.instance`) is representable; cross-instance permission
   data is delegated/cached, never assumed local.
4. **Changes are events** (ADR-0003). A share or assignment is an event that
   could later be delivered to another instance's inbox, not only a row
   mutation.

Actual S2S protocol selection and implementation is deferred to its own future
ADR.

## Consequences

- The API and schema carry opaque IDs, URIs, actor/instance modeling, and an
  event backbone from the start — small costs now, large savings later.
- No federation code or protocol commitment yet; we only avoid contradicting a
  future ActivityPub-like design.
- Some designs are now off the table (e.g. integer auto-increment IDs in API
  responses, or assuming every subject is a local user).

## Alternatives considered

- **Ignore federation until needed** — rejected; the identifier and
  ownership-model choices are not cheaply reversible once data exists.
- **Commit to ActivityPub now** — rejected as premature; we keep compatibility
  without paying implementation cost before the core product exists.
