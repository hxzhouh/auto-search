ALTER TABLE contents ADD COLUMN hidden TINYINT(1) NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_contents_hidden ON contents(hidden, status);
