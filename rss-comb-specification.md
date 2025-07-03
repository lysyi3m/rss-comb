# RSS Comb - Engineering Specification Document

## Project Overview

RSS Comb is a server application that acts as a proxy between existing RSS/Atom feeds and RSS reader applications. It provides feed normalization, deduplication, and content filtering capabilities through YAML-based configuration files.

## Technical Requirements

- **Primary Language**: Go 1.24+
- **Database**: PostgreSQL 17+
- **Cache**: Redis 7+
- **Deployment**: Docker & Docker Compose
- **Supported Feed Formats**: RSS 1.0, RSS 2.0, Atom 1.0

---

## Phase 1: Development Environment Setup

### Objective
Set up the development environment and project structure with all necessary dependencies.

### Deliverables

1. **Project Structure**
```
rss-comb/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── database/
│   ├── feed/
│   ├── parser/
│   ├── cache/
│   └── api/
├── migrations/
│   └── 001_initial_schema.sql
├── feeds/
│   └── .gitkeep
├── docker/
│   └── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

2. **Go Module Initialization**
```bash
go mod init github.com/lysyi3m/rss-comb
```

3. **Required Dependencies**
```go
// go.mod dependencies
require (
    github.com/gin-gonic/gin v1.9.1
    github.com/lib/pq v1.10.9
    github.com/go-redis/redis/v9 v9.5.1
    github.com/golang-migrate/migrate/v4 v4.17.0
    gopkg.in/yaml.v3 v3.0.1
    github.com/google/uuid v1.6.0
    github.com/mmcdole/gofeed v1.2.1
)
```

4. **Docker Compose Configuration**
```yaml
version: '3.8'
services:
  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: rss_bridge
      POSTGRES_USER: rss_user
      POSTGRES_PASSWORD: rss_password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

5. **Makefile**
```makefile
.PHONY: dev-up dev-down test build run migrate

dev-up:
	docker-compose up -d db redis

dev-down:
	docker-compose down

test:
	go test -v ./...

build:
	go build -o bin/rss-comb cmd/server/main.go

run: dev-up
	go run cmd/server/main.go

migrate:
	migrate -path migrations -database "postgres://rss_user:rss_password@localhost:5432/rss_bridge?sslmode=disable" up
```

### Acceptance Criteria
- [ ] Project structure created
- [ ] All dependencies installed successfully
- [ ] Docker Compose starts PostgreSQL and Redis
- [ ] Basic Makefile commands work
- [ ] Can build empty Go application

---

## Phase 2: Database Schema and Migration System

### Objective
Implement database schema with migration system.

### Deliverables

1. **Initial Schema Migration** (`migrations/001_initial_schema.sql`)
```sql
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

-- DOWN Migration
DROP TABLE IF EXISTS feed_items;
DROP TABLE IF EXISTS feeds;
```

2. **Database Connection Package** (`internal/database/connection.go`)
```go
package database

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq"
)

type DB struct {
    *sql.DB
}

func NewConnection(host, port, user, password, dbname string) (*DB, error) {
    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        host, port, user, password, dbname)

    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }

    if err := db.Ping(); err != nil {
        return nil, err
    }

    return &DB{db}, nil
}
```

3. **Migration Runner**
```go
// cmd/migrate/main.go
package main

import (
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
    m, err := migrate.New(
        "file://migrations",
        "postgres://rss_user:rss_password@localhost:5432/rss_bridge?sslmode=disable")
    if err != nil {
        panic(err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        panic(err)
    }
}
```

### Acceptance Criteria
- [ ] Migration files created
- [ ] Can run migrations successfully
- [ ] Database connection package works
- [ ] Tables created with correct schema
- [ ] All indexes created

---

## Phase 3: Configuration System

### Objective
Implement YAML configuration loading and validation system.

### Deliverables

1. **Configuration Types** (`internal/config/types.go`)
```go
package config

import "time"

type FeedConfig struct {
    Feed     FeedInfo     `yaml:"feed"`
    Settings FeedSettings `yaml:"settings"`
    Filters  []Filter     `yaml:"filters"`
}

type FeedInfo struct {
    URL  string `yaml:"url"`
    Name string `yaml:"name"`
}

type FeedSettings struct {
    Enabled         bool          `yaml:"enabled"`
    Deduplication   bool          `yaml:"deduplication"`
    RefreshInterval time.Duration `yaml:"refresh_interval"`
    CacheDuration   time.Duration `yaml:"cache_duration"`
    MaxItems        int           `yaml:"max_items"`
    Timeout         time.Duration `yaml:"timeout"`
    UserAgent       string        `yaml:"user_agent"`
}

type Filter struct {
    Field    string   `yaml:"field"`
    Includes []string `yaml:"includes"`
    Excludes []string `yaml:"excludes"`
}
```

