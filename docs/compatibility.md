# Compatibility investigation

No T3 Code release or DMS version has been verified with this plugin yet. The
bootstrap component is a placeholder; source review is not an integration test.

Initial upstream source review on 2026-09-13 used T3 commit
`c29976458a7dbbb61c83b3d06ea894869b718607`:

- [Thread and shell contracts](https://github.com/pingdotgg/t3code/blob/c29976458a7dbbb61c83b3d06ea894869b718607/packages/contracts/src/orchestration.ts)
- [RPC contracts](https://github.com/pingdotgg/t3code/blob/c29976458a7dbbb61c83b3d06ea894869b718607/packages/contracts/src/rpc.ts)
- [Environment authentication](https://github.com/pingdotgg/t3code/blob/c29976458a7dbbb61c83b3d06ea894869b718607/docs/internals/environment-auth.md)
- [Connection runtime](https://github.com/pingdotgg/t3code/blob/c29976458a7dbbb61c83b3d06ea894869b718607/docs/internals/connection-runtime.md)

## Evidence required before implementation

| Area | Questions to resolve |
|---|---|
| Version | Which released T3 version and DMS version are tested? Is the interface intended for external clients? |
| Setup | Can a local third-party client pair through a supported flow without reading private application stores? |
| Permissions | Which exact read scopes are required? What authority is granted by pairing? |
| State | Which snapshot/subscription fields distinguish working, questions, approvals, plans, success, failure, and unknown? |
| Lifecycle | What happens on sleep/resume, disconnect, revocation, server restart, incompatible version, and replay? |
| Navigation | Which browser routes and desktop opening mechanisms work? What safe fallback exists? |
| Implementation | Can QML own the connection simply, or is a helper justified? What dependencies and Nix packaging follow? |
| Privacy | Which minimum fields are retained, for how long, and how are credentials and diagnostics kept separate? |

Record sanitized results and synthetic protocol fixtures only. A supported
connection is required; do not read live prompts, credential stores, or raw
conversation payloads to perform the investigation.
