package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkoukk/tiktoken-go"
)

type modelOptions string

const (
	GPT4       modelOptions = "gpt-4"
	GPT35Turbo modelOptions = "gpt-3.5-turbo"
)

type AllMessages struct {
	*sync.Mutex
	Messages []Message
}

var AllMessageData = &AllMessages{
	Mutex:    &sync.Mutex{},
	Messages: make([]Message, 0),
}

type Message struct {
	Role    string
	Content string
}

type ErrorMessage struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type ErrorDetail struct {
	Error ErrorMessage `json:"error"`
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

func prepareAndTokenizeData() RequestData {
	limitMessages := LimitMessageTokens(AllMessageData.Messages, 7000, GPT4)

	newData := RequestData{
		Messages: limitMessages,
		Stream:   false,
		Model:    GPT4,
	}

	return newData
}

func GenerateNeuralMessage(messageText string) (string, time.Duration, bool) {
	startTime := time.Now()

	url := os.Getenv("NEURAL_NETWORK_URL") + "?conversation_id=" + os.Getenv("CONVERSATION_ID")
	token := os.Getenv("AUTHORIZATION_TOKEN")

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	AllMessageData.Lock()
	AllMessageData.Messages = append(AllMessageData.Messages, Message{
		Role:    "user",
		Content: messageText,
	})

	requestData := prepareAndTokenizeData()
	AllMessageData.Unlock()

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("Error marshaling requestData: %v\n", err)
		return "", time.Duration(0), false
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return "", time.Duration(0), false
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	res, err := client.Do(req)

	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return "", time.Duration(0), false
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("[ERR] [net/http] %+v\n", err)
		}
	}(res.Body)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return "", time.Duration(0), false
	}

	responseData := &ResponseData{}

	err = json.Unmarshal(body, responseData)
	if err != nil {
		fmt.Printf("[ERR] [unmarshal] %+v\n", err)
	}

	if len(responseData.Choices) > 0 {
		AllMessageData.Lock()
		AllMessageData.Messages = append(AllMessageData.Messages, Message{
			Role:    responseData.Choices[0].Message.Role,
			Content: responseData.Choices[0].Message.Content,
		})
		AllMessageData.Unlock()
	} else {
		e := ErrorDetail{}
		if err := json.Unmarshal(body, &e); err != nil {
			panic(err)
		}

		if e.Error.Type == "engine_overloaded_error" {
			time.Sleep(2 * time.Second)

			return "", time.Duration(0), true
		}

		return "", time.Duration(0), false
	}

	duration := time.Since(startTime)

	return responseData.Choices[0].Message.Content, duration, false
}