2. **Configuration Loader** (`internal/config/loader.go`)
```go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
    "gopkg.in/yaml.v3"
)

type Loader struct {
    feedsDir string
}

func NewLoader(feedsDir string) *Loader {
    return &Loader{feedsDir: feedsDir}
}

func (l *Loader) LoadAll() (map[string]*FeedConfig, error) {
    configs := make(map[string]*FeedConfig)

    files, err := filepath.Glob(filepath.Join(l.feedsDir, "*.yaml"))
    if err != nil {
        return nil, err
    }

    for _, file := range files {
        config, err := l.loadFile(file)
        if err != nil {
            return nil, fmt.Errorf("error loading %s: %w", file, err)
        }

        if err := l.validate(config); err != nil {
            return nil, fmt.Errorf("invalid config %s: %w", file, err)
        }

        configs[file] = config
    }

    return configs, nil
}

func (l *Loader) loadFile(path string) (*FeedConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var config FeedConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    // Set defaults
    if config.Settings.RefreshInterval == 0 {
        config.Settings.RefreshInterval = 3600 * time.Second
    }
    if config.Settings.CacheDuration == 0 {
        config.Settings.CacheDuration = 300 * time.Second
    }
    if config.Settings.MaxItems == 0 {
        config.Settings.MaxItems = 100
    }
    if config.Settings.Timeout == 0 {
        config.Settings.Timeout = 30 * time.Second
    }
    if config.Settings.UserAgent == "" {
        config.Settings.UserAgent = "RSS-Bridge/1.0"
    }

    return &config, nil
}

func (l *Loader) validate(config *FeedConfig) error {
    if config.Feed.URL == "" {
        return fmt.Errorf("feed URL is required")
    }
    if config.Feed.Name == "" {
        return fmt.Errorf("feed name is required")
    }

    validFields := map[string]bool{
        "title": true, "description": true, "content": true,
        "author": true, "link": true, "categories": true,
    }

    for _, filter := range config.Filters {
        if !validFields[filter.Field] {
            return fmt.Errorf("invalid filter field: %s", filter.Field)
        }
    }

    return nil
}
```

3. **Test Configuration File** (`feeds/example.yaml`)
```yaml
feed:
  url: "https://example.com/feed.xml"
  name: "Example Feed"

settings:
  enabled: true
  deduplication: true
  refresh_interval: 3600
  cache_duration: 300
  max_items: 50
  timeout: 30
  user_agent: "RSS-Bridge/1.0"

filters:
  - field: "title"
    includes:
      - "technology"
    excludes:
      - "advertisement"
```

### Acceptance Criteria
- [ ] Configuration types defined
- [ ] YAML files load successfully
- [ ] Validation catches invalid configurations
- [ ] Default values applied correctly
- [ ] Can load multiple configuration files

---

## Phase 4: Feed Parser Implementation

### Objective
Implement feed parsing for RSS 1.0, RSS 2.0, and Atom formats with normalization.

### Deliverables

1. **Parser Types** (`internal/parser/types.go`)
```go
package parser

import "time"

type FeedMetadata struct {
    Title       string
    Link        string
    Description string
    IconURL     string
    Language    string
    Updated     *time.Time
}

type NormalizedItem struct {
    GUID          string
    Title         string
    Link          string
    Description   string
    Content       string
    PublishedDate *time.Time
    UpdatedDate   *time.Time
    AuthorName    string
    AuthorEmail   string
    Categories    []string

    ContentHash   string
    IsDuplicate   bool
    IsFiltered    bool
    FilterReason  string
    DuplicateOf   *string

    RawData       map[string]interface{}
}
```

2. **Feed Parser** (`internal/parser/parser.go`)
```go
package parser

import (
    "bytes"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "github.com/mmcdole/gofeed"
)

type Parser struct {
    gofeedParser *gofeed.Parser
}

func NewParser() *Parser {
    return &Parser{
        gofeedParser: gofeed.NewParser(),
    }
}

func (p *Parser) Parse(data []byte) (*FeedMetadata, []NormalizedItem, error) {
    feed, err := p.gofeedParser.Parse(bytes.NewReader(data))
    if err != nil {
        return nil, nil, err
    }

    metadata := &FeedMetadata{
        Title:       feed.Title,
        Link:        feed.Link,
        Description: feed.Description,
        Language:    feed.Language,
    }

    if feed.Image != nil {
        metadata.IconURL = feed.Image.URL
    }

    if feed.UpdatedParsed != nil {
        metadata.Updated = feed.UpdatedParsed
    }

    items := make([]NormalizedItem, 0, len(feed.Items))
    for _, item := range feed.Items {
        normalized := p.normalizeItem(item)
        normalized.ContentHash = p.generateContentHash(normalized)
        items = append(items, normalized)
    }

    return metadata, items, nil
}

func (p *Parser) normalizeItem(item *gofeed.Item) NormalizedItem {
    normalized := NormalizedItem{
        GUID:        p.coalesce(item.GUID, item.Link),
        Title:       item.Title,
        Link:        item.Link,
        Description: item.Description,
        Content:     item.Content,
    }

    if item.PublishedParsed != nil {
        normalized.PublishedDate = item.PublishedParsed
    }

    if item.UpdatedParsed != nil {
        normalized.UpdatedDate = item.UpdatedParsed
    }

    if item.Author != nil {
        normalized.AuthorName = item.Author.Name
        normalized.AuthorEmail = item.Author.Email
    }

    if item.Categories != nil {
        normalized.Categories = item.Categories
    }

    // Convert to raw data map
    normalized.RawData = p.itemToMap(item)

    return normalized
}

func (p *Parser) generateContentHash(item NormalizedItem) string {
    content := fmt.Sprintf("%s|%s|%s",
        item.Title,
        item.Link,
        item.Description)

    hash := sha256.Sum256([]byte(content))
    return hex.EncodeToString(hash[:])
}

func (p *Parser) coalesce(values ...string) string {
    for _, v := range values {
        if v != "" {
            return v
        }
    }
    return ""
}

func (p *Parser) itemToMap(item *gofeed.Item) map[string]interface{} {
    // Implementation to convert item to map
    return make(map[string]interface{})
}
```

### Acceptance Criteria
- [ ] Can parse RSS 2.0 feeds
- [ ] Can parse Atom feeds
- [ ] Can parse RSS 1.0 feeds
- [ ] All fields normalized correctly
- [ ] Content hash generation works
- [ ] Feed metadata extracted including icons

