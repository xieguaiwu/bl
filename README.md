# bl — Terminal Dictionary Client

**English** | [中文](README.zh-CN.md)

A terminal-based dictionary client written in Go. Supports **Youdao** (EN ⇄ ZH) and **WoerterNet** (German) dictionary lookups with SQLite caching, offline dictionaries, and persistent configuration. Multi-platform bot support (Telegram, DingTalk).

> This project was inspired by [rdict](https://github.com/Guanran928/rdict) — a Rust-based terminal dictionary client. Thanks to the original author for the inspiration.

## Install via COPR (Fedora)

```bash
sudo dnf install dnf-plugins-core
sudo dnf copr enable xieguaiwu/bl
sudo dnf install bl
```

## Quick Start

```bash
go build -o bl . && ./bl hello
```

## Features

- **Multiple sources**: Youdao (EN⇄ZH default), WoerterNet (German via `-g`)
- **Three query modes**: `auto` (offline → cache → online), `offline`, `online`
- **Offline dictionaries**: Pre-built SQLite databases with zlib compression
- **SQLite cache**: Auto-caches results, skippable with `--no-cache`
- **Output formats**: Markdown (default), JSON (`-j`), oneliner (`-o`)
- **ANSI color**: Auto-detected, respects `NO_COLOR`
- **Interactive mode**: REPL prompt with `[bl]#`
- **Bot platforms**: Telegram bot + DingTalk webhook
- **LLM translation**: AI-powered translation between any languages via OpenAI-compatible APIs

## LLM Translation (`--llm`)

Translate any text between any languages using large language models — no more web scraping, works for any language pair.

```bash
# Default LLM translation (configured in ~/.config/bl/config.json)
bl --llm hello

# Specify target language
bl --llm --to-lang "日本語" "good morning"
bl --llm --to-lang "Français" "hello"

# Choose a different provider/model
bl --llm --llm-provider openrouter --llm-model "google/gemma-4-31b-it:free" hello
bl --llm --llm-model nemotron-3-ultra-free hello

# Inline API key (overrides config)
bl --llm --llm-key "$OPENROUTER_API_KEY" hello
```

### Provider Presets

| Provider | Endpoint | Default Model | Free Tier |
|----------|----------|---------------|-----------|
| `openrouter` | `https://openrouter.ai/api/v1` | `qwen/qwen3-next-80b-a3b-instruct:free` | 27+ free models
| `opencode-zen` | `https://opencode.ai/zen/v1` | `deepseek-v4-flash-free` | 5 free models
| `nemotron` | `https://integrate.api.nvidia.com/v1` | `nvidia/nemotron-3-ultra-550b-a55b` | Requires NVIDIA API key
| `custom` | (user-defined) | (user-defined) | Any OpenAI-compatible endpoint

### Rich Linguistic Info

The LLM response includes grammatical details when relevant:
- **Part of speech**: noun/verb/adjective/etc.
- **Gender**: masculine/feminine/neuter (for inflected languages)
- **Plural form**: for countable nouns
- **Comparative / Superlative**: for adjectives/adverbs
- **Pronunciation**: phonetic or pinyin
- **5 vivid example sentences**: concrete scenes with actions and imagery

### Local Config (`.blrc`)

Create a `.blrc` file in your project directory for quick provider/model switching without CLI flags:

```bash
# Reference a named provider from global config
echo '{"provider":"openrouter","model":"google/gemma-4-31b-it:free","target_lang":"Français"}' > .blrc

# Ad-hoc provider (no global config needed)
echo '{"base_url":"https://openrouter.ai/api/v1","model":"google/gemma-4-31b-it:free","api_key":"env:OPENROUTER_API_KEY","target_lang":"日本語"}' > .blrc
```

Priority: `CLI flags` > `.blrc` > `global config`

### Configuration Example

Copy and customize:

```bash
cp config.example.json ~/.config/bl/config.json
# Then edit ~/.config/bl/config.json to set your API keys
```

The config supports multiple providers. Set `api_key` to `env:VAR_NAME` to reference environment variables (`OPENROUTER_API_KEY`, `OPENCODE_API_KEY`, `NVIDIA_API_KEY`).

### Source Language (`--from-lang`)

For words shared between languages (e.g. "Raisonnement" in French vs German, "Handy" in German vs English), specify the source language explicitly:

```bash
bl --llm --from-lang French Raisonnement     # French "reasoning" → 推理
bl --llm --from-lang German Raisonnement     # German "reasoning" → 论证
bl --llm --from-lang German --to-lang English Handy   # German "Handy" → mobile phone
```

When `--from-lang` is omitted, the model auto-detects the source language.

### API Keys

Set these in your shell profile (`~/.bashrc`, `~/.config/fish/config.fish`, etc.):

```bash
export OPENROUTER_API_KEY="sk-or-v1-your-key"
export OPENCODE_API_KEY="sk-your-key"
export NVIDIA_API_KEY="nvapi-your-key"
```

## Usage

```bash
# Default Youdao EN⇄ZH
bl hello

# German dictionary
bl -g Haus
bl -s woerter-net laufen

# JSON output
bl -j hello

# Single-line output
bl -o hello

# Offline mode (no network)
bl --offline hello
bl --offline -g Haus

# Set default mode (persisted to config)
bl --mode offline

# Pipe mode
echo "world" | bl

# Interactive mode
bl

# Dictionary management
bl --dict-status
bl --update-dict          # requires BL_DICT_URL
bl --generate-config
```

## Mode Resolution

Priority: `CLI flag (--offline/--online)` > `BL_MODE env var` > `config file` > `default "auto"`

| Mode     | Behavior                                       |
|----------|------------------------------------------------|
| `auto`   | Try offline dictionary first, fall back to online |
| `offline`| Offline only, error if word not in local dict  |
| `online` | Skip offline dictionary, always fetch from network |

## Build

```bash
# CLI
go build -o bl .

# Bot binaries
go build -o bl-telegram ./cmd/telegram/
go build -o bl-dingtalk ./cmd/dingtalk/

# All
go build ./...
go vet ./...
```

## Project Structure

```
bl/
├── main.go                 # CLI entry point
├── cmd/
│   ├── telegram/main.go    # Telegram bot (long polling)
│   └── dingtalk/main.go    # DingTalk bot (HTTP callback)
├── internal/
│   ├── config/config.go    # Persistent JSON config
│   ├── dict/               # Core: types, sources, offline, engine
│   ├── render/render.go    # Output formatting
│   └── cache/cache.go      # SQLite query cache
├── scripts/build_dict/     # Offline dictionary builder
├── doc/                    # Architecture & AI guide
└── scripts/testdata/       # Sample dictionaries
```

## Dependencies

| Library | Purpose |
|---------|---------|
| `modernc.org/sqlite` | Pure-Go SQLite (no CGO) |
| `github.com/PuerkitoBio/goquery` | HTML parsing with CSS selectors |
| `github.com/go-telegram-bot-api/telegram-bot-api/v5` | Telegram Bot API |

## Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `BL_MODE` | Override default query mode | `BL_MODE=offline` |
| `BL_DICT_URL` | Base URL for `--update-dict` | `BL_DICT_URL=https://example.com/dicts` |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token (required for bl-telegram) | `TELEGRAM_BOT_TOKEN=xxx` |
| `RDICT_SOURCE` | Dictionary source for Telegram bot | `RDICT_SOURCE=woerter-net` |
