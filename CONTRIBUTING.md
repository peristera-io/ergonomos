# Contributing to ergonomos

Thanks for your interest! A few things to know before you start.

## How we work

ergonomos is developed with a specify → red → green → refactor (BDD) loop and a
written project memory. Please read **`CLAUDE.md`** — it is the operating
manual for the repo and applies to humans and LLM agents alike. In short:

- Specify behavior as a Gherkin `.feature` (domain/API level), make it fail,
  then implement the smallest change to pass.
- After meaningful work, **append to `docs/worklog.md`**, add an **ADR** for
  non-obvious decisions, and add to **`docs/guidelines/`** when a reusable
  convention emerges.
- Keep changes small and CI green.

## Developer Certificate of Origin (DCO)

Contributions are accepted under the **Developer Certificate of Origin 1.1**
(see the `DCO` file). This certifies you have the right to submit your work
under the project's license.

Sign off every commit with the `-s` flag, which appends a `Signed-off-by`
line using your real name and email:

```sh
git commit -s -m "Your message"
```

By signing off you agree to the DCO and that your contribution is licensed
under AGPL-3.0-or-later **with** the App Store distribution exception
(`LICENSE-EXCEPTION.md`), so the project can continue to grant that exception.

## License

By contributing you agree that your contributions are licensed as above.
