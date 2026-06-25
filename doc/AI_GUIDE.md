# AI Agent Guide: bl

> This document is written for AI agents (LLMs) that need to understand, modify, or extend the **bl** codebase. It covers interfaces, extension points, invariants, and common pitfalls.

## 1. Interface Contracts

### 1.1 `DictionarySource` (internal/dict/source.go)

```go
type DictionarySource interface {
    Name() string
    FetchURL(word string) string
    Parse(word string, html string) (*TranslationData, error)
}
```

**Contract:**

- `Name()`: Returns a unique, URL-safe identifier (e.g. `"youdao"`, `"woerter-net"`). Used for error messages and source selection. Must be lowercase, no spaces.
- `FetchURL(word)`: Returns the complete URL to fetch. The word should be URL-encoded if it contains special characters. **Must not** make HTTP requests itself — the engine handles networking.
- `Parse(word, html)`: Parses raw HTML from the source website. Returns `*TranslationData` or an error. **Must** return `*NoTranslationResults` when the parse yields no useful data (distinguishing "no results" from "parse failure").

**Adding a new source:**

1. Create a new file `internal/dict/foo.go`
2. Define a struct implementing `DictionarySource`
3. Wire it in `internal/dict/source.go` (the `NewSourceByName` function). The bot binaries (`cmd/telegram/`, `cmd/dingtalk/`) automatically pick up the new source through the factory — no per-bot registration needed.
4. If the result format differs from existing variants, add a new variant to `TranslationType` and `TranslationData` in `types.go`
5. Add a render function in `internal/render/render.go`
6. Register the dispatch in `RenderTranslation()`

### 1.2 `OfflineDictionary` (internal/dict/offline.go)

```go
type OfflineDictionary struct { /* unexported fields */ }

func NewOfflineDict(dbDir, langPair string) (*OfflineDictionary, error)
func (o *OfflineDictionary) Close() error
func (o *OfflineDictionary) Lookup(word string) (*TranslationData, bool)
func (o *OfflineDictionary) Stats() (entries int, size int64, err error)
```

**Contract:**

- `NewOfflineDict(dbDir, langPair)`: Opens an offline dictionary database for the given language pair (`"de-en"`, `"en-zh"`, `"zh-en"`). Returns `(nil, nil)` **without error** if the database file does not exist (callers should treat this as "offline unavailable", not a failure).
- `Lookup(word)`: Searches the offline dictionary. Reads a zlib-compressed JSON blob from SQLite, decompresses, unmarshals into `TranslationData`. Returns `(nil, false)` if the word is not found.
- `Stats()`: Returns entry count and file size. Useful for `--dict-status`.
- The database is opened in **read-only mode** (`PRAGMA query_only = 1`). No writes happen at runtime.
- To create offline dictionaries, use `dict.CreateOfflineDict()` or the standalone `scripts/build_dict/` tool.

**Language pair selection** is handled by `LangForSource()`:

```go
func LangForSource(sourceName, text string) string
```

| Source | Query contains CJK | Result |
|--------|--------------------|--------|
| `woerter-net` | — | `"de-en"` |
| default | yes | `"zh-en"` |
| default | no | `"en-zh"` |

**Adding a new offline dictionary language pair:**

1. Create the dictionary database file named `{lang}.db` (e.g. `fr-en.db`)
2. Add a new case in `LangForSource()` to select the pair based on source name or query content
3. The database must have the schema: `CREATE TABLE IF NOT EXISTS entries (query TEXT PRIMARY KEY, data BLOB) WITHOUT ROWID`
4. Each row stores zlib-compressed JSON: `encode(TranslationData)` → zlib compress → `data BLOB`

### 1.3 `Rdict` (internal/dict/rdict.go)

```go
type Rdict struct { /* unexported fields */ }

func NewRdict(source DictionarySource, cacheDB string) (*Rdict, error)
func NewRdictWithOffline(source DictionarySource, cacheDB string, offlineSource *OfflineDictionary, onlyOffline bool) (*Rdict, error)
func (r *Rdict) Close() error
func (r *Rdict) GetResults(inputText string) (*FetchedResult, error)
```

**Contracts:**

- `NewRdict(source, cacheDB)`: Online-only. If `cacheDB` is empty string, caching is disabled. The cache directory is auto-created. Returns error only if cache initialization fails (not if cache is disabled).
- `NewRdictWithOffline(source, cacheDB, offlineSource, onlyOffline)`: If `offlineSource` is nil, behaves identically to `NewRdict`. If `onlyOffline` is true, `GetResults` returns `*OfflineUnavailable` when the word is not in the offline dictionary, without attempting an online fetch.
- `GetResults(inputText)`: **Query chain** (in order):
  1. **Offline dictionary lookup** (if configured) — fastest path, no network
  2. **Cache check** — returns cached result if available
  3. **Online fetch** — HTTP GET → Parse → cache store → return

  **Always returns a non-nil `*FetchedResult` on success.** The caller checks `result.IsCached`.
