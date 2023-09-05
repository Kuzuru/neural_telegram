package main

import (
	"encoding/json"
	"fmt"
	"io"
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
		actions.GenerateAndSendMessage(bot, fmt.Sprintf("%s (%s): %s", user.FirstName, user.UserName, msg.Message), chatID, msg.MessageID)
	}
}

func ReadPromptsJSON(filename string) (*actions.RequestData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}(file)

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	var requestData actions.RequestData

	err = json.Unmarshal(data, &requestData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	return &requestData, nil
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

					// [[[ WRITING INITIAL DATA ]]]
					requestData, err := ReadPromptsJSON("prompts.json")
					if err != nil {
						fmt.Printf("Error: %v\n", err)
					}

					actions.AllMessageData.Lock()

					for _, message := range requestData.Messages {
						actions.AllMessageData.Messages = append(actions.AllMessageData.Messages, message)
					}

					actions.AllMessageData.Unlock()
					// [[[ EOW ]]]

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
