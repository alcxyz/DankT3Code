# ADR-0003: Supported connection and activity observation

**Status:** Accepted for the HTTP collector; UI integration remains unreleased
**Date:** 2026-09-13
**Applies to:** T3 transport, authentication, state collection

## Context

T3's source contains authenticated RPC connections and thread summaries with
session state, pending approvals, pending user input, and actionable plans.
These implementation contracts are useful evidence, not proof of a supported,
stable third-party API for the version a user has installed.

## Decision

Use one explicitly configured loopback T3 environment and its authenticated HTTP
shell snapshot endpoint. A small Go standard-library helper fetches the environment
descriptor and `GET /api/orchestration/shell`, emitting a minimized JSON observation
for the widget. The first collector supports the tested T3 `0.0.40` release only;
unknown versions report incompatible until their contracts are checked.

Keep credential use, HTTP bounds, response validation, and error handling in Go.
The helper is one-shot; the future DMS integration owns a non-overlapping polling
timer with a five-second starting interval and backoff after errors. No daemon,
database, npm dependency, or WebSocket connection is required at runtime. Keep
transport availability separate from observation freshness. HTTP polling can miss
transitions between samples, so notifications describe observed changes, not a
complete event history.

Pairing must exchange an ordinary one-time T3 pairing token for a bearer with
exactly `orchestration:read`; T3's CLI does not offer a narrower pairing flag.
The collector initially consumes an explicitly supplied owner-only bearer file.
Pairing UI/credential creation and the polling timer remain tracked in issue #2;
no existing T3 credential store is read automatically.

This is the smallest available scope, but T3 also authorizes arbitrary server-side
file reads under it. It is broader than activity-only access. Explain that scope
when pairing; restrict the helper to the two observation endpoints. Bearers last
30 days by default and have no refresh-token endpoint. Expiry or revocation
requires fresh pairing. Do not request operate or administrative scopes.

Consume only the fields needed for session identity, display, state, and navigation.
Do not scrape T3's private database or credential stores, read provider transcripts,
derive work state from process listings, or persist conversation payloads. Use
environment-scoped session identity and treat reconnect snapshots as observations,
not new completion events. Pairing material must stay out of URLs shown to users,
logs, diagnostics, and fixtures.

## Alternatives considered

- Authenticated RPC snapshot plus subscription: verified in the isolated spike,
  but Effect RPC framing, acknowledgements, replay, and socket lifetimes add work
  without adding necessary fields for the first release.
- QML-only HTTP: avoids a helper but couples credential handling and protocol
  validation to presentation and makes failure-path testing harder.
- Node helper using T3 client libraries: adds a runtime dependency without a stable
  public package contract; T3's contracts package is private.
- Bounded HTTP polling: selected because the released endpoint has the required
  activity flags and ordinary Go HTTP/JSON support is sufficient.
- Local database or process scraping: brittle implementation coupling and weak
  activity semantics; not a fallback to bypass an unavailable supported API.

## Evidence and remaining release gates

The isolated `t3@0.0.40` smoke test verifies one-time pairing, scope restriction,
HTTP/WS shell reads, restart continuity, and revocation without touching a live
installation. [Compatibility findings](../compatibility.md) distinguish these
checks from source-only navigation evidence and untested DMS integration.

The first implementation is a collector, not a working widget release. Pairing,
scheduled polling/backoff, activity semantics, navigation, notifications, and
manual DMS validation remain milestone work. Browser thread routing is supported
by the reviewed source; an external desktop thread-opening contract was not found.
Use browser navigation as the initial direction and verify it under issue #5.

The upstream interface is implemented and tested here, but no stable third-party
API guarantee was found. New T3 releases need explicit compatibility checks;
incompatibility must not silently widen permissions or produce a fresh idle state.
