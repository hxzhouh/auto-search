package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSQLiteConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"app": {"name": "auto-search", "env": "test"},
		"database": {
			"driver": "sqlite",
			"mysql": {
				"host": "127.0.0.1",
				"port": 3306,
				"user": "root",
				"database": "auto_search"
			},
			"sqlite": {"path": "./tmp/test.db"}
		},
		"http": {"timeout_seconds": 15},
		"ai": {"provider": "openai", "model": "gpt-4.1-mini"}
	}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Database.Driver != "sqlite" {
		t.Fatalf("期望驱动为 sqlite，实际为 %s", cfg.Database.Driver)
	}
	if cfg.HTTP.TimeoutSeconds != 15 {
		t.Fatalf("期望超时为 15，实际为 %d", cfg.HTTP.TimeoutSeconds)
	}
}
