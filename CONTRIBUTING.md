# Contributing

DankT3Code is in early development. Start with the [first-release plan](docs/roadmap.md)
and [architecture decisions](docs/adr/README.md). The initial HTTP collector is
tested against T3 `0.0.40`; check [the evidence](docs/compatibility.md) before
expanding compatibility or adding live UI integration.

Use GitHub issues for bugs and proposals and link the relevant issue in a pull
request. Target `dev` for ordinary contributions. `main` is reserved for reviewed
release promotion; the initial promotion remains a draft until `v0.1.0` is ready.

Keep changes focused and describe their user-visible behavior and validation.
Use synthetic fixtures and never attach credentials, private thread titles,
prompts, transcripts, raw protocol traffic, or unreviewed diagnostics to issues.

The standard CI validates `plugin.json` and referenced component files, builds the
Go helper, and runs `go test ./...`. Supply meaningful tests for connection
failures, stale state, and replay behavior. Describe manual DMS/T3 checks
separately from static validation. Do not claim live compatibility from a manifest
or syntax check.

The repository uses the [MIT license](LICENSE). This is an independent community
project; report plugin-specific issues here and verified upstream issues to the
relevant upstream project.
