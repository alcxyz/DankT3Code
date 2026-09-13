# Compatibility findings

Reviewed and tested on 2026-09-13. This is a collector compatibility record, not
a claim that the DMS widget is ready for use.

| Component | Evidence |
|---|---|
| T3 Code `v0.0.40` | Published npm `t3@0.0.40`, tested as an isolated server; source tag `09e8de9c655ae85410bf6b00446f272a01da81c7`. |
| Node `22.23.2` | Runs the optional smoke harness and released T3 server; not a dependency of the Go collector. |
| DMS `1.5.3+date=2026-07-27_069ddab` | Installed CLI version recorded. Live widget integration and navigation remain untested. |
| Newer T3 releases/nightlies | Not supported by the initial collector; require an explicit contract check. |

## Connection and permissions

1. Discover identity and server version through `GET /.well-known/t3/environment`.
2. Exchange a user-supplied one-time pairing token using form-encoded
   `POST /oauth/token`, requesting exactly `scope=orchestration:read`.
3. Fetch `GET /api/orchestration/shell` with the returned bearer in the
   `Authorization` header. No agent commands or full conversation requests are needed.

The ordinary `t3 pair` and `t3 auth pairing create` commands do not accept a scope
flag. Narrow the token at exchange, not by inventing CLI flags. Pairing tokens
are one-use and expire after five minutes by default. Failed exchanges can consume
the token, so retry setup with a newly created one. Bearers default to 30 days;
there is no refresh-token endpoint. Expiry/revocation requires fresh pairing.

`orchestration:read` is the narrowest available scope, but it also grants broader
file-reading authority on the T3 server. A paired credential is therefore more
powerful than the two endpoints this helper uses. Store it in an owner-only file
and explain the grant during pairing. The initial collector reads an explicitly
supplied bearer file; it does not implement pairing or inspect existing stores.

Sources: [auth contracts](https://github.com/pingdotgg/t3code/blob/09e8de9c655ae85410bf6b00446f272a01da81c7/packages/contracts/src/auth.ts),
[auth implementation](https://github.com/pingdotgg/t3code/blob/09e8de9c655ae85410bf6b00446f272a01da81c7/apps/server/src/auth/EnvironmentAuth.ts),
[scope boundary](https://github.com/pingdotgg/t3code/blob/09e8de9c655ae85410bf6b00446f272a01da81c7/docs/internals/environment-auth.md).

## Activity and freshness

The HTTP shell contains `snapshotSequence`, projects, threads, and `updatedAt`.
Each thread includes session/latest-turn status, three attention flags, and
optional background liveness. Environment identity comes from the descriptor;
the shell itself has no environment ID. The collector attaches that identity.

Attention booleans describe affected threads, not the number of questions. A
thread may be working while waiting for an async answer. Background `working`
and `monitoring` are distinct from an active turn. Native error strings,
workspace paths, scripts, and conversation details are not needed by the widget
and must be dropped rather than copied into its output.

Polling collects full observations and can miss short-lived transitions. A
snapshot is not a new event stream on reconnect. A server's `updatedAt` can remain
old in a truly idle environment; client observation time establishes collection
freshness. A failed fetch must never become a fresh zero count. The activity
model and notification deduplication are tracked in #3 and #6.

Sources: [shell contract](https://github.com/pingdotgg/t3code/blob/09e8de9c655ae85410bf6b00446f272a01da81c7/packages/contracts/src/orchestration.ts),
[HTTP endpoints](https://github.com/pingdotgg/t3code/blob/09e8de9c655ae85410bf6b00446f272a01da81c7/packages/contracts/src/environmentHttp.ts).

## Navigation evidence

The reviewed web router uses `/<environmentId>/<threadId>`; mobile uses
`/threads/<environmentId>/<threadId>`. Do not interchange these routes.
The desktop `t3code://app/...` scheme is an internal Electron content protocol,
not proof of an external thread-opening handler. The reviewed desktop activation
contract opens projects/new threads and does not accept an existing thread ID.

The initial navigation direction is a validated browser URL to the configured
origin. Actual browser/session behavior and an application-level fallback remain
to be verified in #5. No desktop thread deep link is claimed.

Sources: [web route construction](https://github.com/pingdotgg/t3code/blob/09e8de9c655ae85410bf6b00446f272a01da81c7/apps/web/src/threadRoutes.ts),
[desktop activation](https://github.com/pingdotgg/t3code/blob/09e8de9c655ae85410bf6b00446f272a01da81c7/packages/contracts/src/desktopAppActivation.ts).

## Reproduce the server checks

Use Node 22.16+ and an explicitly installed npm `t3@0.0.40` package. The supplied
entrypoint must be that package's `dist/bin.mjs`; the script checks package name
and version. On Nix, the npm package's `node-pty` native dependency may need a
rebuild with Python, make, and a C++ compiler before the server can start.

```sh
node scripts/compatibility-smoke.mjs /path/to/t3/dist/bin.mjs
```

Optionally pass a built `dankt3code` collector as the second argument to exercise
it against the same server. The harness creates its own temporary base, home,
settings, and credentials, disables providers and telemetry, and binds loopback.
It neither connects to nor reads the running user's T3 instance. Server output
and credentials are suppressed; only fixed check labels are printed. It stops
only its own child and removes the generated state when done.

The live checks cover startup/version, one-time pairing narrowed from the normal
grant, authenticated HTTP shell reads, unauthenticated rejection, denied agent
and access-management operations, a ticketed WebSocket snapshot with completion
marker, environment/bearer continuity across restart, and rejection after
revocation. The WebSocket check compares the alternative transport; the runtime
collector uses HTTP only. All live data is an empty synthetic environment.

Synthetic nonempty snapshots in `tests/fixtures` cover activity states without
running agents or reading conversations. Authentication rejection and malformed/
version-mismatch responses are collector unit-test cases. Bearer expiry is
source-reviewed, not a time-accelerated live-server test.
Frontend behavior, real agent transitions, and end-to-end notifications are
release gates, not claims made by this spike.
