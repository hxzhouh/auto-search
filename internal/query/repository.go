package query

import (
	"context"
	"database/sql"
	"fmt"
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
