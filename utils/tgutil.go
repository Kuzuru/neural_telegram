package tgutil

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"time"
)

func IsGroup(chat *tgbotapi.Chat) bool {
	return chat.IsGroup() || chat.IsSuperGroup() || chat.Type == "channel"
}

func GetLatestUpdateID(bot *tgbotapi.BotAPI) int {
	updates, err := bot.GetUpdates(tgbotapi.UpdateConfig{
		Offset:  0,
		Limit:   100,
		Timeout: 5,
	})

	if err != nil || len(updates) == 0 {
		return 0
	}

	return updates[len(updates)-1].UpdateID
}

func GetFormattedTime() string {
	return time.Now().Format("02.01.2006 15:04:05 UTC-07")
}
