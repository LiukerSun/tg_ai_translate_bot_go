package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"tg-bot-go/config"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type OpenAIChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

type OpenAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

var httpClient = &http.Client{
	Timeout: 60 * time.Second,
}

// GetOpenAIResponse gets a response from OpenAI based on the provided messages history
func GetOpenAIResponse(messages []ChatMessage) (string, error) {
	apiURL := fmt.Sprintf("%s/v1/chat/completions", config.Config.OpenAI.APIURL)
	apiKey := config.Config.OpenAI.APIKey
	model := config.Config.OpenAI.Model

	if strings.TrimSpace(apiKey) == "" {
		return "", fmt.Errorf("openai api key not set")
	}
	if strings.TrimSpace(model) == "" {
		return "", fmt.Errorf("openai model not set")
	}

	// 构建请求体
	requestBody, err := json.Marshal(OpenAIChatRequest{
		Model:    model,
		Messages: messages,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	if ref := config.Config.OpenAI.HTTPReferer; ref != "" {
		req.Header.Set("HTTP-Referer", ref)
	}
	if title := config.Config.OpenAI.XTitle; title != "" {
		req.Header.Set("X-Title", title)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyText := strings.TrimSpace(string(bodyBytes))
		if len(bodyText) > 2000 {
			bodyText = bodyText[:2000]
		}
		return "", fmt.Errorf("openai api error: status %d: %s", resp.StatusCode, bodyText)
	}

	var openAIResp OpenAIChatResponse
	if err := json.Unmarshal(bodyBytes, &openAIResp); err != nil {
		return "", err
	}

	if len(openAIResp.Choices) > 0 {
		return openAIResp.Choices[0].Message.Content, nil
	}

	var openAIError OpenAIErrorResponse
	if err := json.Unmarshal(bodyBytes, &openAIError); err == nil && openAIError.Error.Message != "" {
		return "", fmt.Errorf(openaiErrorMessage(openAIError))
	}

	return "", fmt.Errorf("no response from OpenAI")
}

func openaiErrorMessage(openAIError OpenAIErrorResponse) string {
	if openAIError.Error.Type != "" {
		return fmt.Sprintf("%s: %s", openAIError.Error.Type, openAIError.Error.Message)
	}
	return openAIError.Error.Message
}
