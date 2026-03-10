package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type DiscoverRecord struct {
	Source         string
	QueryID        int64
	QueryText      string
	RSSTitle       string
	RSSLink        string
	RSSSourceSite  string
	RSSSummary     string
	RSSPublishedAt sql.NullTime
	FinalURL       string
	CanonicalURL   string
	URLHash        string
	TitleHash      string
	Status         string
	RawPayload     any
}

type PendingContent struct {
	ID           int64
	RSSTitle     string
	FinalURL     string
	CanonicalURL string
}

type CleaningCandidate struct {
	ID             int64
	RSSTitle       string
	ArticleTitle   string
	RSSSummary     string
	RSSSourceSite  string
	FinalURL       string
	CanonicalURL   string
	RawContentText string
}

type TagInput struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

type CleanedContent struct {
	ID               int64
	QueryText        string
	RSSTitle         string
	RSSSourceSite    string
	RSSPublishedAt   sql.NullTime
	FinalURL         string
	CanonicalURL     string
	CleanedTitle     string
	CleanedSummary   string
	CleanedContent   string
	Language         string
	ContentType      string
	QualityScore     int
	ImportanceScore  int
	WriteworthyScore int
	IsRelevant       bool
	AngleHint        string
	AIReason         string
	UpdatedAt        time.Time
	Tags             []TagInput
}

type ExtractionUpdate struct {
	ID               int64
	FinalURL         string
	CanonicalURL     string
	ArticleTitle     string
	ArticleAuthor    string
	ArticlePublished sql.NullTime
	RawContentText   string
	CleanedSummary   string
	ContentHash      string
	Status           string
}

type CleaningUpdate struct {
	ID               int64
	CleanedTitle     string
	CleanedSummary   string
	CleanedContent   string
	ContentHash      string
	Language         string
	ContentType      string
	QualityScore     int
	ImportanceScore  int
	WriteworthyScore int
	IsRelevant       bool
	AngleHint        string
	AIReason         string
	Status           string
	Tags             []TagInput
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ExistsByURLHash(ctx context.Context, hash string) (bool, error) {
	const sqlText = `SELECT 1 FROM contents WHERE url_hash = ? LIMIT 1`

	var value int
	err := r.db.QueryRowContext(ctx, sqlText, hash).Scan(&value)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("检查 url_hash 去重失败: %w", err)
	}
	return true, nil
}

func (r *Repository) ExistsByTitleHash(ctx context.Context, hash string) (bool, error) {
	const sqlText = `SELECT 1 FROM contents WHERE title_hash = ? LIMIT 1`

	var value int
	err := r.db.QueryRowContext(ctx, sqlText, hash).Scan(&value)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("检查 title_hash 去重失败: %w", err)
	}
	return true, nil
}

