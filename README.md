# bl — Terminal Dictionary Client

**English** | [中文](README.zh-CN.md)

A terminal-based dictionary client written in Go. Supports **Youdao** (EN ⇄ ZH) and **WoerterNet** (German) dictionary lookups with SQLite caching, offline dictionaries, and persistent configuration. Multi-platform bot support (Telegram, DingTalk).

> This project was inspired by [rdict](https://github.com/Guanran928/rdict) — a Rust-based terminal dictionary client. Thanks to the original author for the inspiration.

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
