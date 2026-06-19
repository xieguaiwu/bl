# bl — 终端词典客户端

[English](README.md) | **中文**

一款用 Go 编写的终端词典客户端。支持 **有道词典**（英 ⇄ 中）和 **WoerterNet**（德语）查询，内置 SQLite 缓存、离线词典和持久化配置。同时支持多平台机器人（Telegram、钉钉）。

> 本项目受 [rdict](https://github.com/Guanran928/rdict) 启发而开发，感谢原作者的创意。

## 通过 COPR 安装（Fedora）

```bash
sudo dnf install dnf-plugins-core
sudo dnf copr enable xieguaiwu/bl
sudo dnf install bl
```

## 快速开始

```bash
go build -o bl . && ./bl hello
```

## 功能特性

- **多数据源**：有道词典（默认英⇄中）、WoerterNet（德语，通过 `-g` 参数）
- **三种查询模式**：`auto`（离线→缓存→在线）、`offline`（仅离线）、`online`（仅在线）
- **离线词典**：预构建的 SQLite 数据库，支持 zlib 压缩
- **SQLite 缓存**：自动缓存查询结果，可通过 `--no-cache` 跳过
- **输出格式**：Markdown（默认）、JSON（`-j`）、单行（`-o`）
- **ANSI 颜色**：自动检测终端支持，尊重 `NO_COLOR` 环境变量
- **交互模式**：REPL 提示符 `[bl]#`
- **机器人平台**：Telegram 机器人 + 钉钉 Webhook
- **LLM 翻译**：通过大语言模型实现任意语言之间的 AI 翻译，无需爬取网页

## LLM 翻译 (`--llm`)

使用大语言模型在任意语言之间翻译，支持词性、性数格、比较级等语法信息，并提供 5 个有画面感的例句。

```bash
# 默认 LLM 翻译（由 ~/.config/bl/config.json 决定）
bl --llm hello

# 指定目标语言
bl --llm --to-lang "日本語" "good morning"
bl --llm --to-lang "Français" "hello"

# 切换 Provider / 模型
bl --llm --llm-provider openrouter --llm-model "google/gemma-4-31b-it:free" hello
bl --llm --llm-model nemotron-3-ultra-free hello

# 临时指定 API key
bl --llm --llm-key "$OPENROUTER_API_KEY" hello
```

### Provider 预设

| Provider | 接口地址 | 默认模型 | 免费额度 |
|----------|----------|----------|----------|
| `openrouter` | `https://openrouter.ai/api/v1` | `google/gemma-4-31b-it:free` | 27+ 免费模型
| `opencode-zen` | `https://opencode.ai/zen/v1` | `deepseek-v4-flash-free` | 5 个免费模型
| `nemotron` | `https://integrate.api.nvidia.com/v1` | `nvidia/nemotron-3-ultra-550b-a55b` | 需 NVIDIA API key
| `custom` | 用户自定义 | 用户自定义 | 任意 OpenAI 兼容接口

### 丰富的语法信息

LLM 翻译结果包含以下语法信息（根据语言自动适配）：

- **词性**：名词/动词/形容词/副词等
- **性属**：阳性/阴性/中性（屈折语名词）
- **复数形式**：可数名词的复数
- **比较级 / 最高级**：形容词/副词
- **发音**：音标或拼音
- **5 个生动例句**：有画面感、有动作、有意象的具体场景

### 本地配置 (`.blrc`)

在项目目录创建 `.blrc` 文件，快速切换 Provider 和模型，无需每次传 CLI 参数：

```bash
# 引用全局配置中的 named provider
echo '{"provider":"openrouter","model":"google/gemma-4-31b-it:free","target_lang":"Français"}' > .blrc

# 完全独立定义 provider（不依赖全局配置）
echo '{"base_url":"https://openrouter.ai/api/v1","model":"google/gemma-4-31b-it:free","api_key":"env:OPENROUTER_API_KEY","target_lang":"日本語"}' > .blrc
```

优先级：`CLI 参数` > `.blrc` > `全局配置`

### 配置示例

```bash
cp config.example.json ~/.config/bl/config.json
# 编辑 ~/.config/bl/config.json 填入你的 API key
```

`api_key` 支持 `env:VAR_NAME` 格式引用环境变量 (`OPENROUTER_API_KEY`, `OPENCODE_API_KEY`, `NVIDIA_API_KEY`)。

### 指定源语言 (`--from-lang`)

当单词在多种语言中都存在时（如 "Raisonnement" 同时是法语和德语单词，"Handy" 在德语中意为"手机"），需要明确指定源语言：

```bash
bl --llm --from-lang French Raisonnement     # 法语 → 推理
bl --llm --from-lang German Raisonnement     # 德语 → 论证
bl --llm --from-lang German --to-lang English Handy   # 德语 Handy → mobile phone
```

不指定时，模型会自动判断源语言。

### API 密钥设置

在 shell 配置文件中添加：

```bash
export OPENROUTER_API_KEY="sk-or-v1-your-key"
export OPENCODE_API_KEY="sk-your-key"
export NVIDIA_API_KEY="nvapi-your-key"
```

## 使用方法

```bash
# 默认有道英⇄中
bl hello

# 德语词典
bl -g Haus
bl -s woerter-net laufen

# JSON 输出
bl -j hello

# 单行输出
bl -o hello

# 离线模式（无需网络）
bl --offline hello
bl --offline -g Haus

# 设置默认模式（持久化到配置文件）
bl --mode offline

# 管道模式
echo "world" | bl

# 交互模式
bl

# 词典管理
bl --dict-status
bl --update-dict          # 需要设置 BL_DICT_URL
bl --generate-config
```

## 模式优先级

优先级：`命令行参数 (--offline/--online)` > `BL_MODE 环境变量` > `配置文件` > `默认 "auto"`

| 模式 | 行为 |
|------|------|
| `auto` | 优先尝试离线词典，未命中则回退到在线查询 |
| `offline` | 仅使用离线词典，本地词典中不存在则报错 |
| `online` | 跳过离线词典，始终从网络获取 |

## 构建

```bash
# CLI
go build -o bl .

# 机器人二进制文件
go build -o bl-telegram ./cmd/telegram/
go build -o bl-dingtalk ./cmd/dingtalk/

# 全部
go build ./...
go vet ./...
```

## 项目结构

```
bl/
├── main.go                 # CLI 入口
├── cmd/
│   ├── telegram/main.go    # Telegram 机器人（长轮询）
│   └── dingtalk/main.go    # 钉钉机器人（HTTP 回调）
├── internal/
│   ├── config/config.go    # 持久化 JSON 配置
│   ├── dict/               # 核心：类型、数据源、离线词典、查询引擎
│   ├── render/render.go    # 输出格式化
│   └── cache/cache.go      # SQLite 查询缓存
├── scripts/build_dict/     # 离线词典构建工具
├── doc/                    # 架构与 AI 开发指南
└── scripts/testdata/       # 示例词典
```

## 依赖

| 库 | 用途 |
|----|------|
| `modernc.org/sqlite` | 纯 Go SQLite（无需 CGO） |
| `github.com/PuerkitoBio/goquery` | 使用 CSS 选择器解析 HTML |
| `github.com/go-telegram-bot-api/telegram-bot-api/v5` | Telegram Bot API |

## 环境变量

| 变量 | 用途 | 示例 |
|------|------|------|
| `BL_MODE` | 覆盖默认查询模式 | `BL_MODE=offline` |
| `BL_DICT_URL` | `--update-dict` 的下载地址 | `BL_DICT_URL=https://example.com/dicts` |
| `TELEGRAM_BOT_TOKEN` | Telegram 机器人令牌（bl-telegram 必需） | `TELEGRAM_BOT_TOKEN=xxx` |
| `RDICT_SOURCE` | Telegram 机器人的词典源 | `RDICT_SOURCE=woerter-net` |

## 许可

MIT
