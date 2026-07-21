# Changelog

所有显著变更均记录于此文件。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [v1.8.3] - 2026-07-16

### Added

- **Chinese idiom dictionary** (`bl --cn <concept/idiom>`): LLM-powered Chinese
  dictionary with three modes:
  - **概念查成语**: input a concept (e.g. "形容非常高兴"), get 5-8 matching idioms
    with pinyin, definition, and usage tips
  - **成语查释义**: input an idiom (e.g. "画蛇添足"), get full breakdown with
    origin, synonyms/antonyms, examples, and English equivalent
  - **生僻字查解释**: input a rare character (e.g. "叕"), get pinyin, meaning,
    common compounds, and cultural notes
  - Uses `sendChatRequest()` — new generic chat API helper with customizable
    system prompt and user message

## [v1.8.2] - 2026-07-04

### Added

- **Interactive translation picker** (`bl --llm --pick <word>`): after a forward
  LLM translation, shows numbered candidate list and lets users select which
  words to reverse-check. Each selected word is then translated back to Chinese
  so users can compare full meanings:
  ```
  bl --llm --pick --to-lang English 你好
  # Shows: [1] hello [2] hi
  # Select: 1,2
  # Shows reverse: hello → 中文 (你好/嗨) and hi → 中文 (嗨/你好)
  ```
  - Supports multiple selections: `1,3`, `1-3`, `1 3`, or any combination
  - New: `pickQuery()`, `parseSelection()`, `parseInt()`
  - Reverse queries use swapped source/target languages, cached normally

### Fixed

- **Interactive mode now supports arrow keys, Ctrl+A/E, Home/End**: the
  `interactiveMode` function was rewritten to use a raw-terminal ReadLine
  (adapted from the Essen project) instead of `bufio.Scanner`. Users can now
  edit input freely with proper cursor movement.
  - Left/Right arrows, Home (Ctrl+A), End (Ctrl+E)
  - Backspace, Delete (Ctrl+D), Ctrl+U to clear line
  - Ctrl+C to abort
  - Full UTF-8 / CJK character support
  - Falls back to `bufio.Scanner` when stdin is not a TTY
  - New package: `internal/tui/input.go` with `ReadLine()`

## [v1.8.0] - 2026-07-04

### Added

- **Grammar analysis** (`bl --llm --parse <sentence>`): full grammatical structure
  analysis via LLM. Output includes:
  - Token table: POS, lemma, morphological features, dependency relations
  - Sentence structure: phrase types (NP/VP/PP), clause analysis
  - Key observations: verb position, case marking, agreement, fixed phrases
  - Natural translation
  - Supports German (V2, case system), French (agreement, clitics), English (tense, aspect)
  - New: `grammarQuery()`, `sendRawLLMRequest()`, `grammarAnalysisPrompt`
- **Morphological analysis (lemma extraction)**: LLM now identifies the base/dictionary
  form of inflected words. Examples:
  - "ging" → lemma "gehen" with "Past tense of gehen"
  - "besser" → lemma "gut" with "Comparative form of gut"
  - "ran" → lemma "run" with "past tense of run"
  - "children" → lemma "child" with "Plural of child"
  - New output fields: `# Lemma`, `# Form` sections in LLM translation results
  - New Translation struct fields: `lemma`, `lemma_info`, `inflection`
  - Renderer updated to display lemma/ inflection info
- **Documentation**: open-source grammar parser reference table (spaCy, Stanza,
  TreeTagger, Stanford CoreNLP) with integration examples

### Fixed

- **LLM fallback errors now visible**: previously when the default provider failed and
  bl tried fallback providers, their error messages were silently hidden. Now every
  provider's error is shown inline, so users can see why each one failed.

## [v1.7.1] - 2026-07-04

### Added

- **Cache management**:
  - `bl --clear-cache`: clear all cached query results with size summary
  - Cache UI in config TUI (`[c]` key): shows entry count and size, prompts before clearing
  - VACUUM after clearing to reclaim disk space