---

## Phase 5: Feed Processing Engine

### Objective
Implement the core feed processing logic with filtering and deduplication.

### Deliverables

1. **Database Models** (`internal/database/models.go`)
```go
package database

import (
    "time"
)

type Feed struct {
    ID          string
    ConfigFile  string
    URL         string
    Name        string
    IconURL     string
    LastFetched *time.Time
    LastSuccess *time.Time
    NextFetch   *time.Time
    IsActive    bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Item struct {
    ID            string
    FeedID        string
    GUID          string
    Link          string
    Title         string
    Description   string
    Content       string
    PublishedDate *time.Time
    UpdatedDate   *time.Time
    AuthorName    string
    AuthorEmail   string
    Categories    []string
    IsDuplicate   bool
    IsFiltered    bool
    FilterReason  string
    DuplicateOf   *string
    ContentHash   string
    RawData       map[string]interface{}
    CreatedAt     time.Time
}
```

2. **Feed Repository** (`internal/database/feed_repository.go`)
```go
package database

import (
    "database/sql"
    "time"
)

type FeedRepository struct {
    db *DB
}

func NewFeedRepository(db *DB) *FeedRepository {
    return &FeedRepository{db: db}
}

func (r *FeedRepository) UpsertFeed(configFile, feedURL, feedName string) (string, error) {
    var feedID string
    err := r.db.QueryRow(`
        INSERT INTO feeds (config_file, feed_url, feed_name)
        VALUES ($1, $2, $3)
        ON CONFLICT (config_file)
        DO UPDATE SET
            feed_url = EXCLUDED.feed_url,
            feed_name = EXCLUDED.feed_name,
            updated_at = NOW()
        RETURNING id
    `, configFile, feedURL, feedName).Scan(&feedID)

    return feedID, err
}

func (r *FeedRepository) UpdateFeedMetadata(feedID string, iconURL string) error {
    _, err := r.db.Exec(`
        UPDATE feeds
        SET feed_icon_url = $2, last_success = NOW(), updated_at = NOW()
        WHERE id = $1
    `, feedID, iconURL)

    return err
}

func (r *FeedRepository) UpdateNextFetch(feedID string, nextFetch time.Time) error {
    _, err := r.db.Exec(`
        UPDATE feeds
        SET next_fetch = $2, last_fetched = NOW()
        WHERE id = $1
    `, feedID, nextFetch)

    return err
}

func (r *FeedRepository) GetFeedsDueForRefresh() ([]Feed, error) {
    rows, err := r.db.Query(`
        SELECT id, config_file, feed_url, feed_name
        FROM feeds
        WHERE is_active = true
          AND (next_fetch IS NULL OR next_fetch <= NOW())
        ORDER BY next_fetch
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var feeds []Feed
    for rows.Next() {
        var feed Feed
        err := rows.Scan(&feed.ID, &feed.ConfigFile, &feed.URL, &feed.Name)
        if err != nil {
            return nil, err
        }
        feeds = append(feeds, feed)
    }

    return feeds, rows.Err()
}

func (r *FeedRepository) GetFeedByConfigFile(configFile string) (*Feed, error) {
    var feed Feed
    err := r.db.QueryRow(`
        SELECT id, config_file, feed_url, feed_name, feed_icon_url
        FROM feeds
        WHERE config_file = $1
    `, configFile).Scan(&feed.ID, &feed.ConfigFile, &feed.URL, &feed.Name, &feed.IconURL)

    if err == sql.ErrNoRows {
        return nil, nil
    }

    return &feed, err
}
```

3. **Item Repository** (`internal/database/item_repository.go`)
```go
package database

import (
    "database/sql"
    "encoding/json"
    "github.com/lib/pq"
)

type ItemRepository struct {
    db *DB
}

func NewItemRepository(db *DB) *ItemRepository {
    return &ItemRepository{db: db}
}

func (r *ItemRepository) CheckDuplicate(contentHash, feedID string, global bool) (bool, *string, error) {
    var duplicateID sql.NullString
    var query string

    if global {
        query = `SELECT id FROM feed_items WHERE content_hash = $1 LIMIT 1`
        err := r.db.QueryRow(query, contentHash).Scan(&duplicateID)
        if err == sql.ErrNoRows {
            return false, nil, nil
        }
        if err != nil {
            return false, nil, err
        }
    } else {
        query = `SELECT id FROM feed_items WHERE feed_id = $1 AND content_hash = $2 LIMIT 1`
        err := r.db.QueryRow(query, feedID, contentHash).Scan(&duplicateID)
        if err == sql.ErrNoRows {
            return false, nil, nil
        }
        if err != nil {
            return false, nil, err
        }
    }

    id := duplicateID.String
    return true, &id, nil
}

func (r *ItemRepository) StoreItem(feedID string, item NormalizedItem) error {
    rawDataJSON, err := json.Marshal(item.RawData)
    if err != nil {
        return err
    }

    _, err = r.db.Exec(`
        INSERT INTO feed_items (
            feed_id, guid, link, title, description, content,
            published_date, updated_date, author_name, author_email,
            categories, is_duplicate, is_filtered, filter_reason,
            duplicate_of, content_hash, raw_data
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
        ON CONFLICT (feed_id, guid) DO NOTHING
    `, feedID, item.GUID, item.Link, item.Title, item.Description, item.Content,
        item.PublishedDate, item.UpdatedDate, item.AuthorName, item.AuthorEmail,
        pq.Array(item.Categories), item.IsDuplicate, item.IsFiltered, item.FilterReason,
        item.DuplicateOf, item.ContentHash, rawDataJSON)

    return err
}

