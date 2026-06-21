# bl Project — Architecture Understanding

## Overview

**bl** is a terminal-based dictionary client written in Go (module `bl`, Go 1.25). It supports Youdao (EN⇄ZH) and WoerterNet (German) dictionary lookups with a three-tier query chain: offline dictionary → SQLite cache → online HTTP fetch. Additionally, it has Telegram and DingTalk bot adapters.

---

## Project Structure

```
bl/
├── main.go                       # CLI: flag parsing, pipe mode, interactive REPL
├── cmd/
│   ├── telegram/main.go          # Telegram bot (long-polling via tgbotapi)
│   └── dingtalk/main.go          # DingTalk bot (HTTP callback, 0 new deps)
├── internal/
│   ├── config/config.go          # JSON config (~/.config/bl/config.json)
│   ├── dict/                     # Core dictionary logic
│   │   ├── types.go              # TranslationData (tagged union), error types
│   │   ├── source.go             # DictionarySource interface + factory
│   │   ├── rdict.go              # Engine: offline→cache→HTTP orchestration
│   │   ├── offline.go            # Offline SQLite+zlib dictionaries (de-en/en-zh/zh-en)
│   │   ├── youdao.go             # Youdao EN⇄ZH source (goquery-based)
│   │   └── german.go             # WoerterNet German source (goquery + IPA)
│   └── render/render.go          # Output: Markdown/JSON/oneliner + ANSI color
├── scripts/
│   └── build_dict/main.go        # JSONL → SQLite offline dict builder
├── doc/
│   ├── ARCHITECTURE.md           # Full architecture & data flow
│   ├── AI_GUIDE.md               # Agent guide: interfaces, extensions, pitfalls
│   └── BOT_PLATFORMS.md          # Multi-platform bot analysis & roadmap
└── .sisyphus/                    # AI workflow orchestration (Sisyphus loop)
```

---

## Architecture & Data Flow

### Query Resolution Chain (3-tier)

```
User Input
    │
    ▼
┌──────────────────────────────────────────┐
│  Rdict.GetResults()                      │
│                                          │
│  1. OfflineDictionary.Lookup(word)       │  ← SQLite + zlib, fastest
│     (if configured)                      │
│         │ miss                           │
│         ▼                                │
│  2. Cache.Get(key)                       │  ← auto-populated from prior queries
│     (if not --no-cache)                  │
│         │ miss                           │
│         ▼                                │
│  3. HTTP GET → Parse → Cache.Set()       │  ← network fetch, goquery parse
│                                          │
└──────────────────────────────────────────┘
         │ TranslationData
         ▼
┌─────────────────────┐
│ RenderTranslation()  │  Markdown/JSON/oneliner + ANSI
└─────────────────────┘
```

### Mode Resolution Priority

`CLI flag (--offline/--online)` > `BL_MODE env var` > `config file` > `"auto"` default

| Mode | Behavior |
|------|----------|
| `auto` | Try offline → cache → online |
| `offline` | Offline only, error if not in local dict |
| `online` | Skip offline dict, always online |

---

## Core Type System

### `DictionarySource` Interface — Extension Point #1

```go
type DictionarySource interface {
    Name() string
    FetchURL(word string) string
    Parse(word string, html string) (*TranslationData, error)
}
```

Two implementations:
- **`YoudaoSource`** (`youdao.go`): m.youdao.com, EN⇄ZH
- **`WoerterNetSource`** (`german.go`): verbformen.com, German→English with IPA, CEFR level, gender, word type

**To add a new source:** Create file implementing this interface → register in `NewSourceByName()` (`source.go`).

### `TranslationData` — Tagged Union

```go
type TranslationData struct {
    Type      TranslationType    // TypeToChinese | TypeToEnglish | TypeGerman
    ToChinese *ToChinese
    ToEnglish *ToEnglish
    German    *GermanEntry
}
```

**Invariant:** Exactly one pointer is non-nil, matching `Type`. JSON format: `{"type":"german","data":{...}}`.

