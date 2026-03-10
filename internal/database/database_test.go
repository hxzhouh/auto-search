package database

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"auto-search/internal/config"
	"auto-search/internal/query"
)

func TestRunMigrationsAndListEnabledQueriesWithSQLite(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "auto-search.db")
	db, err := Open(config.DatabaseConfig{
		Driver: "sqlite",
		SQLite: config.SQLiteConfig{Path: dbPath},
	})
	if err != nil {
		t.Fatalf("打开 sqlite 失败: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RunMigrations(ctx, db, "sqlite"); err != nil {
		t.Fatalf("执行 sqlite 迁移失败: %v", err)
	}

	repo := query.NewRepository(db)
	items, err := repo.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("读取启用 query 失败: %v", err)
	}

	if len(items) != 5 {
		t.Fatalf("期望 5 条启用 query，实际为 %d", len(items))
	}
	if items[0].Name != "openai" {
		t.Fatalf("期望第一条 query 为 openai，实际为 %s", items[0].Name)
	}
	if items[2].QueryText != "\"Claude Code\" when:1d" {
		t.Fatalf("期望 claude_code query 正确转义，实际为 %s", items[2].QueryText)
	}
}
