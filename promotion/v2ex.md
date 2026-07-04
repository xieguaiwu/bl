# V2EX — 中文

> 发布节点：分享创造
> 标题：bl — 终端词典客户端，支持离线词典、LLM 翻译、语法解析

---

## 正文

分享一个自己每天在用的终端词典工具 **bl**，用 Go 写的。

### 三种查询模式

```
# 模式一：离线词典（飞机上也能用）
bl --offline hello

# 模式二：在线爬取（Youdao 英汉 / WoerterNet 德语）
bl hello
bl -g Haus          # 德语词典，含动词变位

# 模式三：LLM 翻译（任意语言对）
bl --llm "bonjour"
bl --llm --to-lang "日本語" "good morning"
```

### 语法解析（v1.8.0 新功能）

这是我最常用的功能。输入一个句子，LLM 自动分析每个词的词性、原型、语法特征和依存关系：

```
bl --llm --parse "Der Mann geht nach Hause."

# Lemma  gehen (Past tense of gehen)
# Form   1st/3rd person singular simple past

| Token | POS  | Lemma  | Features            | Dependency |
|-------|------|--------|---------------------|------------|
| Der   | DET  | der    | Masc, Sg, Nom       | det        |
| Mann  | NOUN | Mann   | Masc, Sg, Nom       | nsubj      |
| geht  | VERB | gehen  | Ind, Pres, 3rd, Sg  | root       |
| nach  | ADP  | nach   | —                   | case       |
| Hause | NOUN | Haus   | Neut, Sg, Dat       | obl        |

# Translation  The man is going home.
```

曲折变化的词会自动还原到原型（ging → gehen, besser → gut, ran → run）。

### 技术特点

- **纯 Go** 编译为单文件，~10MB，零运行时依赖
- 三种数据源：SQLite 离线词典 → 查询缓存 → LLM API
- 支持 Telegram 和钉钉机器人（同一个代码库编译）
- 交互式配置界面：`bl -C`
- LLM 多 Provider 自动容灾（一个 API 挂了自动切下一个）

### 安装

```bash
# Fedora COPR
sudo dnf copr enable xieguaiwu/bl && sudo dnf install bl

# 源码编译
git clone https://github.com/xieguaiwu/bl.git && cd bl && go build -o bl .
```

GitHub：https://github.com/xieguaiwu/bl

欢迎 Star、PR、提 Issue。

---

## 掘金 — 中文

> 标题：写了个终端词典工具 bl：支持离线查询、LLM 翻译、语法分析

**文章结构建议：**

标题可以改成更吸引人的如：
- 「我用 Go 写了个终端词典，顺便加了个德语语法分析器」
- 「终端词典 bl v1.8：离线+AI 翻译+语法树，一个二进制搞定」

主图：terminal screenshot showing `bl --llm --parse` output

内容可以用 V2EX 版本的结构，再加一段"为什么用终端查词"的开头：

> 作为开发者，90% 的时间在终端里。为了查个单词切到浏览器 -> 打开词典网站 -> 输入 -> 看结果 -> 切回终端，这一套下来 30 秒没了。bl 把这件事压缩到 1 秒：打字、回车、继续写代码。
>
> 而且终端词典可以实现一些浏览器做不到的事：
> - 离线也能查（预建 SQLite 词典）
> - 管道输入（echo "word" | bl）
> - 配合 vim 用（:!bl % 查当前词）
> - 语法分析结果直接终端显示，不用截图
