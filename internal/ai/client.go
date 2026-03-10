package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"auto-search/internal/config"
	"auto-search/internal/httpclient"
)

type Client struct {
	httpClient  *http.Client
	baseURL     string
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content          any    `json:"content"`
			Reasoning        string `json:"reasoning"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func NewClient(cfg *config.Config) (*Client, error) {
	if strings.TrimSpace(cfg.AI.BaseURL) == "" {
		return nil, fmt.Errorf("ai.base_url 不能为空")
	}
	if strings.TrimSpace(cfg.AI.APIKey) == "" {
		return nil, fmt.Errorf("ai.api_key 不能为空")
	}
	if strings.TrimSpace(cfg.AI.Model) == "" {
		return nil, fmt.Errorf("ai.model 不能为空")
	}

	return &Client{
		httpClient:  newAIHTTPClient(cfg),
		baseURL:     strings.TrimRight(cfg.AI.BaseURL, "/"),
		apiKey:      cfg.AI.APIKey,
		model:       cfg.AI.Model,
		maxTokens:   cfg.AI.MaxTokens,
		temperature: cfg.AI.Temperature,
	}, nil
}

func newAIHTTPClient(cfg *config.Config) *http.Client {
	client := httpclient.New(cfg.HTTP)
	client.Timeout = time.Duration(cfg.AI.TimeoutSeconds) * time.Second
	return client
}

func (c *Client) CreateChatCompletion(ctx context.Context, messages []Message) (string, error) {
	requestBody, err := json.Marshal(chatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
	})
	if err != nil {
		return "", fmt.Errorf("序列化 AI 请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("创建 AI 请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 AI 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 AI 响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI 返回状态异常: %d, body=%s", resp.StatusCode, truncateBody(body))
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("解析 AI 响应失败: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("AI 返回错误: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("AI 响应缺少 choices")
	}

	content, err := extractMessageContent(parsed.Choices[0].Message.Content)
	if err != nil {
		fallback := strings.TrimSpace(parsed.Choices[0].Message.ReasoningContent)
		if fallback == "" {
			fallback = strings.TrimSpace(parsed.Choices[0].Message.Reasoning)
		}
		if fallback == "" {
			return "", err
		}
		return fallback, nil
	}

	return content, nil
}

func extractMessageContent(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), nil
	case []any:
		var builder strings.Builder
		for _, item := range typed {
			part, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if part["type"] == "text" {
				text, _ := part["text"].(string)
				builder.WriteString(text)
			}
		}
		if builder.Len() == 0 {
			return "", fmt.Errorf("AI 响应内容为空")
		}
		return strings.TrimSpace(builder.String()), nil
	default:
		return "", fmt.Errorf("无法识别的 AI 响应内容格式")
	}
}

func truncateBody(body []byte) string {
	const maxLen = 400
	text := strings.TrimSpace(string(body))
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen]
}
