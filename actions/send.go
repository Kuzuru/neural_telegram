package actions

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tgutil "polina_petrilovna/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var MentionKeywords = []string{
	"PolinaPetrilovnaBot",
	"петруха",
	"петриловна",
	"петр",
	"бабка",
	"карга",
	"бабуля",
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
// в звисимости от длины сообщения и длины запроса
func EmulateTyping(bot *tgbotapi.BotAPI, chatID int64, textLength int, alreadyPassed time.Duration) {
	chatActionConfig := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	_, _ = bot.Send(chatActionConfig)

	cps := 230.0
	speedAdjustment := .7

	nominalDuration := (float64(textLength) / cps * 60 * 1000) * speedAdjustment
	nominalDuration -= float64(alreadyPassed.Milliseconds())

	randomAdjustment := rand.Float64() * .3
	adjustedDuration := nominalDuration * (1 - randomAdjustment)

	durationInMilliseconds := int(adjustedDuration)
	durationInTimeFormat := time.Duration(durationInMilliseconds) * time.Millisecond

	fmt.Printf("[%s] [EMUL] Typing Message %+v...\n", tgutil.GetFormattedTime(), durationInTimeFormat)

	time.Sleep(durationInTimeFormat)

	fmt.Printf("[%s] [EMUL] Message sent\n\n", tgutil.GetFormattedTime())
}

func GenerateAndSendMessage(bot *tgbotapi.BotAPI, user *tgbotapi.User, messageText string, chatID int64, messageID int) {
	fmt.Printf("[%s] Working with Message: %+v %+v %+v\n", tgutil.GetFormattedTime(), chatID, messageID, messageText)

	EmulateTyping(bot, chatID, len("Привет, внучек %s (@%s)!"), 1*time.Second)

	userTeleTag := user.UserName
	firstName := user.FirstName

	message := fmt.Sprintf("Привет, внучек %s (@%s)! Ответ на сообщение: %s", firstName, userTeleTag, messageText)

	if len(GroupMessages[chatID]) <= 2 {
		msg := tgbotapi.NewMessage(chatID, message)
		_, _ = bot.Send(msg)
	} else {
		replyMsg := tgbotapi.NewMessage(chatID, message)
		replyMsg.ReplyToMessageID = messageID
		_, _ = bot.Send(replyMsg)
	}

	fmt.Printf("[%s] [Group Capacity] %d\n", tgutil.GetFormattedTime(), len(GroupMessages[chatID]))
}
