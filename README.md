# ergonomos

A hybrid **task, project, and documentation manager** — multi-user, team-aware,
with fine-grained sharing (read-only / editable / public) and a design that
keeps the door open to **federation** between independent ergonomos servers.

- **Backend:** Go
- **Clients:** Flutter (web, iOS, Android), pinned with [fvm](https://fvm.app)
- **API contract:** REST, spec-first via OpenAPI (`api/openapi.yaml`)
- **Database:** PostgreSQL
- **Authorization:** relationship-based (ReBAC) via **OpenFGA, embedded
  in-process** behind an `Authorizer` interface
- **Realtime:** websocket change-sync + presence (v1); CRDT co-editing deferred
- **Tested with:** Go tests + **BDD acceptance specs** (Gherkin + godog)
- **License:** AGPL-3.0-or-later, with an App Store distribution exception

The decisions behind each of these live in [`docs/adr/`](docs/adr/).

## Repository layout

```
api/              OpenAPI contract — the source of truth for the REST API
server/           Go backend (cmd/, internal/, features/*.feature)
app/              Flutter client (fvm-pinned)
docs/
  adr/            Architecture Decision Records — why things are the way they are
  worklog.md      Running log of what has been done (project memory)
  guidelines/     Conventions discovered while building
CLAUDE.md         Operating manual for humans and LLM agents working here
```

## Getting started

Prerequisites: Go 1.26+, Docker + Compose. (`task` is optional but convenient:
`brew install go-task`.)

```sh
# start local dependencies (Postgres)
docker compose up -d            # or: task up

# build, test and run the server
cd server
go test ./...                   # or: task server:test
go run ./cmd/ergonomos          # serves http://localhost:8080/healthz
```

### Client (when you start on it)

```sh
brew install fvm                # or: dart pub global activate fvm
cd app
fvm install                     # installs the version pinned in app/.fvmrc
fvm flutter pub get
```

## Status

Early scaffolding. The first feature slice (user registration & authentication)
is specified in `server/features/auth.feature` and is the next thing to be
implemented red→green.

> **Note:** the Go module path (`github.com/ergonomos/ergonomos/server`) is a
> placeholder until the GitHub org/repo is finalized — update it with
> `go mod edit -module=...` and adjust imports.

## License

AGPL-3.0-or-later (`LICENSE`) with the App Store distribution exception
(`LICENSE-EXCEPTION.md`). Contributions are accepted under a Contributor
License Agreement that lets the project relicense in future — see
`CONTRIBUTING.md` and `CLA.md`.
