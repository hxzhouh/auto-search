package query

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type FeedQuery struct {
	ID        int64
	Name      string
	QueryText string
	Lang      string
	Region    string
	Enabled   bool
	Priority  int
	CreatedAt string
	UpdatedAt string
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListEnabled(ctx context.Context) ([]FeedQuery, error) {
	const sqlText = `
SELECT id, name, query_text, lang, region, enabled, priority, created_at, updated_at
FROM feed_queries
WHERE enabled = 1
ORDER BY priority DESC, id ASC
`

	rows, err := r.db.QueryContext(ctx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("查询启用 query 失败: %w", err)
	}
	defer rows.Close()

	items := make([]FeedQuery, 0)
	for rows.Next() {
		var item FeedQuery
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.QueryText,
			&item.Lang,
			&item.Region,
			&item.Enabled,
			&item.Priority,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描 query 失败: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 query 失败: %w", err)
	}

	return items, nil
}

func (r *Repository) ListAll(ctx context.Context) ([]FeedQuery, error) {
	const sqlText = `
SELECT id, name, query_text, lang, region, enabled, priority, created_at, updated_at
FROM feed_queries
ORDER BY priority DESC, id ASC
`

	rows, err := r.db.QueryContext(ctx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("查询所有 query 失败: %w", err)
	}
	defer rows.Close()

	items := make([]FeedQuery, 0)
	for rows.Next() {
		var item FeedQuery
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.QueryText,
			&item.Lang,
			&item.Region,
			&item.Enabled,
			&item.Priority,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描 query 失败: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) Insert(ctx context.Context, q FeedQuery) (int64, error) {
	now := time.Now().UTC().Format(time.DateTime)
	enabled := 0
	if q.Enabled {
		enabled = 1
	}

	result, err := r.db.ExecContext(ctx, `
INSERT INTO feed_queries (name, query_text, lang, region, enabled, priority, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		q.Name, q.QueryText, q.Lang, q.Region, enabled, q.Priority, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("插入 query 失败: %w", err)
	}

	return result.LastInsertId()
}

func (r *Repository) Update(ctx context.Context, q FeedQuery) error {
	now := time.Now().UTC().Format(time.DateTime)
	enabled := 0
	if q.Enabled {
		enabled = 1
	}

	_, err := r.db.ExecContext(ctx, `
UPDATE feed_queries
SET name=?, query_text=?, lang=?, region=?, enabled=?, priority=?, updated_at=?
WHERE id=?`,
		q.Name, q.QueryText, q.Lang, q.Region, enabled, q.Priority, now, q.ID,
	)
	if err != nil {
		return fmt.Errorf("更新 query 失败: %w", err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM feed_queries WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("删除 query 失败: %w", err)
	}

	return nil
}
