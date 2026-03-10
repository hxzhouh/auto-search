package discovery

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"auto-search/internal/config"
	"auto-search/internal/content"
	"auto-search/internal/database"
	"auto-search/internal/query"
	"auto-search/internal/resolver"
	"auto-search/internal/rss"
)

type stubQueryRepo struct {
	items []query.FeedQuery
	err   error
}

func (s stubQueryRepo) ListEnabled(_ context.Context) ([]query.FeedQuery, error) {
	return s.items, s.err
}

type stubRSSFetcher struct {
	items []rss.Item
	err   error
}

func (s stubRSSFetcher) Fetch(_ context.Context, _ query.FeedQuery) ([]rss.Item, error) {
	return s.items, s.err
}

type stubResolver struct {
	result resolver.Result
	err    error
}

func (s stubResolver) Resolve(_ context.Context, _ string) (resolver.Result, error) {
	return s.result, s.err
}

func TestServiceRunInsertsAndDeduplicates(t *testing.T) {
	t.Parallel()

	db := openDiscoveryTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	service := &Service{
		queryRepo: stubQueryRepo{
			items: []query.FeedQuery{
				{ID: 1, Name: "openai", QueryText: "OpenAI when:1d", Lang: "en", Region: "US"},
			},
		},
		contentRepo: contentRepoForTest(db),
		rssClient: stubRSSFetcher{
			items: []rss.Item{
				{
					QueryID:        1,
					QueryName:      "openai",
					QueryText:      "OpenAI when:1d",
					QueryLang:      "en",
					QueryRegion:    "US",
					Title:          "OpenAI 发布新模型",
					Link:           "https://news.google.com/articles/1",
					Description:    "摘要",
					PublishedAtRaw: "Tue, 10 Mar 2026 10:00:00 +0800",
					SourceSite:     "Example",
				},
				{
					QueryID:        1,
					QueryName:      "openai",
					QueryText:      "OpenAI when:1d",
					QueryLang:      "en",
					QueryRegion:    "US",
					Title:          "OpenAI 发布新模型",
					Link:           "https://news.google.com/articles/2",
					Description:    "摘要2",
					PublishedAtRaw: "Tue, 10 Mar 2026 11:00:00 +0800",
					SourceSite:     "Example",
				},
			},
		},
		linkResolver: stubResolver{
			result: resolver.Result{
				FinalURL:     "https://example.com/article?id=1&utm_source=test",
				CanonicalURL: "https://example.com/article?id=1",
			},
		},
	}

	stats, err := service.Run(ctx)
	if err != nil {
		t.Fatalf("执行 discovery 失败: %v", err)
	}

	if stats.Inserted != 1 {
		t.Fatalf("期望插入 1 条，实际为 %d", stats.Inserted)
	}
	if stats.URLDuplicates != 1 {
		t.Fatalf("期望 URL 去重 1 条，实际为 %d", stats.URLDuplicates)
	}
}

func openDiscoveryTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "auto-search.db")
	db, err := database.Open(config.DatabaseConfig{
		Driver: "sqlite",
		SQLite: config.SQLiteConfig{Path: dbPath},
	})
	if err != nil {
		t.Fatalf("打开 sqlite 失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.RunMigrations(ctx, db, "sqlite"); err != nil {
		db.Close()
		t.Fatalf("执行迁移失败: %v", err)
	}

	return db
}

func contentRepoForTest(db *sql.DB) *content.Repository {
	return content.NewRepository(db)
}
