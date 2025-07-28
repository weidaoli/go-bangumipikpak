package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramNotifier struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

func (tn *TelegramNotifier) SendMessage(message string) error {
	msg := tgbotapi.NewMessage(tn.chatID, message)
	msg.ParseMode = "Markdown"

	_, err := tn.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %v", err)
	}

	return nil
}
