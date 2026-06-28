# bl — Session Context

> Current version: v1.6.3. For AI agents that need to understand, modify, or extend the project.

## Project Overview

Terminal dictionary client written in Go. Three query paths:

1. **Traditional scraping**: Youdao (EN↔ZH), WoerterNet (German) via goquery HTML parsing
2. **Offline dictionaries**: Pre-built SQLite + zlib-compressed blobs
3. **LLM translation**: OpenAI-compatible API for any language pair

## Architecture

```
main.go                    CLI entry point, flag parsing, LLM fallback loop
internal/
  config/config.go         Persistent JSON config (mode, LLM providers)
  dict/
    types.go               TranslationData, Translation struct, Translator interface
    source.go              DictionarySource factory
    llm.go                 LLMSource (OpenAI-compatible chat completions)
    youdao.go              Youdao HTML scraping
    german.go              WoerterNet German dictionary scraping
    offline.go             SQLite offline dictionary (read-only)
    rdict.go               Engine: offline→cache→online (HTML or API)
  render/render.go         Output formatting (Markdown, JSON, Oneliner)
  cache/cache.go           SQLite query cache
config.example.json        Template config (all env:VAR, no real keys)
doc/
  AI_GUIDE.md              Full architectural guide for AI agents
  session_context.md       This file — quick context for handoff
```

## LLM Translation Design

### Translator Interface (`dict/rdict.go`)

```go
type Translator interface {
    Translate(word string) (*TranslationData, error)
}
```

`Rdict.GetResults()` detects `Translator` via type assertion. If the source implements it, calls `Translate()` instead of `fetchSourceHTML()` + `Parse()`.

### LLMSource (`dict/llm.go`)

- Implements both `DictionarySource` (stubbed `FetchURL`/`Parse`) and `Translator`
- Sends POST to `{baseUrl}/chat/completions` with OpenAI-compatible format
- System prompt requests structured JSON output
- `buildPrompt()` constructs the prompt with direction: `"to 中文"` or `"from French to English"`

### System Prompt Rules

1. Return ONLY JSON (no markdown, no fences)
2. JSON structure: `translations[]`, `pronunciation`, `part_of_speech`, `gender`, `plural`, `comparative`, `superlative`, `examples[{en, zh}]`
3. Up to 3 translations (multiple meanings)
4. Exactly 5 vivid, concrete scene-based examples
5. Gender/plural for inflected languages (German, French)
6. Comparative/superlative for adjectives
7. Language-specific examples: English → English only, German → German + Chinese
8. Unset fields → empty string/array

### Cache Key

`llm:{provider}:{sourceLang}:{targetLang}` — prevents collision across providers/languages.

## Provider Fallback (v1.6+)

In `main.go`, `llmQuery()` implements:

1. Try default provider (from config or `.blrc`)
2. On API error → try next provider in order
3. On input error → stop immediately (no point retrying)
4. If fallback succeeds → save as new default (`config.Save()`)

`isAPIError()` checks for: rate limit, timeout, HTTP 5xx, connection refused, EOF, TLS errors, "Provider returned error". Everything else is treated as input error.

## Provider Configuration

### Personal (`~/.config/bl/config.json`)

```json
{
  "provider": "opencode-zen",
  "providers": [
    {"name":"opencode-zen","base_url":"https://opencode.ai/zen/v1","model":"deepseek-v4-flash-free","api_key":"env:OPENCODE_API_KEY"},
    {"name":"openrouter",  "base_url":"https://openrouter.ai/api/v1",    "model":"qwen/qwen3-next-80b-a3b-instruct:free","api_key":"env:OPENROUTER_API_KEY"},
    {"name":"nemotron",    "base_url":"https://integrate.api.nvidia.com/v1","model":"nvidia/nemotron-3-ultra-550b-a55b",   "api_key":"env:NVIDIA_API_KEY"}
  ]
}
```

### Local Config (`.blrc`)

Place in project root. Overrides global config:
```json
{"provider":"openrouter", "model":"google/gemma-4-31b-it:free", "target_lang":"Français", "source_lang":"English"}
```

Priority: `CLI flags` > `.blrc` > `global config`

## User Preferences (from MEMORY.md)

- Examples: exactly 5, vivid concrete scenes with actions/imagery
- English source → English-only examples (zh field empty)
- German source → German original + Chinese translation
- All API keys via env vars only, never hardcoded in files

## API Keys (in ~/.config/fish/config.fish)

| Env Var | Purpose |
|---------|---------|
| `OPENCODE_API_KEY` | OpenCode Zen (default provider, no rate limits) |
| `OPENROUTER_API_KEY` | OpenRouter (27+ free models, 50 req/day limit) |
| `NVIDIA_API_KEY` | NVIDIA Nemotron direct API |

## Released Versions

| Tag | Key Feature |
|-----|-------------|
| v1.0.0 | Initial: youdao + verbformen + offline dicts |
| v1.1.0 | LLM translation, 4 provider presets |
| v1.2.0 | Gender/plural/comparative/superlative grammar |
| v1.3.0 | `--from-lang` for ambiguous words |
| v1.4.0 | Default: qwen3-next-80b-a3b-instruct:free |
| v1.5.0 | max_tokens 4096 for reasoning models |
| v1.6.0 | Automatic provider fallback |
| v1.6.1 | Smart fallback: skip retry on bad user input |
| v1.6.2 | Compact translation output on single line |
| v1.6.3 | Provider label on each query, full sentence support |

## Common Tasks

### Add a new provider preset
1. Add entry to `config/DefaultLLMConfig()` in `config.go`
2. User adds to `config.json` with `api_key: "env:VAR_NAME"`

### Add a new LLM output field
1. Add field to `Translation` struct in `types.go`
2. Add display in `RenderTranslationResult()` in `render.go`
3. Update system prompt to request it

### Debug a failing provider
```bash
bl --llm --llm-provider opencode-zen hello    # test specific provider
bl --llm -j hello                              # raw JSON output
```

## TODO

- [ ] Bot platform (Telegram/DingTalk) LLM support
- [ ] Renew COPR token + rebuild: copr-cli build bl https://github.com/xieguaiwu/bl/archive/v1.6.3.tar.gz
- [ ] Reasoning model support (strip `reasoning_content` from response)
