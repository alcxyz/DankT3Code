# DankT3Code

A planned [DankMaterialShell](https://github.com/AvengeMedia/DankMaterialShell)
companion for [T3 Code](https://github.com/pingdotgg/t3code): see what your agents
are doing, notice when they need attention, and return to the relevant thread.

**Status: bootstrap only.** The QML component displays a development placeholder.
It does not connect to T3 Code, report agent activity, or send notifications yet.
Version `0.0.0` identifies the scaffold; `v0.1.0` is the first working release target.
This is an independent community plugin, not an official T3 Code integration.

## First release

- A compact bar summary of working sessions and sessions needing attention.
- A dropdown with attention items first, followed by running and recent sessions.
- Project, provider, and environment context, with honest connection/freshness states.
- Open T3 Code and, where verified, the selected thread.
- Optional completion, failure, approval, and question notifications.

The first implementation will target one explicitly configured local environment.
Multiple environments, remote pairing, and relay support are later work unless the
compatibility investigation shows they are necessary for the local connection.
No helper language or supported T3 version has been selected yet.

## Relationship to DankAIUsage

[DankAIUsage](https://github.com/alcxyz/DankAIUsage) owns quotas, credits, reset
countdowns, and local token history. DankT3Code owns T3 session activity and
navigation. Either plugin can be installed independently; they share no service,
credentials, or state. There is no quota bridge in the first release.

Agent approvals, prompt submission, stop/restart controls, account management,
and detailed usage analytics remain in T3 Code.

## Development

GitHub is canonical for source, issues, pull requests, CI, and releases. Work on
`dev`; promote to `main` through a GitHub pull request. During bootstrap, `dev`
contains the scaffold and planning documents. The promotion PR stays in draft
until the first release is ready. The standard workflow releases versions from
`plugin.json` on pushes to `main`, so the scaffold must not be promoted as a
working release.

The copied CI workflow validates the manifest and component paths. It builds/tests
a Go helper if `go.mod` is introduced and runs `test.sh` if present. The bootstrap
has no helper and no runtime test suite; manifest validation does not establish
DMS or T3 compatibility.

For a development-only preview, copy `plugin.json` and `DankT3CodeWidget.qml`
into `~/.config/DankMaterialShell/plugins/DankT3Code/`, then enable **T3 Code**
and add it to the bar. Expect only the placeholder. Packaging, screenshots, and
plugin registry submission are release-readiness tasks.

## Planning

- [First-release plan](docs/roadmap.md)
- [Architecture decisions](docs/adr/README.md)
- [Compatibility investigation](docs/compatibility.md)
- [GitHub issues](https://github.com/alcxyz/DankT3Code/issues)
- [GitHub milestones](https://github.com/alcxyz/DankT3Code/milestones)

Licensed under [MIT](LICENSE).
