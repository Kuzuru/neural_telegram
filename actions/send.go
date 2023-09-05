package actions

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	tgutil "polina_petrilovna/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type RequestData struct {
	Messages []Message    `json:"messages"`
	Stream   bool         `json:"stream"`
	Model    modelOptions `json:"model"`
}

type ResponseData struct {
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

var MentionKeywords = []string{
	"PolinaPetrilovnaBot",
	"петруха",
	"петриловна",
	"петр",
	"бабка",
	"карга",
	"бабуля",
	"бабуль",
	"бабушка",
	"баба",
	"полина",
	"поля",
	"полинка",
}

type GroupMessage struct {
	UserID    int
	Message   string
	MessageID int
}

var GroupMessages = make(map[int64]chan GroupMessage)

func ShouldAnswer(update tgbotapi.Update) bool {
	for _, keyword := range MentionKeywords {
		if strings.Contains(strings.ToLower(update.Message.Text), strings.ToLower(keyword)) {
			return true
		}
	}

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	chance := r1.Float64()
	if chance <= 0.25 {
		return true
	}

	return false
}

// EmulateTyping Отправляет эмуляцию печати в беседу
func EmulateTyping(bot *tgbotapi.BotAPI, chatID int64) {
	chatActionConfig := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	_, _ = bot.Send(chatActionConfig)
}

func GenerateAndSendMessage(bot *tgbotapi.BotAPI, messageText string, chatID int64, messageID int) {
	fmt.Printf("[%s] [%+v // %+v] Working with Message: %+v\n", tgutil.GetFormattedTime(), chatID, messageID, messageText)

	// Emulate
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)

	// Call EmulateTyping in parallel until GenerateNeuralMessage ends
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				EmulateTyping(bot, chatID)
			}
		}
	}()

	// Generate
	message, _ := GenerateNeuralMessage(messageText)
	cancel()

	if len(GroupMessages[chatID]) <= 2 {
		msg := tgbotapi.NewMessage(chatID, message)
		_, _ = bot.Send(msg)
	} else {
		replyMsg := tgbotapi.NewMessage(chatID, message)
		replyMsg.ReplyToMessageID = messageID
		_, _ = bot.Send(replyMsg)
	}

	fmt.Printf("[%s] Message sent\n\n", tgutil.GetFormattedTime())
	fmt.Printf("[%s] [Group Capacity] %d\n", tgutil.GetFormattedTime(), len(GroupMessages[chatID]))
}
