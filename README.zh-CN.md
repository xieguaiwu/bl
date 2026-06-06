# bl — 终端词典客户端

[English](README.md) | **中文**

一款用 Go 编写的终端词典客户端。支持 **有道词典**（英 ⇄ 中）和 **WoerterNet**（德语）查询，内置 SQLite 缓存、离线词典和持久化配置。同时支持多平台机器人（Telegram、钉钉）。

> 本项目受 [rdict](https://github.com/Guanran928/rdict) 启发而开发，感谢原作者的创意。

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
