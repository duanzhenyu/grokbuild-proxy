# grokbuild-proxy v0.1.0

[简体中文](#简体中文) | [English](#english)

## 简体中文

`grokbuild-proxy` 首个公开版本：将使用者本人合法持有的 Grok Build 账号
接入 Claude Code、Anthropic SDK 和 OpenAI 兼容客户端。

> [!CAUTION]
> 本项目非官方，仅用于技术学习、协议互操作研究和个人自托管实验。使用
> 前请阅读[完整免责声明](https://github.com/GreyGunG/grokbuild-proxy/blob/main/DISCLAIMER.md)。

### 主要功能

- Anthropic Messages、OpenAI Responses、Chat Completions 和 Models API
- Anthropic / Chat Completions 增量 SSE 转换
- 客户端工具、并行调用、Tool Result 回放和 Grok 内置 Web Search
- Adaptive / Manual Thinking 与 CPA 风格 Signature Bridge
- JSON Schema 结构化输出
- 多账号 OAuth、刷新、冷却、会话粘滞和故障切换
- 浏览器 Device Login 和 Grok CLI 凭据导入
- 内嵌 Admin Web UI
- 带锁、原子写入、备份恢复的本地 JSON 存储
- Readiness、Prometheus 指标、结构化日志和 Request ID

### 一键安装

Linux / macOS：

```bash
curl -fsSL \
  https://raw.githubusercontent.com/GreyGunG/grokbuild-proxy/main/scripts/install.sh \
  | sh
```

Windows PowerShell：

```powershell
irm https://raw.githubusercontent.com/GreyGunG/grokbuild-proxy/main/scripts/install.ps1 | iex
```

Docker：

```bash
docker pull ghcr.io/greygung/grokbuild-proxy:0.1.0
```

### 手动下载

本 Release 提供：

- Linux：amd64 / arm64
- macOS：Intel / Apple Silicon
- Windows：amd64 / arm64
- SHA-256 Checksums
- 每个归档对应的 SBOM
- Checksums Sigstore Bundle

详细使用方式见
[README](https://github.com/GreyGunG/grokbuild-proxy#readme) 和
[构建指南](https://github.com/GreyGunG/grokbuild-proxy/blob/main/docs/build-and-run.md)。

### 已知限制

- 上游 Grok CLI 协议不稳定，可能随时发生变化。
- Anthropic `count_tokens` 尚未实现。
- Thinking Signature 仅限本代理和原模型/账号路径。
- 部分 Anthropic 推理参数采用近似映射。
- 只有 Server Web Search 做了专门的 Server Tool 映射。
- OAuth 刷新由请求触发，尚无后台预刷新调度器。

## English

Initial public release of `grokbuild-proxy`, a local compatibility proxy for
using an operator-owned Grok Build account with Claude Code, Anthropic SDKs,
and OpenAI-compatible clients.

> [!CAUTION]
> This is an unofficial project for technical learning, interoperability
> research, and personal self-hosting experiments. Read the
> [full disclaimer](https://github.com/GreyGunG/grokbuild-proxy/blob/main/DISCLAIMER.md)
> before use.

### Highlights

- Anthropic Messages, OpenAI Responses, Chat Completions, and Models APIs
- Incremental Anthropic and Chat Completions SSE translation
- Client tools, parallel calls, tool-result replay, and Grok-hosted web search
- Adaptive/manual effort and CPA-style thinking/signature bridge
- JSON Schema structured output
- Multi-account OAuth selection, refresh, cooldown, sticky sessions, failover
- Browser device login and Grok CLI credential import
- Embedded Admin Web UI
- Locked atomic JSON storage with backup recovery
- Readiness, Prometheus metrics, structured logs, and request IDs

### One-command installation

Linux / macOS:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/GreyGunG/grokbuild-proxy/main/scripts/install.sh \
  | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/GreyGunG/grokbuild-proxy/main/scripts/install.ps1 | iex
```

Docker:

```bash
docker pull ghcr.io/greygung/grokbuild-proxy:0.1.0
```

### Release artifacts

- Linux amd64 / arm64
- macOS Intel / Apple Silicon
- Windows amd64 / arm64
- SHA-256 checksums
- Per-archive SBOMs
- Sigstore bundle for the checksum manifest

See the
[README](https://github.com/GreyGunG/grokbuild-proxy/blob/main/README_EN.md)
and
[build guide](https://github.com/GreyGunG/grokbuild-proxy/blob/main/docs/build-and-run.md).

### Known limitations

- The upstream Grok CLI protocol is unstable and may change.
- Anthropic `count_tokens` is not implemented.
- Thinking signatures are proxy/model/account scoped.
- Some Anthropic reasoning controls are approximated.
- Only hosted web search has a dedicated server-tool mapping.
- OAuth refresh is request-driven rather than scheduled in the background.

## License

MIT
