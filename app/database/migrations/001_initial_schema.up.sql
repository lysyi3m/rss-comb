-- RSS Comb SQLite Schema (consolidated from PostgreSQL migrations 001-014)

CREATE TABLE feeds (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT UNIQUE NOT NULL,
    feed_url TEXT NOT NULL,
    link TEXT,
    title TEXT,
    source_title TEXT,
    description TEXT,
    image_url TEXT,
    language TEXT,
    last_fetched_at TEXT,
    next_fetch_at TEXT,
    feed_published_at TEXT,
    feed_updated_at TEXT,
    feed_type TEXT NOT NULL DEFAULT '',
    is_enabled INTEGER NOT NULL DEFAULT 1,
    settings TEXT,
    filters TEXT,
    config_hash TEXT,
    itunes_author TEXT,
    itunes_image TEXT,
    itunes_explicit TEXT,
    itunes_owner_name TEXT,
    itunes_owner_email TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE feed_items (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    feed_id TEXT NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    guid TEXT NOT NULL,
    link TEXT,
    title TEXT,
    description TEXT,
    content TEXT,
    published_at TEXT NOT NULL,
    updated_at TEXT,
    authors TEXT DEFAULT '[]',
    categories TEXT DEFAULT '[]',
    is_filtered INTEGER NOT NULL DEFAULT 0,
    content_hash TEXT NOT NULL,
    enclosure_url TEXT,
    enclosure_length INTEGER,
    enclosure_type TEXT,
    itunes_duration INTEGER,
    itunes_episode INTEGER,
    itunes_season INTEGER,
    itunes_episode_type TEXT,
    itunes_image TEXT,
    content_extraction_status TEXT,
    media_status TEXT,
    media_path TEXT,
    media_size INTEGER,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(feed_id, guid)
);

CREATE TABLE jobs (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    job_type TEXT NOT NULL,
    feed_id TEXT NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    item_id TEXT REFERENCES feed_items(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    retries INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    run_after TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Performance indexes
CREATE INDEX idx_feeds_next_fetch_at ON feeds(next_fetch_at);
CREATE INDEX idx_feeds_is_enabled ON feeds(is_enabled);
CREATE INDEX idx_content_hash ON feed_items(content_hash);
CREATE INDEX idx_feed_items_visible ON feed_items(feed_id, published_at DESC)
    WHERE is_filtered = 0;
CREATE INDEX idx_feed_items_extraction_pending ON feed_items(id)
    WHERE content_extraction_status = 'pending';
CREATE INDEX idx_feed_items_media_pending ON feed_items(id)
    WHERE media_status = 'pending';
CREATE INDEX idx_feed_items_media_path ON feed_items(media_path)
    WHERE media_status = 'ready';
CREATE INDEX idx_jobs_pending ON jobs(created_at)
    WHERE status = 'pending';
CREATE INDEX idx_jobs_dedup ON jobs(feed_id, job_type, item_id)
    WHERE status IN ('pending', 'processing');
