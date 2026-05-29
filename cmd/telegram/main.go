package main

import (
	"log"
	"os"
	"strings"

	"bl/internal/dict"
	"bl/internal/render"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN not set")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("create bot: %v", err)
	}
	log.Printf("authorized as %s", bot.Self.UserName)

	sourceName := os.Getenv("RDICT_SOURCE")
	if sourceName == "" {
		sourceName = "youdao"
	}

	source := dict.NewSourceByName(sourceName)

	client, err := dict.NewRdict(source, "")
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	defer client.Close()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil || !update.Message.IsCommand() {
			continue
		}

		switch update.Message.Command() {
		case "help":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"/translate <text> - translate the given text\n/help - show this message")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)

		case "translate":
			text := update.Message.CommandArguments()
			if text == "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID,
					"usage: /translate <text>")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			result, err := client.GetResults(text)
			if err != nil {
				log.Printf("get results: %v", err)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID,
					"translation failed. please try again later.")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			output := render.RenderTranslation(&result.Data, dict.FormatMarkdown, false)
			wrapped := "<pre><code class=\"language-markdown\">" +
				escapeHTML(output) + "</code></pre>"

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, wrapped)
			msg.ReplyToMessageID = update.Message.MessageID
			msg.ParseMode = "HTML"
			bot.Send(msg)
		}
	}
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
