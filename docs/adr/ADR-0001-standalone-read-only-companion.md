# ADR-0001: Standalone, read-only T3 Code companion

**Status:** Accepted
**Date:** 2026-09-13
**Applies to:** Product scope, UI, integrations

## Context

A desktop bar can expose ongoing T3 Code work and outstanding questions without
requiring the user to keep the application visible. DankAIUsage already answers
a different question: remaining provider allowance and local consumption history.
Combining these responsibilities would couple quota collection to live session
connections and make the quota dropdown more complex.

## Decision

Create DankT3Code as an independently installable DMS plugin. Its first release
shows working sessions, sessions needing attention, recent outcomes, environment
and project context, and a way to open the relevant T3 UI. Optional notifications
are off by default. Target one explicitly configured local T3 environment first.

The integration observes state. Approval responses, prompts, stop/restart actions,
account changes, and other agent commands remain in T3 Code. Pairing a client is
connection setup, not permission to perform agent actions.

DankAIUsage retains quotas, credits, resets, and token history. No shared service,
plugin dependency, cross-plugin credentials, quota bridge, or token import is
required. Detailed analytics can be opened in T3's existing UI. Support the T3
providers represented by its normalized activity data rather than hardcoding
DankAIUsage's two-provider limit.

Unknown or stale state is distinct from idle. Attention flags may coexist with
ongoing work, so counts and labels must explain overlapping categories. An async
question does not necessarily mean the agent has stopped.

## Alternatives considered

- Extend DankAIUsage: convenient placement, but combines different data lifecycles,
  user needs, and maintenance obligations.
- Build a general agent-monitoring framework: potentially broader support, but
  adds adapter abstractions before a second integration exists.
- Build another T3 control UI: gives more desktop actions, but duplicates the
  conversation UI and expands permissions and state-management requirements.

## Consequences

Users can install either plugin or both. T3 connection failures do not affect
quota monitoring. Multiple/remote environments and any future quota bridge need
separate scope and identity decisions after the first release is proven.
