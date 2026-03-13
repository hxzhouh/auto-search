ALTER TABLE contents ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_contents_hidden ON contents(hidden, status);
