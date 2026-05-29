# bl Architecture

> A terminal-based dictionary client written in Go. Supports Youdao (EN <-> ZH) and Netzverb (German) dictionary lookups with SQLite caching, offline dictionaries (kd-style), and persistent configuration. Multi-platform bot support (Telegram, DingTalk).

## Project Structure

```
bl/
├── main.go                    # CLI entry point
├── cmd/
│   ├── telegram/main.go       # Telegram bot (long polling)
│   └── dingtalk/main.go       # DingTalk bot (HTTP callback)
├── internal/
│   ├── config/
│   │   └── config.go          # Persistent configuration (JSON)
│   ├── dict/
│   │   ├── types.go           # Data structures, JSON codec, error types
│   │   ├── source.go          # DictionarySource interface
│   │   ├── offline.go         # OfflineDictionary (SQLite + zlib)
│   │   ├── youdao.go          # Youdao dictionary source
│   │   ├── german.go          # WoerterNet German dictionary source
│   │   └── rdict.go           # Engine: offline→cache→HTTP orchestration
│   ├── render/
│   │   └── render.go          # Output formatting (ANSI color + plain)
│   └── cache/
│       └── cache.go           # SQLite cache layer
├── scripts/
│   ├── build_dict/
│   │   └── main.go            # Offline dictionary builder (JSONL → SQLite)
│   └── testdata/
│       ├── de-en.jsonl        # Sample German→English word list
│       ├── en-zh.jsonl        # Sample English→Chinese word list
│       ├── zh-en.jsonl        # Sample Chinese→English word list
│       ├── de-en.db           # Sample built dictionary
│       ├── en-zh.db           # Sample built dictionary
│       └── zh-en.db           # Sample built dictionary
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
┌────────────────────────────────────────────────┐
│  Rdict.GetResults()                             │
│  internal/dict/rdict.go                        │
│                                                │
│  Query chain (in order):                       │
│                                                │
│  1. OfflineDictionary.Lookup(word)  ── hit ──►│
│     (if configured & --offline mode)           │
│         │ miss                                 │
│         ▼                                      │
│  2. cache.Get(key)              ── hit ──►    │
│     (if not --no-cache)                        │
│         │ miss                                 │
│         ▼                                      │
│  3. fetchSourceHTML()            ──► Parse ──►│
│     HTTP GET via net/http         │            │
│      source.FetchURL()            │            │
│      source.Parse()               │            │
│                                   ▼            │
│                              cache.Set()       │
└────────────────────────────────────────────────┘
         │ TranslationData
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

- **Data types** (`types.go`): All domain structs plus `TranslationData` (a tagged union with JSON format `{"type":"to_chinese","data":{...}}`), error types (`HttpError`, `NoTranslationResults`, `OfflineUnavailable`), and utility functions (`IsCJK`)
- **Interface** (`source.go`): `DictionarySource` — the single abstraction point for adding new dictionaries
- **Offline dictionary** (`offline.go`): `OfflineDictionary` — read-only SQLite databases with zlib-compressed TranslationData BLOBs. Supports three language pairs: `de-en`, `en-zh`, `zh-en`. Language selection by `LangForSource()` using source name + CJK detection.
- **Sources** (`youdao.go`, `german.go`): Two implementations of `DictionarySource`
- **Engine** (`rdict.go`): `Rdict` struct that wires offline + HTTP + cache together. `NewRdictWithOffline()` takes an optional `OfflineDictionary`. `GetResults()` follows the query chain: offline → cache → online.

### `internal/cache` — SQLite cache layer

Stores translation results as raw JSON strings. Uses `modernc.org/sqlite` (pure Go — no CGO). Communicates with `dict` via raw strings to avoid circular imports. The table schema is `cache_results(text TEXT PRIMARY KEY, data TEXT NOT NULL)`.

### `internal/config` — Persistent configuration

JSON-based config file stored at `~/.config/bl/config.json`:

```json
{"mode": "offline"}
```

Supported modes:
- `"auto"` — try offline first, fall back to online (default)
- `"offline"` — offline-only, error if word not in local dictionary
- `"online"` — skip offline dictionary, always fetch from network

Resolution priority: `CLI flag (--offline/--online)` > `BL_MODE env var` > `config file` > `"auto"`.

### `internal/render` — Output formatting

Three rendering functions (`RenderChinese`, `RenderEnglish`, `RenderGermanEntry`) plus a dispatcher (`RenderTranslation`). Each function takes a `colored bool` parameter: when true, emits ANSI escape codes; when false, emits plain text.

### `main.go` — CLI entry point

Three operating modes:

| Mode | Detection | Behavior |
|------|-----------|----------|
| Direct query | `flag.Args()` non-empty | Fetch and output immediately |
| Pipe mode | stdin is not a character device | Read stdin line, fetch and output |
| Interactive mode | No args + stdin is terminal | REPL loop with `[bl]# ` prompt |

