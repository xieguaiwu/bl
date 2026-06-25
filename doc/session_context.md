# bl — Session Context

## Project

Terminal dictionary client written in Go. Supports web scraping (Youdao, WoerterNet), offline SQLite dictionaries, and LLM-based translation via OpenAI-compatible APIs.

## Architecture

```
main.go                           CLI entry point
internal/config/config.go         Persistent JSON config (mode, LLM providers)
internal/dict/
  types.go                        TranslationData, Translator interface
  source.go                       DictionarySource factory
  llm.go                          LLM translation source (OpenAI-compatible)
  youdao.go / german.go           HTML scraping sources
  offline.go                      SQLite offline dictionaries
  rdict.go                        Engine (offline→cache→online)
internal/render/render.go         Output formatting (Markdown/JSON/Oneliner)
internal/cache/cache.go           SQLite query cache
config.example.json               Template config (env-only keys, no secrets)
```

## LLM Translation Design

- `Translator` interface in `rdict.go` — engine detects API-based sources
- `LLMSource` in `llm.go` — OpenAI-compatible chat completions
- System prompt: structured JSON with translations, pronunciation, POS, gender, plural, comparative, superlative, 5 vivid examples
- Source language support via `--from-lang` (prompt becomes "from X to Y")
- Cache key: `llm:{provider}:{sourceLang}:{targetLang}`

## Provider Configuration

### Personal config (~/.config/bl/config.json)
- Default: `opencode-zen` + `deepseek-v4-flash-free` (no rate limits)
- Fallbacks: openrouter (qwen3), nemotron (NVIDIA direct)

### Providers

| Provider | Endpoint | Key Env Var |
|----------|----------|-------------|
| opencode-zen | https://opencode.ai/zen/v1 | OPENCODE_API_KEY |
| openrouter | https://openrouter.ai/api/v1 | OPENROUTER_API_KEY |
| nemotron | https://integrate.api.nvidia.com/v1 | NVIDIA_API_KEY |

### Free Models

**OpenCode Zen**: deepseek-v4-flash-free, nemotron-3-ultra-free, big-pickle, mimo-v2.5-free, north-mini-code-free
**OpenRouter (27+)**: google/gemma-4-31b-it:free, qwen/qwen3-next-80b-a3b-instruct:free, nvidia/nemotron-3-ultra-550b-a55b:free, meta-llama/llama-3.3-70b-instruct:free, etc.

### Local Config (.blrc)
Place in project root: `{"provider":"...", "model":"...", "target_lang":"...", "source_lang":"..."}`
Priority: CLI flags > .blrc > global config

## Preferences (from MEMORY.md)

- Examples: exactly 5, vivid concrete scenes with actions/imagery
- English source: English-only examples (no Chinese)
- German source: German + Chinese translation
- All API keys via env vars, never hardcoded

## Released Versions

| Tag | Key Feature |
|-----|-------------|
| v1.0.0 | Initial release (youdao + verbformen) |
| v1.1.0 | LLM translation, 4 provider presets |
| v1.2.0 | Gender/plural/comparative/superlative grammar info |
| v1.3.0 | --from-lang flag for ambiguous words |
| v1.4.0 | Default model: qwen3-next-80b-a3b-instruct:free |
| v1.5.0 | max_tokens 4096 for reasoning models |
| v1.6.0 | Automatic provider fallback |

## Provider Fallback (v1.6.0)

- When `llmQuery()` is called, it tries the configured provider first
- If that provider fails (rate limit, timeout, network error, empty response), bl iterates through remaining providers in order
- The first successful provider is cached to `~/.config/bl/config.json` as the new default for subsequent queries
- Interactive (`-i`) mode and bot platforms skip fallback — they use the primary provider only
- Cache key includes provider name, so fallback responses are properly isolated

## TODO

- [ ] Test opencode-zen nemotron-3-ultra-free (was timing out)
- [ ] Consider COPR package update for Fedora users
- [ ] Bot platform (Telegram/DingTalk) support for LLM translation