- New `cache.Clear()` + `cache.Stats()` methods for programmatic access

### Fixed

- **LLM fallback errors now visible**: previously when the default provider failed and
  bl tried fallback providers, their error messages were silently hidden. Now every
  provider's error is shown inline, so users can see why each one failed.

## [v1.7.0] - 2026-07-04

### Added

- **Interactive LLM Configuration UI** (`bl --config-ui` / `bl -C`):
  - Select default provider by entering its number
  - Change model for any provider
  - Toggle LLM translation on/off
  - Change target/source language
  - Add, edit, or delete providers
  - Changes saved to `~/.config/bl/config.json` on exit
  - Pure standard-library TUI — no external dependencies

## [v1.6.4] - 2026-06-25

### Fixed

- `-g` flag now correctly uses German dictionary instead of LLM mode when LLM is enabled in config
- Traditional source flags (`-g`, `-s`) now take priority over `cfg.llm.enabled`; explicit `--llm` still overrides everything

## [v1.6.3] - 2026-06-25

### Added

- **Provider label**: every LLM translation now shows `[provider / model]` so users know which backend is serving

## [v1.6.2] - 2026-06-25

### Changed

- Compact translation output: translations now shown on one line separated by " / " instead of a multi-line bullet list

## [v1.6.1] - 2026-06-25

### Fixed

- Smart fallback: only retry next provider on API-level errors (rate limit, timeout, HTTP 5xx, connection issues). Input-level errors (bad word, no translation results) now stop immediately instead of wasting retries on remaining providers.

## [v1.6.0] - 2026-06-19

### Added

- **Automatic provider fallback**: when an LLM provider fails (rate limit, timeout, etc.), bl automatically tries the next configured provider in order
- **Auto-save working provider**: if a different provider succeeds, it's saved as the new default in ~/.config/bl/config.json for future queries

### Changed

- LLM query path refactored into llmQuery() with fallback loop
- Interactive mode unchanged (uses primary provider only)

## [v1.5.0] - 2026-06-19

### Fixed

- Increased max_tokens from 1024 to 4096 for reasoning models (DeepSeek V4 Flash, etc.) — reasoning traces were consuming the token budget, leaving empty responses
- Stop tracking pre-built binary files bl-dingtalk and bl-telegram in git

## [v1.4.0] - 2026-06-19

### Changed

- Default model switched to `qwen/qwen3-next-80b-a3b-instruct:free` — MoE 80B/3B active, fastest inference, no thinking traces, excellent multilingual support
- Updated config.example.json and README to reflect the new default

## [v1.3.0] - 2026-06-19

### Added

- `--from-lang` flag: specify source language explicitly for ambiguous shared words (e.g. French vs German "Raisonnement")
- `source_lang` field in global config and `.blrc` local config

### Changed

- LLM prompt now includes source language when specified: "Translate from {source} to {target}"
- Cache key includes source language for proper isolation

### Docs

- README (EN + ZH): added `--from-lang` usage section with examples

## [v1.2.0] - 2026-06-19

### Added

- **Rich grammatical info in LLM output**: Gender, Plural, Comparative, Superlative fields for inflected languages
- **`config.example.json` template** — all API keys removed, safe for sharing
- **LLM translation section in README** (EN + ZH) with provider table and configuration guide

### Changed

- **System prompt redesigned**: now requests gender / plural / comparative / superlative for inflected languages
- **Custom system prompts** now correctly interpolate target language (via `%s` or auto-prepend)
- **Default model** switched to `google/gemma-4-31b-it:free` (27+ free models available on OpenRouter)

### Fixed

- **`--to-lang` flag** now correctly overrides target language even when custom system prompts are active

## [v1.1.0] - 2026-06-19

### Added