func (r *Repository) InsertDiscovered(ctx context.Context, record DiscoverRecord) error {
	payload, err := json.Marshal(record.RawPayload)
	if err != nil {
		return fmt.Errorf("序列化原始 payload 失败: %w", err)
	}

	const sqlText = `
INSERT INTO contents (
	source,
	query_id,
	query_text,
	rss_title,
	rss_link,
	rss_source_site,
	rss_summary,
	rss_published_at,
	final_url,
	canonical_url,
	url_hash,
	title_hash,
	status,
	raw_payload,
	created_at,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	now := time.Now()
	_, err = r.db.ExecContext(
		ctx,
		sqlText,
		record.Source,
		record.QueryID,
		record.QueryText,
		record.RSSTitle,
		record.RSSLink,
		record.RSSSourceSite,
		record.RSSSummary,
		record.RSSPublishedAt,
		record.FinalURL,
		record.CanonicalURL,
		record.URLHash,
		record.TitleHash,
		record.Status,
		string(payload),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("插入发现记录失败: %w", err)
	}

	return nil
}

func (r *Repository) ListPendingForExtraction(ctx context.Context, limit int) ([]PendingContent, error) {
	const baseSQL = `
SELECT id, rss_title, final_url, canonical_url
FROM contents
WHERE status = 'pending'
ORDER BY rss_published_at DESC, id ASC
`

	sqlText := baseSQL
	args := []any{}
	if limit > 0 {
		sqlText += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("查询待提取内容失败: %w", err)
	}
	defer rows.Close()

	items := make([]PendingContent, 0)
	for rows.Next() {
		var item PendingContent
		if err := rows.Scan(&item.ID, &item.RSSTitle, &item.FinalURL, &item.CanonicalURL); err != nil {
			return nil, fmt.Errorf("扫描待提取内容失败: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历待提取内容失败: %w", err)
	}

	return items, nil
}

func (r *Repository) UpdateExtractionResult(ctx context.Context, update ExtractionUpdate) error {
	const sqlText = `
UPDATE contents
SET
		final_url = ?,
		canonical_url = ?,
		article_title = ?,
		article_author = ?,
		article_published_at = ?,
		raw_content_text = ?,
	cleaned_summary = ?,
	content_hash = ?,
	status = ?,
	updated_at = ?
WHERE id = ?
`

	_, err := r.db.ExecContext(
		ctx,
		sqlText,
		update.FinalURL,
		update.CanonicalURL,
		update.ArticleTitle,
		update.ArticleAuthor,
		update.ArticlePublished,
		update.RawContentText,
		update.CleanedSummary,
		update.ContentHash,
		update.Status,
		time.Now(),
		update.ID,
	)
	if err != nil {
		return fmt.Errorf("更新提取结果失败: %w", err)
	}

	return nil
}

func (r *Repository) ListExtractedForCleaning(ctx context.Context, limit int) ([]CleaningCandidate, error) {
	const baseSQL = `
SELECT id, rss_title, article_title, rss_summary, rss_source_site, final_url, canonical_url, raw_content_text
FROM contents
WHERE status = 'extracted' AND raw_content_text IS NOT NULL AND raw_content_text != ''
ORDER BY updated_at ASC, id ASC
`

	sqlText := baseSQL
	args := []any{}
	if limit > 0 {
		sqlText += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("查询待清洗内容失败: %w", err)
	}
	defer rows.Close()

	items := make([]CleaningCandidate, 0)
	for rows.Next() {
		var item CleaningCandidate
		if err := rows.Scan(
			&item.ID,
			&item.RSSTitle,
			&item.ArticleTitle,
			&item.RSSSummary,
			&item.RSSSourceSite,
			&item.FinalURL,
			&item.CanonicalURL,
			&item.RawContentText,
		); err != nil {
			return nil, fmt.Errorf("扫描待清洗内容失败: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历待清洗内容失败: %w", err)
	}

	return items, nil
}

func (r *Repository) SaveCleaningResult(ctx context.Context, update CleaningUpdate) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启清洗事务失败: %w", err)
	}
	defer tx.Rollback()

	const updateSQL = `
UPDATE contents
SET
	cleaned_title = ?,
	cleaned_summary = ?,
	cleaned_content = ?,
	content_hash = ?,
	language = ?,
	content_type = ?,
	quality_score = ?,
	importance_score = ?,
	writeworthy_score = ?,
	is_relevant = ?,
	angle_hint = ?,
	ai_reason = ?,
	status = ?,
	updated_at = ?
WHERE id = ?
`

	isRelevant := 0
	if update.IsRelevant {
		isRelevant = 1
	}

	if _, err := tx.ExecContext(
		ctx,
		updateSQL,
		update.CleanedTitle,
		update.CleanedSummary,
		update.CleanedContent,
		update.ContentHash,
		update.Language,
		update.ContentType,
		update.QualityScore,
		update.ImportanceScore,
		update.WriteworthyScore,
		isRelevant,
		update.AngleHint,
		update.AIReason,
		update.Status,
		time.Now(),
		update.ID,
	); err != nil {
		return fmt.Errorf("更新清洗结果失败: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM content_tags WHERE content_id = ?`, update.ID); err != nil {
		return fmt.Errorf("清理旧标签关系失败: %w", err)
	}

	for _, item := range dedupeTags(update.Tags) {
		tagID, err := upsertTag(ctx, tx, item)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO content_tags (content_id, tag_id, created_at) VALUES (?, ?, ?)`,
			update.ID,
			tagID,
			time.Now(),
		); err != nil {
			return fmt.Errorf("写入标签关系失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交清洗事务失败: %w", err)
	}

	return nil
}

func (r *Repository) ListCleaned(ctx context.Context, limit int) ([]CleanedContent, error) {
	const baseSQL = `
SELECT
	id,
	query_text,
	rss_title,
	rss_source_site,
	rss_published_at,
	final_url,
	canonical_url,
	cleaned_title,
	cleaned_summary,
	cleaned_content,
	language,
	content_type,
	quality_score,
	importance_score,
	writeworthy_score,
	is_relevant,
	angle_hint,
	ai_reason,
	updated_at
FROM contents
WHERE status = 'cleaned'
ORDER BY updated_at DESC, id DESC
`

	sqlText := baseSQL
	args := []any{}
	if limit > 0 {
		sqlText += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("查询已清洗内容失败: %w", err)
	}
	defer rows.Close()

	items := make([]CleanedContent, 0)
	ids := make([]int64, 0)
	for rows.Next() {
		var item CleanedContent
		var isRelevant int
		if err := rows.Scan(
			&item.ID,
			&item.QueryText,
			&item.RSSTitle,
			&item.RSSSourceSite,
			&item.RSSPublishedAt,
			&item.FinalURL,
			&item.CanonicalURL,
			&item.CleanedTitle,
			&item.CleanedSummary,
			&item.CleanedContent,
			&item.Language,
			&item.ContentType,
			&item.QualityScore,
			&item.ImportanceScore,
			&item.WriteworthyScore,
			&isRelevant,
			&item.AngleHint,
			&item.AIReason,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描已清洗内容失败: %w", err)
		}
		item.IsRelevant = isRelevant == 1
		items = append(items, item)
		ids = append(ids, item.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历已清洗内容失败: %w", err)
	}

	if len(items) == 0 {
		return items, nil
	}

	tagMap, err := r.listTagsForContentIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].Tags = tagMap[items[i].ID]
	}

	return items, nil
}

func upsertTag(ctx context.Context, tx *sql.Tx, input TagInput) (int64, error) {
	var id int64
	err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM tags WHERE name = ? AND category = ? LIMIT 1`,
		input.Name,
		input.Category,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("查询标签失败: %w", err)
	}

	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO tags (name, category, created_at) VALUES (?, ?, ?)`,
		input.Name,
		input.Category,
		time.Now(),
	)
	if err != nil {
		err = tx.QueryRowContext(
			ctx,
			`SELECT id FROM tags WHERE name = ? AND category = ? LIMIT 1`,
			input.Name,
			input.Category,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("写入标签失败: %w", err)
		}
		return id, nil
	}

	id, err = result.LastInsertId()
	if err == nil && id > 0 {
		return id, nil
	}

	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM tags WHERE name = ? AND category = ? LIMIT 1`,
		input.Name,
		input.Category,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("回查标签失败: %w", err)
	}

	return id, nil
}

func dedupeTags(values []TagInput) []TagInput {
	set := make(map[string]struct{})
	result := make([]TagInput, 0, len(values))
	for _, value := range values {
		if value.Name == "" || value.Category == "" {
			continue
		}

		key := value.Category + ":" + value.Name
		if _, exists := set[key]; exists {
			continue
		}
		set[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (r *Repository) listTagsForContentIDs(ctx context.Context, ids []int64) (map[int64][]TagInput, error) {
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}

	sqlText := fmt.Sprintf(`
SELECT ct.content_id, t.name, t.category
FROM content_tags ct
JOIN tags t ON t.id = ct.tag_id
WHERE ct.content_id IN (%s)
ORDER BY ct.content_id ASC, t.category ASC, t.name ASC
`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("查询内容标签失败: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]TagInput, len(ids))
	for rows.Next() {
		var contentID int64
		var item TagInput
		if err := rows.Scan(&contentID, &item.Name, &item.Category); err != nil {
			return nil, fmt.Errorf("扫描内容标签失败: %w", err)
		}
		result[contentID] = append(result[contentID], item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历内容标签失败: %w", err)
	}

	return result, nil
}
