CREATE TABLE IF NOT EXISTS feed_queries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    query_text TEXT NOT NULL,
    lang TEXT NOT NULL DEFAULT 'en',
    region TEXT NOT NULL DEFAULT 'US',
    enabled INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS contents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL,
    query_id INTEGER,
    query_text TEXT NOT NULL,
    rss_title TEXT NOT NULL,
    rss_link TEXT NOT NULL,
    rss_source_site TEXT NOT NULL DEFAULT '',
    rss_summary TEXT,
    rss_published_at DATETIME,
    final_url TEXT,
    canonical_url TEXT,
    url_hash TEXT NOT NULL DEFAULT '',
    title_hash TEXT NOT NULL DEFAULT '',
    content_hash TEXT NOT NULL DEFAULT '',
    article_title TEXT NOT NULL DEFAULT '',
    article_author TEXT NOT NULL DEFAULT '',
    article_published_at DATETIME,
    raw_html TEXT,
    raw_content_text TEXT,
    cleaned_title TEXT NOT NULL DEFAULT '',
    cleaned_summary TEXT,
    cleaned_content TEXT,
    language TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT '',
    quality_score INTEGER NOT NULL DEFAULT 0,
    importance_score INTEGER NOT NULL DEFAULT 0,
    writeworthy_score INTEGER NOT NULL DEFAULT 0,
    is_relevant INTEGER NOT NULL DEFAULT 0,
    angle_hint TEXT NOT NULL DEFAULT '',
    ai_reason TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    raw_payload TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_contents_url_hash ON contents(url_hash);
CREATE INDEX IF NOT EXISTS idx_contents_query_id ON contents(query_id);
CREATE INDEX IF NOT EXISTS idx_contents_status ON contents(status);
CREATE INDEX IF NOT EXISTS idx_contents_published_at ON contents(rss_published_at);
CREATE INDEX IF NOT EXISTS idx_contents_relevant ON contents(is_relevant, importance_score);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, category)
);

CREATE TABLE IF NOT EXISTS content_tags (
    content_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
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
ON CONFLICT(name) DO UPDATE SET
    query_text = excluded.query_text,
    lang = excluded.lang,
    region = excluded.region,
    enabled = excluded.enabled,
    priority = excluded.priority,
    updated_at = CURRENT_TIMESTAMP;
