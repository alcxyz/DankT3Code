# First-release plan

The GitHub milestone [v0.1.0 — Read-only T3 companion](https://github.com/alcxyz/DankT3Code/milestone/1)
owns the implementation backlog. GitHub issues are the planning source of truth.

## Outcome

A user can connect one local T3 environment, see truthful working/attention
states in DMS, open the relevant T3 UI, and opt into useful notifications. Both
horizontal and vertical bars work. Disconnects and replay do not invent idle
state or duplicate alerts.

## Tracked work

| Issue | Work |
|---|---|
| [#1](https://github.com/alcxyz/DankT3Code/issues/1) | research: verify T3 connection, activity, and navigation contracts |
| [#2](https://github.com/alcxyz/DankT3Code/issues/2) | feat: implement the read-only T3 connection lifecycle |
| [#3](https://github.com/alcxyz/DankT3Code/issues/3) | feat: define activity, attention, and freshness semantics |
| [#4](https://github.com/alcxyz/DankT3Code/issues/4) | feat: build the DMS activity bar and attention dropdown |
| [#5](https://github.com/alcxyz/DankT3Code/issues/5) | feat: open T3 Code and the selected thread |
| [#6](https://github.com/alcxyz/DankT3Code/issues/6) | feat: add opt-in activity and attention notifications |
| [#7](https://github.com/alcxyz/DankT3Code/issues/7) | chore: validate and publish the first working v0.1.0 release |
| [#8](https://github.com/alcxyz/DankT3Code/issues/8) | chore: distribute the released plugin through the aggregate and DMS registry (after release; outside milestone) |

## Delivery order

1. Verify the supported T3 connection and settle ADR-0003. The HTTP collector
   decision now has isolated `0.0.40` evidence; UI navigation remains unverified.
2. Implement connection lifecycle and activity projection with synthetic fixtures.
3. Build the DMS summary/dropdown and verified navigation.
4. Add opt-in notifications and validate reconnect behavior.
5. Complete documentation, packaging, compatibility checks, and release promotion.

Each implementation issue has acceptance criteria. Successful manifest CI on the
scaffold is not completion of any runtime acceptance criterion.

The current slice supplies a one-shot collector, synthetic fixtures, and the
isolated server smoke harness. Pairing and scheduled polling remain in #2;
the widget is still a placeholder. GitHub issue checklists distinguish completed
evidence from remaining milestone work.

## Release gate

- Supported versions and setup are documented and tested end to end.
- Fresh/stale/disconnected/idle states and overlapping activity/attention are tested.
- Notifications are off by default and replay-safe.
- Accessibility, horizontal/vertical layout, screenshots, and clean install are checked.
- Packaging is implemented for the chosen architecture, with any aggregate changes
  made through their own repository workflow. Registry submission follows a working release.
- `plugin.json` is bumped from `0.0.0` to `0.1.0`; CI passes; the promotion PR is
  reviewed and merged before the standard workflow publishes the release.

## Later candidates

Multiple/remote environments, relay support, a quota bridge, and agent control are
outside this milestone. Evaluate them separately after the first release; their
presence here is not a commitment to implement them.
