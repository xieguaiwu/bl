# Bot Platform Support: Multi-Platform Remote Dictionary

> Analysis of building bl bots for Telegram, QQ, DingTalk, WeChat, and other platforms.

## Current Status

| Platform | Status | Lines of Code | 
|----------|--------|---------------|
| CLI (direct) | Done | `main.go` (155) |
| Telegram | Done | `cmd/telegram/main.go` (98) |
| QQ | Not started | — |
| WeChat (Official Account) | Not started | — |
| WeChat Work Bot | Not started | — |

## Platform Feasibility Analysis

### Telegram Bot — ⭐⭐⭐⭐⭐ (已实现)

**Connection**: Long polling via `getUpdates`
**Auth**: Bot token from @BotFather
**Server**: No public server needed (outbound only)
**Registration**: Instant, no review
**API**: `github.com/go-telegram-bot-api/telegram-bot-api/v5`
**Formatting**: Full HTML / Markdown support
**Deploy**: `TELEGRAM_BOT_TOKEN=xxx ./bl-telegram`

**Assessment**: Gold standard for bot development. Zero friction, full featured.

---

### QQ Official Bot — ⭐⭐⭐⭐ (推荐下一个实现)

**Connection**: WebSocket (no public server needed)
**Auth**: Bot token + App ID from [bot.q.qq.com](https://bot.q.qq.com)
**Server**: No public server needed (WebSocket outbound)
**Registration**: Requires实名认证 + bot review (3-7 days)
**API**: `github.com/tencent-connect/botgo` (official Go SDK)
**Formatting**: Plain text, Markdown (limited)
**Commands**: Supports slash commands like Telegram
**Limitations**: 
  - Bot needs to be added to a group or friends list first
  - Rate limits apply
  - Message types differ between QQ and QQ Guild

**Key differences from Telegram**:
```go
// QQ uses WebSocket instead of long polling
// botgo event-driven architecture:
func main() {
    bot := botgo.NewBot(token, appID)
    bot.AddHandler(handlers{
        OnMessage: func(msg *Message) {
            reply := engine.HandleMessage(msg.Content)
            bot.Reply(msg, reply)
        },
    })
    bot.Start()
}
```

**Assessment**: Very feasible. The Go SDK is well-maintained by Tencent. WebSocket model means no public server. Main barrier is实名认证.

---

### QQ Guild Bot — ⭐⭐⭐

**Connection**: WebSocket
**Auth**: Bot token from QQ Guild platform
**Registration**: Simpler than QQ bot, less review
**API**: Same `botgo` SDK
**Limitations**: QQ Guild (频道) has smaller user base

**Assessment**: Feasible but QQ Guild adoption is limited in China.

---

### WeChat Official Account (公众号) — ⭐⭐

**Connection**: Webhook (requires public HTTPS server)
**Auth**: AppID + AppSecret from WeChat Official Account admin
**Server**: **REQUIRES** public HTTPS server with domain (no Ngrok for production)
**Registration**: 
  - Personal subscription account: cannot add custom bot logic (only menu + auto-reply)
  - Enterprise verified account: can use custom development (requires business license)
**API**: `wechat.dev` official SDK (no mature Go library, need to wrap REST API)
**Formatting**: XML-based, very limited
**Flow**:
```
User → WeChat Server → (HTTP POST) → Your HTTPS Server → (response XML) → WeChat → User
```

**Critical limitation**: Personal accounts cannot run custom bots. Only enterprise verified accounts ($30/year + business license) can.

**Assessment**: Impractical for this project. The registration barrier and server requirement are too high for a personal project.

---

### WeChat Work Bot (企业微信群机器人) — ⭐

**Connection**: Webhook (outbound only)
**Auth**: Webhook URL from group settings
**Capability**: Can only **push** notifications, cannot **receive** messages
**API**: Simple HTTP POST to webhook URL

**Assessment**: Not suitable for a query-response bot. Can only send notifications one-way.

---

### Personal WeChat Unofficial — ⭐ (不推荐)

**Approach**: Reverse-engineered protocols (ItChat, WeChatFerry, ComWeChatRobot, etc.)
**Risk**: Account ban on detection. WeChat actively detects and blocks automated clients.
**Go libraries**: Some exist but are unreliable and break on WeChat updates.

**Assessment**: Do not build on this. Account is too valuable to risk.

---

---

### DingTalk Outgoing Bot (钉钉出站机器人) — ⭐⭐⭐⭐⭐ (推荐·已实现)

**Connection**: HTTP callback (DingTalk POSTs to your server when @mentioned)
**Auth**: None (URL-based, configured in DingTalk group bot settings)
**Server**: **REQUIRES** public HTTPS URL (use ngrok for dev, nginx + Let's Encrypt for prod)
**Registration**: Instant (add a custom bot in any DingTalk group → enable Outgoing)
**API**: None needed (raw HTTP JSON in/out, zero Go dependencies)
**Formatting**: Plain text only
**Dependencies added**: **Zero** (uses `net/http` from stdlib)
**Source**: `cmd/dingtalk/main.go` (fully implemented)
**Deploy**:
  ```bash
  # Development
  ngrok http 8080 &
  DINGTALK_URL=https://xxxx.ngrok.io ./bl-dingtalk

  # Production (with TLS)
  ./bl-dingtalk -addr :443 -cert cert.pem -key privkey.pem
  ```

**Data flow**:
```
User: "@bl hello" in DingTalk group
  │
  ▼
DingTalk Server ──HTTP POST──► your /webhook
  │  {                          │
  │    "text":{"content":       │  parse text
  │      "@bl hello"},       │  → "hello"
  │    "senderNick":"张三",       │  → Rdict.GetResults("hello")
  │    ...                      │  → render
  │  }                          │
  │                             ▼
  │  {                          │
  │    "msgtype":"text",        │  JSON response
  │    "text":{"content":       │
  │      "# Pronunciation\n...  │
  │    }                        │
  │  }                          │
  │◄──── HTTP 200 ──────────────┘
  ▼
DingChat group shows reply
```

**Setup steps (5 minutes)**:

1. `go build -o bl-dingtalk ./cmd/dingtalk/`
2. `ngrok http 8080` → get `https://xxxx.ngrok.io`
3. In DingTalk group → Group Settings → Bots → Add Bot → Custom → 
   - Name: bl
   - Callback URL: `https://xxxx.ngrok.io/webhook`
   - Token: (leave empty for dev)
4. `./bl-dingtalk`
5. Type `@bl hello` in the group

**Advantages**:
- Zero new dependencies (pure `net/http`)
- Simplest possible integration model (HTTP in/out)
- Works in any DingTalk group instantly
- Group admin can add it without app store approval
- Can run behind any reverse proxy (nginx, caddy, Cloudflare)
- Same binary, same codebase, same Rdict engine
- No WebSocket, no SDK, no third-party library

**Limitations**:
- Requires public HTTPS URL (hard requirement from DingTalk)
- Plain text only (no rich cards without official bot registration)
- Outgoing mode only (bot responds when @mentioned, cannot initiate)
- Single callback URL per bot (one endpoint handles all groups)

**Assessment**: Simplest bot to implement (zero new deps, raw HTTP). Best for development and small teams. Only requires a public HTTPS endpoint.

---

### DingTalk Official Bot (钉钉开放平台机器人) — ⭐⭐⭐

**Connection**: HTTP callback (same as Outgoing Bot but with token verification)
**Auth**: AppKey + AppSecret from [open-dev.dingtalk.com](https://open-dev.dingtalk.com)
**Server**: Requires public HTTPS URL
**Registration**: Register application → get AppKey/AppSecret → publish
**API**: Official DingTalk SDK or raw HTTP
**Formatting**: Rich cards, markdown, interactive components
**Extra capabilities**: 
  - Bot can be installed from DingTalk app store
  - Supports interactive card callbacks
  - Persistent bot profile across the org

**Assessment**: More powerful but heavier setup. Only worth it if you need rich cards or app store distribution. The Outgoing Bot covers most use cases with zero registration friction.

---

### Matrix / Discord — ⭐⭐⭐⭐ (备选)

Not mentioned but worth considering as alternatives:

**Discord**: Excellent bot API (`discordgo`), similar to Telegram but popular globally
**Matrix**: Decentralized protocol, good Go SDK (`mautrix/go`)

## Unified Architecture

Rather than each bot being a standalone implementation, a shared `BotEngine` abstracts the common logic:

```
cmd/
  telegram/main.go          ← Telegram adapter (uses botengine)
  qq/main.go                ← QQ adapter (uses botengine)
  wechat/main.go            ← WeChat adapter (uses botengine, if implemented)
  
internal/
  botengine/
    engine.go               ← Shared bot logic
    message.go              ← Platform-agnostic message types
  dict/                     ← Unchanged
  render/                   ← Unchanged
  cache/                    ← Unchanged
```

### `internal/botengine/engine.go` — Core Design

```go
package botengine

import (
"bl/internal/dict"
	"bl/internal/render"
)

// Engine provides platform-agnostic query handling.
// Every bot platform shares this code path.
type Engine struct {
    client *dict.Rdict
}

func New(cacheDB string) (*Engine, error) {
    source := dict.NewYoudaoSource("https://m.youdao.com")
    client, err := dict.NewRdict(source, cacheDB)
    if err != nil {
        return nil, err
    }
    return &Engine{client: client}, nil
}

func (e *Engine) Close() error {
    return e.client.Close()
}

// Handle processes a plain-text query and returns a plain-text reply.
// Output is always plain text for maximum platform compatibility.
func (e *Engine) Handle(text string) string {
    if text == "" {
        return "usage: send me a word or phrase to translate"
    }
    result, err := e.client.GetResults(text)
    if err != nil {
        return "error: " + err.Error()
    }
    out := render.RenderTranslation(&result.Data, dict.FormatMarkdown, false)
    if result.IsCached {
        out += "\n\n(cached)"
    }
    return out
}

// SetSource switches the dictionary source at runtime.
func (e *Engine) SetSource(s dict.DictionarySource) {
    e.client = dict.NewRdict(s, "")
}
```

### Platform Adapter Pattern

Each platform's `main.go` becomes a thin adapter:

```go
// cmd/telegram/main.go — Telegram adapter (simplified)
func main() {
    engine := botengine.New("")
    defer engine.Close()

    bot, _ := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
    for update := range bot.GetUpdatesChan(tgbotapi.NewUpdate(0)) {
        if update.Message == nil { continue }
        reply := engine.Handle(update.Message.Text)
        bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, reply))
    }
}
```

```go
// cmd/qq/main.go — QQ adapter (design sketch)
func main() {
    engine := botengine.New("")
    defer engine.Close()

    bot := botgo.NewBot(os.Getenv("QQ_APP_ID"), os.Getenv("QQ_BOT_TOKEN"))
    bot.OnMessage(func(msg *botgo.Message) {
        reply := engine.Handle(msg.Content)
        bot.ReplyText(msg, reply)
    })
    bot.Start()
}
```

### Comparison of Adapter Complexity

| Platform | Adapter Lines | Connection | Auth Env Vars |
|----------|--------------|------------|---------------|
| Telegram | ~40 LOC | Long polling | `TELEGRAM_BOT_TOKEN` |
| QQ | ~50 LOC | WebSocket | `QQ_APP_ID`, `QQ_BOT_TOKEN` |
| QQ Guild | ~40 LOC | WebSocket | `GUILD_APP_ID`, `GUILD_BOT_TOKEN` |
| Discord | ~40 LOC | WebSocket | `DISCORD_BOT_TOKEN` |
| WeChat OA | ~80 LOC | Webhook HTTP | `WECHAT_APP_ID`, `WECHAT_APP_SECRET` |

## Recommendation

### Priority Order

```
1. Telegram      ✅ Done — keep as reference implementation
2. QQ Bot         ← Recommended next. Feasible, official, good Go SDK.
3. QQ Guild       ← Alternative if QQ Bot review is too slow.
4. Discord        ← If international users are a target.
5. WeChat OA      ← Only if enterprise account is available.
6. Personal WeChat ← Never. Risk > reward.
```

### Implementation Roadmap

**Phase 1 (current)**: Telegram bot working

**Phase 2**: Extract `BotEngine` (1-2 hours)
- Move shared logic to `internal/botengine/`
- Refactor `cmd/telegram/main.go` to use it
- No functional change

**Phase 3**: QQ bot (3-5 hours)
- Register on bot.q.qq.com
- Implement adapter using botgo SDK
- Test in QQ group

**Phase 4 (optional)**: Discord / more
- Each additional platform is ~40-80 LOC

### Key Insight

All bot platforms follow the same pattern:

```
Receive text → Engine.Handle(text) → Send reply
```

The `BotEngine` abstraction makes adding new platforms a mechanical task of writing a thin adapter. The core dictionary logic never needs to change.

The biggest differentiator between platforms is not the code but the **registration process** and **deployment requirements**. Telegram and QQ (WebSocket) can run on any machine with internet access. WeChat requires a public HTTPS server.
