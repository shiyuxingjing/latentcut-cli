# novelo-cli

A command-line interface for the Novelo AI novel-to-drama video pipeline. Convert your novel text into short drama video episodes with AI-powered parsing, asset generation, and video assembly.

## Overview

novelo-cli orchestrates the complete Novelo workflow:

1. **AI Parsing** — Analyze novel text and extract story structure, characters, and scenes
2. **Asset Generation** — Generate character images, location artwork, and voice audio
3. **Video Production** — Create keyframes, synchronize audio, and render video clips
4. **Episode Assembly** — Combine shots into polished drama episodes

The CLI integrates with latentCut-server (primary), with legacy support for direct Mastra server integration.

## Prerequisites

Before using novelo-cli, you need:

- **Go 1.25+** (to build from source)
- Three running services:
  - latentCut-server (port 7001) — Orchestrates the full pipeline
  - Mastra (port 4111) — LLM-powered novel parsing
  - RunningHub — Image, audio, and video generation backend
- MySQL + Redis — Database and caching for latentCut-server
- FFmpeg (optional, for local video merging)
- A registered user account with latentCut-server

## Installation

### Build from Source

```bash
cd novelo-cli
go build -buildvcs=false -o novelo-cli .
```

The binary will be created as `./novelo-cli` in the current directory.

### Verify Installation

```bash
./novelo-cli version
```

You should see output like:
```
novelo-cli dev (commit: none, built: unknown)
```

## Quick Start

### 1. Login

Authenticate with latentCut-server to obtain a JWT token.

Interactive mode (prompts for credentials):
```bash
novelo-cli login
```

Command-line arguments:
```bash
novelo-cli login --account user@example.com --password yourpassword
```

Your credentials are saved to `~/.novelo/config.yaml`.

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

Authenticate with latentCut-server and save a JWT token.

**Usage:**
```
novelo-cli login [--account <email>] [--password <password>]
```

**Flags:**
- `--account <email>` — User account (email or username)
- `--password <password>` — Password (prompted if not provided)

**Examples:**
```bash
novelo-cli login --account user@example.com --password mypass
novelo-cli login  # Interactive prompt
```

**Output:**
- Logs in with your password to obtain a short-lived JWT
- Automatically creates a persistent API key (`novelo-cli`) using that JWT
- Saves the API key to `~/.novelo/config.yaml` as `api_key_latentcut`
- The JWT is discarded — only the API key is stored
- API keys do not expire (unlike the 7-day JWT expiry)

You can also set the API key manually without logging in:
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
Thread: thread-abc123
[streaming response text...]
[thread: thread-abc123]
```

With `--json`:
```json
{
  "text": "response content...",
  "threadId": "thread-abc123"
}
```

**Thread Caching:**
The CLI automatically caches the last thread ID to `~/.novelo/config.yaml` under `last_thread_id`. Subsequent calls reuse this thread unless you specify a different `--thread-id` or use `--new-thread`.

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

1. **AI Parsing** — Analyzes novel text and extracts story structure, characters, and locations (SSE progress stream)
2. **Asset Generation** — Generates character images, character voices, and location images
3. **Shot Batch Generation** — Creates keyframes, audio, and video clips
4. **Episode Assembly** — Combines shots into polished episodes
5. **Download & Merge** — Downloads episode videos and merges into final drama.mp4 (optional)

**Output:**

Progress display (stderr):
```
Creating project "novel" on http://localhost:7001...
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

**Note:** This command uses Mastra directly (port 4111) instead of latentCut-server. For new projects, use `produce` instead.

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
- `api-key` — Legacy API key for Mastra server
- `server-url` — Mastra server URL (default: http://localhost:4111)
- `output-dir` — Output directory (default: novelo-output)
- `latentcut-url` — latentCut-server URL (default: http://localhost:7001)
- `token` — JWT token from login (set automatically by `login` command)
- `account` — Account name (set automatically by `login` command)

**Examples:**
```bash
novelo-cli config set api-key mykey123
novelo-cli config set server-url http://mastra.example.com:4111
novelo-cli config set output-dir ./my-videos
novelo-cli config set latentcut-url http://latentcut.example.com:7001
```

#### config get

Get a configuration value.

**Usage:**
```
novelo-cli config get <key>
```

**Examples:**
```bash
novelo-cli config get api-key
novelo-cli config get latentcut-url
novelo-cli config get account
```

Sensitive values (api-key, token) are masked in output.

#### config list

Display all configuration values.

**Usage:**
```
novelo-cli config list
```

**Output:**
```
api-key:       ****3k33
server-url:    http://localhost:4111
output-dir:    novelo-output
latentcut-url: http://localhost:7001
account:       user@example.com
token:         ****abcd

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
| `api_key` | Legacy Mastra API key | unset | `config set api-key` |
| `api_key_latentcut` | API key for latentCut-server (primary auth) | unset | `login` command, `config set api-key-latentcut` |
| `server_url` | Mastra server URL | http://localhost:4111 | `config set server-url` |
| `output_dir` | Output directory for videos | novelo-output | `config set output-dir`, `produce --output-dir` |
| `latentcut_url` | latentCut-server URL | http://localhost:7001 | `config set latentcut-url` |
| `token` | Legacy JWT token (backward compat) | unset | `login` command (older versions) |
| `account` | Logged-in account | unset | `login` command |
| `last_thread_id` | Cached chat thread ID | unset | `chat` command (auto-cached) |

### Example Config File

```yaml
api_key: xxxxxxxxxxxxxxxx
api_key_latentcut: nv-abc123...
server_url: http://localhost:4111
output_dir: novelo-output
latentcut_url: http://localhost:7001
account: user@example.com
last_thread_id: thread-abc123def456
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
# 1. Login
novelo-cli login --account user@example.com --password secret

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

### "connection refused" or "dial tcp"

latentCut-server is not running on the configured URL. Check:
```bash
novelo-cli config get latentcut-url
curl http://localhost:7001/health  # Test connectivity
```

If the server is on a different host:
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
    ↓ (HTTP + SSE)
latentCut-server (port 7001)
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
