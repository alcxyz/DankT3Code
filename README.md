# DankT3Code

A planned [DankMaterialShell](https://github.com/AvengeMedia/DankMaterialShell)
companion for [T3 Code](https://github.com/pingdotgg/t3code): see what your agents
are doing, notice when they need attention, and return to the relevant thread.

**Status: early development.** A Go collector can read a minimized activity
snapshot from T3 Code `0.0.40`. The QML component still displays a development
placeholder; pairing, live bar updates, navigation, and notifications are not
implemented yet. Version `0.0.0` identifies development; `v0.1.0` is the first
working widget release target.
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
The collector uses Go's standard library and authenticated HTTP snapshots.
T3 `0.0.40` is the initial tested version; other releases/nightlies currently
report incompatible. See [compatibility findings](docs/compatibility.md).

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

The copied CI workflow validates the manifest and component paths and builds/tests
the Go collector. Unit tests use synthetic snapshots and local HTTP test servers.
The optional [isolated release smoke test](docs/compatibility.md#reproduce-the-server-checks)
exercises a published T3 server without accessing a running installation.
Neither manifest validation nor collector tests establish live DMS compatibility.

### Collector development

With Go 1.22 or newer:

```sh
go test ./...
go build -o /tmp/dankt3code ./cmd/dankt3code
/tmp/dankt3code snapshot --endpoint http://127.0.0.1:3773 --token-file /path/to/bearer-token
```

Use your configured loopback server port. The token file must already contain a
T3 bearer restricted to `orchestration:read`, owned by the current user and
inaccessible to other users. Automated pairing is still pending; do not copy
credentials from T3's private stores. No token is accepted as a command-line value.
T3's read scope includes broader file access, although this collector calls only
the descriptor and shell endpoints. See the permission notes in the compatibility
document before arranging a live credential.

The command emits schema-versioned JSON with either a connected observation or a
defined error and nonzero exit status. Display labels belong to the private UI
payload; do not paste live output into an issue. Raw errors, scripts, workspace
paths, and conversation data are dropped. It writes no cache and starts no timer.

For a development-only preview, copy `plugin.json` and `DankT3CodeWidget.qml`
into `~/.config/DankMaterialShell/plugins/DankT3Code/`, then enable **T3 Code**
and add it to the bar. Expect only the placeholder. Packaging, screenshots, and
plugin registry submission are release-readiness tasks.

## Planning

- [First-release plan](docs/roadmap.md)
- [Architecture decisions](docs/adr/README.md)
- [Compatibility findings](docs/compatibility.md)
- [GitHub issues](https://github.com/alcxyz/DankT3Code/issues)
- [GitHub milestones](https://github.com/alcxyz/DankT3Code/milestones)

Licensed under [MIT](LICENSE).
