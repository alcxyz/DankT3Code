# ADR-0003: Supported connection and activity observation

**Status:** Proposed
**Date:** 2026-09-13
**Applies to:** T3 transport, authentication, state collection

## Context

T3's source contains authenticated RPC connections and thread summaries with
session state, pending approvals, pending user input, and actionable plans.
These implementation contracts are useful evidence, not proof of a supported,
stable third-party API for the version a user has installed.

## Proposed direction

Use an explicitly configured T3 environment and the supported client connection
flow. Request only the scopes needed for observation. Prefer an initial snapshot
and event subscription to frequent polling when the tested API supports it.
Keep transport connection status separate from snapshot freshness.

Keep protocol handling outside presentation components if its complexity warrants
a helper. Choose QML-only versus a helper, and any helper language, after a small
compatibility spike measures authentication, decoding, reconnect, and packaging
costs. No shared daemon or general adapter framework is committed here.

Consume only the fields needed for session identity, display, state, and navigation.
Do not scrape T3's private database or credential stores, read provider transcripts,
derive work state from process listings, or persist conversation payloads. Use
environment-scoped session identity and treat reconnect snapshots as observations,
not new completion events. Pairing material must stay out of URLs shown to users,
logs, diagnostics, and fixtures.

## Alternatives to evaluate

- Authenticated RPC snapshot plus subscription: timely data, but compatibility and
  connection lifecycle need verification.
- A supported lightweight status endpoint/export: potentially simpler if upstream
  provides one with enough activity data.
- Bounded polling of supported read methods: simpler lifecycle, with latency and
  server-load tradeoffs to measure.
- Local database or process scraping: brittle implementation coupling and weak
  activity semantics; not a fallback to bypass an unavailable supported API.

## Acceptance gate

Before accepting this ADR, record the tested T3 release/commit and DMS version,
actual RPC methods and required scopes, pairing/storage strategy, snapshot fields,
reconnect/replay behavior, and browser/desktop navigation results. Compare
implementation/dependency options and document the chosen one with evidence.

If the available API cannot support the read-only MVP, record the gap and seek
an upstream contract or revise the plan; do not silently expand permissions or
claim live compatibility. See [the investigation checklist](../compatibility.md).
