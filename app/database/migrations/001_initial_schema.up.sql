-- RSS Comb Initial Database Schema
-- Creates the complete database structure for RSS feed processing

-- Enable UUID extension for primary keys
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Feeds table: stores feed metadata and processing status
CREATE TABLE feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,                    -- Feed identifier derived from configuration filename
    feed_url TEXT NOT NULL,                       -- RSS/Atom feed URL from configuration
    link TEXT,                                    -- Homepage URL from feed's <link> element
    title TEXT,                                   -- Feed title from RSS/Atom source
    description TEXT,                             -- Feed description from RSS/Atom source
    image_url TEXT,                               -- Feed image/logo URL
    language TEXT,                                -- Feed language code
    last_fetched_at TIMESTAMP,                    -- Last successful fetch attempt
    next_fetch_at TIMESTAMP,                      -- Scheduled time for next fetch
    feed_published_at TIMESTAMP,                  -- Feed's own publication date from RSS/Atom
    created_at TIMESTAMP DEFAULT NOW(),           -- Record creation time
    updated_at TIMESTAMP DEFAULT NOW()            -- Last successful processing time
);

-- Feed items table: stores individual feed items with content and metadata
CREATE TABLE feed_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID REFERENCES feeds(id) ON DELETE CASCADE,
    guid TEXT NOT NULL,                           -- Unique identifier from RSS/Atom
    link TEXT,                                    -- Item URL
    title TEXT,                                   -- Item title
    description TEXT,                             -- Item description/summary
    content TEXT,                                 -- Full item content (may be extracted)
    published_at TIMESTAMP NOT NULL,              -- Item publication date (required)
    updated_at TIMESTAMP,                         -- Item last modified date
    authors TEXT[],                               -- Authors in format "email (name)" or "name"
    categories TEXT[],                            -- Item categories/tags
    is_filtered BOOLEAN DEFAULT false,            -- Whether item was filtered out
    filter_reason TEXT,                           -- Reason for filtering
    content_hash TEXT NOT NULL,                   -- Hash for deduplication
    created_at TIMESTAMP DEFAULT NOW(),           -- Record creation time
    
    -- Content extraction tracking
    content_extracted_at TIMESTAMP,              -- When content was extracted
    content_extraction_status TEXT DEFAULT 'pending', -- pending, success, failed, skipped
    content_extraction_error TEXT,               -- Error message if extraction failed
    extraction_attempts INTEGER DEFAULT 0,       -- Number of extraction attempts
    
    -- RSS enclosure support (media attachments)
    enclosure_url TEXT,                          -- Enclosure URL
    enclosure_length BIGINT,                     -- Enclosure size in bytes
    enclosure_type TEXT,                         -- Enclosure MIME type
    
    UNIQUE(feed_id, guid)                        -- Prevent duplicate items per feed
);

-- Performance indexes
CREATE INDEX idx_content_hash ON feed_items(content_hash);
CREATE INDEX idx_feed_items_visible ON feed_items(feed_id, published_at DESC) 
    WHERE is_filtered = false;
CREATE INDEX idx_feeds_next_fetch_at ON feeds(next_fetch_at);
CREATE INDEX idx_content_extraction_status ON feed_items(content_extraction_status);
CREATE INDEX idx_extraction_attempts ON feed_items(extraction_attempts) 
    WHERE content_extraction_status = 'failed';