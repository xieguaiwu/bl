# Show HN: bl — Terminal Dictionary Client with LLM Translation & Grammar Analysis

> Target: https://news.ycombinator.com/submit
> Title (81 chars): Show HN: bl – terminal dictionary with LLM, grammar parsing, offline dicts

---

## Post body (Markdown)

I built a terminal dictionary client in Go. It started as a simple Youdao scraper, but grew into something more useful.

### What it does

```
bl --llm --parse "Der Mann geht nach Hause."

# Lemma  gehen  (Past tense of gehen)
# Form   1st/3rd person singular simple past

| Token | POS  | Lemma  | Features                  | Dependency |
|-------|------|--------|---------------------------|------------|
| Der   | DET  | der    | Definite, Masc, Sg, Nom   | det        |
| Mann  | NOUN | Mann   | Masc, Sg, Nom             | nsubj      |
| geht  | VERB | gehen  | Ind, Pres, 3rd, Sg        | root       |
| nach  | ADP  | nach   | —                         | case       |
| Hause | NOUN | Haus   | Neut, Sg, Dat             | obl        |

# Translation  The man is going home.
```

### Key features

**3 query paths in one binary:**
1. **Offline dictionaries** — SQLite + zlib, no network needed
2. **Web scraping** — Youdao (EN⇄ZH) + WoerterNet (German verb conjugation)
3. **LLM translation** — any language pair via OpenAI-compatible APIs

**LLM features (v1.8.0):**
- Lemma extraction: `ging` → lemma `gehen` with morphological context
- Grammar parsing: full sentence analysis with token table, POS tags, dependency relations, phrase structure tree — works for German (V2, case), French (agreement, clitics), English
- Multiple provider support with automatic fallback (rate limit → next provider)
- Local `.blrc` config for project-specific model switching
- Interactive TUI for provider/model management (`bl -C`)

**Bot support:** Telegram + DingTalk bots included in the repo.

### Tech stack
- Pure Go, single binary, ~10MB stripped
- goquery for HTML parsing (youdao, verbformen)
- modernc.org/sqlite (pure-Go SQLite, no CGO)
- Multiple output formats: Markdown, JSON, oneliner
- Config: `~/.config/bl/config.json`

### Install
```bash
# Fedora (COPR)
sudo dnf copr enable xieguaiwu/bl && sudo dnf install bl

# Build from source
git clone https://github.com/xieguaiwu/bl.git && cd bl && go build -o bl .
```

### Why another dictionary?

I wanted something that works entirely in the terminal across three scenarios:
- Offline on a plane (pre-built SQLite dictionaries)
- Quick lookup with caching for common words
- Deep LLM analysis for understanding inflected forms when learning German

It's at https://github.com/xieguaiwu/bl — feedback welcome!

---

## Comments to watch for

Potential HN questions & prepared answers:

**Q: How is this different from just using `curl` against an API?**
A: Three-tier architecture: offline→cache→online. Offline dictionaries work without any network. The cache layer (SQLite) stores results so repeated lookups are instant. LLM is only invoked when you explicitly pass `--llm`.

**Q: What about `dict` / `gdict` / `sdcv`?**
A: Those use DICT protocol / StarDict dictionaries. bl supports those formats too (via the offline dict system), but also adds web scraping for Youdao (best EN⇄ZH source) and LLM-based translation for any language pair. The grammar analysis feature (`--parse`) is something no terminal dict client offers.

**Q: How do you handle API costs for LLM?**
A: Default provider uses OpenRouter's free tier (27+ free models). There's also a pure offline mode that never touches any API.

**Q: Why Go?**
A: Single binary deployment. The Telegram and DingTalk bots are built from the same codebase — `go build -o bl-telegram ./cmd/telegram`. No runtime dependencies.