- **Thread safety**: `Rdict` is **not** safe for concurrent use (the embedded `http.Client` is safe, but the `DictionarySource.Parse` may have state). Use `sync.Mutex` if concurrent access is needed.
- If `Rdict` is created with `NewRdictWithOffline` and `onlyOffline=true`, the HTTP client is never used (no network requests).

### 1.4 `Cache` (internal/cache/cache.go)

```go
type Cache struct { /* unexported fields */ }

func New(dbPath string) (*Cache, error)
func (c *Cache) Close() error
func (c *Cache) Get(text string) (string, error)
func (c *Cache) Set(text string, jsonData string) error
func (c *Cache) Delete(text string) error
```

**Contract:**

- `Get(text)`: Returns `("", nil)` on cache miss — **not** an error. Errors are reserved for I/O/database failures.
- `Set(text, jsonData)`: Upserts by primary key (`INSERT OR REPLACE`). The `jsonData` parameter must be valid JSON (the caller is responsible for marshaling).
- The cache stores raw strings to avoid circular imports with `internal/dict`. The caller (`Rdict`) handles JSON marshaling/unmarshaling.
- `modernc.org/sqlite` implements `database/sql` interface. Connection uses WAL journal mode.

### 1.5 `Config` (internal/config/config.go)

```go
type Config struct {
    Mode Mode   // "auto" | "offline" | "online"
}

func Load() (*Config, error)
func Save(cfg *Config) error
func ConfigPath() (string, error)
func GenerateConfig() (bool, error)
func DefaultConfig() *Config
```

**Contract:**

- `Load()`: Reads `~/.config/bl/config.json`. Returns `DefaultConfig()` (mode `"auto"`) if the file does not exist.
- `Save()`: Writes config to `~/.config/bl/config.json`. Creates the directory if needed.
- `GenerateConfig()`: Creates a default config file only if one does not already exist. Returns `(true, nil)` on new creation.
- **Priority chain** for mode resolution: `CLI flag (--offline/--online)` > `BL_MODE env var` > `config file` > `default "auto"`.

## 2. Type System Invariants

### 2.1 `TranslationData` — Tagged Union (types.go)

```go
type TranslationData struct {
    Type      TranslationType    // TypeToChinese (0), TypeToEnglish (1), TypeGerman (2)
    ToChinese *ToChinese         // non-nil only when Type==TypeToChinese
    ToEnglish *ToEnglish         // non-nil only when Type==TypeToEnglish
    German    *GermanEntry       // non-nil only when Type==TypeGerman
}
```

**Critical invariant:** Exactly one pointer is non-nil at any time, matching `Type`. Code must not assume that a non-nil pointer implies a specific `Type`. Use type switches or `switch data.Type` for dispatch.

**JSON format** (tagged union):

```json
{"type":"to_chinese","data":{...}}
{"type":"to_english","data":{...}}
{"type":"german","data":{...}}
```

### 2.2 Error Types

| Type | When | Fields |
|------|------|--------|
| `HttpError` | HTTP status != 200 | `Code`, `Source`, `Word` |
| `NoTranslationResults` | Parse succeeded but no data | `word` (unexported) |
| `OfflineUnavailable` | `--offline` mode + word not in offline dict | `word` (unexported) |

All implement the `error` interface. `NoTranslationResults` is distinct from a regular `error` — callers may want to handle it differently (e.g. "no results found" vs "something went wrong").

### 2.3 Optional Fields

In the structs, empty string (`""`) means "absent". There are no pointer-to-string fields. When adding fields, prefer zero-value semantics over pointers for simplicity.

## 3. HTML Parsing Rules

### 3.1 goquery API

All HTML parsing uses `github.com/PuerkitoBio/goquery`. The key API:

```go
doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))

// Find by CSS selector (multiple)
doc.Find(".classname").Each(func(i int, s *goquery.Selection) { ... })

// Find first
doc.Find(".classname").First()

// Find within a selection
selection.Find(".child-class")

// Get text content (includes all descendant text)
selection.Text()

// Get attribute value
href, exists := selection.Attr("href")

// Get HTML content
innerHtml, err := selection.Html()
```

