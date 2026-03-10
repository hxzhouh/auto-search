CREATE TABLE IF NOT EXISTS feed_queries (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    query_text VARCHAR(255) NOT NULL,
    lang VARCHAR(16) NOT NULL DEFAULT 'en',
    region VARCHAR(16) NOT NULL DEFAULT 'US',
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    priority INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_feed_queries_name (name)
);

CREATE TABLE IF NOT EXISTS contents (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    source VARCHAR(32) NOT NULL,
    query_id BIGINT UNSIGNED NULL,
    query_text VARCHAR(255) NOT NULL,
    rss_title VARCHAR(512) NOT NULL,
    rss_link TEXT NOT NULL,
    rss_source_site VARCHAR(255) NOT NULL DEFAULT '',
    rss_summary TEXT NULL,
    rss_published_at DATETIME NULL,
    final_url TEXT NULL,
    canonical_url TEXT NULL,
    url_hash CHAR(64) NOT NULL DEFAULT '',
    title_hash CHAR(64) NOT NULL DEFAULT '',
    content_hash CHAR(64) NOT NULL DEFAULT '',
    article_title VARCHAR(512) NOT NULL DEFAULT '',
    article_author VARCHAR(255) NOT NULL DEFAULT '',
    article_published_at DATETIME NULL,
    raw_html LONGTEXT NULL,
    raw_content_text LONGTEXT NULL,
    cleaned_title VARCHAR(512) NOT NULL DEFAULT '',
    cleaned_summary TEXT NULL,
    cleaned_content LONGTEXT NULL,
    language VARCHAR(16) NOT NULL DEFAULT '',
    content_type VARCHAR(64) NOT NULL DEFAULT '',
    quality_score INT NOT NULL DEFAULT 0,
    importance_score INT NOT NULL DEFAULT 0,
    writeworthy_score INT NOT NULL DEFAULT 0,
    is_relevant TINYINT(1) NOT NULL DEFAULT 0,
    angle_hint VARCHAR(255) NOT NULL DEFAULT '',
    ai_reason TEXT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    raw_payload JSON NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_contents_url_hash (url_hash),
    KEY idx_contents_query_id (query_id),
    KEY idx_contents_status (status),
    KEY idx_contents_published_at (rss_published_at),
    KEY idx_contents_relevant (is_relevant, importance_score)
);

CREATE TABLE IF NOT EXISTS tags (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    category VARCHAR(32) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_tags_name_category (name, category)
);

CREATE TABLE IF NOT EXISTS content_tags (
    content_id BIGINT UNSIGNED NOT NULL,
    tag_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (content_id, tag_id)
);

INSERT INTO feed_queries (name, query_text, lang, region, enabled, priority)
VALUES
    ('openai', 'OpenAI when:1d', 'en', 'US', 1, 100),
    ('anthropic', 'Anthropic when:1d', 'en', 'US', 1, 95),
    ('google_ai', 'Google AI when:1d', 'en', 'US', 1, 90),
    ('claude_code', '"Claude Code" when:1d', 'en', 'US', 1, 95),
    ('ai_coding', 'AI coding when:1d', 'en', 'US', 1, 90)
ON DUPLICATE KEY UPDATE
    query_text = VALUES(query_text),
    lang = VALUES(lang),
    region = VALUES(region),
    enabled = VALUES(enabled),
    priority = VALUES(priority);
