package actions

import (
	"github.com/pkoukk/tiktoken-go"
	"strings"
)

type modelOptions string

const (
	GPT4          modelOptions = "gpt-4"
	GPT432k       modelOptions = "gpt-4-32k"
	GPT35Turbo    modelOptions = "gpt-3.5-turbo"
	GPT35Turbo16k modelOptions = "gpt-3.5-turbo-16k"
)

type Message struct {
	Role    string
	Content string
}

var TKE *tiktoken.Tiktoken

func getChatGPTEncoding(messages []Message, model modelOptions) []int {
	isGpt3 := model == GPT35Turbo

	msgSep := ""
	roleSep := "<|im_sep|>"

	if isGpt3 {
		msgSep = "\n"
		roleSep = "\n"
	}

	serializedMessages := make([]string, len(messages))

	for i, msg := range messages {
		serializedMessages[i] = "<|im_start|>" + msg.Role + roleSep + msg.Content + "<|im_end|>"
	}

	serialized := strings.Join(serializedMessages, msgSep) + "<|im_start|>assistant" + roleSep

	return TKE.Encode(serialized, nil, nil)
}

func countTokens(messages []Message, model modelOptions) int {
	if len(messages) == 0 {
		return 0
	}

	return len(getChatGPTEncoding(messages, model))
}

func LimitMessageTokens(messages []Message, limit int, model modelOptions) []Message {
	if limit == 0 {
		limit = 4096
	}

	limitedMessages := make([]Message, 0)
	tokenCount := 0

	isSystemFirstMessage := messages[0].Role == "system"
	retainSystemMessage := false

	if isSystemFirstMessage {
		systemTokenCount := countTokens([]Message{messages[0]}, model)

		if systemTokenCount < limit {
			tokenCount += systemTokenCount
			retainSystemMessage = true
		}
	}

	for i := len(messages) - 1; i >= 1; i-- {
		count := countTokens([]Message{messages[i]}, model)

		if (count + tokenCount) > limit {
			break
		}

		tokenCount += count
		limitedMessages = append([]Message{messages[i]}, limitedMessages...)
	}

	if retainSystemMessage {
		index := len(limitedMessages) - 3

		if index < 0 {
			index = 0
		}

		limitedMessages = append(limitedMessages[:index], append([]Message{messages[0]}, limitedMessages[index:]...)...)
	} else if !isSystemFirstMessage {
		firstMessageTokenCount := countTokens([]Message{messages[0]}, model)

		if firstMessageTokenCount+tokenCount < limit {
			limitedMessages = append([]Message{messages[0]}, limitedMessages...)
		}
	}

	return limitedMessages
}
