# Synthetic T3 shell fixtures

These JSON files are synthetic responses for `GET /api/orchestration/shell`,
based on the pinned T3 Code v0.0.40 contracts at commit
`09e8de9c655ae85410bf6b00446f272a01da81c7`.

- `fixtures/shell-empty.json` is an idle empty environment.
- `fixtures/shell-states.json` is one small project containing running work with
  overlapping async input and background work, a pending approval, completed and
  error terminal outcomes, and background `working` and `monitoring` states.

All identifiers, paths, titles, timestamps, model names, and error text are
invented. These fixtures contain no credentials, URLs, prompts, transcript data,
or private session metadata.

The shell response is environment-scoped by the connection that fetched it; the
JSON itself has no `environmentId`. Approval and user-input fields are summary
booleans. Request IDs and question details require a thread detail snapshot and
activity reduction. A shell snapshot also has no process exit code; consumers
should use `latestTurn.state`, `completedAt`, and `session.status`. The native
`session.lastError` may contain private text and must not be forwarded to the UI
or diagnostics; the collector drops it.

`backgroundLiveness` is independent of `latestTurn`: a row can have no active
turn while still reporting native background work. Reconnect code must treat a
snapshot as the current observation and suppress duplicate completion or
notification effects. The initial HTTP collector fetches full snapshots. The
upstream WebSocket protocol also supports sequence replay, but it is not a runtime
dependency of this collector.

The `providerName` and `modelSelection.instanceId` values use the lowercase
`codex` provider identifier. `synthetic-model` is only a valid non-empty model
placeholder for contract decoding; it is not a real model recommendation.
