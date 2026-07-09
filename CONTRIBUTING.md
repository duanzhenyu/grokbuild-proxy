# Contributing

Thanks for helping improve grokbuild-proxy.

## Development

Requirements:

- Go 1.26.5 or newer;
- Docker for container smoke tests;
- no real credentials in fixtures, logs, commits, or issue reports.

Run the local checks before submitting a change:

```bash
gofmt -w ./cmd ./internal
go vet ./...
go test -race ./...
go build ./cmd/grokbuild-proxy
```

Use synthetic tokens such as `access-token-fixture` in tests. Tests must not
depend on a live xAI/Grok account or network access. Live smoke tests belong in
an explicitly opt-in workflow.

## Pull requests

- Keep changes focused and explain user-visible behavior.
- Add tests for protocol conversion, retry/failover, auth, or storage changes.
- Update the compatibility matrix and configuration documentation when public
  behavior changes.
- Preserve loopback-first defaults and avoid logging prompts or secrets.
- Do not add SaaS, multi-tenant, database, or provider-abstraction scope without
  an accepted design discussion.

## Compatibility changes

The OpenAI and Anthropic endpoints intentionally implement documented subsets.
Do not silently discard newly accepted fields. Either map them, preserve them
when safe, or return a clear validation error and update the compatibility
matrix.

## License

By submitting a contribution, you agree that it is licensed under the
MIT License.
