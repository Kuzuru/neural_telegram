package main

import (
	"fmt"
	"log"
	"os"

	"polina_petrilovna/actions"
	tgutil "polina_petrilovna/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/joho/godotenv/autoload"
	"github.com/pkoukk/tiktoken-go"
)

func processGroupMessages(bot *tgbotapi.BotAPI, user *tgbotapi.User, chatID int64) {
	for msg := range actions.GroupMessages[chatID] {
		actions.GenerateAndSendMessage(bot, user, msg.Message, chatID, msg.MessageID)
	}
}

func bootstrap() (*tgbotapi.BotAPI, tgbotapi.UpdateConfig, error) {
	tke, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, tgbotapi.UpdateConfig{}, err
	}

	// Initialize TikToken
	actions.TKE = tke

	// Get .env
	botToken := os.Getenv("BOT_TOKEN")

	// Main bootstrap
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, tgbotapi.UpdateConfig{}, err
	}

	latestUpdateID := tgutil.GetLatestUpdateID(bot)
	u := tgbotapi.NewUpdate(latestUpdateID + 1)
	u.Timeout = 60

	fmt.Printf("[%s] Bot initiated!\n", tgutil.GetFormattedTime())
	fmt.Printf("[%s] Last update: %+v\n", tgutil.GetFormattedTime(), latestUpdateID)

	return bot, u, err
}

func main() {
	// Initialize bot
	bot, u, err := bootstrap()
	if err != nil {
		log.Fatalln(err)
	}

	updates, err := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message != nil && update.Message.Text != "" {
			if tgutil.IsGroup(update.Message.Chat) {
				chatID := update.Message.Chat.ID

				if actions.GroupMessages[chatID] == nil {
					fmt.Printf("[%s] [MSG] %s: %s\n", tgutil.GetFormattedTime(), update.Message.From.UserName, update.Message.Text)

					actions.GroupMessages[chatID] = make(chan actions.GroupMessage)
					go processGroupMessages(bot, update.Message.From, chatID)
				}

				if actions.ShouldAnswer(update) {
					actions.GroupMessages[chatID] <- actions.GroupMessage{
						UserID:    update.Message.From.ID,
						Message:   update.Message.Text,
						MessageID: update.Message.MessageID,
					}
				}
			}
		}
	}
}
