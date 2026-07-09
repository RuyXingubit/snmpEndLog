#!/bin/bash
set -e

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Error: Version argument is required"
    exit 1
fi

echo "Building and pushing images for version: $VERSION"

# Build and push nms-web
docker buildx build --platform linux/amd64,linux/arm64 -t xingubit/nms-web:$VERSION -t xingubit/nms-web:latest --push ./web

# Build and push nms-collector (using root context to include db folder)
docker buildx build --platform linux/amd64,linux/arm64 -f collector/Dockerfile -t xingubit/nms-collector:$VERSION -t xingubit/nms-collector:latest --push .

# Build and push nms-db
docker buildx build --platform linux/amd64,linux/arm64 -t xingubit/nms-db:$VERSION -t xingubit/nms-db:latest --push ./db

echo "Successfully built and pushed all images for version $VERSION"
