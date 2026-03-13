package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"auto-search/internal/config"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

func Open(cfg config.DatabaseConfig) (*sql.DB, error) {
	switch cfg.Driver {
	case "mysql":
		return openMySQL(cfg.MySQL)
	case "sqlite":
		return openSQLite(cfg.SQLite)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.Driver)
	}
}

func openMySQL(cfg config.MySQLConfig) (*sql.DB, error) {
	params := cfg.Params
	if params == "" {
		params = "charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai"
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		params,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("MySQL Ping 失败: %w", err)
	}

	return db, nil
}

func openSQLite(cfg config.SQLiteConfig) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
		return nil, fmt.Errorf("创建 sqlite 目录失败: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("连接 SQLite 失败: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("SQLite Ping 失败: %w", err)
	}

	return db, nil
}

func RunMigrations(ctx context.Context, db *sql.DB, driver string) error {
	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version TEXT NOT NULL PRIMARY KEY,
	applied_at DATETIME NOT NULL
)`); err != nil {
		return fmt.Errorf("创建迁移记录表失败: %w", err)
	}

	dir, err := resolveMigrationsDir(driver)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("读取迁移目录失败: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	slices.Sort(names)

	for _, name := range names {
		var applied int
		_ = db.QueryRowContext(ctx, `SELECT 1 FROM schema_migrations WHERE version = ?`, name).Scan(&applied)
		if applied == 1 {
			continue
		}

		path := filepath.Join(dir, name)
		sqlText, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取迁移文件失败: %s: %w", path, err)
		}

		statements := splitStatements(string(sqlText))
		for _, stmt := range statements {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("执行迁移失败: %s: %w", path, err)
			}
		}

		if _, err := db.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			name, time.Now(),
		); err != nil {
			return fmt.Errorf("记录迁移版本失败: %s: %w", name, err)
		}
	}

	return nil
}

func resolveMigrationsDir(driver string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("获取当前目录失败: %w", err)
	}

	current := wd
	for range 8 {
		candidate := filepath.Join(current, "migrations", driver)
		info, statErr := os.Stat(candidate)
		if statErr == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", fmt.Errorf("未找到迁移目录: migrations/%s", driver)
}

func splitStatements(sqlText string) []string {
	lines := strings.Split(sqlText, "\n")
	var builder strings.Builder
	var statements []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}

		builder.WriteString(line)
		builder.WriteString("\n")

		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(builder.String())
			stmt = strings.TrimSuffix(stmt, ";")
			if stmt != "" {
				statements = append(statements, stmt)
			}
			builder.Reset()
		}
	}

	if leftover := strings.TrimSpace(builder.String()); leftover != "" {
		statements = append(statements, leftover)
	}

	return statements
}
