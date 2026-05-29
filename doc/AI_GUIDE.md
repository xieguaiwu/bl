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

### 1.2 `Rdict` (internal/dict/rdict.go)

```go
type Rdict struct { /* unexported fields */ }

func NewRdict(source DictionarySource, cacheDB string) (*Rdict, error)
func (r *Rdict) Close() error
func (r *Rdict) GetResults(inputText string) (*FetchedResult, error)
```

**Contract:**

- `NewRdict(source, cacheDB)`: If `cacheDB` is empty string, caching is disabled. The cache directory is auto-created. Returns error only if cache initialization fails (not if cache is disabled).
- `GetResults(inputText)`: First checks cache, then fetches HTML, parses, stores in cache, returns result. **Always returns a non-nil `*FetchedResult` on success.** The caller checks `result.IsCached`.
- **Thread safety**: `Rdict` is **not** safe for concurrent use (the embedded `http.Client` is safe, but the `DictionarySource.Parse` may have state). Use `sync.Mutex` if concurrent access is needed.

### 1.3 `Cache` (internal/cache/cache.go)

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

**JSON format** (compatible with Rust serde-tagged):

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

Both implement the `error` interface. `NoTranslationResults` is distinct from a regular `error` — callers may want to handle it differently (e.g. "no results found" vs "something went wrong").

### 2.3 Optional Fields

In the structs, empty string (`""`) means "absent". There are no pointer-to-string fields. This differs from Rust's `Option<String>` and C++'s `std::optional<std::string>`. When adding fields, prefer zero-value semantics over pointers for simplicity.

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
| `--no-cache` | bool | `false` | Disable SQLite caching |
| positional | string[] | (empty) | Word(s) to translate |

### 4.2 Cache Path

Cache database location (when not `--no-cache`):

- Linux: `~/.cache/bl/cache.db`
- macOS: `~/Library/Caches/bl/cache.db`
- Windows: `%LOCALAPPDATA%\bl\cache.db`

Uses `os.UserCacheDir()` per XDG spec.

### 4.3 Color Detection

ANSI color is enabled when `TERM` is set and not `"dumb"`, and `NO_COLOR` is not set (per https://no-color.org).

## 5. Common Pitfalls for AI Agents

### 5.1 Import Cycle: `dict` ↔ `cache`

**`internal/cache` must not import `internal/dict`.** The cache stores raw JSON strings (`string`) instead of `*TranslationData` to avoid a circular import. All JSON marshaling/unmarshaling happens in `internal/dict/rdict.go`.

If you need to add a new type to the cache, **do not** import `internal/dict` from `internal/cache`. Either:
- Keep it string-based and marshal in the caller
- Or extract the shared type into a third package

### 5.2 goquery Memory

`goquery` builds a full DOM tree in memory. For very large HTML pages, this could be slow. Current usage (Youdao mobile + verbformen.com) is well within acceptable limits. If adding a source with very large pages, consider streaming parsing or limiting selector scope.

### 5.3 modernc.org/sqlite Build Time

`modernc.org/sqlite` is a transpiled C library (CCGO). It adds ~30-60 seconds to the first `go build` and increases binary size by ~10MB. Subsequent builds are cached. If build time is critical, consider `github.com/mattn/go-sqlite3` (requires CGO) instead.

### 5.4 Telegram Bot Token

The Telegram bot reads `TELEGRAM_BOT_TOKEN` from environment. If this is not set, the bot binary will exit immediately with `log.Fatal`. The `bl` CLI binary does **not** need this token.

### 5.5 Source URL Stability

The dictionary source URLs (`m.youdao.com`, `www.verbformen.com`) are hardcoded. If the upstream sites change their HTML structure, the goquery selectors in `youdao.go` and `german.go` will break. This is an inherent risk with screen-scraping.

**Selector consolidation:** All CSS selectors are inline in the goquery calls. If a site changes selectors frequently, consider extracting them into constants at the package level.

### 5.6 Concurrent Access

`Rdict` is **not** goroutine-safe. The `http.Client` is safe for concurrent use, but `DictionarySource.Parse` is not guaranteed to be. If adding concurrent features:

```go
var mu sync.Mutex
mu.Lock()
result, err := client.GetResults(text)
mu.Unlock()
```

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

### New Output Format

- [ ] Add format to `Format` enum in `types.go`
- [ ] Handle format in `RenderTranslation()` in `render.go`
- [ ] Add flag in `main.go`
- [ ] Handle format in `output()` function in `main.go`

### New Cache Backend

- [ ] Implement the same interface as `internal/cache/cache.go` (Get/Set/Delete/Close)
- [ ] Wire in `internal/dict/rdict.go` (import the new package)

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
echo "world" | ./bl           # pipe mode
TELEGRAM_BOT_TOKEN=xxx ./bl-telegram
./bl-dingtalk -addr :8080           # then tunnel :8080 via ngrok
```
