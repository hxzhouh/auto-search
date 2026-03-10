package ai

import (
	"context"
	"testing"

	"auto-search/internal/content"
)

type stubChatClient struct {
	response string
	err      error
}

func (s stubChatClient) CreateChatCompletion(_ context.Context, _ []Message) (string, error) {
	return s.response, s.err
}

func TestCleanerClean(t *testing.T) {
	t.Parallel()

	cleaner := NewCleaner(stubChatClient{
		response: `{
			"cleaned_title":"OpenAI 发布新模型",
			"cleaned_summary":"摘要",
			"cleaned_content":"正文",
			"language":"en",
			"content_type":"news",
			"companies":["OpenAI"],
			"products":["GPT-5"],
			"topics":["AI Coding"],
			"quality_score":8,
			"importance_score":9,
			"writeworthy_score":7,
			"is_relevant":true,
			"angle_hint":"模型能力继续上探",
			"reason":"信息量高"
		}`,
	})

	result, err := cleaner.Clean(context.Background(), content.CleaningCandidate{
		RSSTitle:       "RSS 标题",
		ArticleTitle:   "文章标题",
		RawContentText: "正文原文",
	})
	if err != nil {
		t.Fatalf("清洗失败: %v", err)
	}

	if result.CleanedTitle != "OpenAI 发布新模型" {
		t.Fatalf("标题错误: %s", result.CleanedTitle)
	}
	if len(result.Companies) != 1 || result.Companies[0] != "openai" {
		t.Fatalf("公司标签错误: %+v", result.Companies)
	}
	if len(result.Topics) != 1 || result.Topics[0] != "ai_coding" {
		t.Fatalf("主题标签错误: %+v", result.Topics)
	}
}
