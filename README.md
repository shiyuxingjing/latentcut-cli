# novelo-cli

[English](#english) | [中文](#中文)

---

<a id="english"></a>

# English

A command-line interface for the Novelo AI novel-to-drama video pipeline. Convert your novel text into short drama video episodes with AI-powered parsing, asset generation, and video assembly.

## Overview

novelo-cli orchestrates the complete Novelo workflow:

1. **AI Parsing** — Analyze novel text and extract story structure, characters, and scenes
2. **Asset Generation** — Generate character images, location artwork, and voice audio
3. **Video Production** — Create keyframes, synchronize audio, and render video clips
4. **Episode Assembly** — Combine shots into polished drama episodes

The CLI integrates with latentCut-server (primary), with legacy support for direct Mastra server integration.

## Prerequisites

- An API key from [shiyuxingjing.com](https://shiyuxingjing.com)
- FFmpeg (optional, for local video merging)

<details>
<summary>For developers building from source</summary>

- **Go 1.25+**
- See the [Development](#development) section below

</details>

## Installation

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/novelo-ai/novelo-cli/releases).

**macOS (Apple Silicon):**
```bash
curl -Lo novelo-cli.tar.gz https://github.com/novelo-ai/novelo-cli/releases/latest/download/novelo-cli_$(curl -s https://api.github.com/repos/novelo-ai/novelo-cli/releases/latest | grep tag_name | cut -d'"' -f4 | sed 's/^v//')_darwin_arm64.tar.gz
tar xzf novelo-cli.tar.gz
chmod +x novelo-cli
sudo mv novelo-cli /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -Lo novelo-cli.tar.gz https://github.com/novelo-ai/novelo-cli/releases/latest/download/novelo-cli_$(curl -s https://api.github.com/repos/novelo-ai/novelo-cli/releases/latest | grep tag_name | cut -d'"' -f4 | sed 's/^v//')_darwin_amd64.tar.gz
tar xzf novelo-cli.tar.gz
chmod +x novelo-cli
sudo mv novelo-cli /usr/local/bin/
```

**Linux (x86_64):**
```bash
curl -Lo novelo-cli.tar.gz https://github.com/novelo-ai/novelo-cli/releases/latest/download/novelo-cli_$(curl -s https://api.github.com/repos/novelo-ai/novelo-cli/releases/latest | grep tag_name | cut -d'"' -f4 | sed 's/^v//')_linux_amd64.tar.gz
tar xzf novelo-cli.tar.gz
chmod +x novelo-cli
sudo mv novelo-cli /usr/local/bin/
```

**Windows:** Download the `.zip` file from [Releases](https://github.com/novelo-ai/novelo-cli/releases) and add `novelo-cli.exe` to your PATH.

### Build from Source

```bash
cd novelo-cli
go build -buildvcs=false -o novelo-cli .
```

The binary will be created as `./novelo-cli` in the current directory.

### Verify Installation

```bash
novelo-cli version
```

You should see output like:
```
novelo-cli 1.0.0 (commit: abc1234, built: 2025-04-04T12:00:00Z)
```

## Authentication

novelo-cli uses **API Key** authentication to communicate with latentCut-server. API keys are persistent, do not expire, and are the only authentication method.

### How It Works

1. Get your API key from [shiyuxingjing.com](https://shiyuxingjing.com) (starts with `nv-`)
2. Run `novelo-cli login --api-key nv-xxx...` or `novelo-cli login` (interactive prompt)
3. The API key is saved to `~/.novelo/config.yaml` as `api_key_latentcut`
4. All requests use the `X-API-Key` header with your API key

### Authentication Priority

The CLI determines the auth token via `EffectiveToken()`:
1. `api_key_latentcut` (preferred — API key, does not expire)
2. `token` (fallback — legacy JWT, expires in 7 days)

## Quick Start

### 1. Configure API Key

Set your API key (get one from [shiyuxingjing.com](https://shiyuxingjing.com)):

Interactive mode:
```bash
novelo-cli login
```

Direct:
```bash
novelo-cli login --api-key nv-abc123...
```

Your API key is saved to `~/.novelo/config.yaml`.

### 2. Brainstorm with Chat

Use the creative-video-agent to brainstorm story ideas and develop scripts.

Single message:
```bash
novelo-cli chat -m "我想写一个仙侠故事"
```

Continue the conversation:
```bash
novelo-cli chat -m "主角有什么特殊能力?" --thread-id thread-xxx
```

Force a new conversation:
```bash
novelo-cli chat -m "换个话题" --new-thread
```

Structured output (for scripts):
```bash
novelo-cli chat -m "就这个方向，开始写吧" --json > novel.txt
```

The agent caches the last thread ID, so subsequent calls reuse the same conversation context unless you use `--new-thread`.

### 3. Produce Video

Convert a novel text file into drama episodes.

Basic usage:
```bash
novelo-cli produce novel.txt
```

With visual style:
```bash
novelo-cli produce novel.txt --style "精致国漫/仙侠风"
```

Custom output directory:
```bash
novelo-cli produce novel.txt --output-dir ./my-drama
```

Print video URLs without downloading:
```bash
novelo-cli produce novel.txt --no-merge
```

## Commands

### login

Configure your API key for latentCut-server authentication. Get your API key from [shiyuxingjing.com](https://shiyuxingjing.com).

**Usage:**
```
novelo-cli login [--api-key <key>]
```

**Flags:**
- `--api-key <key>` — API key (starts with `nv-`). If not provided, prompts interactively.

**Examples:**
```bash
novelo-cli login --api-key nv-abc123def456...
novelo-cli login  # Interactive prompt
```

**What happens:**
1. Saves the API key to `~/.novelo/config.yaml` as `api_key_latentcut`
2. Verifies the API key by calling the server
3. Displays the result (success or warning)

**Output:**
```
Enter your API key (get one from https://shiyuxingjing.com):
API Key: nv-abc123...
Verifying API key with https://api.shiyuxingjing.com...
API key verified and saved successfully!
Config: /Users/username/.novelo/config.yaml
```

You can also set the API key directly via config:
```bash
novelo-cli config set api-key-latentcut nv-xxx...
```

---

### chat

Send messages to the creative-video-agent for brainstorming and script development. Supports multi-turn conversations with automatic thread caching.

**Usage:**
```
novelo-cli chat -m <message> [--thread-id <id>] [--new-thread] [--json]
```

**Flags:**
- `-m, --message <message>` — Message to send (required)
- `--thread-id <id>` — Reuse a specific conversation thread
- `--new-thread` — Force create a new thread (ignore cached thread ID)
- `--json` — Output JSON instead of streaming text

**Examples:**
```bash
# Brainstorm a story idea
novelo-cli chat -m "我想写一个仙侠故事"

# Continue a previous conversation
novelo-cli chat -m "主角有什么特殊能力?" --thread-id thread-abc123

# Start fresh
novelo-cli chat -m "换个话题" --new-thread

# Get structured JSON output (for scripts)
novelo-cli chat -m "就这个方向，开始写吧" --json
```

**Output:**

Default (streaming text):
```
Thread: cli-thread-1712345678000
[streaming response text...]

[thread: cli-thread-1712345678000]
```

With `--json`:
```json
{
  "text": "response content...",
  "threadId": "cli-thread-1712345678000"
}
```

**Thread Caching:**
- When no `--thread-id` is given and `--new-thread` is not set, the CLI reuses the cached `last_thread_id` from `~/.novelo/config.yaml`
- If no cached thread exists, a new client-side thread ID is generated (`cli-thread-<timestamp>`)
- After each call, the effective thread ID is cached for subsequent use

**Conversation History:**
- Local conversation history is stored per thread ID for context continuity
- Previous messages are sent to the agent to maintain conversation context

---

### produce

Convert a novel text file into drama video episodes. Orchestrates the complete pipeline: AI parsing, asset generation, video production, and episode assembly.

**Usage:**
```
novelo-cli produce <input-file> [--style <style>] [--output-dir <dir>] [--no-merge]
```

**Arguments:**
- `<input-file>` — Path to novel text file (required, minimum 100 characters)

**Flags:**
- `--style <style>` — Visual style/aesthetic (e.g., "精致国漫/仙侠风", "写实风格")
- `--output-dir <dir>` — Output directory (default: novelo-output)
- `--no-merge` — Skip download and merging, just print video URLs
- `-v, --verbose` — Enable debug logging
- `--json` — Output progress as JSONL instead of progress bars

**Examples:**
```bash
# Basic production with defaults
novelo-cli produce novel.txt

# Specify visual style
novelo-cli produce novel.txt --style "精致国漫/仙侠风"

# Custom output location
novelo-cli produce novel.txt --output-dir ./drama-output

# Get URLs without downloading
novelo-cli produce novel.txt --no-merge

# With debugging
novelo-cli produce novel.txt -v
```

**Phases:**

| Phase | Description | Method |
|-------|-------------|--------|
| 1. AI Parsing | Analyzes novel text, extracts story structure, characters, and locations | SSE progress stream |
| 2a. Character & Location Assets | Generates character images, character voices, and location images | HTTP + polling |
| 2b. Shot Batch Generation | Creates keyframes, audio, and video clips for all shots | HTTP + polling |
| 3. Episode Assembly | Combines shots into polished episode videos | SSE + polling fallback |
| 4. Download & Merge | Downloads episodes and merges into final drama.mp4 | FFmpeg (optional) |

**Output:**

Progress display (stderr):
```
Creating project "novel" on https://api.shiyuxingjing.com...
Project: uuid-xxx
Task: uuid-yyy

--- Phase 1: AI Parsing ---
  [====================] 100% Parsing novel

AI parsing complete!
Fetching project structure...
  5 episodes, 42 shots, 8 characters, 12 locations

--- Phase 2a: Character & Location Assets ---
  Generating 8 character images...
  Generating 8 character voices...
  Generating 12 location images...
  Waiting for character & location assets...
  [====================] 100% 28/28 assets

--- Phase 2b: Shot Generation (keyframes + audio + video) ---
  42 shots to generate, estimated cost: 500 credits
  Batch started: 42 shots accepted
  [====================] 100% 42/42 tasks done

Asset generation complete!

--- Phase 3: Episode Video Assembly ---
  Episode 1 (Opening): assembling 8 shots...
    Done!
  Episode 2 (Rising Action): assembling 9 shots...
    Done!
  ...

--- Results: 5 episode videos ---
  Episode 1: https://cdn.example.com/ep1.mp4
  Episode 2: https://cdn.example.com/ep2.mp4
  ...

Downloading 5 episode videos...
  Downloading episode 1...
  Downloading episode 2...
  ...
Merging 5 episodes into novelo-output/drama.mp4...

Output: novelo-output/drama.mp4
```

With `--json`:
```
{"type":"event","name":"drama_progress","data":{"progress":25,"currentStep":"Analyzing characters"}}
{"type":"event","name":"drama_done"}
...
```

---

### run

Legacy pipeline via Mastra server (pre-latentCut integration). Kept for backward compatibility.

**Usage:**
```
novelo-cli run <input-file> [--style <style>] [--output-dir <dir>] [--server-url <url>] [--no-merge]
```

**Arguments:**
- `<input-file>` — Path to novel text file (required)

**Flags:**
- `--style <style>` — Visual style override
- `--output-dir <dir>` — Output directory (default: novelo-output)
- `--server-url <url>` — Override Mastra server URL
- `--no-merge` — Skip ffmpeg merge, just print video URLs
- `-v, --verbose` — Enable debug logging
- `--json` — Output progress as JSONL

**Examples:**
```bash
novelo-cli run input.txt --style cinematic
novelo-cli run input.txt --output-dir ./output
novelo-cli run input.txt --json
```

**Note:** This command uses Mastra directly (port 4111) instead of latentCut-server. It requires the legacy `api_key` config (not `api_key_latentcut`). For new projects, use `produce` instead.

---

### config

Manage persistent CLI configuration stored in `~/.novelo/config.yaml`.

#### config set

Set a configuration value.

**Usage:**
```
novelo-cli config set <key> <value>
```

**Supported Keys:**

| Key | Aliases | Purpose |
|-----|---------|---------|
| `api-key` | `api_key` | Legacy API key for Mastra server |
| `api-key-latentcut` | `api_key_latentcut` | API key for latentCut-server (primary auth) |
| `server-url` | `server_url` | Mastra server URL |
| `output-dir` | `output_dir` | Output directory for videos |
| `latentcut-url` | `latentcut_url` | latentCut-server URL |
| `token` | — | Legacy JWT token |
| `account` | — | Account name |

**Examples:**
```bash
novelo-cli config set api-key-latentcut nv-abc123...
novelo-cli config set latentcut-url http://latentcut.example.com:7001
novelo-cli config set output-dir ./my-videos
novelo-cli config set server-url http://mastra.example.com:4111
```

#### config get

Get a configuration value.

**Usage:**
```
novelo-cli config get <key>
```

**Examples:**
```bash
novelo-cli config get api-key-latentcut
novelo-cli config get latentcut-url
novelo-cli config get account
```

Sensitive values (`api-key`, `api-key-latentcut`, `token`) are masked in output.

#### config list

Display all configuration values.

**Usage:**
```
novelo-cli config list
```

**Output:**
```
api-key:            (not set)
api-key-latentcut:  ****ef01
server-url:         http://localhost:4111
output-dir:         novelo-output
latentcut-url:      https://api.shiyuxingjing.com
account:            user@example.com
token:              (not set)

Config path: /Users/username/.novelo/config.yaml
```

---

### credits

Show current credit balance.

**Usage:**
```
novelo-cli credits
novelo-cli credits --json
```

**Output:**
```
Credits: 10060 (daily: 60, purchased: 10000)
```

**JSON output (`--json`):**
```json
{"credits":10060,"credits_daily":60,"credits_purchased":10000}
```

---

### recharge

Redeem a credit code to add credits to your account.

**Usage:**
```
novelo-cli recharge --code <redeem-code>
novelo-cli recharge -c <redeem-code>
```

**Flags:**

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--code` | `-c` | Yes | Redeem code to apply |

**Example:**
```bash
novelo-cli recharge -c ABC123XYZ
# Redeeming code: ABC123XYZ...
# Redeemed successfully!
# Current credits: 15060
```

**Other payment methods:**

CLI supports redeem code recharging only. For Alipay or WeChat Pay, please visit the official website: https://shiyuxingjing.com

---

### version

Print version information.

**Usage:**
```
novelo-cli version
```

**Output:**
```
novelo-cli 1.0.0 (commit: abc1234, built: 2025-04-04T12:00:00Z)
```

## Configuration

### Config File Location

Configuration is stored in `~/.novelo/config.yaml` (in your home directory).

### Available Keys

| Key | Purpose | Default | Set via |
|-----|---------|---------|---------|
| `api_key` | Legacy Mastra API key | (not set) | `config set api-key` |
| `api_key_latentcut` | API key for latentCut-server (primary auth, `nv-` prefix) | (not set) | `login` command, `config set api-key-latentcut` |
| `server_url` | Mastra server URL | `http://localhost:4111` | `config set server-url` |
| `output_dir` | Output directory for videos | `novelo-output` | `config set output-dir`, `produce --output-dir` |
| `latentcut_url` | latentCut-server URL | `https://api.shiyuxingjing.com` | `config set latentcut-url` |
| `token` | Legacy JWT token (backward compat, expires in 7 days) | (not set) | Legacy login |
| `account` | Logged-in account | (not set) | `login` command |
| `last_thread_id` | Cached chat thread ID | (not set) | `chat` command (auto-cached) |

### Example Config File

```yaml
api_key_latentcut: nv-abc123def456...
server_url: http://localhost:4111
output_dir: novelo-output
latentcut_url: https://api.shiyuxingjing.com
account: user@example.com
last_thread_id: cli-thread-1712345678000
```

## Global Flags

These flags apply to all commands:

- `-v, --verbose` — Enable verbose debug logging
- `--json` — Output results as JSONL instead of human-readable text/progress bars

**Examples:**
```bash
# Verbose debugging
novelo-cli produce novel.txt -v

# JSON output for parsing
novelo-cli produce novel.txt --json | jq .
```

## Workflow Examples

### Complete Novel-to-Drama Workflow

```bash
# 1. Configure API key
novelo-cli login --api-key nv-abc123...

# 2. Brainstorm and draft
novelo-cli chat -m "我想写一个仙侠故事" --json > draft.txt

# 3. Refine with follow-ups (thread auto-cached)
novelo-cli chat -m "主角应该有什么背景?" --json >> draft.txt

# 4. Generate drama episodes
novelo-cli produce draft.txt --style "精致国漫/仙侠风"

# Result: novelo-output/drama.mp4
```

### Get Video URLs Without Local Merging

```bash
novelo-cli produce novel.txt --no-merge > episode_urls.txt
```

This prints URLs like:
```
https://cdn.example.com/episode_01.mp4
https://cdn.example.com/episode_02.mp4
https://cdn.example.com/episode_03.mp4
```

### Debug a Failed Production

```bash
novelo-cli produce novel.txt -v 2>&1 | tee production.log
```

Check `production.log` for detailed error messages.

### Change Output Location

```bash
# Option 1: Command-line flag
novelo-cli produce novel.txt --output-dir /tmp/drama

# Option 2: Configuration
novelo-cli config set output-dir /tmp/drama
novelo-cli produce novel.txt  # Uses configured output-dir
```

## Troubleshooting

### "not logged in. Run: novelo-cli login"

You need to authenticate first:
```bash
novelo-cli login
```

### "unauthorized (401): token expired or invalid, run: novelo-cli login"

Your API key may have been revoked or deleted. Re-authenticate:
```bash
novelo-cli login
```

### "connection refused" or "dial tcp"

Cannot reach the Novelo server. Check your network connection and server URL:
```bash
novelo-cli config get latentcut-url
```

If you need to use a custom server:
```bash
novelo-cli config set latentcut-url http://your-server:7001
```

### FFmpeg not found

FFmpeg is required for the final video merge step. Install it:

**macOS:**
```bash
brew install ffmpeg
```

**Ubuntu/Debian:**
```bash
sudo apt-get install ffmpeg
```

**Fedora/CentOS:**
```bash
sudo yum install ffmpeg
```

Alternatively, use `--no-merge` to skip merging and get raw episode URLs instead.

### Config file permissions error

If you get permission errors reading/writing `~/.novelo/config.yaml`:

```bash
# Fix permissions
chmod 600 ~/.novelo/config.yaml

# Or recreate
rm ~/.novelo/config.yaml
novelo-cli login
```

### "novel text too short (X chars), minimum 100 characters"

Your input file is less than 100 characters. The AI parser requires enough context. Provide a longer text sample.

### Production times out or stalls

Large novels may take extended time. Check the web UI for the project to see actual progress:
- Project UUID and Task UUID are printed at the start
- Web UI shows detailed step-by-step progress
- Character/location assets can take several minutes
- Video generation is the longest phase (5-30 minutes depending on shot count)

## Architecture

```
novelo-cli
    ↓ (HTTP + SSE, X-API-Key auth)
latentCut-server (shiyuxingjing.com)
    ├─→ Mastra (port 4111) — LLM novel parsing + script generation
    ├─→ RunningHub — Character/location image generation, audio synthesis
    └─→ Aliyun OSS — Asset storage (URLs returned to CLI)
```

**Legacy Path:**
```
novelo-cli
    ↓ (WebSocket + HTTP)
Mastra server (port 4111) — Direct novel parsing
    ↓
RunningHub — Asset generation
```

## Environment

novelo-cli reads configuration from:

1. Command-line flags (highest priority)
2. `~/.novelo/config.yaml` (persistent configuration)
3. Built-in defaults (lowest priority)

It does not read from environment variables.

## Development

### Build Options

```bash
# Standard build
go build -o novelo-cli .

# Build without version control info (e.g., in CI)
go build -buildvcs=false -o novelo-cli .

# Build with version info (goreleaser style)
go build -ldflags="-X main.version=1.0.0 -X main.commit=abc1234 -X main.date=2025-04-04" -o novelo-cli .
```

### Dependencies

- **github.com/spf13/cobra** — CLI framework
- **golang.org/x/term** — Password input masking
- **gopkg.in/yaml.v3** — Config file parsing
- **github.com/coder/websocket** — WebSocket client (legacy pipeline)
- **github.com/schollz/progressbar/v3** — Progress display

Run `go mod tidy` to verify dependencies.

## License

See LICENSE file in the repository.

## Support

For issues, feature requests, or questions:

1. Check this README and troubleshooting section
2. Review error messages (use `--verbose` for details)
3. Check latentCut-server logs
4. Contact the Novelo team with your project UUID and Task UUID

## Changelog

See CHANGELOG.md for version history and changes.

---

<a id="中文"></a>

# 中文

Novelo AI 小说转短剧视频的命令行工具。利用 AI 驱动的解析、素材生成和视频合成，将小说文本转化为短剧视频。

## 概述

novelo-cli 编排完整的 Novelo 工作流：

1. **AI 解析** — 分析小说文本，提取故事结构、角色和场景
2. **素材生成** — 生成角色形象、场景画面和语音音频
3. **视频制作** — 创建关键帧、同步音频、渲染视频片段
4. **剧集合成** — 将镜头组合为完整的短剧集数

CLI 主要集成 latentCut-server，同时保留对 Mastra 服务器的旧版支持。

## 前置要求

- API 密钥（在 [shiyuxingjing.com](https://shiyuxingjing.com) 获取）
- FFmpeg（可选，用于本地视频合并）

<details>
<summary>从源码构建的开发者</summary>

- **Go 1.25+**
- 参见下方 [开发指南](#开发指南) 部分

</details>

## 安装

### 下载二进制文件

从 [GitHub Releases](https://github.com/novelo-ai/novelo-cli/releases) 下载适合你平台的最新版本。

**macOS (Apple Silicon):**
```bash
curl -Lo novelo-cli.tar.gz https://github.com/novelo-ai/novelo-cli/releases/latest/download/novelo-cli_$(curl -s https://api.github.com/repos/novelo-ai/novelo-cli/releases/latest | grep tag_name | cut -d'"' -f4 | sed 's/^v//')_darwin_arm64.tar.gz
tar xzf novelo-cli.tar.gz
chmod +x novelo-cli
sudo mv novelo-cli /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -Lo novelo-cli.tar.gz https://github.com/novelo-ai/novelo-cli/releases/latest/download/novelo-cli_$(curl -s https://api.github.com/repos/novelo-ai/novelo-cli/releases/latest | grep tag_name | cut -d'"' -f4 | sed 's/^v//')_darwin_amd64.tar.gz
tar xzf novelo-cli.tar.gz
chmod +x novelo-cli
sudo mv novelo-cli /usr/local/bin/
```

**Linux (x86_64):**
```bash
curl -Lo novelo-cli.tar.gz https://github.com/novelo-ai/novelo-cli/releases/latest/download/novelo-cli_$(curl -s https://api.github.com/repos/novelo-ai/novelo-cli/releases/latest | grep tag_name | cut -d'"' -f4 | sed 's/^v//')_linux_amd64.tar.gz
tar xzf novelo-cli.tar.gz
chmod +x novelo-cli
sudo mv novelo-cli /usr/local/bin/
```

**Windows：** 从 [Releases](https://github.com/novelo-ai/novelo-cli/releases) 下载 `.zip` 文件，将 `novelo-cli.exe` 添加到 PATH。

### 从源码构建

```bash
cd novelo-cli
go build -buildvcs=false -o novelo-cli .
```

构建产物为当前目录下的 `./novelo-cli`。

### 验证安装

```bash
novelo-cli version
```

预期输出：
```
novelo-cli 1.0.0 (commit: abc1234, built: 2025-04-04T12:00:00Z)
```

## 认证机制

novelo-cli 使用 **API Key** 与 latentCut-server 通信。API Key 是持久化的、不会过期的，是唯一的认证方式。

### 工作原理

1. 从 [shiyuxingjing.com](https://shiyuxingjing.com) 获取你的 API Key（以 `nv-` 为前缀）
2. 运行 `novelo-cli login --api-key nv-xxx...` 或 `novelo-cli login`（交互式输入）
3. API Key 保存到 `~/.novelo/config.yaml` 的 `api_key_latentcut` 字段
4. 后续所有请求通过 `X-API-Key` 请求头携带 API Key

### 认证优先级

CLI 通过 `EffectiveToken()` 确定使用哪个凭证：
1. `api_key_latentcut`（优先 — API Key，永不过期）
2. `token`（兜底 — 旧版 JWT，7 天过期）

### 服务端认证逻辑

latentCut-server 的认证中间件支持以下方式（按优先级排列）：

1. **`X-API-Key` 请求头** — 直接传入 API Key
2. **`Authorization: Bearer nv-xxx` 请求头** — 以 `nv-` 开头的 Bearer 令牌自动识别为 API Key
3. **`Authorization: Bearer <jwt>` 请求头** — 传统 JWT 令牌认证
4. **`?token=` 查询参数或请求体** — 兼容旧版传参方式

## 快速开始

### 1. 配置 API Key

设置你的 API Key（从 [shiyuxingjing.com](https://shiyuxingjing.com) 获取）：

交互模式：
```bash
novelo-cli login
```

直接设置：
```bash
novelo-cli login --api-key nv-abc123...
```

你的 API Key 将保存到 `~/.novelo/config.yaml`。

### 2. 与 AI 创意对话

使用创意视频 Agent 进行故事创意头脑风暴和剧本开发。

发送单条消息：
```bash
novelo-cli chat -m "我想写一个仙侠故事"
```

继续对话：
```bash
novelo-cli chat -m "主角有什么特殊能力?" --thread-id thread-xxx
```

强制开始新对话：
```bash
novelo-cli chat -m "换个话题" --new-thread
```

结构化输出（用于编写剧本）：
```bash
novelo-cli chat -m "就这个方向，开始写吧" --json > novel.txt
```

Agent 会缓存上次的会话 ID，后续调用自动复用同一对话上下文，除非使用 `--new-thread`。

### 3. 制作视频

将小说文本文件转化为短剧集数。

基本用法：
```bash
novelo-cli produce novel.txt
```

指定视觉风格：
```bash
novelo-cli produce novel.txt --style "精致国漫/仙侠风"
```

自定义输出目录：
```bash
novelo-cli produce novel.txt --output-dir ./my-drama
```

只打印视频 URL，不下载：
```bash
novelo-cli produce novel.txt --no-merge
```

## 命令参考

### login — 配置 API Key

配置 latentCut-server 的 API Key 认证。从 [shiyuxingjing.com](https://shiyuxingjing.com) 获取你的 API Key。

**用法：**
```
novelo-cli login [--api-key <密钥>]
```

**参数：**
- `--api-key <密钥>` — API Key（以 `nv-` 开头）。未提供时交互式输入。

**示例：**
```bash
novelo-cli login --api-key nv-abc123def456...
novelo-cli login  # 交互式输入
```

**执行过程：**
1. 将 API Key 保存到 `~/.novelo/config.yaml` 的 `api_key_latentcut` 字段
2. 调用服务器验证 API Key 是否有效
3. 显示验证结果（成功或警告）

**输出：**
```
Enter your API key (get one from https://shiyuxingjing.com):
API Key: nv-abc123...
Verifying API key with https://api.shiyuxingjing.com...
API key verified and saved successfully!
Config: /Users/username/.novelo/config.yaml
```

也可以通过 config 命令直接设置：
```bash
novelo-cli config set api-key-latentcut nv-xxx...
```

---

### chat — 创意对话

向创意视频 Agent 发送消息，用于头脑风暴和剧本开发。支持多轮对话，自动缓存会话。

**用法：**
```
novelo-cli chat -m <消息> [--thread-id <id>] [--new-thread] [--json]
```

**参数：**
- `-m, --message <消息>` — 要发送的消息（必填）
- `--thread-id <id>` — 指定复用的对话 ID
- `--new-thread` — 强制创建新对话（忽略缓存的对话 ID）
- `--json` — 输出 JSON 格式而非流式文本

**示例：**
```bash
# 头脑风暴故事创意
novelo-cli chat -m "我想写一个仙侠故事"

# 继续之前的对话
novelo-cli chat -m "主角有什么特殊能力?" --thread-id thread-abc123

# 开始新对话
novelo-cli chat -m "换个话题" --new-thread

# 获取结构化 JSON 输出
novelo-cli chat -m "就这个方向，开始写吧" --json
```

**输出：**

默认（流式文本）：
```
Thread: cli-thread-1712345678000
[流式响应文本...]

[thread: cli-thread-1712345678000]
```

使用 `--json`：
```json
{
  "text": "响应内容...",
  "threadId": "cli-thread-1712345678000"
}
```

**会话缓存机制：**
- 未指定 `--thread-id` 且未使用 `--new-thread` 时，自动复用 `~/.novelo/config.yaml` 中缓存的 `last_thread_id`
- 若无缓存的会话 ID，自动生成客户端会话 ID（`cli-thread-<时间戳>`）
- 每次调用后，有效的会话 ID 会被缓存以供后续使用

**对话历史：**
- 每个会话 ID 的对话历史存储在本地，保证上下文连续性
- 历史消息会随请求发送给 Agent 以维持对话上下文

---

### produce — 制作视频

将小说文本文件转化为短剧视频。编排完整流水线：AI 解析、素材生成、视频制作和剧集合成。

**用法：**
```
novelo-cli produce <输入文件> [--style <风格>] [--output-dir <目录>] [--no-merge]
```

**位置参数：**
- `<输入文件>` — 小说文本文件路径（必填，最少 100 个字符）

**可选参数：**
- `--style <风格>` — 视觉风格（如 "精致国漫/仙侠风"、"写实风格"）
- `--output-dir <目录>` — 输出目录（默认：novelo-output）
- `--no-merge` — 跳过下载和合并，只打印视频 URL
- `-v, --verbose` — 启用调试日志
- `--json` — 以 JSONL 格式输出进度

**示例：**
```bash
# 默认参数制作
novelo-cli produce novel.txt

# 指定视觉风格
novelo-cli produce novel.txt --style "精致国漫/仙侠风"

# 自定义输出位置
novelo-cli produce novel.txt --output-dir ./drama-output

# 获取 URL 不下载
novelo-cli produce novel.txt --no-merge

# 调试模式
novelo-cli produce novel.txt -v
```

**流程阶段：**

| 阶段 | 描述 | 方式 |
|------|------|------|
| 1. AI 解析 | 分析小说文本，提取故事结构、角色和场景 | SSE 进度流 |
| 2a. 角色和场景素材 | 生成角色形象、角色语音和场景图片 | HTTP + 轮询 |
| 2b. 镜头批量生成 | 为所有镜头创建关键帧、音频和视频 | HTTP + 轮询 |
| 3. 剧集合成 | 将镜头组合为完整的剧集视频 | SSE + 轮询兜底 |
| 4. 下载与合并 | 下载剧集并合并为最终的 drama.mp4 | FFmpeg（可选） |

**输出示例：**
```
Creating project "novel" on https://api.shiyuxingjing.com...
Project: uuid-xxx
Task: uuid-yyy

--- Phase 1: AI Parsing ---
  [====================] 100% Parsing novel

AI parsing complete!
Fetching project structure...
  5 episodes, 42 shots, 8 characters, 12 locations

--- Phase 2a: Character & Location Assets ---
  Generating 8 character images...
  Generating 8 character voices...
  Generating 12 location images...
  Waiting for character & location assets...
  [====================] 100% 28/28 assets

--- Phase 2b: Shot Generation (keyframes + audio + video) ---
  42 shots to generate, estimated cost: 500 credits
  Batch started: 42 shots accepted
  [====================] 100% 42/42 tasks done

Asset generation complete!

--- Phase 3: Episode Video Assembly ---
  Episode 1 (Opening): assembling 8 shots...
    Done!
  ...

--- Results: 5 episode videos ---
  Episode 1: https://cdn.example.com/ep1.mp4
  ...

Downloading 5 episode videos...
Merging 5 episodes into novelo-output/drama.mp4...

Output: novelo-output/drama.mp4
```

---

### run — 运行旧版管线

通过 Mastra 服务器运行的旧版管线（latentCut 集成之前的版本），保留用于向后兼容。

**用法：**
```
novelo-cli run <输入文件> [--style <风格>] [--output-dir <目录>] [--server-url <url>] [--no-merge]
```

**参数：**
- `<输入文件>` — 小说文本文件路径（必填）
- `--style <风格>` — 视觉风格覆盖
- `--output-dir <目录>` — 输出目录（默认：novelo-output）
- `--server-url <url>` — 覆盖 Mastra 服务器 URL
- `--no-merge` — 跳过 ffmpeg 合并，仅打印视频 URL
- `-v, --verbose` — 启用调试日志
- `--json` — 以 JSONL 格式输出进度

**注意：** 此命令直接使用 Mastra（端口 4111）而非 latentCut-server。需要旧版 `api_key` 配置（非 `api_key_latentcut`）。新项目请使用 `produce`。

---

### config — 配置管理

管理存储在 `~/.novelo/config.yaml` 中的持久化 CLI 配置。

#### config set — 设置配置值

**用法：**
```
novelo-cli config set <键> <值>
```

**支持的键：**

| 键 | 别名 | 用途 |
|-----|------|------|
| `api-key` | `api_key` | Mastra 服务器的旧版 API Key |
| `api-key-latentcut` | `api_key_latentcut` | latentCut-server 的 API Key（主要认证） |
| `server-url` | `server_url` | Mastra 服务器 URL |
| `output-dir` | `output_dir` | 视频输出目录 |
| `latentcut-url` | `latentcut_url` | latentCut-server URL |
| `token` | — | 旧版 JWT 令牌 |
| `account` | — | 账号名 |

**示例：**
```bash
novelo-cli config set api-key-latentcut nv-abc123...
novelo-cli config set latentcut-url http://latentcut.example.com:7001
novelo-cli config set output-dir ./my-videos
```

#### config get — 获取配置值

**用法：**
```
novelo-cli config get <键>
```

**示例：**
```bash
novelo-cli config get api-key-latentcut
novelo-cli config get latentcut-url
novelo-cli config get account
```

敏感值（`api-key`、`api-key-latentcut`、`token`）在输出中会被脱敏显示。

#### config list — 列出所有配置

**用法：**
```
novelo-cli config list
```

**输出：**
```
api-key:            (not set)
api-key-latentcut:  ****ef01
server-url:         http://localhost:4111
output-dir:         novelo-output
latentcut-url:      https://api.shiyuxingjing.com
account:            user@example.com
token:              (not set)

Config path: /Users/username/.novelo/config.yaml
```

---

### credits — 查询积分

显示当前积分余额。

**用法：**
```
novelo-cli credits
novelo-cli credits --json
```

**输出：**
```
Credits: 10060 (daily: 60, purchased: 10000)
```

**JSON 输出（`--json`）：**
```json
{"credits":10060,"credits_daily":60,"credits_purchased":10000}
```

---

### recharge — 充值积分

兑换充值码，为账户添加积分。

**用法：**
```
novelo-cli recharge --code <兑换码>
novelo-cli recharge -c <兑换码>
```

**参数：**

| 参数 | 简写 | 必填 | 描述 |
|------|------|------|------|
| `--code` | `-c` | 是 | 兑换码 |

**示例：**
```bash
novelo-cli recharge -c ABC123XYZ
# Redeeming code: ABC123XYZ...
# Redeemed successfully!
# Current credits: 15060
```

**其他充值方式：**

CLI 仅支持兑换码充值。若需支付宝或微信支付，请访问官网：https://shiyuxingjing.com

---

### version — 版本信息

打印版本信息。

**用法：**
```
novelo-cli version
```

**输出：**
```
novelo-cli 1.0.0 (commit: abc1234, built: 2025-04-04T12:00:00Z)
```

## 配置说明

### 配置文件位置

配置存储在 `~/.novelo/config.yaml`（用户主目录下）。

### 可用配置项

| 键 | 用途 | 默认值 | 设置方式 |
|-----|------|--------|---------|
| `api_key` | Mastra 旧版 API Key | （未设置） | `config set api-key` |
| `api_key_latentcut` | latentCut-server 的 API Key（主要认证，`nv-` 前缀） | （未设置） | `login` 命令，`config set api-key-latentcut` |
| `server_url` | Mastra 服务器 URL | `http://localhost:4111` | `config set server-url` |
| `output_dir` | 视频输出目录 | `novelo-output` | `config set output-dir`，`produce --output-dir` |
| `latentcut_url` | latentCut-server URL | `https://api.shiyuxingjing.com` | `config set latentcut-url` |
| `token` | 旧版 JWT 令牌（向后兼容，7 天过期） | （未设置） | 旧版 login |
| `account` | 登录账号 | （未设置） | `login` 命令 |
| `last_thread_id` | 缓存的对话 ID | （未设置） | `chat` 命令（自动缓存） |

### 配置文件示例

```yaml
api_key_latentcut: nv-abc123def456...
server_url: http://localhost:4111
output_dir: novelo-output
latentcut_url: https://api.shiyuxingjing.com
account: user@example.com
last_thread_id: cli-thread-1712345678000
```

## 全局参数

以下参数适用于所有命令：

- `-v, --verbose` — 启用详细调试日志
- `--json` — 以 JSONL 格式输出结果，替代人类可读的文本/进度条

**示例：**
```bash
# 调试模式
novelo-cli produce novel.txt -v

# JSON 输出便于解析
novelo-cli produce novel.txt --json | jq .
```

## 工作流示例

### 完整的小说转短剧流程

```bash
# 1. 配置 API Key
novelo-cli login --api-key nv-abc123...

# 2. 头脑风暴并撰写初稿
novelo-cli chat -m "我想写一个仙侠故事" --json > draft.txt

# 3. 继续细化（会话自动缓存）
novelo-cli chat -m "主角应该有什么背景?" --json >> draft.txt

# 4. 生成短剧集数
novelo-cli produce draft.txt --style "精致国漫/仙侠风"

# 结果：novelo-output/drama.mp4
```

### 仅获取视频 URL

```bash
novelo-cli produce novel.txt --no-merge > episode_urls.txt
```

输出格式：
```
https://cdn.example.com/episode_01.mp4
https://cdn.example.com/episode_02.mp4
https://cdn.example.com/episode_03.mp4
```

### 调试失败的制作

```bash
novelo-cli produce novel.txt -v 2>&1 | tee production.log
```

检查 `production.log` 获取详细错误信息。

### 更改输出位置

```bash
# 方式 1：命令行参数
novelo-cli produce novel.txt --output-dir /tmp/drama

# 方式 2：修改配置
novelo-cli config set output-dir /tmp/drama
novelo-cli produce novel.txt  # 使用配置的输出目录
```

## 常见问题

### "not logged in. Run: novelo-cli login"

需要先认证：
```bash
novelo-cli login
```

### "unauthorized (401): token expired or invalid, run: novelo-cli login"

你的 API Key 可能已被撤销或删除。重新认证：
```bash
novelo-cli login
```

### "connection refused" 或 "dial tcp"

无法连接到 Novelo 服务器。检查网络连接和服务器 URL：
```bash
novelo-cli config get latentcut-url
```

如需使用自定义服务器：
```bash
novelo-cli config set latentcut-url http://your-server:7001
```

### 找不到 FFmpeg

FFmpeg 用于最终的视频合并步骤。安装方法：

**macOS:**
```bash
brew install ffmpeg
```

**Ubuntu/Debian:**
```bash
sudo apt-get install ffmpeg
```

**Fedora/CentOS:**
```bash
sudo yum install ffmpeg
```

或者使用 `--no-merge` 跳过合并，直接获取剧集视频 URL。

### 配置文件权限错误

如果读写 `~/.novelo/config.yaml` 时出现权限错误：

```bash
# 修复权限
chmod 600 ~/.novelo/config.yaml

# 或重新创建
rm ~/.novelo/config.yaml
novelo-cli login
```

### "novel text too short (X chars), minimum 100 characters"

输入文件不足 100 个字符。AI 解析器需要足够的上下文，请提供更长的文本。

### 制作超时或卡住

大型小说可能需要较长时间。在 Web UI 中查看项目的实际进度：
- 项目 UUID 和任务 UUID 在开始时会打印出来
- Web UI 显示详细的逐步进度
- 角色/场景素材可能需要几分钟
- 视频生成是最长的阶段（取决于镜头数量，5-30 分钟）

## 系统架构

```
novelo-cli
    ↓ (HTTP + SSE, X-API-Key 认证)
latentCut-server (shiyuxingjing.com)
    ├─→ Mastra (端口 4111) — LLM 小说解析 + 剧本生成
    ├─→ RunningHub — 角色/场景图像生成、音频合成
    └─→ 阿里云 OSS — 素材存储（URL 返回给 CLI）
```

**旧版路径：**
```
novelo-cli
    ↓ (WebSocket + HTTP)
Mastra 服务器 (端口 4111) — 直接小说解析
    ↓
RunningHub — 素材生成
```

## 运行环境

novelo-cli 按以下优先级读取配置：

1. 命令行参数（最高优先级）
2. `~/.novelo/config.yaml`（持久化配置）
3. 内置默认值（最低优先级）

不读取环境变量。

## 开发指南

### 构建选项

```bash
# 标准构建
go build -o novelo-cli .

# 不包含版本控制信息的构建（如在 CI 中）
go build -buildvcs=false -o novelo-cli .

# 包含版本信息的构建（goreleaser 风格）
go build -ldflags="-X main.version=1.0.0 -X main.commit=abc1234 -X main.date=2025-04-04" -o novelo-cli .
```

### 依赖

- **github.com/spf13/cobra** — CLI 框架
- **golang.org/x/term** — 密码输入掩码
- **gopkg.in/yaml.v3** — 配置文件解析
- **github.com/coder/websocket** — WebSocket 客户端（旧版管线）
- **github.com/schollz/progressbar/v3** — 进度条显示

运行 `go mod tidy` 验证依赖。

## 许可证

见仓库中的 LICENSE 文件。

## 支持

遇到问题、功能建议或疑问：

1. 查阅本 README 和常见问题部分
2. 查看错误信息（使用 `--verbose` 获取详情）
3. 检查 latentCut-server 日志
4. 联系 Novelo 团队并提供你的项目 UUID 和任务 UUID

## 更新日志

见 CHANGELOG.md 了解版本历史和变更记录。
