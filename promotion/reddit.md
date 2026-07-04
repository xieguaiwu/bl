# Reddit Posts

---

## r/coolgithubprojects — English

> Post as link: https://github.com/xieguaiwu/bl
> Title: bl — terminal dictionary with LLM translation, grammar parsing, and offline support

**Comment (optional):**

I built a terminal dictionary client in Go that has three lookup modes:
1. Offline SQLite dictionaries (no network needed)
2. Web scraping (Youdao EN⇄ZH, WoerterNet German)
3. LLM translation via OpenAI-compatible API

The latest version also does grammatical analysis of full sentences — token table with POS tags, dependency relations, phrase structure trees. Works for German (V2 word order, case marking), French, and English.

Features:
- Single binary, ~10MB, no runtime deps
- `bl --llm --parse "Der Mann geht nach Hause."` for grammar breakdown
- Auto-detects inflected forms: `ging` → lemma `gehen` (past tense)
- Multiple LLM providers with automatic fallback
- Telegram & DingTalk bots included
- Config UI: `bl --config-ui`

```bash
sudo dnf copr enable xieguaiwu/bl && sudo dnf install bl
```

https://github.com/xieguaiwu/bl

---

## r/golang — English

> Post as link: https://github.com/xieguaiwu/bl
> Title: bl — a Go terminal dictionary with goquery, pure-Go SQLite, and LLM integration

**Comment:**

I wrote this terminal dictionary client in Go and wanted to share some technical decisions:

**Stack:** Pure Go. Uses `modernc.org/sqlite` (no CGO), `PuerkitoBio/goquery` for HTML parsing, and standard library for everything else. Single ~10MB stripped binary.

**Three-tier architecture:**
1. Offline SQLite dictionaries (read-only, zlib-compressed blobs) — instant, no network
2. SQLite query cache (WAL mode) — auto-populated, skippable with `--no-cache`
3. LLM API — only when `--llm` flag is passed

**LLM integration design:**
- The `DictionarySource` interface has a `Translator` sub-interface detected via type assertion
- Implements automatic provider fallback — if one API fails (rate limit, timeout), tries the next configured provider
- If a fallback succeeds, saves it as the new default
- New in v1.8.0: grammar analysis mode bypasses JSON parsing entirely, uses `sendRawLLMRequest()` for raw markdown output

**Bot builds:** The same codebase compiles to Telegram and DingTalk bots via `./cmd/` — just `go build -o bl-telegram ./cmd/telegram`.

Would love feedback on the architecture: https://github.com/xieguaiwu/bl

---

## r/commandline — English

> Post as link: https://github.com/xieguaiwu/bl
> Title: bl — terminal dictionary with offline support, LLM translation, and grammar analysis

**Comment:**

A CLI dictionary tool I've been using daily. Key terminal-friendly features:

- `bl hello` → instant EN⇄ZH lookup
- `bl -g Haus` → German dictionary (verb conjugations, noun declensions)
- `bl --llm "bonjour"` → LLM translation to Chinese (or any language)
- `bl --llm --parse "Je ne parle pas français."` → grammar analysis with token table
- `bl -C` → interactive config UI for managing providers/models
- `bl --clear-cache` → clear cached results
- `bl` (no args) → interactive REPL mode

COPR install for Fedora, or `go build` for others.

https://github.com/xieguaiwu/bl
