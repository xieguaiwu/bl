# 发布指南

## 执行顺序（推荐）

```
Day 1:   Hacker News Show HN  +  r/golang
Day 2:   r/coolgithubprojects  +  r/commandline
Day 3:   V2EX
Day 4:   掘金
Day 7:   Product Hunt
```

间隔 1-2 天是为了让 star 增长曲线自然，避免各平台爆发式涌入同一天。

---

## 1. Hacker News

**URL:** https://news.ycombinator.com/submit
**标题 (81字符):** Show HN: bl – terminal dictionary with LLM, grammar parsing, offline dicts
**URL:** https://github.com/xieguaiwu/bl

正文用 promotion/hackernews_showhn.md 的内容。
注意：HN 正文不支持 Markdown 表格，发布前把 token table 换成纯文本格式。

**最佳发布时间:** 周一至周四 14:00-16:00 UTC（对应美东 10:00-12:00，美西 7:00-9:00）

---

## 2. Reddit

### r/golang
**URL:** https://old.reddit.com/r/golang/submit
**类型:** link post
**URL:** https://github.com/xieguaiwu/bl
**标题:** bl — a Go terminal dictionary with goquery, pure-Go SQLite, and LLM integration

### r/coolgithubprojects
**URL:** https://old.reddit.com/r/coolgithubprojects/submit
**类型:** link post
**标题:** bl — terminal dictionary with LLM translation, grammar parsing, and offline support

### r/commandline
**URL:** https://old.reddit.com/r/commandline/submit
**类型:** link post
**标题:** bl — terminal dictionary with offline support, LLM translation, and grammar analysis

---

## 3. V2EX

**URL:** https://v2ex.com/go/create
**节点:** 分享创造
**标题:** bl — 终端词典客户端，支持离线词典、LLM 翻译、语法解析
**正文:** 见 promotion/v2ex.md

V2EX 支持 Markdown，可以直接粘贴。
**最佳时间:** 工作日上午 10:00-11:00（北京时间）

---

## 4. 掘金

**URL:** https://juejin.cn/post
**标题:** 写了个终端词典工具 bl：离线查询 + LLM 翻译 + 语法分析
**标签:** Go、开源、命令行工具、词典

注意掘金的编辑器是富文本，建议先在 Markdown 写好再粘贴。
配图建议：终端截图 + GitHub star 历史图。

---

## 5. Product Hunt

**URL:** https://www.producthunt.com/posts/new
**Hunter:** 可以自己 launch，也可以找朋友帮忙 hunter
**标题:** bl — Terminal dictionary with LLM grammar analysis
**Tagline:** Offline dictionaries, AI translation, and grammatical parsing in your terminal
**Description:**

> bl is a terminal dictionary client written in Go. It supports three query modes:
> - Offline SQLite dictionaries (works without network)
> - Web scraping (Youdao EN⇄ZH, WoerterNet German)
> - LLM translation (any language pair via OpenAI API)
>
> Version 1.8 adds grammatical analysis — parse full sentences with token-level POS tagging, dependency relations, and structural trees. Works for German (V2, case system), French (agreement, clitics), and English.
>
> Features a config UI (`bl -C`), automatic provider fallback, Telegram/DingTalk bots, and interactive REPL mode. Single ~10MB Go binary.
>
> Install: `sudo dnf copr enable xieguaiwu/bl && sudo dnf install bl`

**Topics:** Developer Tools, Open Source, Artificial Intelligence
**Product URL:** https://github.com/xieguaiwu/bl
**Video:** optional

---

## 发布后的维护

1. **HN 评论区** — 发布后 2 小时内常刷新，及时回复问题。常见问题见 hackernews_showhn.md 的预备回答
2. **Reddit 评论区** — 隔几小时看一次
3. **Star 增长** — 正常节奏是 HN 当天 50-200 stars，Reddit 20-50，V2EX 30-80，Product Hunt 50-300
4. **检查异常** — 如果某个平台带来大量 star 但没有任何 issue/feedback/comment，可能被 GitHub 标记。正常情况会伴随 issue 和 discussion