func (r *ItemRepository) GetVisibleItems(feedID string, limit int) ([]Item, error) {
    rows, err := r.db.Query(`
        SELECT guid, title, link, description, content,
               published_date, author_name, categories
        FROM feed_items
        WHERE feed_id = $1
          AND is_duplicate = false
          AND is_filtered = false
        ORDER BY published_date DESC
        LIMIT $2
    `, feedID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var items []Item
    for rows.Next() {
        var item Item
        err := rows.Scan(
            &item.GUID, &item.Title, &item.Link, &item.Description,
            &item.Content, &item.PublishedDate, &item.AuthorName,
            pq.Array(&item.Categories))
        if err != nil {
            return nil, err
        }
        items = append(items, item)
    }

    return items, rows.Err()
}
```

4. **Feed Processor** (`internal/feed/processor.go`)
```go
package feed

import (
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"

    "github.com/lysyi3m/rss-comb/internal/config"
    "github.com/lysyi3m/rss-comb/internal/database"
    "github.com/lysyi3m/rss-comb/internal/parser"
)

type Processor struct {
    parser    *parser.Parser
    feedRepo  *database.FeedRepository
    itemRepo  *database.ItemRepository
    config    map[string]*config.FeedConfig
    client    *http.Client
}

func NewProcessor(p *parser.Parser, fr *database.FeedRepository,
                  ir *database.ItemRepository, configs map[string]*config.FeedConfig) *Processor {
    return &Processor{
        parser:   p,
        feedRepo: fr,
        itemRepo: ir,
        config:   configs,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (p *Processor) ProcessFeed(feedID, configFile string) error {
    config, ok := p.config[configFile]
    if !ok {
        return fmt.Errorf("configuration not found: %s", configFile)
    }

    if !config.Settings.Enabled {
        return nil
    }

    // Fetch feed
    data, err := p.fetchFeed(config.Feed.URL, config.Settings)
    if err != nil {
        return fmt.Errorf("failed to fetch feed: %w", err)
    }

    // Parse feed
    metadata, items, err := p.parser.Parse(data)
    if err != nil {
        return fmt.Errorf("failed to parse feed: %w", err)
    }

    // Update feed metadata
    if err := p.feedRepo.UpdateFeedMetadata(feedID, metadata.IconURL); err != nil {
        return fmt.Errorf("failed to update feed metadata: %w", err)
    }

    // Process items
    processedCount := 0
    for _, item := range items {
        // Check duplicates
        if config.Settings.Deduplication {
            isDup, dupID, err := p.itemRepo.CheckDuplicate(item.ContentHash, feedID, false)
            if err != nil {
                return fmt.Errorf("failed to check duplicate: %w", err)
            }
            if isDup {
                item.IsDuplicate = true
                item.DuplicateOf = dupID
            }
        }

        // Apply filters
        if !item.IsDuplicate {
            for _, filter := range config.Filters {
                if !p.matchFilter(item, filter) {
                    item.IsFiltered = true
                    item.FilterReason = fmt.Sprintf("Excluded by %s filter", filter.Field)
                    break
                }
            }
        }

        // Store item
        if err := p.itemRepo.StoreItem(feedID, item); err != nil {
            return fmt.Errorf("failed to store item: %w", err)
        }

        processedCount++
        if processedCount >= config.Settings.MaxItems {
            break
        }
    }

    // Update next fetch time
    nextFetch := time.Now().Add(config.Settings.RefreshInterval)
    return p.feedRepo.UpdateNextFetch(feedID, nextFetch)
}

func (p *Processor) fetchFeed(url string, settings config.FeedSettings) ([]byte, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("User-Agent", settings.UserAgent)

    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
    }

    return io.ReadAll(resp.Body)
}

func (p *Processor) matchFilter(item parser.NormalizedItem, filter config.Filter) bool {
    value := p.getFieldValue(item, filter.Field)

    // Check excludes first
    for _, exclude := range filter.Excludes {
        if strings.Contains(strings.ToLower(value), strings.ToLower(exclude)) {
            return false
        }
    }

    // If includes specified, at least one must match
    if len(filter.Includes) > 0 {
        for _, include := range filter.Includes {
            if strings.Contains(strings.ToLower(value), strings.ToLower(include)) {
                return true
            }
        }
        return false
    }

    return true
}

func (p *Processor) getFieldValue(item parser.NormalizedItem, field string) string {
    switch field {
    case "title":
        return item.Title
    case "description":
        return item.Description
    case "content":
        return item.Content
    case "author":
        return item.AuthorName
    case "link":
        return item.Link
    case "categories":
        return strings.Join(item.Categories, " ")
    default:
        return ""
    }
}
```

### Acceptance Criteria
- [ ] Feed fetching works with timeout
- [ ] Deduplication logic implemented
- [ ] Filter matching works correctly
- [ ] Items stored in database
- [ ] Next fetch time updated
- [ ] Respects max items limit

---

## Phase 6: Background Scheduler

### Objective
Implement background job scheduler for periodic feed updates.

### Deliverables

1. **Scheduler** (`internal/scheduler/scheduler.go`)
```go
package scheduler

import (
    "context"
    "log"
    "sync"
    "time"

    "github.com/lysyi3m/rss-comb/internal/database"
    "github.com/lysyi3m/rss-comb/internal/feed"
)

type Scheduler struct {
    processor    *feed.Processor
    feedRepo     *database.FeedRepository
    interval     time.Duration
    workerCount  int
    ctx          context.Context
    cancel       context.CancelFunc
    wg           sync.WaitGroup
}

func NewScheduler(processor *feed.Processor, feedRepo *database.FeedRepository,
                  interval time.Duration, workerCount int) *Scheduler {
    ctx, cancel := context.WithCancel(context.Background())
    return &Scheduler{
        processor:   processor,
        feedRepo:    feedRepo,
        interval:    interval,
        workerCount: workerCount,
        ctx:         ctx,
        cancel:      cancel,
    }
}

func (s *Scheduler) Start() {
    log.Println("Starting scheduler with", s.workerCount, "workers")

    // Start worker pool
    jobs := make(chan database.Feed, 100)

    for i := 0; i < s.workerCount; i++ {
        s.wg.Add(1)
        go s.worker(i, jobs)
    }

    // Start scheduler loop
    s.wg.Add(1)
    go s.schedulerLoop(jobs)
}

func (s *Scheduler) Stop() {
    log.Println("Stopping scheduler")
    s.cancel()
    s.wg.Wait()
}

func (s *Scheduler) schedulerLoop(jobs chan<- database.Feed) {
    defer s.wg.Done()

    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()

    // Process immediately on start
    s.enqueueDueFeeds(jobs)

    for {
        select {
        case <-s.ctx.Done():
            close(jobs)
            return
        case <-ticker.C:
            s.enqueueDueFeeds(jobs)
        }
    }
}

func (s *Scheduler) enqueueDueFeeds(jobs chan<- database.Feed) {
    feeds, err := s.feedRepo.GetFeedsDueForRefresh()
    if err != nil {
        log.Printf("Error getting due feeds: %v", err)
        return
    }

    for _, feed := range feeds {
        select {
        case jobs <- feed:
        case <-s.ctx.Done():
            return
        }
    }
}

func (s *Scheduler) worker(id int, jobs <-chan database.Feed) {
    defer s.wg.Done()

    for {
        select {
        case feed, ok := <-jobs:
            if !ok {
                return
            }

            log.Printf("Worker %d processing feed: %s", id, feed.Name)
            start := time.Now()

            if err := s.processor.ProcessFeed(feed.ID, feed.ConfigFile); err != nil {
                log.Printf("Worker %d error processing feed %s: %v", id, feed.Name, err)
            } else {
                log.Printf("Worker %d processed feed %s in %v", id, feed.Name, time.Since(start))
            }

        case <-s.ctx.Done():
            return
        }
    }
}
```

### Acceptance Criteria
- [ ] Scheduler starts and stops gracefully
- [ ] Worker pool processes feeds concurrently
- [ ] Feeds processed on schedule
- [ ] Errors logged but don't crash scheduler
- [ ] Immediate processing on startup

---

## Phase 7: Caching Layer

### Objective
Implement Redis caching for processed feeds.

### Deliverables

1. **Cache Client** (`internal/cache/redis.go`)
```go
package cache

import (
    "context"
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "time"
    "github.com/go-redis/redis/v9"
)

type Cache struct {
    client *redis.Client
    ctx    context.Context
}

func NewCache(addr string) (*Cache, error) {
    client := redis.NewClient(&redis.Options{
        Addr: addr,
    })

    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, err
    }

    return &Cache{
        client: client,
        ctx:    ctx,
    }, nil
}

func (c *Cache) Get(key string) (string, error) {
    val, err := c.client.Get(c.ctx, key).Result()
    if err == redis.Nil {
        return "", nil
    }
    return val, err
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }

    return c.client.Set(c.ctx, key, data, ttl).Err()
}