**IMPORTANT:** `goquery` parses malformed HTML without error. Always check `.Length() > 0` or `.First().Length() > 0` before accessing `.Text()`. Never assume a selector will match.

### 3.2 IPA Extraction (german.go)

The `extractIPA()` function uses a custom scanner rather than goquery because IPA notation appears as bare `/fənetɪk/` text in the page body, not inside specific elements. It:

1. Scans the first 3000 characters of body text
2. Finds patterns between forward slashes (`/.../`)
3. Validates the candidate contains IPA-specific Unicode characters

**If the IPA regex misses characters**, add them to the `containsIPASymbols()` function. The current set covers Latin IPA extensions and common diacritics.

### 3.3 German Word Type Guessing (german.go)

`guessWordType()` uses suffix heuristics to determine the URL path:

| Suffix Examples | Guessed Type | URL Path |
|----------------|-------------|----------|
| `-ieren`, `-eln`, `-ern`, `-en` | verb | `conjugation/{word}.htm` |
| `-lich`, `-isch`, `-ig`, `-bar`, `-sam`, `-los`, `-haft` | adjective | `declension/adjectives/{word}.htm` |
| anything else | noun | `declension/nouns/{word}.htm` |

This is an approximation. Some German verbs match the noun suffix pattern (and vice versa). The page `<title>` serves as the authoritative source.

## 4. Configuration & Environment

### 4.1 CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-g` / `--german` | bool | `false` | Use German dictionary (woerter-net) |
| `-s` / `--source` | string | `"youdao"` | Dictionary source (`youdao`, `woerter-net`) |
| `-j` / `--json` | bool | `false` | Output raw JSON instead of formatted text |
| `-o` / `--oneliner` | bool | `false` | Single-line compact output |
| `--no-cache` | bool | `false` | Disable SQLite caching |
| `--offline` | bool | `false` | Offline-only mode (no network) |
| `--online` | bool | `false` | Skip offline dictionary, force network |
| `--mode` | string | `""` | Set & save default mode (`auto`/`offline`/`online`) |
| `--generate-config` | bool | `false` | Create default config file |
| `--update-dict` | bool | `false` | Download offline dictionaries |
| `--dict-status` | bool | `false` | Show offline dictionary info |
| positional | string[] | (empty) | Word(s) to translate |

### 4.2 Config File

Configuration is stored in `~/.config/bl/config.json`:

```json
{
  "mode": "offline"
}
```

Supported modes:

| Mode | Behavior |
|------|----------|
| `"auto"` | Try offline dictionary first, fall back to online (default) |
| `"offline"` | Offline-only, error if word not in local dictionary |
| `"online"` | Skip offline dictionary, always fetch from network |

Resolution order: `CLI flag (--offline/--online)` > `BL_MODE env var` > `config file mode` > `"auto"` default.

### 4.3 Cache Path

Cache database location (when not `--no-cache`):

- Linux: `~/.cache/bl/cache.db`
- macOS: `~/Library/Caches/bl/cache.db`
- Windows: `%LOCALAPPDATA%\bl\cache.db`

Uses `os.UserCacheDir()` per XDG spec.

### 4.4 Offline Dictionary Path

Offline dictionary databases are stored in `~/.config/bl/dict/`:

- `~/.config/bl/dict/de-en.db` — German→English
- `~/.config/bl/dict/en-zh.db` — English→Chinese
- `~/.config/bl/dict/zh-en.db` — Chinese→English

Dictionary files are **read-only** SQLite databases created by `scripts/build_dict/` or downloaded via `bl --update-dict` (requires `BL_DICT_URL` env var).

### 4.5 Color Detection