### Error Types

| Type | When |
|------|------|
| `HttpError` | HTTP status != 200 |
| `NoTranslationResults` | Parse succeeded but no data |
| `OfflineUnavailable` | `--offline` mode + word not in offline dict |

---

## Offline Dictionary System

### Storage Format

SQLite database with a single table:
```sql
CREATE TABLE entries (
    query TEXT NOT NULL PRIMARY KEY,
    data  BLOB NOT NULL
) WITHOUT ROWID;
```

- `data` stores **zlib-compressed** JSON (`TranslationData` in tagged union format)
- Read-only at runtime (`PRAGMA query_only = 1`)
- Default location: `~/.config/bl/dict/{lang}.db`

### Language Pairs

Currently supported: `de-en` (German→English), `en-zh` (English→Chinese), `zh-en` (Chinese→English)

Mapping logic in `LangForSource()`:
- `woerter-net` source → always `"de-en"`
- default source + CJK text → `"zh-en"`
- default source + non-CJK → `"en-zh"`

### Builder Tool

`scripts/build_dict/main.go` converts JSONL → SQLite offline dictionary:
```bash
go run scripts/build_dict/ -lang de-en -input words.jsonl -output ~/.config/bl/dict/de-en.db
```

Each JSONL line is one word entry with all metadata (word, definitions, gender, article, phonetic, CEFR level, word type, examples).

---

## Offline Dictionary Status — Current Gap

**There is no German offline dictionary (`de-en.db`) installed yet.**

The codebase has:
- ✅ Offline dictionary infrastructure (`offline.go`): SQLite read/write, zlib compression, schema
- ✅ Builder tool (`scripts/build_dict/main.go`): JSONL→SQLite conversion
- ✅ Integration in `LangForSource()`: `"woerter-net"` → `"de-en"`
- ✅ Integration in `updateDictCmd()`: downloads `de-en.db` from `BL_DICT_URL`
- ✅ Integration in `dictStatusCmd()`: checks for `de-en.db`
- ✅ Sample testdata: `scripts/testdata/de-en.jsonl` + `scripts/testdata/de-en.db`
- ❌ **No pre-built downloadable `de-en.db`** hosted at a `BL_DICT_URL`
- ❌ **No comprehensive German word list** as JSONL input for the builder

The German online source (WoerterNet via verbformen.com) works via `german.go`, but an offline dictionary would require either:
1. Building `de-en.db` from a JSONL word list using `scripts/build_dict/main.go`
2. Hosting it at a URL and downloading via `bl --update-dict` with `BL_DICT_URL` set

---

## Bot Platform Architecture

| Platform | Status | Connection | Dependencies |
|----------|--------|-----------|--------------|
| Telegram | ✅ Done | Long polling | `tgbotapi/v5` |
| DingTalk | ✅ Done | HTTP callback | None (pure `net/http`) |

All bots share the same core (`Rdict` engine + render). A proposed `internal/botengine/` abstraction (documented in `BOT_PLATFORMS.md`) would reduce adapter duplication but is not yet implemented.

---

## Key Architectural Invariants

1. **No circular import between `dict` and `cache`**: Cache stores raw JSON strings, not `*TranslationData`
2. **Offline dict is read-only at runtime**: No writes; separate from the read-write query cache
3. **`goquery` for all HTML parsing**: CSS selectors; always check `.Length() > 0` before accessing `.Text()`
4. **`modernc.org/sqlite` (pure Go, no CGO)**: Adds ~10MB binary size
5. **`Rdict` is not goroutine-safe**: `http.Client` is safe but `DictionarySource.Parse` may have state

---

## Sisyphus Loop Context

The `.sisyphus/ralph-loop.local.md` file records this session as an **ultrawork loop** (iteration 1, strategy: continue). The loop mechanism tracks work until verified completion via Oracle. This architectural understanding document (`session_context.md`) is the output of the "exploration + documentation" phase.
