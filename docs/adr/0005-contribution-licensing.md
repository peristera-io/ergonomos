# ADR-0005: Contribution licensing — CLA with relicensing grant

- **Status:** Accepted
- **Date:** 2026-06-19
- **Amends:** ADR-0001 (its DCO decision)

## Context

ADR-0001 set up a Developer Certificate of Origin (DCO) for contributions. A
DCO only certifies provenance and keeps contributions under the project's
current license ("inbound = outbound"); each contributor retains copyright and
grants nothing more. That means the project could **not** relicense or
dual-license in the future without the explicit consent of every past
contributor (or rewriting their code).

We want to preserve the *option* to relicense later (e.g. dual-licensing or
changing copyleft terms) without precluding it now. The project is young, so
this is the cheapest moment to put the right mechanism in place — retrofitting
a licensing agreement onto existing contributors is the painful part.

A full copyright **assignment** was considered but rejected: it is a heavier
ask that deters contributors, it is viewed with suspicion in copyleft
communities, and true assignment is legally constrained in Luxembourg/EU
author's-rights ("droit d'auteur") jurisdictions.

## Decision

Replace the DCO with a **Contributor License Agreement (CLA) that includes a
relicensing grant** (`CLA.md`). Contributors **keep their copyright** but grant
the project a broad, irrevocable license — explicitly including the right to
sublicense and to license their contributions under **any** terms now or in the
future. This gives relicensing flexibility without anyone signing away
ownership, and it works within EU author's-rights law (using a broad license
rather than assignment, plus a moral-rights non-assertion clause).

Signing is enforced via the **CLA Assistant** GitHub Action
(`.github/workflows/cla.yml`): contributors sign once by commenting on their
PR, and signatures are stored in `signatures/cla.json`. The DCO file and the
`git commit -s` sign-off requirement are removed.

## Consequences

- The project can change its license or dual-license in the future without
  hunting down past contributors.
- The App Store exception (`LICENSE-EXCEPTION.md`) remains grantable, since the
  CLA's broad outbound grant covers it.
- **Added friction:** contributors must sign the CLA (one-time, via a bot
  comment) instead of just signing off commits. Some contributors decline CLAs
  on principle; this is an accepted trade-off.
- This is a *template* CLA; it should be reviewed by legal counsel before being
  relied upon, and the governing-jurisdiction and "Us" party must be finalized.
- Corporate contributors need a separate Corporate CLA (noted in `CLA.md`).

## Alternatives considered

- **Keep the DCO** — lowest friction and best community goodwill, but no
  relicensing flexibility. Rejected because preserving optionality is a stated
  goal.
- **Full copyright assignment / FSFE FLA** — strongest consolidation, but
  heavier ask and EU-assignment constraints. The FLA remains a fallback if true
  consolidation is ever required.
