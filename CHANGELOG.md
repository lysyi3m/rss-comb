# Changelog

All notable changes to this project will be documented in this file.

## [2.2.0] - 2026-03-25

### Changed
- **FeedType interface** — replaced monolithic `Parse()`/`GenerateRSS()` with `FeedType` interface and three implementations: `basicType`, `podcastType`, `youtubeType`. Each type owns its parsing and RSS building logic.
- **Dissolved `services/` package** — `ProcessFeed` moved to `jobs/`, `ConfigSync` and `Refilter` moved to `feed/`. No more indirection layer.
- **Newest-item duplicate check** — replaced feed-level content hash (which triggered false positives on metadata-only changes) with a check on the newest parsed item. Dropped `content_hash` column from feeds table.
- **Feed type via config** — new `type` field in YAML config (`"youtube"`, `"podcast"`, or omit for basic). Replaces `extract_media` setting. New `feed_type` column in database.
- **Title override** — new `title` field in YAML config overrides source feed title. Existing `title` column renamed to `source_title`.
- Renamed `MediaExtraction`/`media_extraction` to `ExtractMedia`/`extract_media` for consistency, then replaced entirely by feed type system.
- Reduced log noise: removed per-worker, per-job, per-item, and per-file logs. Fixed 404 logged as error.
- Removed redundant tests, obvious comments, and trivial helper wrappers (`decodeHTMLEntities`, `isURL`).
- Simplified `normalizeWhitespace` with `strings.Fields`.

### Added
- `feed/feed_type.go` — `FeedType` interface with `ForType()` factory
- `feed/basic.go`, `feed/podcast.go`, `feed/youtube.go` — type-specific implementations
- `feed/helpers.go` — shared parsing/building utilities
- `feed/config_sync.go` — config sync to database (was `services/sync_feed_config.go`)
- `feed/refilter.go` — re-apply filters (was `services/refilter_feed.go`)
- `jobs/process.go`, `jobs/fetch.go` — feed processing logic (was `services/process_feed.go`)
- YouTube metadata extraction: `media:description`, `media:thumbnail`, inferred feed image and author
- Migrations 011 (title/source_title split), 012 (feed_type column), 013 (drop feed content_hash)

### Removed
- `app/services/` package
- `feed/parsing.go`, `feed/generator.go` (replaced by type-specific files)
- `ExtractMedia` setting from `Settings` struct
- Feed-level `content_hash` column and `GetFeedContentHash()` method
- Redundant test files: `database/connection_test.go`, `cfg/loader_test.go`, `feed/extraction_test.go`

## [2.1.0] - 2026-03-25

### Changed
- Renamed `MediaExtraction`/`media_extraction` to `ExtractMedia`/`extract_media` for consistency with `ExtractContent`.

## [2.0.0] - 2026-03-24

### Added
- **Job queue system** — PostgreSQL-backed with `FOR UPDATE SKIP LOCKED`, worker pool, scheduler.
- **Media downloading** — YouTube video to podcast audio conversion via yt-dlp with three-layer dedup (DB, filesystem, download).
- **Content extraction as async job** — items hidden until extraction completes.
- `WORKER_COUNT`, `MEDIA_DIR`, `YT_DLP_CMD`, `YT_DLP_ARGS`, `YT_DLP_UPDATE` environment variables.
- `/media/<filename>` endpoint for serving downloaded audio files.
- Immediate scheduler tick on startup.

### Changed
- Replaced synchronous ticker-based processing with job queue worker pool.
- Content extraction moved from inline to separate `extract_content` job type.
- yt-dlp bundled directly in Docker image (pip install + ffmpeg).

### Fixed
- `fetch_feed` retry storm (jobs with `max_retries=0` were never deleted on failure).
- Media cleanup now only deletes `.mp3` files (was deleting cookies files).

## [1.9.0] - 2026-03-24

### Added
- Job queue infrastructure (PostgreSQL-backed, `FOR UPDATE SKIP LOCKED`).
- Content extraction as separate job type with retry support.

## [1.8.0] - 2026-02-16

### Added
- **Regex pattern support** in filters — wrap patterns in `/slashes/` for regex matching.
- Automatically case-insensitive, compiled once and cached.
- Invalid regex falls back to literal substring matching.

## [1.7.2] - 2026-02-02

### Fixed
- Whitespace normalization (NBSP, tabs, newlines) in filter matching.
- Unicode normalization (NFC) for consistent Cyrillic character matching.

## [1.7.1] - 2026-01-11

### Fixed
- Feeds with NULL `next_fetch_at` now included in due feeds query.

## [1.7.0] - 2026-01-08

### Added
- Complete iTunes podcast RSS extension support (author, image, explicit, owner, duration, episode, season, type).
- iTunes namespace added conditionally only when podcast data is present.

## [1.6.0] - 2026-01-08

### Changed
- Improved content extraction with noise reduction (custom Readability settings, SVG removal).

## [1.5.2] - 2026-01-08

### Fixed
- Restored missing `/feeds/:name` RSS feed endpoint.

## [1.5.1] - 2026-01-07

### Fixed
- Added `link` field support in feed filter validation.

## [1.5.0] - 2026-01-07

### Changed
- Simplified configuration management, removed unnecessary wrapper functions.

## [1.4.0] - 2025-08-04

### Changed
- Replaced timestamp-based feed optimization with SHA-256 content hash comparison.

## [1.3.0] - 2025-08-03

### Added
- Feed timestamp optimization to skip processing for unchanged feeds.

## [1.2.0] - 2025-08-01

### Fixed
- Array filtering bug (categories/authors matched against joined strings instead of individual elements).
- Removed redundant `filter_reason` database field.

## [1.1.1] - 2025-07-31

### Fixed
- HTML entity decoding for feed titles and descriptions.

## [1.1.0] - 2025-07-24

### Added
- URL normalization to strip tracking parameters (UTM, fbclid, gclid, etc.) and prevent duplicate items.

## [1.0.0] - 2025-07-23

### Added
- Initial release: RSS/Atom feed proxy with normalization, deduplication, and YAML-based filtering.
- PostgreSQL storage with embedded migrations.
- Content extraction via go-readability.
- Docker multi-arch builds (amd64, arm64).
- Health endpoint and API authentication.
