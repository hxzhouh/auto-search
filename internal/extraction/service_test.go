package extraction

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"auto-search/internal/content"
	"auto-search/internal/extractor"
	"auto-search/internal/resolver"
)

type stubContentRepo struct {
	items     []content.PendingContent
	updates   []content.ExtractionUpdate
	listErr   error
	updateErr error
}

func (s *stubContentRepo) ListPendingForExtraction(_ context.Context, _ int) ([]content.PendingContent, error) {
	return s.items, s.listErr
}

func (s *stubContentRepo) UpdateExtractionResult(_ context.Context, update content.ExtractionUpdate) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updates = append(s.updates, update)
	return nil
}

type stubArticleExtractor struct {
	article extractor.Article
	err     error
}

func (s stubArticleExtractor) Extract(_ context.Context, _ string) (extractor.Article, error) {
	return s.article, s.err
}

type stubArticleResolver struct {
	result resolver.Result
	err    error
}

func (s stubArticleResolver) Resolve(_ context.Context, _ string) (resolver.Result, error) {
	return s.result, s.err
}

func TestServiceRun(t *testing.T) {
	t.Parallel()

	published := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	repo := &stubContentRepo{
		items: []content.PendingContent{
			{
				ID:           11,
				RSSTitle:     "RSS 标题",
				CanonicalURL: "https://example.com/article",
			},
		},
	}

	service := &Service{
		contentRepo: repo,
		resolver: stubArticleResolver{
			result: resolver.Result{CanonicalURL: "https://example.com/article"},
		},
		extractor: stubArticleExtractor{
			article: extractor.Article{
				Title:       "正文标题",
				Author:      "Alice",
				Description: "摘要",
				PublishedAt: &published,
				Markdown:    "# 正文\n\n内容",
			},
		},
	}

	stats, err := service.Run(context.Background(), 10)
	if err != nil {
		t.Fatalf("执行提取服务失败: %v", err)
	}

	if stats.Selected != 1 || stats.Extracted != 1 || stats.Failed != 0 {
		t.Fatalf("统计结果不正确: %+v", stats)
	}
	if len(repo.updates) != 1 {
		t.Fatalf("期望 1 次更新，实际为 %d", len(repo.updates))
	}
	if repo.updates[0].Status != "extracted" {
		t.Fatalf("状态更新错误: %s", repo.updates[0].Status)
	}
	if !repo.updates[0].ArticlePublished.Valid || !repo.updates[0].ArticlePublished.Time.Equal(published) {
		t.Fatalf("发布时间更新错误: %+v", repo.updates[0].ArticlePublished)
	}
	if repo.updates[0].CanonicalURL != "https://example.com/article" {
		t.Fatalf("canonical_url 更新错误: %s", repo.updates[0].CanonicalURL)
	}
}

func TestNullableTime(t *testing.T) {
	t.Parallel()

	if value := nullableTime(nil); value != (sql.NullTime{}) {
		t.Fatalf("空时间转换错误: %+v", value)
	}
}
