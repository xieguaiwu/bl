package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"bl/internal/dict"
	"bl/internal/render"
)

// DingTalk outgoing bot request body.
type dingtalkRequest struct {
	ConversationTitle string `json:"conversationTitle"`
	BotName           string `json:"botName"`
	SenderNick        string `json:"senderNick"`
	Text              struct {
		Content string `json:"content"`
	} `json:"text"`
	MsgType          string `json:"msgtype"`
	IsInAtList       bool   `json:"isInAtList"`
	SessionWebhook   string `json:"sessionWebhook"`
	ChatbotUserID    string `json:"chatbotUserId"`
	SenderID         string `json:"senderId"`
	ConversationType string `json:"conversationType"`
	SenderStaffID    string `json:"senderStaffId"`
}

type dingtalkResponse struct {
	MsgType string        `json:"msgtype"`
	Text    dingtalkText  `json:"text"`
}

type dingtalkText struct {
	Content string `json:"content"`
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	certFile := flag.String("cert", "", "TLS cert file (optional)")
	keyFile := flag.String("key", "", "TLS key file (optional)")
	sourceName := flag.String("source", "", "dictionary source (youdao, woerter-net)")
	flag.Parse()

	source := resolveSource(*sourceName)
	client, err := dict.NewRdict(source, "")
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	defer client.Close()

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req dingtalkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("bad request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		text := extractQuery(req.Text.Content)
		log.Printf("query from %s [%s]: %q", req.SenderNick, req.ConversationTitle, text)

		var reply string
		if text == "" {
			reply = "请发送要翻译的单词或短语\n用法: @" + req.BotName + " <text>"
		} else {
			result, err := client.GetResults(text)
			if err != nil {
				reply = "翻译失败: " + err.Error()
				log.Printf("get results for %q: %v", text, err)
			} else {
				reply = render.RenderTranslation(&result.Data, dict.FormatMarkdown, false)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dingtalkResponse{
			MsgType: "text",
			Text:    dingtalkText{Content: reply},
		})
	})

	log.Printf("DingTalk bot listening on %s", *addr)
	if *certFile != "" && *keyFile != "" {
		log.Fatal(http.ListenAndServeTLS(*addr, *certFile, *keyFile, nil))
	} else {
		log.Fatal(http.ListenAndServe(*addr, nil))
	}
}

func resolveSource(name string) dict.DictionarySource {
	if name == "" {
		name = os.Getenv("BL_SOURCE")
	}
	switch strings.ToLower(name) {
	case "woerter-net":
		return dict.NewWoerterNetSource("https://www.verbformen.com")
	default:
		return dict.NewYoudaoSource("https://m.youdao.com")
	}
}

func extractQuery(content string) string {
	content = strings.TrimSpace(content)
	if idx := strings.Index(content, " "); idx >= 0 {
		prefix := content[:idx]
		if strings.HasPrefix(prefix, "@") {
			content = strings.TrimSpace(content[idx+1:])
		}
	} else if strings.HasPrefix(content, "@") {
		content = ""
	}
	return content
}
