# ADR-0001: Foundational decisions

- **Status:** Accepted
- **Date:** 2026-06-19
- **Amended:** the DCO-for-contributions decision below is superseded by
  ADR-0005 (CLA with relicensing grant).

## Context

ergonomos is a greenfield, hybrid task/project/documentation manager. It is
multi-user with teams and fine-grained sharing, built in the open on GitHub,
and intended to be developed substantially with the help of LLM agents. We
needed to fix the stack and the working method before writing code.

## Decision

- **Stack:** Go backend, Flutter clients (web, iOS, Android), PostgreSQL.
- **Monorepo** with `server/` (Go) and `app/` (Flutter), tied together by a
  root `Taskfile.yml`. Each toolchain keeps its native build.
- **API contract: REST, spec-first via OpenAPI** (`api/openapi.yaml`). The
  spec is the source of truth and doubles as documentation; the typed Dart
  client is generated from it. Chosen over gRPC/Connect for tooling maturity
  in Dart and because LLMs are highly fluent in REST/OpenAPI.
- **Flutter version pinned with fvm** (`app/.fvmrc`) for reproducible builds.
- **Working method: specification by example (BDD).** Gherkin `.feature`
  files run by godog document behavior at the domain/API level; ordinary Go
  tests cover mechanics. See `CLAUDE.md`.
- **Project memory:** ADRs (decisions), `docs/worklog.md` (running log), and
  `docs/guidelines/` (discovered conventions).
- **License:** AGPL-3.0-or-later with an App Store distribution exception
  (`LICENSE-EXCEPTION.md`), and a DCO for contributions (`CONTRIBUTING.md`).
- **v1 scope:** online + realtime collaboration (websocket change-sync and
  presence). Offline-first and character-level CRDT co-editing are out of v1.

## Consequences

- A single contract artifact keeps server and client in sync and is easy for
  agents to read and modify.
- Choosing realtime in v1 commits us to a websocket layer and an event stream
  (see ADR-0003) earlier than a purely request/response app would need.
- AGPL + the App Store conflict is resolved by the exception rather than by
  splitting licenses; this requires the DCO so the exception stays grantable.

## Alternatives considered

- **gRPC/Connect** for the contract — rejected for now due to less mature
  Dart tooling and thinner LLM training coverage.
- **Split licensing** (server AGPL, client permissive) — viable, but the
  single-license-plus-exception approach was preferred for simplicity and to
  keep the whole project copylefted.
- **Online-only / offline-first** for sync — rejected in favor of realtime as
  the v1 target, while explicitly deferring CRDT co-editing.
