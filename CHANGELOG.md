# Changelog

All notable changes are documented here. The project follows Semantic
Versioning and keeps the latest release under GitHub Releases.

## [Unreleased]

## [0.1.0] - 2026-07-10

### Added

- Official OpenAI and Anthropic Go SDK contract tests.
- Persistent credential health, 402/429 failover, idempotent imports, and
  process lifetime storage locking.
- Readiness and metrics endpoints, JSON request logs, request IDs, pool
  summaries, and browser-based OAuth device login.
- Multi-platform archive, checksum, SBOM, checksum signature, and container
  release automation.
- Opt-in credentialed live probe for CPA thinking blocks, signature replay, and
  summarized/omitted streams.

### Changed

- Anthropic and Chat Completions streaming now use per-item state machines and
  surface failed or truncated streams as errors.
- Claude Code adaptive/manual thinking strength now maps to Grok Responses
  reasoning effort, while CPA-style thinking blocks preserve Grok summaries and
  encrypted reasoning through Claude Code tool turns.
- Native Responses encrypted-reasoning items now survive stateless tool-loop
  replay, and conflicting effort spellings fail validation.
- Anthropic attribution metadata is stripped before Grok Build requests because
  the upstream rejects that field.
- Anthropic structured-output schemas now map to Responses `text.format`;
  incompatible `top_k`/stop hints are consumed, and effort remains usable when
  Claude Code explicitly disables thinking.
- Anthropic versioned `web_search_*` server tools now use Grok's built-in web
  search instead of being returned as unexecutable client tool calls; forced
  server-tool choices are normalized to xAI-compatible automatic selection.
- Runtime listen environment overrides are applied before configuration
  validation, allowing a public config bind to be safely narrowed to loopback.
- Public listeners require explicit opt-in.
- Bootstrap secrets are no longer printed to logs.

### Security

- Strict YAML field validation, external request deadlines, revocable bootstrap
  client keys, safe data-directory validation, backups, and durable writes.

[Unreleased]: https://github.com/GreyGunG/grokbuild-proxy/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/GreyGunG/grokbuild-proxy/releases/tag/v0.1.0
