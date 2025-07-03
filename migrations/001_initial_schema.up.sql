-- UP Migration
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_file TEXT UNIQUE NOT NULL,
    feed_url TEXT NOT NULL,
    feed_name TEXT,
    feed_icon_url TEXT,
    last_fetched TIMESTAMP,
    last_success TIMESTAMP,
    next_fetch TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE feed_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID REFERENCES feeds(id) ON DELETE CASCADE,
    guid TEXT NOT NULL,
    link TEXT,
    title TEXT,
    description TEXT,
    content TEXT,
    published_date TIMESTAMP,
    updated_date TIMESTAMP,
    author_name TEXT,
    author_email TEXT,
    categories TEXT[],
    is_duplicate BOOLEAN DEFAULT false,
    is_filtered BOOLEAN DEFAULT false,
    filter_reason TEXT,
    duplicate_of UUID,
    content_hash TEXT NOT NULL,
    raw_data JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(feed_id, guid)
);

CREATE INDEX idx_content_hash ON feed_items(content_hash);
CREATE INDEX idx_feed_items_visible ON feed_items(feed_id, published_date DESC)
    WHERE is_duplicate = false AND is_filtered = false;
CREATE INDEX idx_feeds_next_fetch ON feeds(next_fetch) WHERE is_active = true;