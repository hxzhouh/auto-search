package cleaning

import (
	"context"
	"testing"

	"auto-search/internal/ai"
	"auto-search/internal/content"
)

type stubCleaningRepo struct {
	items   []content.CleaningCandidate
	updates []content.CleaningUpdate
	listErr error
	saveErr error
}

func (s *stubCleaningRepo) ListExtractedForCleaning(_ context.Context, _ int) ([]content.CleaningCandidate, error) {
	return s.items, s.listErr
}

func (s *stubCleaningRepo) SaveCleaningResult(_ context.Context, update content.CleaningUpdate) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.updates = append(s.updates, update)
	return nil
}

type stubCleaner struct {
	result ai.CleanResult
	err    error
}

func (s stubCleaner) Clean(_ context.Context, _ content.CleaningCandidate) (ai.CleanResult, error) {
	return s.result, s.err
}

func TestServiceRun(t *testing.T) {
	t.Parallel()

	repo := &stubCleaningRepo{
		items: []content.CleaningCandidate{
			{
				ID:             1,
				RSSTitle:       "RSS 标题",
				ArticleTitle:   "文章标题",
				RawContentText: "正文内容",
			},
		},
	}

	service := &Service{
		repo: repo,
		cleaner: stubCleaner{
			result: ai.CleanResult{
				CleanedTitle:     "清洗标题",
				CleanedSummary:   "摘要",
				CleanedContent:   "清洗正文",
				Language:         "en",
				ContentType:      "news",
				Companies:        []string{"openai"},
				Topics:           []string{"ai_coding"},
				QualityScore:     8,
				ImportanceScore:  9,
				WriteworthyScore: 7,
				IsRelevant:       true,
				AngleHint:        "角度",
				Reason:           "原因",
			},
		},
	}

	stats, err := service.Run(context.Background(), 5)
	if err != nil {
		t.Fatalf("执行清洗失败: %v", err)
	}

	if stats.Selected != 1 || stats.Cleaned != 1 || stats.Failed != 0 {
		t.Fatalf("统计错误: %+v", stats)
	}
	if len(repo.updates) != 1 {
		t.Fatalf("更新次数错误: %d", len(repo.updates))
	}
	if repo.updates[0].Status != "cleaned" {
		t.Fatalf("状态错误: %s", repo.updates[0].Status)
	}
	if len(repo.updates[0].Tags) < 3 {
		t.Fatalf("标签数量错误: %+v", repo.updates[0].Tags)
	}
}