ANSI color is enabled when `TERM` is set and not `"dumb"`, and `NO_COLOR` is not set (per https://no-color.org).

### 4.6 Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `BL_MODE` | Override default query mode | `BL_MODE=offline` |
| `BL_DICT_URL` | Base URL for `--update-dict` | `BL_DICT_URL=https://example.com/dicts` |

## 5. Common Pitfalls for AI Agents

### 5.1 Import Cycle: `dict` ↔ `cache`

**`internal/cache` must not import `internal/dict`.** The cache stores raw JSON strings (`string`) instead of `*TranslationData` to avoid a circular import. All JSON marshaling/unmarshaling happens in `internal/dict/rdict.go`.

If you need to add a new type to the cache, **do not** import `internal/dict` from `internal/cache`. Either:
- Keep it string-based and marshal in the caller
- Or extract the shared type into a third package

### 5.2 Offline Dictionary vs Cache: Separate Concerns

The offline dictionary (`internal/dict/offline.go`) and the query cache (`internal/cache/cache.go`) serve different purposes:

| Aspect | Offline Dictionary | Query Cache |
|--------|--------------------|-------------|
| **Source** | Pre-built data files | Auto-populated from queries |
| **Lifecycle** | Installed by user, read-only | Created on use, auto-evicted |
| **Update mechanism** | `bl --update-dict` | `--no-cache` to skip |
| **Storage** | `~/.config/bl/dict/{lang}.db` | `~/.cache/bl/cache.db` |
| **Read/write** | Read-only at runtime | Read/write |
| **Data format** | zlib-compressed BLOBs | Plain JSON strings |

Never create a circular dependency between these two systems.

### 5.3 goquery Memory

`goquery` builds a full DOM tree in memory. For very large HTML pages, this could be slow. Current usage (Youdao mobile + verbformen.com) is well within acceptable limits. If adding a source with very large pages, consider streaming parsing or limiting selector scope.

### 5.4 modernc.org/sqlite Build Time

`modernc.org/sqlite` is a transpiled C library (CCGO). It adds ~30-60 seconds to the first `go build` and increases binary size by ~10MB. Subsequent builds are cached. If build time is critical, consider `github.com/mattn/go-sqlite3` (requires CGO) instead.

### 5.5 Telegram Bot Token

The Telegram bot reads `TELEGRAM_BOT_TOKEN` from environment. If this is not set, the bot binary will exit immediately with `log.Fatal`. The `bl` CLI binary does **not** need this token.

### 5.6 Source URL Stability

The dictionary source URLs (`m.youdao.com`, `www.verbformen.com`) are hardcoded. If the upstream sites change their HTML structure, the goquery selectors in `youdao.go` and `german.go` will break. This is an inherent risk with screen-scraping.

**Offline dictionaries mitigate this risk** — if an online source breaks, offline dictionaries continue to work indefinitely.

**Selector consolidation:** All CSS selectors are inline in the goquery calls. If a site changes selectors frequently, consider extracting them into constants at the package level.

### 5.7 Concurrent Access

`Rdict` is **not** goroutine-safe. The `http.Client` is safe for concurrent use, but `DictionarySource.Parse` is not guaranteed to be. If adding concurrent features:

```go
var mu sync.Mutex
mu.Lock()
result, err := client.GetResults(text)
mu.Unlock()
```

### 5.8 OfflineUnavailable vs NoTranslationResults

- `NoTranslationResults`: The parser ran successfully but found no data (e.g., word not in upstream dictionary).
- `OfflineUnavailable`: `--offline` mode is active and the word was not in the offline dictionary.

These are separate error types. Callers should check both independently.

## 6. Adding a New Feature Checklist

### New Dictionary Source

- [ ] Create `internal/dict/foo.go` implementing `DictionarySource`
- [ ] Add variant to `TranslationType` in `types.go`
- [ ] Add pointer field to `TranslationData` struct in `types.go`
- [ ] Update `MarshalJSON` / `UnmarshalJSON` in `types.go`
- [ ] Add render function in `internal/render/render.go`
- [ ] Register dispatch in `RenderTranslation()`
- [ ] Register source in `main.go` (`NewSourceByName`)
- [ ] Build: `go build -o bl . && go vet ./...`

### New Offline Dictionary Language Pair

- [ ] Create the `.db` file using `scripts/build_dict/` or `dict.CreateOfflineDict()`
- [ ] Add language pair case in `dict.LangForSource()` to map source→pair
- [ ] Add pair to `updateDictCmd()` and `dictStatusCmd()` download list in `main.go`
- [ ] Update docs: add pair to `ARCHITECTURE.md` and `AI_GUIDE.md`
- [ ] Verify with `bl --offline <word>` and `bl --dict-status`

### New Output Format

- [ ] Add format to `Format` enum in `types.go`
- [ ] Handle format in `RenderTranslation()` in `render.go`
- [ ] Add flag in `main.go`
- [ ] Handle format in `output()` function in `main.go`

### New Cache Backend

- [ ] Implement the same interface as `internal/cache/cache.go` (Get/Set/Delete/Close)
- [ ] Wire in `internal/dict/rdict.go` (import the new package)

## 6. LLM Translation (`--llm`)

### 6.1 Overview

LLM translation bypasses HTML scraping entirely. A new `Translator` interface in `rdict.go` is detected via type assertion:

```go
type Translator interface {
    Translate(word string) (*TranslationData, error)
}
```

When `Rdict.GetResults()` finds that the source implements `Translator`, it calls `Translate()` instead of `fetchSourceHTML()` + `Parse()`.

### 6.2 LLMSource (`internal/dict/llm.go`)

`LLMSource` implements `DictionarySource` (stubs `FetchURL`/`Parse`) + `Translator`:

- Sends POST to OpenAI-compatible `/chat/completions` endpoint
- Request body: system prompt + user message with the word
- Response is parsed as structured JSON into `TranslationData{Type: TypeTranslation}`

### 6.3 System Prompt

Defined as `defaultTranslationPrompt` in `llm.go`. Uses `%s` for direction:
- `"Translate the given text to 中文."` (default)
- `"Translate the given text from French to English."` (with `--from-lang`)

Custom prompts can be set via `config.json` → `llm.system_prompt`. If they contain `%s`, it's interpolated with direction. Otherwise the direction is prepended.

Prompt rules (hardcoded):
1. Return ONLY a JSON object
2. JSON structure: `translations`, `pronunciation`, `part_of_speech`, `gender`, `plural`, `comparative`, `superlative`, `examples`
3. Up to 3 translations
4. Exactly 5 vivid, scene-based example sentences
5. Gender/plural for inflected languages
6. Comparative/superlative for adjectives
7. Language-specific examples: English→English only, German→German+Chinese
8. Unset fields → empty string/array

### 6.4 Cache Key

`llm:{provider}:{sourceLang}:{targetLang}` — isolates across providers and languages.

### 6.5 Provider Fallback (v1.6+)

`llmQuery()` in `main.go` implements fallback logic:
1. Try the configured default provider
2. On API error (rate limit, timeout, HTTP 5xx), try the next provider in order
3. On input error (bad word, no translation), stop immediately
4. If a fallback provider succeeds, save it as new default in `config.json`

`isAPIError()` checks error strings against known API failure patterns.

### 6.6 Type Translation (`internal/dict/types.go`)

```go
type Translation struct {
    InputText     string    `json:"input_text"`
    SourceLang    string    `json:"source_lang,omitempty"`
    TargetLang    string    `json:"target_lang,omitempty"`
    Translations  []string  `json:"translations"`
    Pronunciation string    `json:"pronunciation,omitempty"`
    PartOfSpeech  string    `json:"part_of_speech,omitempty"`
    Gender        string    `json:"gender,omitempty"`
    Plural        string    `json:"plural,omitempty"`
    Comparative   string    `json:"comparative,omitempty"`
    Superlative   string    `json:"superlative,omitempty"`
    Examples      []Example `json:"examples,omitempty"`
}
```

### 6.7 Renderer

`RenderTranslationResult()` in `render.go` handles `TypeTranslation`, displaying:
- Translations as bullet list
- Part of Speech, Gender, Plural, Comparative, Superlative in header lines
- Pronunciation
- 5 example sentences

### 6.8 Rendering Example (`-j` JSON)

```json
{
  "data": {
    "input_text": "morning",
    "translations": ["早上", "上午", "早晨"],
    "pronunciation": "zǎoshang",
    "part_of_speech": "noun",
    "gender": "",
    "plural": "",
    "comparative": "",
    "superlative": "",
    "examples": [
      {"en": "She waved her hand and said hello to the mailman.", "zh": ""},
      {"en": "He picked up the phone and said hello.", "zh": ""}
    ]
  },
  "type": "translation"
}
```

Note: `zh` is empty for English source (per prompt rule 7). For German source, `zh` contains Chinese translation.

### 6.9 CLI Flags Summary

| Flag | Description |
|------|-------------|
| `--llm` | Enable LLM translation (overrides traditional sources) |
| `--llm-provider` | Select provider by name (openrouter, opencode-zen, nemotron) |
| `--llm-model` | Override model ID |
| `--llm-key` | Override API key |
| `--to-lang` | Target language |
| `--from-lang` | Source language (auto-detect if empty) |

## 7. Build & Test

```bash
# Build CLI
go build -o bl .                   # CLI
go build -o bl-telegram ./cmd/telegram/
go build -o bl-dingtalk ./cmd/dingtalk/
go build ./...
go vet ./...

# Run
./bl hello                   # Youdao (default)
./bl -g Haus                 # German dictionary
./bl -j 你好                  # JSON output
./bl -g -j laufen            # German + JSON
./bl --offline hello         # Offline mode only
./bl --mode offline          # Set offline as default mode
echo "world" | ./bl           # pipe mode
TELEGRAM_BOT_TOKEN=xxx ./bl-telegram
./bl-dingtalk -addr :8080           # then tunnel :8080 via ngrok
```