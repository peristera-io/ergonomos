# Architecture Decision Records

Each ADR captures one decision: the **context** that forced it, the
**decision** taken, and the **consequences** (good and bad). ADRs are
append-only history — when a decision changes, write a new ADR that supersedes
the old one rather than editing the original.

## Index

| #    | Title                                   | Status   |
|------|-----------------------------------------|----------|
| 0001 | Foundational decisions                  | Accepted |
| 0002 | Authorization: embed OpenFGA behind an interface | Accepted |
| 0003 | Domain-event / transactional-outbox backbone | Accepted |
| 0004 | Federation-readiness constraints        | Accepted |
| 0005 | Contribution licensing — CLA with relicensing grant | Accepted (amends 0001) |
| 0006 | ergonomos as a network-embeddable engine (API + federation) | Accepted |
| 0007 | HTTP router and middleware — adopt go-chi/chi | Accepted |
| 0008 | Persistence — pgx, goose, and a context-propagated transaction | Accepted |

## Template

```markdown
# ADR-NNNN: Title

- **Status:** Proposed | Accepted | Superseded by ADR-XXXX
- **Date:** YYYY-MM-DD

## Context
What is the situation and what forces are at play?

## Decision
What we decided to do.

## Consequences
What becomes easier, harder, or constrained as a result.

## Alternatives considered
What else we weighed and why we did not pick it.
```
