# Worklog

The running memory of ergonomos: what was done, when, and why. Newest entries
at the top. One entry per meaningful unit of work. Keep it honest — record
what was skipped or left unfinished too.

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
