# bl Architecture

> A terminal-based dictionary client written in Go. Supports Youdao (EN <-> ZH) and Netzverb (German) dictionary lookups with SQLite caching. Multi-platform bot support (Telegram, DingTalk).

## Project Structure

```
bl/
├── main.go                    # CLI entry point
├── cmd/
│   ├── telegram/main.go       # Telegram bot (long polling)
│   └── dingtalk/main.go       # DingTalk bot (HTTP callback)
├── internal/
│   ├── dict/
│   │   ├── types.go           # Data structures, JSON codec, error types
│   │   ├── source.go          # DictionarySource interface
│   │   ├── youdao.go          # Youdao dictionary source
│   │   ├── german.go          # WoerterNet German dictionary source
│   │   └── rdict.go           # Engine: HTTP fetch + cache orchestration
│   ├── render/
│   │   └── render.go          # Output formatting (ANSI color + plain)
│   └── cache/
│       └── cache.go           # SQLite cache layer
└── doc/
    ├── ARCHITECTURE.md        # This file
    ├── AI_GUIDE.md            # Guide for AI agents
    └── BOT_PLATFORMS.md       # Multi-platform bot analysis
```

## Data Flow

```
User Input
    │
    ▼
┌─────────────────────┐
│      main.go         │  CLI: arg / pipe / interactive mode
│  / cmd/telegram/    │  Telegram: /translate command
│  / cmd/dingtalk/    │  DingTalk: @bot mention
└────────┬────────────┘
         │ text
         ▼
┌─────────────────────┐
│  Rdict.GetResults() │  internal/dict/rdict.go
└────────┬────────────┘
    ├── cache.Get(text) ────── cache hit ──► return cached result
    │     (miss)
    ▼
┌─────────────────────┐
│ fetchSourceHTML()    │  HTTP GET via net/http
│  source.FetchURL()   │  URL construction by source
└────────┬────────────┘
         │ raw HTML
         ▼
┌─────────────────────┐
│ source.Parse()       │  goquery CSS selector parsing
│  → TranslationData   │  returns tagged variant
└────────┬────────────┘
    │
    ▼
┌─────────────────────┐
│ cache.Set()          │  store as JSON
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│ render.Render*()     │  ANSI colored / plain text output
│      OR              │
│ json.Marshal()       │  raw JSON output (--json flag)
└─────────────────────┘
```

## Package Design

### `internal/dict` — Core types and dictionary sources

This is the central package. It contains:

- **Data types** (`types.go`): All domain structs plus `TranslationData` (a tagged union matching Rust serde's `{"type":"to_chinese","data":{...}}` format), error types (`HttpError`, `NoTranslationResults`), and utility functions (`IsCJK`)
- **Interface** (`source.go`): `DictionarySource` — the single abstraction point for adding new dictionaries
- **Sources** (`youdao.go`, `german.go`): Two implementations of `DictionarySource`
- **Engine** (`rdict.go`): `Rdict` struct that wires source + HTTP + cache together

### `internal/cache` — SQLite cache layer

Stores translation results as raw JSON strings. Uses `modernc.org/sqlite` (pure Go — no CGO). Communicates with `dict` via raw strings to avoid circular imports. The table schema is `cache_results(text TEXT PRIMARY KEY, data TEXT NOT NULL)` — identical to the Rust original.

### `internal/render` — Output formatting

Three rendering functions (`RenderChinese`, `RenderEnglish`, `RenderGermanEntry`) plus a dispatcher (`RenderTranslation`). Each function takes a `colored bool` parameter: when true, emits ANSI escape codes; when false, emits plain text.

### `main.go` — CLI entry point

Three operating modes:

| Mode | Detection | Behavior |
|------|-----------|----------|
| Direct query | `flag.Args()` non-empty | Fetch and output immediately |
| Pipe mode | stdin is not a character device | Read stdin line, fetch and output |
| Interactive mode | No args + stdin is terminal | REPL loop with `[bl]# ` prompt |

### `cmd/telegram/main.go` — Telegram bot

Uses long-polling via `telegram-bot-api/v5`. Responds to `/translate <text>` and `/help`. Environment config via `TELEGRAM_BOT_TOKEN` and `RDICT_SOURCE`.

## Dependencies

| Library | Purpose | Justification |
|---------|---------|--------------|
| `net/http` (stdlib) | HTTP client | Zero dependency, built-in |
| `encoding/json` (stdlib) | JSON serialization | Zero dependency |
| `flag` (stdlib) | CLI argument parsing | Zero dependency |
| `github.com/PuerkitoBio/goquery` | HTML parsing with CSS selectors | jQuery-like API, exactly replaces Rust's `scraper` crate |
| `modernc.org/sqlite` | SQLite database | Pure Go implementation, no CGO required |
| `github.com/go-telegram-bot-api/telegram-bot-api/v5` | Telegram Bot API | Most maintained Go Telegram library |

```bash
go build -o bl .                        # CLI
go build -o bl-telegram ./cmd/telegram/  # Telegram bot
go build -o bl-dingtalk ./cmd/dingtalk/  # DingTalk bot
```

All produce single static binaries with no system dependencies.

Usage:
```bash
# Youdao EN<->ZH (default)
./bl hello

# German dictionary
./bl -g Haus
./bl -s woerter-net Haus

# JSON output
./bl -j hello

# Pipe mode
echo "world" | ./bl

# Interactive mode
./bl
```

## JSON Cache Format

The cache stores `TranslationData` in a tagged JSON format compatible with Rust serde:

```json
{"type":"to_chinese","data":{"input_text":"hello","pronunciation":{...},"meanings":[...],"examples":[...]}}
{"type":"to_english","data":{"input_text":"你好","meanings":[...],"examples":[...]}}
{"type":"german","data":{"word":"Haus","definitions":[...],"examples":[...]}}
```

This is produced/consumed by `TranslationData.MarshalJSON()` / `TranslationData.UnmarshalJSON()` in `types.go`.

## Cross-Reference: Rust Original

This Go rewrite (named **bl**) maps directly from the Rust project at https://github.com/Guanran928/rdict:

| Rust Crate | Go Package | Key Difference |
|-----------|-----------|----------------|
| `rdict-core/src/parse.rs` | `internal/dict/types.go` | Rust `enum` → Go tagged struct with custom JSON |
| `rdict-core/src/source.rs` | `internal/dict/source.go` | Rust `trait` → Go `interface` |
| `rdict-core/src/youdao.rs` | `internal/dict/youdao.go` | Rust `scraper` CSS selectors → Go `goquery` CSS selectors |
| `rdict-core/src/german.rs` | `internal/dict/german.go` | Same logic, different syntax |
| `rdict-core/src/rdict.rs` | `internal/dict/rdict.go` | Rust async `reqwest` + `sqlx` → Go sync `net/http` + `database/sql` |
| `rdict-core/src/lib.rs` | `internal/dict/types.go` | Rust `thiserror` enum → Go error types |
| `rdict-cli/src/main.rs` | `main.go` | Rust `clap` + `rustyline` → Go `flag` + `bufio.Scanner` |
| `rdict-telegram/src/main.rs` | `cmd/telegram/main.go` | Rust `teloxide` → Go `telegram-bot-api/v5` |
