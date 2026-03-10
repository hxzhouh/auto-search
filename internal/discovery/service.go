package discovery

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"auto-search/internal/config"
	"auto-search/internal/content"
	"auto-search/internal/dedupe"
	"auto-search/internal/query"
	"auto-search/internal/resolver"
	"auto-search/internal/rss"
)

type Stats struct {
	Queries         int
	FeedItems       int
	Inserted        int
	URLDuplicates   int
	TitleDuplicates int
	ResolveFailures int
	FetchFailures   int
}

type Service struct {
	queryRepo    queryLister
	contentRepo  contentStore
	rssClient    rssFetcher
	linkResolver linkResolver
	requestDelay time.Duration
}

type queryLister interface {
	ListEnabled(ctx context.Context) ([]query.FeedQuery, error)
}

type contentStore interface {
	ExistsByURLHash(ctx context.Context, hash string) (bool, error)
	ExistsByTitleHash(ctx context.Context, hash string) (bool, error)
	InsertDiscovered(ctx context.Context, record content.DiscoverRecord) error
}

type rssFetcher interface {
	Fetch(ctx context.Context, q query.FeedQuery) ([]rss.Item, error)
}

type linkResolver interface {
	Resolve(ctx context.Context, rawURL string) (resolver.Result, error)
}

func NewService(db *sql.DB, cfg *config.Config) *Service {
	return &Service{
		queryRepo:    query.NewRepository(db),
		contentRepo:  content.NewRepository(db),
		rssClient:    rss.NewGoogleNewsClient(cfg.HTTP),
		linkResolver: resolver.NewGoogleNewsResolver(cfg.HTTP),
		requestDelay: time.Duration(cfg.HTTP.RequestIntervalMS) * time.Millisecond,
	}
}

func (s *Service) Run(ctx context.Context) (Stats, error) {
	items, err := s.queryRepo.ListEnabled(ctx)
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{Queries: len(items)}
	for _, feedQuery := range items {
		feedItems, err := s.rssClient.Fetch(ctx, feedQuery)
		if err != nil {
			stats.FetchFailures++
			continue
		}

		for _, item := range feedItems {
			stats.FeedItems++

			result, err := s.linkResolver.Resolve(ctx, item.Link)
			if err != nil {
				stats.ResolveFailures++
				continue
			}

			urlHash := dedupe.URLHash(result.CanonicalURL)
			titleHash := dedupe.TitleHash(item.Title)

			isURLDuplicate, err := s.contentRepo.ExistsByURLHash(ctx, urlHash)
			if err != nil {
				return stats, err
			}
			if isURLDuplicate {
				stats.URLDuplicates++
				continue
			}

			isTitleDuplicate, err := s.contentRepo.ExistsByTitleHash(ctx, titleHash)
			if err != nil {
				return stats, err
			}
			if isTitleDuplicate {
				stats.TitleDuplicates++
				continue
			}

			record, err := buildDiscoverRecord(item, result, urlHash, titleHash)
			if err != nil {
				return stats, err
			}
			if err := s.contentRepo.InsertDiscovered(ctx, record); err != nil {
				return stats, err
			}
			stats.Inserted++

			if s.requestDelay > 0 {
				select {
				case <-ctx.Done():
					return stats, ctx.Err()
				case <-time.After(s.requestDelay):
				}
			}
		}
	}

	return stats, nil
}

func buildDiscoverRecord(item rss.Item, result resolver.Result, urlHash, titleHash string) (content.DiscoverRecord, error) {
	publishedAt := parsePublishedAt(item.PublishedAtRaw)

	return content.DiscoverRecord{
		Source:         "google_news",
		QueryID:        item.QueryID,
		QueryText:      item.QueryText,
		RSSTitle:       item.Title,
		RSSLink:        item.Link,
		RSSSourceSite:  item.SourceSite,
		RSSSummary:     item.Description,
		RSSPublishedAt: publishedAt,
		FinalURL:       result.FinalURL,
		CanonicalURL:   result.CanonicalURL,
		URLHash:        urlHash,
		TitleHash:      titleHash,
		Status:         "pending",
		RawPayload: map[string]any{
			"query_name":       item.QueryName,
			"query_text":       item.QueryText,
			"query_lang":       item.QueryLang,
			"query_region":     item.QueryRegion,
			"rss_title":        item.Title,
			"rss_link":         item.Link,
			"rss_source_site":  item.SourceSite,
			"rss_summary":      item.Description,
			"rss_published_at": item.PublishedAtRaw,
			"guid":             item.GUID,
			"final_url":        result.FinalURL,
			"canonical_url":    result.CanonicalURL,
		},
	}, nil
}

func parsePublishedAt(raw string) sql.NullTime {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return sql.NullTime{}
	}

	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return sql.NullTime{Time: parsed, Valid: true}
		}
	}

	return sql.NullTime{}
}
