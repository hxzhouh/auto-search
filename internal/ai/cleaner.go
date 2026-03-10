package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"auto-search/internal/content"
)

type Cleaner struct {
	client chatCompletionClient
}

type chatCompletionClient interface {
	CreateChatCompletion(ctx context.Context, messages []Message) (string, error)
}

type CleanResult struct {
	CleanedTitle     string   `json:"cleaned_title"`
	CleanedSummary   string   `json:"cleaned_summary"`
	CleanedContent   string   `json:"cleaned_content"`
	Language         string   `json:"language"`
	ContentType      string   `json:"content_type"`
	Companies        []string `json:"companies"`
	Products         []string `json:"products"`
	Topics           []string `json:"topics"`
	QualityScore     int      `json:"quality_score"`
	ImportanceScore  int      `json:"importance_score"`
	WriteworthyScore int      `json:"writeworthy_score"`
	IsRelevant       bool     `json:"is_relevant"`
	AngleHint        string   `json:"angle_hint"`
	Reason           string   `json:"reason"`
	RawResponse      string   `json:"-"`
}

func NewCleaner(client chatCompletionClient) *Cleaner {
	return &Cleaner{client: client}
}

func (c *Cleaner) Clean(ctx context.Context, item content.CleaningCandidate) (CleanResult, error) {
	raw, err := c.runPrompt(ctx, item, false)
	if err != nil {
		return CleanResult{}, err
	}

	jsonText, err := extractJSONObject(raw)
	if err != nil {
		raw, err = c.runPrompt(ctx, item, true)
		if err != nil {
			return CleanResult{}, err
		}
		jsonText, err = extractJSONObject(raw)
		if err != nil {
			return CleanResult{}, err
		}
	}

	var result CleanResult
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		return CleanResult{}, fmt.Errorf("解析 AI 清洗结果失败: %w", err)
	}

	normalizeResult(&result, item)
	result.RawResponse = raw
	return result, nil
}

func (c *Cleaner) runPrompt(ctx context.Context, item content.CleaningCandidate, compact bool) (string, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: "你是 AI 新闻素材清洗助手。你只输出 JSON，不要输出 Markdown、解释或额外文本。",
		},
		{
			Role:    "user",
			Content: buildCleaningPrompt(item, compact),
		},
	}

	return c.client.CreateChatCompletion(ctx, messages)
}

func buildCleaningPrompt(item content.CleaningCandidate, compact bool) string {
	var builder strings.Builder
	builder.WriteString("请把下面的新闻正文清洗成结构化 JSON。\n")
	builder.WriteString("要求：\n")
	builder.WriteString("1. 仅返回一个 JSON 对象。\n")
	builder.WriteString("2. 保留事实，不编造信息。\n")
	builder.WriteString("3. cleaned_content 输出干净正文，去掉导航、广告、版权、推荐阅读、无关尾巴。\n")
	builder.WriteString("4. language 只允许: zh, en, mixed。\n")
	builder.WriteString("5. content_type 只允许: news, release, tutorial, opinion, discussion, funding, security, benchmark。\n")
	builder.WriteString("6. companies/products/topics 都是字符串数组，使用简洁英文 snake_case 或小写短语。\n")
	builder.WriteString("7. quality_score、importance_score、writeworthy_score 范围是 1-10。\n")
	builder.WriteString("8. is_relevant 输出 true 或 false。\n")
	builder.WriteString("9. angle_hint 最多 30 个中文字符。\n")
	builder.WriteString("10. reason 用中文简述判断依据。\n\n")
	if compact {
		builder.WriteString("11. cleaned_content 控制在 600 字以内。\n")
		builder.WriteString("12. 不要输出思考过程，立刻输出 JSON。\n\n")
	}
	builder.WriteString("JSON 字段必须包含：\n")
	builder.WriteString(`{"cleaned_title":"","cleaned_summary":"","cleaned_content":"","language":"","content_type":"","companies":[],"products":[],"topics":[],"quality_score":0,"importance_score":0,"writeworthy_score":0,"is_relevant":false,"angle_hint":"","reason":""}`)
	builder.WriteString("\n\n")
	builder.WriteString("输入内容：\n")
	builder.WriteString("rss_title: " + item.RSSTitle + "\n")
	builder.WriteString("article_title: " + item.ArticleTitle + "\n")
	builder.WriteString("rss_summary: " + item.RSSSummary + "\n")
	builder.WriteString("source_site: " + item.RSSSourceSite + "\n")
	builder.WriteString("url: " + firstNonEmpty(item.CanonicalURL, item.FinalURL) + "\n")
	builder.WriteString("markdown正文:\n")
	builder.WriteString(truncateForPrompt(item.RawContentText, compact))
	return builder.String()
}

func extractJSONObject(raw string) (string, error) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end < start {
		return "", fmt.Errorf("AI 返回内容不是合法 JSON 对象")
	}
	return strings.TrimSpace(raw[start : end+1]), nil
}

func normalizeResult(result *CleanResult, item content.CleaningCandidate) {
	result.CleanedTitle = firstNonEmpty(strings.TrimSpace(result.CleanedTitle), item.ArticleTitle, item.RSSTitle)
	result.CleanedSummary = strings.TrimSpace(result.CleanedSummary)
	result.CleanedContent = strings.TrimSpace(result.CleanedContent)
	if result.CleanedContent == "" {
		result.CleanedContent = strings.TrimSpace(item.RawContentText)
	}
	result.Language = normalizeLanguage(result.Language)
	result.ContentType = normalizeContentType(result.ContentType)
	result.Companies = normalizeTags(result.Companies)
	result.Products = normalizeTags(result.Products)
	result.Topics = normalizeTags(result.Topics)
	result.QualityScore = clampScore(result.QualityScore)
	result.ImportanceScore = clampScore(result.ImportanceScore)
	result.WriteworthyScore = clampScore(result.WriteworthyScore)
	result.AngleHint = strings.TrimSpace(result.AngleHint)
	result.Reason = strings.TrimSpace(result.Reason)
}

func normalizeLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "zh", "en", "mixed":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "mixed"
	}
}

func normalizeContentType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "news", "release", "tutorial", "opinion", "discussion", "funding", "security", "benchmark":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "news"
	}
}

func normalizeTags(values []string) []string {
	set := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		tag := strings.ToLower(strings.TrimSpace(value))
		tag = strings.Join(strings.Fields(tag), "_")
		if tag == "" {
			continue
		}
		if _, exists := set[tag]; exists {
			continue
		}
		set[tag] = struct{}{}
		result = append(result, tag)
	}
	return result
}

func clampScore(value int) int {
	switch {
	case value < 1:
		return 1
	case value > 10:
		return 10
	default:
		return value
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func truncateForPrompt(text string, compact bool) string {
	text = strings.TrimSpace(text)
	limit := 3000
	if compact {
		limit = 1500
	}

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "\n\n[内容已截断]"
}