func (c *Cache) Delete(key string) error {
    return c.client.Del(c.ctx, key).Err()
}

func (c *Cache) GenerateFeedKey(feedURL string) string {
    hash := sha256.Sum256([]byte(feedURL))
    return fmt.Sprintf("feed:%x", hash)
}
```

### Acceptance Criteria
- [ ] Redis connection established
- [ ] Can cache and retrieve feed data
- [ ] TTL works correctly
- [ ] Cache keys generated consistently

---

## Phase 8: HTTP API Server

### Objective
Implement HTTP server with the feed endpoint.

### Deliverables

1. **RSS Generator** (`internal/api/rss_generator.go`)
```go
package api

import (
    "bytes"
    "encoding/xml"
    "fmt"
    "time"

    "github.com/lysyi3m/rss-comb/internal/database"
)

type RSSGenerator struct{}

func NewRSSGenerator() *RSSGenerator {
    return &RSSGenerator{}
}

func (g *RSSGenerator) Generate(feed database.Feed, items []database.Item) (string, error) {
    var buf bytes.Buffer

    buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
    buf.WriteString("\n")
    buf.WriteString(`<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">`)
    buf.WriteString("\n  <channel>\n")

    // Channel metadata
    buf.WriteString("    <title>")
    xml.EscapeText(&buf, []byte(feed.Name))
    buf.WriteString("</title>\n")

    buf.WriteString("    <link>")
    xml.EscapeText(&buf, []byte(feed.URL))
    buf.WriteString("</link>\n")

    buf.WriteString("    <description>Processed feed from ")
    xml.EscapeText(&buf, []byte(feed.URL))
    buf.WriteString("</description>\n")

    buf.WriteString("    <lastBuildDate>")
    buf.WriteString(time.Now().Format(time.RFC1123Z))
    buf.WriteString("</lastBuildDate>\n")

    buf.WriteString("    <generator>RSS-Bridge/1.0</generator>\n")

    // Feed icon if available
    if feed.IconURL != "" {
        buf.WriteString("    <image>\n")
        buf.WriteString("      <url>")
        xml.EscapeText(&buf, []byte(feed.IconURL))
        buf.WriteString("</url>\n")
        buf.WriteString("      <title>")
        xml.EscapeText(&buf, []byte(feed.Name))
        buf.WriteString("</title>\n")
        buf.WriteString("      <link>")
        xml.EscapeText(&buf, []byte(feed.URL))
        buf.WriteString("</link>\n")
        buf.WriteString("    </image>\n")
    }

    // Items
    for _, item := range items {
        g.writeItem(&buf, item)
    }

    buf.WriteString("  </channel>\n</rss>")

    return buf.String(), nil
}

