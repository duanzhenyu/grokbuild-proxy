# Release checklist

## Automated gates

- [ ] `gofmt` clean
- [ ] `go vet ./...`
- [ ] `go test -race ./...`
- [ ] official OpenAI and Anthropic SDK contract tests pass
- [ ] `govulncheck ./...`
- [ ] Docker build and `/healthz` smoke pass
- [ ] GoReleaser snapshot succeeds for every target
- [ ] no secrets or generated runtime data in the diff

## Compatibility and operations

- [ ] `COMPATIBILITY.md` matches accepted/rejected fields and stream behavior
- [ ] configuration changes are documented and strict parsing tested
- [ ] backup/restore and upgrade notes are current
- [ ] migration from the previous release is tested against a copied data dir
- [ ] 401 refresh, 402/429 failover, cooldown restart, and truncated SSE tests pass
- [ ] Admin device login, key revocation, readiness, and pool summary tests pass

## Optional live gate

With an operator-owned test account and explicit consent:

```bash
GROKBUILD_LIVE_SMOKE=1 \
GROKBUILD_API_KEY=... \
GROKBUILD_LIVE_MODEL=grok-4.5 \
bash scripts/live-smoke.sh
```

For Claude Code thinking/signature compatibility:

```bash
GROKBUILD_LIVE_SMOKE=1 \
GROKBUILD_API_KEY=... \
GROKBUILD_LIVE_MODEL=claude-opus-4-6 \
go run ./scripts/live-thinking-probe.go
```

Never run live smoke in pull requests or with production credentials.

## Publishing

- [ ] update `CHANGELOG.md`
- [ ] create a signed `vMAJOR.MINOR.PATCH` tag
- [ ] verify archives, checksums, SBOMs, Sigstore bundle, and GHCR manifests
- [ ] test one downloaded archive on a clean host
- [ ] publish upgrade and rollback notes
