# Security policy

## Supported versions

Security fixes are provided for the latest tagged release. Pre-release builds
may change configuration or storage formats; read the release notes before
upgrading.

## Reporting a vulnerability

Do not open a public issue for a vulnerability that could expose OAuth tokens,
admin keys, client keys, prompts, or upstream quota.

Use GitHub's private vulnerability reporting feature for this repository. If
that feature is unavailable, contact the repository maintainer privately and
include:

- the affected version or commit;
- reproduction steps and impact;
- whether credentials or user content may have been exposed;
- any suggested mitigation.

Please allow a reasonable remediation window before public disclosure.

## Security model

grokbuild-proxy is designed for a single trusted operator on a loopback or
otherwise trusted network. It is not a multi-tenant security boundary.

- Keep the listener on `127.0.0.1` unless `allow_public_listen` is explicitly
  enabled and a trusted TLS/authenticating reverse proxy protects it.
- Treat `data/credentials.json`, `data/meta.json`, `config.yaml`, admin keys,
  client keys, and browser admin sessions as secrets.
- Treat Anthropic thinking signatures and Grok encrypted reasoning as opaque,
  prompt-equivalent secrets. Do not log or modify them, and replay them only
  through the same trusted proxy/model/account context.
- Do not publish logs or support bundles without checking them for secrets.
- Use only accounts and upstream access that you control and are permitted to
  automate under the relevant provider terms.

See the README for deployment hardening and backup guidance.