func (g *RSSGenerator) writeItem(buf *bytes.Buffer, item database.Item) {
    buf.WriteString("    <item>\n")

    // GUID
    buf.WriteString(`      <guid isPermaLink="false">`)
    xml.EscapeText(buf, []byte(item.GUID))
    buf.WriteString("</guid>\n")

    // Title
    if item.Title != "" {
        buf.WriteString("      <title>")
        xml.EscapeText(buf, []byte(item.Title))
        buf.WriteString("</title>\n")
    }

    // Link
    if item.Link != "" {
        buf.WriteString("      <link>")
        xml.EscapeText(buf, []byte(item.Link))
        buf.WriteString("</link>\n")
    }

    // Description
    if item.Description != "" {
        buf.WriteString("      <description>")
        xml.EscapeText(buf, []byte(item.Description))
        buf.WriteString("</description>\n")
    }

    // Content
    if item.Content != "" {
        buf.WriteString("      <content:encoded><![CDATA[")
        buf.WriteString(item.Content)
        buf.WriteString("]]></content:encoded>\n")
    }

    // Published date
    if item.PublishedDate != nil {
        buf.WriteString("      <pubDate>")
        buf.WriteString(item.PublishedDate.Format(time.RFC1123Z))
        buf.WriteString("</pubDate>\n")
    }

    // Author
    if item.AuthorName != "" {
        buf.WriteString("      <author>")
        xml.EscapeText(buf, []byte(item.AuthorName))
        buf.WriteString("</author>\n")
    }

    // Categories
    for _, category := range item.Categories {
        buf.WriteString("      <category>")
        xml.EscapeText(buf, []byte(category))
        buf.WriteString("</category>\n")
    }

    buf.WriteString("    </item>\n")
}

