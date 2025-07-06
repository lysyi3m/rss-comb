#!/bin/bash
set -e

echo "🚀 Deploying RSS Comb..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Load environment variables if .env file exists
if [ -f .env ]; then
    print_status "Loading environment variables from .env file..."
    export $(cat .env | grep -v '^#' | xargs)
    print_success "Environment variables loaded"
else
    print_warning "No .env file found. Using default values."
fi

# Check if required directories exist
if [ ! -d "feeds" ]; then
    print_warning "feeds/ directory not found. Creating it..."
    mkdir -p feeds
    echo "# Place your feed configuration files here" > feeds/README.md
fi

# Build the application
print_status "Building application..."
if ./scripts/build.sh; then
    print_success "Build completed successfully"
else
    print_error "Build failed"
    exit 1
fi

# Stop existing containers
print_status "Stopping existing containers..."
docker-compose -f docker-compose.prod.yml down --remove-orphans

# Pull latest base images
print_status "Pulling latest base images..."
docker-compose -f docker-compose.prod.yml pull db

# Start new containers
print_status "Starting containers..."
docker-compose -f docker-compose.prod.yml up -d

# Wait for services to be healthy
print_status "Waiting for services to be healthy..."
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if docker-compose -f docker-compose.prod.yml ps | grep -q "healthy"; then
        break
    fi
    
    attempt=$((attempt + 1))
    print_status "Waiting for services... (attempt $attempt/$max_attempts)"
    sleep 10
done

# Check service status
print_status "Checking service status..."
docker-compose -f docker-compose.prod.yml ps

# Run database migrations
print_status "Running database migrations..."
if docker-compose -f docker-compose.prod.yml run --rm migrations; then
    print_success "Database migrations completed"
else
    print_warning "Migration step completed (may have been already applied)"
fi

# Verify deployment
print_status "Verifying deployment..."
sleep 5

# Check if the application is responding
if curl -f -s http://localhost:${PORT:-8080}/health > /dev/null; then
    print_success "Application is responding to health checks"
    
    # Show application info
    print_status "Application information:"
    curl -s http://localhost:${PORT:-8080}/health | python3 -m json.tool 2>/dev/null || curl -s http://localhost:${PORT:-8080}/health
    
    print_success "Deployment completed successfully! 🎉"
    print_status "RSS Comb is running at http://localhost:${PORT:-8080}"
    print_status ""
    print_status "Available endpoints:"
    print_status "  Feed:          http://localhost:${PORT:-8080}/feeds/<id>"
    print_status "  Health check:  http://localhost:${PORT:-8080}/health"
    print_status "  Statistics:    http://localhost:${PORT:-8080}/stats"
    print_status "  List feeds:    http://localhost:${PORT:-8080}/api/v1/feeds"
    print_status "  Feed details:  http://localhost:${PORT:-8080}/api/v1/feeds/details?url=<feed-url>"
    print_status ""
    print_status "To view logs: docker-compose -f docker-compose.prod.yml logs -f"
    print_status "To stop: docker-compose -f docker-compose.prod.yml down"
else
    print_error "Application is not responding. Check logs:"
    print_error "docker-compose -f docker-compose.prod.yml logs app"
    exit 1
fi