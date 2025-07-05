#!/bin/bash
set -e

echo "🚀 Building RSS Comb..."

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

# Check if required tools are installed
print_status "Checking prerequisites..."

if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
print_status "Go version: $GO_VERSION"

# Run tests
print_status "Running tests..."
if go test -v ./...; then
    print_success "All tests passed!"
else
    print_error "Tests failed. Please fix the issues before building."
    exit 1
fi

# Build binary
print_status "Building binary..."
if go build -o bin/rss-comb app/main.go; then
    print_success "Binary built successfully: bin/rss-comb"
else
    print_error "Failed to build binary"
    exit 1
fi

# Build Docker image
print_status "Building Docker image..."
if docker build -f Dockerfile -t rss-comb:latest .; then
    print_success "Docker image built successfully: rss-comb:latest"
else
    print_error "Failed to build Docker image"
    exit 1
fi

# Tag with version if provided
if [ -n "$1" ]; then
    VERSION_TAG="$1"
    print_status "Tagging image with version: $VERSION_TAG"
    docker tag rss-comb:latest rss-comb:$VERSION_TAG
    print_success "Tagged image: rss-comb:$VERSION_TAG"
fi

# Display image info
print_status "Docker image information:"
docker images rss-comb --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedAt}}\t{{.Size}}"

print_success "Build completed successfully! 🎉"
print_status "To run the application:"
print_status "  Local: ./bin/rss-comb"
print_status "  Docker: docker-compose -f docker-compose.prod.yml up"