func (g *RSSGenerator) GenerateEmpty(feedName string) string {
    return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>%s</title>
    <link>http://localhost:8080/feed</link>
    <description>Processed RSS Feed</description>
    <lastBuildDate>%s</lastBuildDate>
    <generator>RSS-Bridge/1.0</generator>
  </channel>
</rss>`, feedName, time.Now().Format(time.RFC1123Z))
}
```

2. **HTTP Handlers** (`internal/api/handlers.go`)
```go
package api

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/lysyi3m/rss-comb/internal/cache"
    "github.com/lysyi3m/rss-comb/internal/config"
    "github.com/lysyi3m/rss-comb/internal/database"
)

type Handler struct {
    feedRepo  *database.FeedRepository
    itemRepo  *database.ItemRepository
    cache     *cache.Cache
    generator *RSSGenerator
    configs   map[string]*config.FeedConfig
}

func NewHandler(fr *database.FeedRepository, ir *database.ItemRepository,
                c *cache.Cache, configs map[string]*config.FeedConfig) *Handler {
    return &Handler{
        feedRepo:  fr,
        itemRepo:  ir,
        cache:     c,
        generator: NewRSSGenerator(),
        configs:   configs,
    }
}

func (h *Handler) GetFeed(c *gin.Context) {
    feedURL := c.Query("url")
    if feedURL == "" {
        c.String(http.StatusBadRequest, "Missing 'url' parameter")
        return
    }

    // Check if feed is registered
    var configFile string
    var feedConfig *config.FeedConfig

    for file, cfg := range h.configs {
        if cfg.Feed.URL == feedURL {
            configFile = file
            feedConfig = cfg
            break
        }
    }

    // If not registered, redirect to original
    if feedConfig == nil {
        c.Redirect(http.StatusFound, feedURL)
        return
    }

    // Check cache
    cacheKey := h.cache.GenerateFeedKey(feedURL)
    cached, err := h.cache.Get(cacheKey)
    if err == nil && cached != "" {
        c.Header("Content-Type", "application/rss+xml; charset=utf-8")
        c.Header("X-Cache", "HIT")
        c.String(http.StatusOK, cached)
        return
    }

    // Get feed from database
    feed, err := h.feedRepo.GetFeedByConfigFile(configFile)
    if err != nil || feed == nil {
        // Feed not processed yet - return empty feed
        c.Header("Content-Type", "application/rss+xml; charset=utf-8")
        c.String(http.StatusOK, h.generator.GenerateEmpty(feedConfig.Feed.Name))
        return
    }

    // Get items
    items, err := h.itemRepo.GetVisibleItems(feed.ID, feedConfig.Settings.MaxItems)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to retrieve items")
        return
    }

    // Generate RSS
    rss, err := h.generator.Generate(*feed, items)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to generate RSS")
        return
    }

    // Cache the result
    if err := h.cache.Set(cacheKey, rss, feedConfig.Settings.CacheDuration); err != nil {
        // Log error but don't fail the request
    }

    c.Header("Content-Type", "application/rss+xml; charset=utf-8")
    c.Header("X-Cache", "MISS")
    c.String(http.StatusOK, rss)
}

func (h *Handler) HealthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status": "healthy",
        "feeds":  len(h.configs),
    })
}
```

3. **Server Setup** (`internal/api/server.go`)
```go
package api

import (
    "github.com/gin-gonic/gin"
)

func NewServer(handler *Handler) *gin.Engine {
    r := gin.Default()

    // Middleware
    r.Use(gin.Recovery())
    r.Use(gin.Logger())

    // Routes
    r.GET("/feed", handler.GetFeed)
    r.GET("/health", handler.HealthCheck)

    return r
}
```

### Acceptance Criteria
- [ ] HTTP server starts on port 8080
- [ ] /feed endpoint works with url parameter
- [ ] Unregistered feeds redirect correctly
- [ ] Empty RSS returned for not-yet-processed feeds
- [ ] Caching works correctly
- [ ] Health check endpoint responds

---

## Phase 9: Main Application

### Objective
Wire everything together in the main application.

### Deliverables

1. **Main Application** (`cmd/server/main.go`)
```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/lysyi3m/rss-comb/internal/api"
    "github.com/lysyi3m/rss-comb/internal/cache"
    "github.com/lysyi3m/rss-comb/internal/config"
    "github.com/lysyi3m/rss-comb/internal/database"
    "github.com/lysyi3m/rss-comb/internal/feed"
    "github.com/lysyi3m/rss-comb/internal/parser"
    "github.com/lysyi3m/rss-comb/internal/scheduler"
)

func main() {
    // Configuration
    dbHost := getEnv("DB_HOST", "localhost")
    dbPort := getEnv("DB_PORT", "5432")
    dbUser := getEnv("DB_USER", "rss_user")
    dbPass := getEnv("DB_PASSWORD", "rss_password")
    dbName := getEnv("DB_NAME", "rss_bridge")
    redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
    feedsDir := getEnv("FEEDS_DIR", "./feeds")
    port := getEnv("PORT", "8080")

    // Database connection
    log.Println("Connecting to database...")
    db, err := database.NewConnection(dbHost, dbPort, dbUser, dbPass, dbName)
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.Close()

    // Redis connection
    log.Println("Connecting to Redis...")
    cache, err := cache.NewCache(redisAddr)
    if err != nil {
        log.Fatal("Failed to connect to Redis:", err)
    }

    // Load configurations
    log.Println("Loading feed configurations...")
    loader := config.NewLoader(feedsDir)
    configs, err := loader.LoadAll()
    if err != nil {
        log.Fatal("Failed to load configurations:", err)
    }
    log.Printf("Loaded %d feed configurations", len(configs))

    // Initialize repositories
    feedRepo := database.NewFeedRepository(db)
    itemRepo := database.NewItemRepository(db)

    // Register feeds
    for configFile, cfg := range configs {
        feedID, err := feedRepo.UpsertFeed(configFile, cfg.Feed.URL, cfg.Feed.Name)
        if err != nil {
            log.Printf("Failed to register feed %s: %v", configFile, err)
            continue
        }
        log.Printf("Registered feed: %s (ID: %s)", cfg.Feed.Name, feedID)
    }

    // Initialize components
    parser := parser.NewParser()
    processor := feed.NewProcessor(parser, feedRepo, itemRepo, configs)

    // Start scheduler
    scheduler := scheduler.NewScheduler(processor, feedRepo, 30*time.Second, 5)
    scheduler.Start()
    defer scheduler.Stop()

    // Initialize HTTP server
    handler := api.NewHandler(feedRepo, itemRepo, cache, configs)
    server := api.NewServer(handler)

    // Start server in goroutine
    go func() {
        log.Printf("Starting HTTP server on port %s", port)
        if err := server.Run(":" + port); err != nil {
            log.Fatal("Failed to start server:", err)
        }
    }()

    // Wait for interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down...")
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

### Acceptance Criteria
- [ ] Application starts successfully
- [ ] All components initialized
- [ ] Feeds registered on startup
- [ ] Graceful shutdown works
- [ ] Environment variables respected

---

## Phase 10: Docker Packaging and Deployment

### Objective
Create production-ready Docker image and compose setup.

### Deliverables

1. **Production Dockerfile** (`docker/Dockerfile`)
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rss-comb cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/rss-comb .
RUN mkdir -p /app/feeds

EXPOSE 8080

CMD ["./rss-comb"]
```

2. **Production Docker Compose** (`docker-compose.prod.yml`)
```yaml
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: docker/Dockerfile
    restart: always
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=db
      - DB_PORT=5432
      - DB_USER=rss_user
      - DB_PASSWORD=${DB_PASSWORD:-rss_password}
      - DB_NAME=rss_bridge
      - REDIS_ADDR=redis:6379
      - FEEDS_DIR=/app/feeds
    volumes:
      - ./feeds:/app/feeds:ro
    depends_on:
      - db
      - redis

  db:
    image: postgres:15-alpine
    restart: always
    environment:
      - POSTGRES_DB=rss_bridge
      - POSTGRES_USER=rss_user
      - POSTGRES_PASSWORD=${DB_PASSWORD:-rss_password}
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    restart: always
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

3. **Build Script** (`scripts/build.sh`)
```bash
#!/bin/bash
set -e

echo "Building RSS Comb..."

# Run tests
echo "Running tests..."
go test -v ./...

# Build binary
echo "Building binary..."
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/rss-comb cmd/server/main.go

# Build Docker image
echo "Building Docker image..."
docker build -f docker/Dockerfile -t rss-comb:latest .

echo "Build complete!"
```

4. **Deployment Script** (`scripts/deploy.sh`)
```bash
#!/bin/bash
set -e

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | xargs)
fi

echo "Deploying RSS Comb..."

# Build the application
./scripts/build.sh

# Stop existing containers
docker-compose -f docker-compose.prod.yml down

# Start new containers
docker-compose -f docker-compose.prod.yml up -d

# Run migrations
echo "Running database migrations..."
docker-compose -f docker-compose.prod.yml exec app ./rss-comb migrate

