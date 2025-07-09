# RSS Comb

[![CI/CD](https://github.com/lysyi3m/rss-comb/actions/workflows/ci.yml/badge.svg)](https://github.com/lysyi3m/rss-comb/actions/workflows/ci.yml)
[![Docker](https://img.shields.io/badge/docker-ghcr.io%2Flysyi3m%2Frss--comb-blue)](https://github.com/lysyi3m/rss-comb/pkgs/container/rss-comb)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

RSS Comb is a high-performance Go server application that acts as a proxy between existing RSS/Atom feeds and RSS reader applications. It provides feed normalization, deduplication, and content filtering capabilities through YAML-based configuration files.

## ✨ Features

- **🔄 Feed Normalization**: Converts RSS 1.0, RSS 2.0, and Atom feeds to standardized RSS 2.0 format
- **🔍 Content Deduplication**: Eliminates duplicate items based on content hashing
- **🎯 Flexible Filtering**: Configurable content filtering using include/exclude rules
- **⚡ Background Processing**: Automated feed updates with configurable refresh intervals
- **📊 Statistics & Monitoring**: Built-in stats endpoint and comprehensive logging
- **🔒 API Authentication**: Secure API endpoints with configurable access keys
- **🌐 Multi-format Support**: Handles various RSS/Atom feed formats seamlessly
- **🐳 Docker Ready**: Fully containerized with optimized multi-stage builds

## 🚀 Quick Start

### Using Docker (Recommended)

1. **Pull the image**:
   ```bash
   # Latest version
   docker pull ghcr.io/lysyi3m/rss-comb:latest
   
   # Specific version (recommended for production)
   docker pull ghcr.io/lysyi3m/rss-comb:1.0.0
   ```

2. **Run with Docker Compose**:
   ```bash
   # Create docker-compose.yml
   cat > docker-compose.yml << EOF
   version: '3.8'
   services:
     rss-comb:
       image: ghcr.io/lysyi3m/rss-comb:latest
       ports:
         - "8080:8080"
       environment:
         - DB_HOST=db
         - DB_USER=rss_user
         - DB_PASSWORD=rss_password
         - DB_NAME=rss_comb
       volumes:
         - ./feeds:/app/feeds:ro
       depends_on:
         - db
     
     db:
       image: postgres:15-alpine
       environment:
         - POSTGRES_DB=rss_comb
         - POSTGRES_USER=rss_user
         - POSTGRES_PASSWORD=rss_password
       volumes:
         - postgres_data:/var/lib/postgresql/data
   
   volumes:
     postgres_data:
   EOF
   
   # Start services
   docker-compose up -d
   ```

3. **Access your feeds**:
   - Statistics: `http://localhost:8080/stats`
   - Feed example: `http://localhost:8080/feeds/your-feed-id`

### Development Setup

1. **Prerequisites**:
   - Go 1.24+
   - PostgreSQL 17+
   - Docker & Docker Compose

2. **Clone and setup**:
   ```bash
   git clone https://github.com/lysyi3m/rss-comb.git
   cd rss-comb
   
   # Start database
   make db-up
   
   # Run application
   make run
   ```

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | localhost | Database host |
| `DB_PORT` | 5432 | Database port |
| `DB_USER` | rss_user | Database username |
| `DB_PASSWORD` | *required* | Database password |
| `DB_NAME` | rss_comb | Database name |
| `FEEDS_DIR` | ./feeds | Directory containing feed configuration files |
| `PORT` | 8080 | HTTP server port |
| `WORKER_COUNT` | 5 | Number of background workers |
| `SCHEDULER_INTERVAL` | 30 | Scheduler interval in seconds |
| `API_ACCESS_KEY` | *optional* | API access key for authentication |
| `USER_AGENT` | "RSS Comb/1.0" | User agent for HTTP requests |
| `TZ` | UTC | Timezone for timestamps |
| `DISABLE_MIGRATE` | false | Disable automatic database migrations |

### Feed Configuration

Create YAML configuration files in the `feeds/` directory:

```yaml
feed:
  id: "tech-news"              # Unique identifier for URL routing
  url: "https://example.com/feed.xml"
  title: "Tech News Feed"

settings:
  enabled: true
  deduplication: true
  refresh_interval: 1800       # 30 minutes (recommended)
  max_items: 50
  timeout: 30                  # seconds

filters:
  - field: "title"
    includes:
      - "technology"
      - "programming"
    excludes:
      - "advertisement"
      - "sponsored"
  - field: "description"
    excludes:
      - "clickbait"
```

**Key Configuration Notes:**
- The `id` field is used in URLs: `/feeds/<id>`
- Feed IDs must be unique and URL-safe
- Refresh intervals should be at least 30 minutes to respect source servers
- Filters support `title`, `description`, and `content` fields

## 🔌 API Endpoints

### Public Endpoints

- **`GET /feeds/<id>`** - Get processed RSS feed by feed ID
- **`GET /stats`** - Application statistics and health check

### Authenticated Endpoints

Require `X-API-Key` header or `Authorization: Bearer <token>`:

- **`GET /api/feeds`** - List all configured feeds with status
- **`GET /api/feeds/<id>/details`** - Detailed feed information and statistics
- **`POST /api/feeds/<id>/refilter`** - Re-apply filters to all feed items

### Example API Usage

```bash
# Get feed statistics
curl http://localhost:8080/stats

# Access processed feed
curl http://localhost:8080/feeds/tech-news

# List all feeds (with API key)
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/feeds

# Re-apply filters
curl -X POST -H "X-API-Key: your-api-key" http://localhost:8080/api/feeds/tech-news/refilter
```

## 🛠️ Development

### Available Commands

```bash
# Database management
make db-up          # Start PostgreSQL database
make db-down        # Stop database
make db-logs        # View database logs

# Development
make build          # Build the application
make run            # Run with auto-database startup
make test           # Run all tests

# Docker
make docker-build   # Build Docker image
make clean          # Clean project-specific artifacts, containers, and volumes
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test -v ./app/config
```

### Database Migrations

Database migrations are embedded in the application binary and run automatically on startup. To disable auto-migration:

```bash
DISABLE_MIGRATE=true make run
```

## 🚢 Deployment

### Production Deployment

RSS Comb uses automated CI/CD with GitHub Actions:

1. **Create a release tag**:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub Actions automatically**:
   - Runs tests
   - Builds multi-architecture Docker images
   - Pushes to GitHub Container Registry
   - Applies OpenContainers annotations

3. **Deploy the container**:
   ```bash
   # Available tags:
   # ghcr.io/lysyi3m/rss-comb:latest     (always latest)
   # ghcr.io/lysyi3m/rss-comb:1.0.0     (exact version)
   # ghcr.io/lysyi3m/rss-comb:1.0       (major.minor)
   # ghcr.io/lysyi3m/rss-comb:1         (major)
   
   docker pull ghcr.io/lysyi3m/rss-comb:1.0.0
   # Configure your environment and run
   ```

### Docker Image Features

- **Multi-stage builds** for minimal image size
- **Multi-architecture support** (amd64, arm64)
- **Non-root user** for security
- **Health checks** built-in
- **Optimized layer caching** for faster builds

## 🔧 Troubleshooting

### Common Issues

**Feed not updating:**
- Check `enabled: true` in feed configuration
- Verify `refresh_interval` is not too frequent
- Review scheduler logs for errors

**Missing feed items:**
- Review filter configuration for over-filtering
- Check `max_items` setting
- Examine `is_filtered` flags in database

**Database connection issues:**
- Verify database credentials
- Ensure PostgreSQL is running
- Check network connectivity

### Monitoring

- **Health check**: `GET /stats` should return 200 OK
- **Logs**: Application logs include detailed processing information
- **Database**: Monitor connection pool and query performance

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## 🔗 Links

- [GitHub Repository](https://github.com/lysyi3m/rss-comb)
- [Docker Images](https://github.com/lysyi3m/rss-comb/pkgs/container/rss-comb)
- [Issues & Bug Reports](https://github.com/lysyi3m/rss-comb/issues)

---

**Built with ❤️ in Go**