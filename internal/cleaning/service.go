package cleaning

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"auto-search/internal/ai"
	"auto-search/internal/config"
	"auto-search/internal/content"
	"auto-search/internal/dedupe"
)

type Stats struct {
	Selected int
	Cleaned  int
	Failed   int
}

type cleaningRepository interface {
	ListExtractedForCleaning(ctx context.Context, limit int) ([]content.CleaningCandidate, error)
	SaveCleaningResult(ctx context.Context, update content.CleaningUpdate) error
}

type cleaner interface {
	Clean(ctx context.Context, item content.CleaningCandidate) (ai.CleanResult, error)
}

type Service struct {
	repo      cleaningRepository
	cleaner   cleaner
	waitAfter time.Duration
}

func NewService(db *sql.DB, cfg *config.Config) (*Service, error) {
	client, err := ai.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Service{
		repo:      content.NewRepository(db),
		cleaner:   ai.NewCleaner(client),
		waitAfter: rateLimitInterval(cfg.AI.RPMLimit),
	}, nil
}

func (s *Service) Run(ctx context.Context, limit int) (Stats, error) {
	items, err := s.repo.ListExtractedForCleaning(ctx, limit)
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{Selected: len(items)}
	for _, item := range items {
		result, err := s.cleaner.Clean(ctx, item)
		if err != nil {
			fmt.Printf("clean 失败: id=%d err=%v\n", item.ID, err)
			stats.Failed++
			continue
		}

		update := content.CleaningUpdate{
			ID:               item.ID,
			CleanedTitle:     result.CleanedTitle,
			CleanedSummary:   result.CleanedSummary,
			CleanedContent:   result.CleanedContent,
			ContentHash:      dedupe.ContentHash(result.CleanedContent),
			Language:         result.Language,
			ContentType:      result.ContentType,
			QualityScore:     result.QualityScore,
			ImportanceScore:  result.ImportanceScore,
			WriteworthyScore: result.WriteworthyScore,
			IsRelevant:       result.IsRelevant,
			AngleHint:        result.AngleHint,
			AIReason:         result.Reason,
			Status:           "cleaned",
			Tags:             buildTags(result),
		}
		if err := s.repo.SaveCleaningResult(ctx, update); err != nil {
			return stats, err
		}
		stats.Cleaned++

		if s.waitAfter > 0 {
			select {
			case <-ctx.Done():
				return stats, ctx.Err()
			case <-time.After(s.waitAfter):
			}
		}
	}

	return stats, nil
}

func buildTags(result ai.CleanResult) []content.TagInput {
	tags := []content.TagInput{
		{Name: result.ContentType, Category: "type"},
		{Name: result.Language, Category: "language"},
	}

	for _, item := range result.Companies {
		tags = append(tags, content.TagInput{Name: item, Category: "company"})
	}
	for _, item := range result.Products {
		tags = append(tags, content.TagInput{Name: item, Category: "product"})
	}
	for _, item := range result.Topics {
		tags = append(tags, content.TagInput{Name: item, Category: "topic"})
	}

	return tags
}

func rateLimitInterval(rpm int) time.Duration {
	if rpm <= 0 {
		return 0
	}

	base := time.Minute / time.Duration(rpm)
	return base + 250*time.Millisecond
}
