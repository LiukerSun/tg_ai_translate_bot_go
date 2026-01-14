package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"tg-bot-go/logger"
	"tg-bot-go/models"
	"tg-bot-go/openai"
	"time"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) HandleMessage(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	// 1. 限流检查 (每分钟 10 条)
	if h.isRateLimited(chatID) {
		msg := tgbotapi.NewMessage(chatID, "您发送消息太快了，请稍后再试。")
		h.Bot.Send(msg)
		return
	}

	// 检查是否是命令（以/开头）
	if strings.HasPrefix(text, "/") {
		h.handleCommand(update)
		return
	}
	// 检查用户是否在白名单中且未过期
	isValid, validErr := models.IsUserValid(h.DB, chatID)
	if validErr != nil {
		logger.LogRuntime(fmt.Sprintf("检查用户有效性失败：%v", validErr))
		msg := tgbotapi.NewMessage(chatID, "系统错误，请稍后再试。")
		h.Bot.Send(msg)
		return
	}
	if !isValid {
		msg := tgbotapi.NewMessage(chatID, "您的使用权限已过期或未获得授权。")
		h.Bot.Send(msg)
		return
	}

	// 记录用户消息
	logger.LogUserMessage(chatID, text)

	contextKey := fmt.Sprintf("user:%d:context", chatID)
	presetKey := fmt.Sprintf("user:%d:preset", chatID)

	// 1. 获取 System Prompt (预设)
	userPreset, _ := h.Redis.Get(ctx, presetKey).Result()
	if userPreset == "" {
		userPreset = "你是一个有帮助的助手。"
	}

	// 2. 获取历史记录 (Redis List)
	historyStrs, err := h.Redis.LRange(ctx, contextKey, 0, -1).Result()
	if err != nil {
		logger.LogRuntime(fmt.Sprintf("Failed to get context: %v", err))
		historyStrs = []string{}
	}

	var messages []openai.ChatMessage
	messages = append(messages, openai.ChatMessage{Role: "system", Content: userPreset})
	
	currentLength := utf8.RuneCountInString(userPreset)
	
	var historyMessages []openai.ChatMessage
	for _, s := range historyStrs {
		var msg openai.ChatMessage
		if err := json.Unmarshal([]byte(s), &msg); err == nil {
			historyMessages = append(historyMessages, msg)
			currentLength += utf8.RuneCountInString(msg.Content)
		}
	}
	
	userMsg := openai.ChatMessage{Role: "user", Content: text}
	currentLength += utf8.RuneCountInString(text)
	
	// 3. 上下文长度控制
	removedCount := 0
	for currentLength > MAX_CONTEXT_LENGTH && len(historyMessages) > 0 {
		removedMsg := historyMessages[0]
		historyMessages = historyMessages[1:]
		currentLength -= utf8.RuneCountInString(removedMsg.Content)
		removedCount++
	}

	if removedCount > 0 {
		for i := 0; i < removedCount; i++ {
			h.Redis.LPop(ctx, contextKey)
		}
		logger.LogRuntime(fmt.Sprintf("User %d context trimmed by %d messages", chatID, removedCount))
	}

	messages = append(messages, historyMessages...)
	messages = append(messages, userMsg)

	// 4. 调用 OpenAI
	response, err := openai.GetOpenAIResponse(messages)
	if err != nil {
		logger.LogRuntime(fmt.Sprintf("OpenAI API error: %v", err))
		msg := tgbotapi.NewMessage(chatID, "获取响应失败，请稍后再试。")
		h.Bot.Send(msg)
		return
	}

	// 5. 发送响应
	msg := tgbotapi.NewMessage(chatID, response)
	h.Bot.Send(msg)
	logger.LogUserMessage(chatID, response)

	// 6. 保存新消息到 Redis Context
	userJson, _ := json.Marshal(userMsg)
	assistantMsg := openai.ChatMessage{Role: "assistant", Content: response}
	assistJson, _ := json.Marshal(assistantMsg)

	pipe := h.Redis.Pipeline()
	pipe.RPush(ctx, contextKey, string(userJson))
	pipe.RPush(ctx, contextKey, string(assistJson))
	pipe.Expire(ctx, contextKey, 30*time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		logger.LogRuntime(fmt.Sprintf("Failed to update context in Redis: %v", err))
	}
}

// isRateLimited 检查用户是否触发限流 (10次/分钟)
func (h *Handler) isRateLimited(userID int64) bool {
	key := fmt.Sprintf("ratelimit:%d", userID)
	limit := 10
	
	// 使用 Redis INCR 计数
	count, err := h.Redis.Incr(ctx, key).Result()
	if err != nil {
		logger.LogRuntime(fmt.Sprintf("Rate limit error: %v", err))
		return false // Redis 出错时放行，保证可用性
	}
	
	if count == 1 {
		// 第一次访问，设置过期时间
		h.Redis.Expire(ctx, key, time.Minute)
	}
	
	return count > int64(limit)
}