Key flags:

| Flag | Purpose |
|------|---------|
| `--offline` | Offline-only mode (no network) |
| `--online` | Skip offline dictionary, force network |
| `--mode auto/offline/online` | Set & save default mode to config |
| `--update-dict` | Download offline dictionaries |
| `--dict-status` | Show installed dictionary info |
| `--generate-config` | Create default config file |

### `scripts/build_dict/` — Offline dictionary builder

Standalone Go CLI tool that converts JSONL word lists into SQLite offline dictionaries:

```bash
go run scripts/build_dict/ -lang de-en -input words.jsonl -output de-en.db
```

Input format (JSONL, one entry per line):
```jsonl
{"word":"Haus","definitions":["house","home"],"type":"german","gender":"neuter","article":"das"}
{"word":"hello","definitions":["你好"],"type":"to_chinese","pronunciation":{"uk":"həˈləʊ"}}
{"word":"你好","definitions":["hello","hi"],"type":"to_english"}
```

### `cmd/telegram/main.go` — Telegram bot

Uses long-polling via `telegram-bot-api/v5`. Responds to `/translate <text>` and `/help`. Environment config via `TELEGRAM_BOT_TOKEN` and `RDICT_SOURCE`.

## Dependencies

| Library | Purpose | Justification |
|---------|---------|--------------|
| `net/http` (stdlib) | HTTP client | Zero dependency, built-in |
| `encoding/json` (stdlib) | JSON serialization | Zero dependency |
| `flag` (stdlib) | CLI argument parsing | Zero dependency |
| `compress/zlib` (stdlib) | Offline dictionary BLOB compression | Zero dependency |
| `github.com/PuerkitoBio/goquery` | HTML parsing with CSS selectors | jQuery-like API for goquery-based HTML extraction |
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

# Offline mode (no network)
./bl --offline hello
./bl --offline -g Haus

# Set offline as default (saves to config)
bl --mode offline

# German dictionary
./bl -g Haus
./bl -s woerter-net Haus

# JSON output
./bl -j hello

# Single-line output
./bl -o hello

# Pipe mode
echo "world" | ./bl

# Interactive mode
./bl

# Dictionary management
./bl --dict-status
./bl --update-dict          # requires BL_DICT_URL
./bl --generate-config
```

## Offline Dictionary Storage Format

Each offline dictionary is a SQLite database with a single table:

```sql
CREATE TABLE IF NOT EXISTS entries (
    query TEXT NOT NULL PRIMARY KEY,
    data  BLOB NOT NULL
) WITHOUT ROWID;
```

The `data` column stores zlib-compressed JSON. The JSON format matches the online `TranslationData` format:

```json
{"type":"german","data":{"word":"Haus","definitions":["house","home"],"gender":"neuter","article":"das"}}
```

**Compression**: `compress/zlib` (stdlib). No external compression library needed.
**Read mode**: `PRAGMA query_only = 1` — the database is never written at runtime.

## Configuration File Format

```json
{
  "mode": "offline"
}
```

File location: `~/.config/bl/config.json`

| Mode | Behavior |
|------|----------|
| `"auto"` | Try offline dict → cache → online (default) |
| `"offline"` | Offline only, error if not found |
| `"online"` | Skip offline dict, always fetch online |

## JSON Cache Format

The cache stores `TranslationData` in a tagged JSON format:

```json
{"type":"to_chinese","data":{"input_text":"hello","pronunciation":{...},"meanings":[...],"examples":[...]}}
{"type":"to_english","data":{"input_text":"你好","meanings":[...],"examples":[...]}}
{"type":"german","data":{"word":"Haus","definitions":[...],"examples":[...]}}
```

This is produced/consumed by `TranslationData.MarshalJSON()` / `TranslationData.UnmarshalJSON()` in `types.go`.


