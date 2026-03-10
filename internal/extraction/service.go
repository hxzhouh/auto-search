package extraction

import (
	"context"
	"database/sql"
	"time"

	"auto-search/internal/config"
	"auto-search/internal/content"
	"auto-search/internal/dedupe"
	"auto-search/internal/extractor"
	"auto-search/internal/resolver"
)

type Stats struct {
	Selected  int
	Extracted int
	Failed    int
}

type pendingContentLister interface {
	ListPendingForExtraction(ctx context.Context, limit int) ([]content.PendingContent, error)
	UpdateExtractionResult(ctx context.Context, update content.ExtractionUpdate) error
}

type articleExtractor interface {
	Extract(ctx context.Context, targetURL string) (extractor.Article, error)
}

type articleResolver interface {
	Resolve(ctx context.Context, rawURL string) (resolver.Result, error)
}

type Service struct {
	contentRepo  pendingContentLister
	extractor    articleExtractor
	resolver     articleResolver
	requestDelay time.Duration
}

func NewService(db *sql.DB, cfg *config.Config) *Service {
	return &Service{
		contentRepo:  content.NewRepository(db),
		extractor:    extractor.NewDefuddleClient(cfg),
		resolver:     resolver.NewGoogleNewsResolver(cfg.HTTP),
		requestDelay: time.Duration(cfg.HTTP.RequestIntervalMS) * time.Millisecond,
	}
}

func (s *Service) Run(ctx context.Context, limit int) (Stats, error) {
	items, err := s.contentRepo.ListPendingForExtraction(ctx, limit)
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{Selected: len(items)}
	for _, item := range items {
		targetURL := item.CanonicalURL
		if targetURL == "" {
			targetURL = item.FinalURL
		}
		resolvedFinalURL := item.FinalURL
		resolvedCanonicalURL := targetURL
		if resolved, err := s.resolver.Resolve(ctx, targetURL); err == nil {
			resolvedFinalURL = resolved.FinalURL
			resolvedCanonicalURL = resolved.CanonicalURL
			targetURL = resolved.CanonicalURL
		}

		article, err := s.extractor.Extract(ctx, targetURL)
		if err != nil {
			stats.Failed++
			continue
		}

		update := content.ExtractionUpdate{
			ID:               item.ID,
			FinalURL:         resolvedFinalURL,
			CanonicalURL:     resolvedCanonicalURL,
			ArticleTitle:     firstNonEmpty(article.Title, item.RSSTitle),
			ArticleAuthor:    article.Author,
			ArticlePublished: nullableTime(article.PublishedAt),
			RawContentText:   article.Markdown,
			CleanedSummary:   article.Description,
			ContentHash:      dedupe.ContentHash(article.Markdown),
			Status:           "extracted",
		}
		if err := s.contentRepo.UpdateExtractionResult(ctx, update); err != nil {
			return stats, err
		}

		stats.Extracted++
		if s.requestDelay > 0 {
			select {
			case <-ctx.Done():
				return stats, ctx.Err()
			case <-time.After(s.requestDelay):
			}
		}
	}

	return stats, nil
}

func nullableTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
