# Changelog

所有显著变更均记录于此文件。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

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

[v1.3.0]: https://github.com/xieguaiwu/bl/compare/v1.2.0...v1.3.0
[v1.2.0]: https://github.com/xieguaiwu/bl/compare/v1.1.0...v1.2.0
[v1.1.0]: https://github.com/xieguaiwu/bl/compare/v1.0.0...v1.1.0
[v1.0.0]: https://github.com/xieguaiwu/bl/releases/tag/v1.0.0