echo "Deployment complete!"
echo "RSS Comb is running at http://localhost:8080"
```

### Acceptance Criteria
- [ ] Docker image builds successfully
- [ ] Container starts with docker-compose
- [ ] Feeds directory mounted correctly
- [ ] All services communicate properly
- [ ] Application accessible on port 8080

---

## Testing Guidelines

### Unit Tests

Each package should have corresponding test files:

1. **Config Loader Test** (`internal/config/loader_test.go`)
```go
package config

import (
    "os"
    "path/filepath"
    "testing"
)

func TestLoadValidConfig(t *testing.T) {
    // Create temp directory
    tempDir := t.TempDir()

    // Create test YAML file
    content := `
feed:
  url: "https://example.com/feed.xml"
  name: "Test Feed"
settings:
  enabled: true
  deduplication: true
`

    err := os.WriteFile(filepath.Join(tempDir, "test.yaml"), []byte(content), 0644)
    if err != nil {
        t.Fatal(err)
    }

    // Load configuration
    loader := NewLoader(tempDir)
    configs, err := loader.LoadAll()
    if err != nil {
        t.Fatal(err)
    }

    if len(configs) != 1 {
        t.Errorf("Expected 1 config, got %d", len(configs))
    }
}
```

2. **Parser Test** (`internal/parser/parser_test.go`)
```go
package parser

import (
    "testing"
)

func TestParseRSS2(t *testing.T) {
    rssData := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <description>Test Description</description>
    <item>
      <title>Test Item</title>
      <link>https://example.com/item</link>
      <description>Test Item Description</description>
      <guid>item-1</guid>
    </item>
  </channel>
</rss>`

    parser := NewParser()
    metadata, items, err := parser.Parse([]byte(rssData))
    if err != nil {
        t.Fatal(err)
    }

    if metadata.Title != "Test Feed" {
        t.Errorf("Expected title 'Test Feed', got '%s'", metadata.Title)
    }

    if len(items) != 1 {
        t.Errorf("Expected 1 item, got %d", len(items))
    }
}
```

### Integration Tests

1. **End-to-End Test** (`test/integration_test.go`)
```go
package test

import (
    "net/http"
    "testing"
    "time"
)

func TestFeedProcessing(t *testing.T) {
    // Start test server
    // Create test feed configuration
    // Wait for processing
    // Verify feed endpoint returns processed data
}
```

### Manual Testing Checklist

1. **Setup**
   - [ ] Run `docker-compose up -d`
   - [ ] Run migrations
   - [ ] Verify services are healthy

2. **Feed Configuration**
   - [ ] Create sample YAML in `feeds/` directory
   - [ ] Verify feed registration in logs
   - [ ] Check database for feed record

3. **Processing**
   - [ ] Wait for scheduler to process feed
   - [ ] Check logs for processing status
   - [ ] Verify items in database

4. **API Testing**
   - [ ] Access `/feed?url=<feed-url>`
   - [ ] Verify RSS output is valid
   - [ ] Check X-Cache header
   - [ ] Test unregistered feed redirect

5. **Filtering**
   - [ ] Create feed with filters
   - [ ] Verify filtered items marked correctly
   - [ ] Check only visible items in output

---

## Monitoring and Maintenance

### Logging

Application logs should include:
- Feed registration events
- Processing start/end with duration
- Errors with context
- Cache hits/misses
- HTTP request logs

### Metrics to Monitor

1. **Application Metrics**
   - Feed processing duration
   - Items processed per feed
   - Filter effectiveness
   - Deduplication rate

2. **System Metrics**
   - Database connection pool
   - Redis memory usage
   - HTTP response times
   - Worker pool utilization

### Maintenance Tasks

1. **Database**
   ```sql
   -- Clean up old items (optional)
   DELETE FROM feed_items
   WHERE created_at < NOW() - INTERVAL '30 days';

   -- Analyze tables for query optimization
   ANALYZE feeds;
   ANALYZE feed_items;
   ```

2. **Redis**
   ```bash
   # Monitor memory usage
   redis-cli INFO memory

   # Clear all cache (if needed)
   redis-cli FLUSHDB
   ```

---

## Troubleshooting Guide

### Common Issues

1. **Feed not updating**
   - Check feed configuration is enabled
   - Verify next_fetch time in database
   - Check scheduler logs for errors

2. **Items missing**
   - Check filter configuration
   - Verify deduplication settings
   - Look for filtered/duplicate flags in database

3. **Cache not working**
   - Verify Redis connection
   - Check cache duration setting
   - Monitor Redis memory usage

### Debug Queries

```sql
-- Check feed status
SELECT * FROM feeds WHERE feed_url = 'URL';

-- Count items by status
SELECT
    COUNT(*) as total,
    SUM(CASE WHEN is_filtered THEN 1 ELSE 0 END) as filtered,
    SUM(CASE WHEN is_duplicate THEN 1 ELSE 0 END) as duplicates
FROM feed_items WHERE feed_id = 'FEED_ID';

-- Recent processing activity
SELECT feed_name, last_fetched, last_success, next_fetch
FROM feeds
ORDER BY last_fetched DESC;
```

---

## Future Enhancements

1. **Authentication Support**
   - Basic Auth for feeds
   - API key management
   - OAuth support

2. **Advanced Features**
   - Feed merging
   - Content enrichment
   - Full-text search
   - Analytics dashboard

3. **Performance**
   - Horizontal scaling
   - Read replicas
   - CDN integration
   - Webhook notifications

4. **Operations**
   - Prometheus metrics
   - Grafana dashboards
   - Automated backups
   - Health check endpoints