- **LLM 翻译功能** (`--llm` 标志)：支持通过 OpenAI 兼容 API 调用大模型进行任意语言翻译。
  - 内置 4 个 Provider 预设：nemotron、bigpickle、opencode、custom
  - `--to-lang` 指定目标语言（默认中文）
  - 缓存支持：以 `llm:{provider}:{lang}` 复合键隔离不同 provider/语言
- **本地配置文件 `.blrc`**：支持在项目根目录放置 `.blrc` 快速切换 LLM provider/model，无需反复传 CLI 参数。

### Changed

- 引擎架构扩展：新增 `Translator` 接口，支持 LLM 作为翻译源与现有词典源并存
- 新增翻译结果渲染器 `RenderTranslationResult`
- CLI 新增 5 个 flags：`--llm`, `--llm-provider`, `--llm-model`, `--llm-key`, `--to-lang`
- 配置系统扩展：`Config` 新增 `LLMConfig` 和 `LLMProvider` 字段

### Fixed

- **LLM 渲染异常**：
  - 输出中多余的代码围栏（code fence stripping）已去除
  - 重复的 "Pronunciation" 标签已修复
- **LLM JSON 解析健壮性**：支持括号深度匹配提取 JSON，应对模型输出中多个 JSON 对象或额外文本
- **Provider 默认值**：旧配置加载后空 provider 字段自动填充默认值
- **API key 错误提示**：显示实际配置来源而非猜测的环境变量名
- **空响应/错误响应**：提前判空并检查 JSON 结构，防止错误内容被包装为正常翻译结果
- **自定义 provider 校验**：前置检查 `base_url` 和 `model` 非空，避免无效配置静默失败

### Docs

- README 新增 COPR 安装说明（Fedora/RHEL 系）

## [v1.0.0] - 2026-06-17

### Added

- 初始发布
- 命令行词典客户端，支持 Youdao（英汉）和 verbformen（德语）爬取
- 离线 SQLite 词典支持（kd 风格）
- 缓存系统，减少重复查询
- Telegram / DingTalk 机器人通知
- 持久化配置（online / offline 模式切换）
- AI_GUIDE.md 和 ARCHITECTURE.md 文档
- MIT 许可证，COPR RPM 打包配置
- 中英文双 README，带语言切换器

[v1.8.3]: https://github.com/xieguaiwu/bl/compare/v1.8.2...v1.8.3
[v1.8.2]: https://github.com/xieguaiwu/bl/compare/v1.8.1...v1.8.2
[v1.8.1]: https://github.com/xieguaiwu/bl/compare/v1.8.0...v1.8.1
[v1.8.0]: https://github.com/xieguaiwu/bl/compare/v1.7.1...v1.8.0
[v1.7.1]: https://github.com/xieguaiwu/bl/compare/v1.7.0...v1.7.1
[v1.7.0]: https://github.com/xieguaiwu/bl/compare/v1.6.4...v1.7.0
[v1.6.4]: https://github.com/xieguaiwu/bl/compare/v1.6.3...v1.6.4
[v1.6.3]: https://github.com/xieguaiwu/bl/compare/v1.6.2...v1.6.3
[v1.6.2]: https://github.com/xieguaiwu/bl/compare/v1.6.1...v1.6.2
[v1.6.1]: https://github.com/xieguaiwu/bl/compare/v1.6.0...v1.6.1
[v1.6.0]: https://github.com/xieguaiwu/bl/compare/v1.5.0...v1.6.0
[v1.5.0]: https://github.com/xieguaiwu/bl/compare/v1.4.0...v1.5.0
[v1.4.0]: https://github.com/xieguaiwu/bl/compare/v1.3.0...v1.4.0
[v1.3.0]: https://github.com/xieguaiwu/bl/compare/v1.2.0...v1.3.0
[v1.2.0]: https://github.com/xieguaiwu/bl/compare/v1.1.0...v1.2.0
[v1.1.0]: https://github.com/xieguaiwu/bl/compare/v1.0.0...v1.1.0
[v1.0.0]: https://github.com/xieguaiwu/bl/releases/tag/v1.0.0
