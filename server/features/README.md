# Acceptance specs (BDD)

These `.feature` files are the behavioral specification of ergonomos, written
in Gherkin and meant to be readable by anyone. They are executed as acceptance
tests with [godog](https://github.com/cucumber/godog).

The runner (`features_test.go`) and step definitions (`*_steps_test.go`) live
alongside the specs. Each scenario spins up an ergonomos instance with
in-memory adapters and drives it through its real HTTP surface (`internal/rest`),
so the specs exercise the full stack end to end. Run them with `go test ./...`
(or `task server:test`).

See `CLAUDE.md` for the specify → red → green → refactor loop.
