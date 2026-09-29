# DankT3Code development

This is a DankMaterialShell plugin, not a Codex plugin. Read `docs/adr/README.md`
and relevant accepted decisions before changing its architecture.

- GitHub is canonical. Work on `dev`, push to GitHub `origin`, and promote through
  a pull request to `main`. Squash-merge feature PRs, merge promotions with a
  merge commit, and retain the long-lived `dev` branch. Never push directly to `main`.
- Keep `.github/workflows/ci.yml` identical to the canonical
  `templates/github/workflows/plugin-ci.yml` in the maintained `dms-plugins`
  aggregate. The copy must remain self-contained for forks. When working under
  that aggregate, run `scripts/check-plugin-ci.sh DankT3Code` from its root.
- `main` pushes trigger versioned releases. Do not promote the `0.0.0`
  development scaffold. Complete the milestone's pre-promotion checks and bump
  `plugin.json` before marking the bootstrap promotion ready. Verify the published
  release and close the milestone after promotion.
- Keep the first release read-only with respect to T3 agents. Connection pairing
  may create a client session; it does not authorize agent commands.
- Keep T3-specific connection/state ownership here. Do not introduce a dependency
  on DankAIUsage or merge its token totals with T3 usage summaries.
- Do not infer idle from disconnected, stale, or failed observations. Scope thread
  identity to its environment and suppress replayed notifications.
- Do not inspect or copy application credential stores, provider homes, prompts,
  or transcripts to discover an integration. Use the supported connection flow.
  Never print credentials, bearer URLs, raw protocol payloads, or private session
  metadata into logs, diagnostics, fixtures, or issues.
- Use sanitized synthetic fixtures. Keep runtime diagnostics limited to defined
  categories and timestamps; avoid raw server error text.
- ADR-0003 selects a Go standard-library, one-shot HTTP collector. Do not add a
  daemon, adapter framework, WebSocket runtime, or dependency without new evidence.
  The minimized collector stdout is private UI data, not a diagnostic report;
  never attach live output to issues without review.
- Run checks appropriate to the implementation. Use `test.sh` for plugin checks
  when introduced; Go helpers use `go.mod`, `gofmt`, and affected-module tests.
- Track unfinished work in GitHub issues and milestones. Use conventional-ish
  commit prefixes such as `feat:`, `fix:`, `docs:`, and `chore:`.
