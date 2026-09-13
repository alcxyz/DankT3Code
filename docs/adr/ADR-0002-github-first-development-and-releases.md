# ADR-0002: GitHub-first development and standard plugin releases

**Status:** Accepted
**Date:** 2026-09-13
**Applies to:** Repository governance, CI, release lifecycle

## Context

The maintained DMS plugin set uses independent public repositories with a common
CI workflow. This plugin should fit that workflow and remain usable by forks.

## Decision

GitHub is canonical for source, issues, milestones, pull requests, CI, and releases
at `alcxyz/DankT3Code`. Any future continuity mirror copies from GitHub; it never
pushes back into the canonical repository.

Develop on `dev`. Protect `main`, require pull requests, and promote `dev` through
a GitHub PR after checks pass. Squash merge by default and retain `dev`. GitHub
initializes the repository with a README/license baseline; subsequent development
does not write directly to `main`.

Copy the maintained aggregate's canonical workflow byte-for-byte to
`.github/workflows/ci.yml`. Keep it self-contained. It validates `plugin.json`,
builds/tests Go when `go.mod` exists, and runs `test.sh` when present. No helper
language is selected by adopting this workflow.

Releases use `plugin.json`'s `X.Y.Z` version and the standard `main`-push workflow.
During bootstrap, `dev` is the default branch and the `dev` to `main` promotion is
a draft. `0.0.0` is a development scaffold, not a published working version.
Complete the pre-promotion checks tracked in the `v0.1.0` milestone, bump the
manifest, then promote and switch the default branch to `main`. Verify the
published release before closing the release issue and milestone. Do not add a
custom release bypass for the scaffold.

## Alternatives considered

- Forgejo-first hosting: appropriate for other repository classes, but not this
  public plugin's selected collaboration surface.
- Shared reusable workflow: reduces copying, but couples forks to a central repo.
- Publish the scaffold immediately: exposes an incomplete plugin as a normal
  release and risks premature package/registry consumption.

## Consequences

Planning and development are public immediately; installation is explicitly
development-only. First-release promotion and aggregate/registry registration
are tracked work. A mirror or changes to other repositories are separate tasks.
