package webui

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"auto-search/internal/config"
	"auto-search/internal/content"
	"auto-search/internal/database"
)

func TestHandleCleaned(t *testing.T) {
	t.Parallel()

	db := openWebUITestDB(t)
	defer db.Close()

	repo := content.NewRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := repo.SaveCleaningResult(ctx, content.CleaningUpdate{
		ID:               insertCleanedSeed(t, db),
		CleanedTitle:     "测试标题",
		CleanedSummary:   "测试摘要",
		CleanedContent:   "测试正文",
		ContentHash:      "hash",
		Language:         "zh",
		ContentType:      "news",
		QualityScore:     8,
		ImportanceScore:  9,
		WriteworthyScore: 7,
		IsRelevant:       true,
		AngleHint:        "测试角度",
		AIReason:         "测试原因",
		Status:           "cleaned",
		Tags: []content.TagInput{
			{Name: "openai", Category: "company"},
			{Name: "ai", Category: "topic"},
		},
	}); err != nil {
		t.Fatalf("写入 cleaned 数据失败: %v", err)
	}

	server := NewServer(db)
	req := httptest.NewRequest(http.MethodGet, "/api/cleaned?limit=10", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码错误: %d", recorder.Code)
	}

	var items []cleanedItemResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &items); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("期望 1 条数据，实际为 %d", len(items))
	}
	if items[0].CleanedTitle != "测试标题" {
		t.Fatalf("标题错误: %s", items[0].CleanedTitle)
	}
	if len(items[0].Tags) != 2 {
		t.Fatalf("标签数量错误: %+v", items[0].Tags)
	}
}

func TestHandleIndex(t *testing.T) {
	t.Parallel()

	server := NewServer(nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码错误: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "只看已清洗的素材") {
		t.Fatalf("页面内容错误: %s", recorder.Body.String())
	}
}

func openWebUITestDB(t *testing.T) *sql.DB {
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

func insertCleanedSeed(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	result, err := db.Exec(`
INSERT INTO contents (
	source, query_text, rss_title, rss_link, rss_source_site, url_hash, title_hash, content_hash,
	final_url, canonical_url, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		"google_news",
		"OpenAI when:1d",
		"原始标题",
		"https://news.google.com/test",
		"Example",
		"urlhash",
		"titlehash",
		"seedhash",
		"https://example.com/article",
		"https://example.com/article",
		"extracted",
	)
	if err != nil {
		t.Fatalf("插入 seed 失败: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("读取 seed id 失败: %v", err)
	}
	return id
}
