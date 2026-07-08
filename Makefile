.PHONY: test test-web test-collector up down build

# Run all tests
test: test-collector test-web

# Run Python collector tests inside the collector container
# We install pytest on the fly if it's not installed, to avoid needing a full rebuild just for tests, 
# although requirements.txt already includes it for new builds.
test-collector:
	@echo "Running Collector Tests..."
	docker compose exec -T collector bash -c "pip install pytest && pytest tests/"

# Run Go web tests using a golang container (since web image doesn't have go toolchain)
test-web:
	@echo "Running Web Tests..."
	docker run --rm -v $(PWD)/web:/app -w /app golang:1.23 go test ./... -v

# Run the full stack
up:
	docker compose up -d

# Stop the full stack
down:
	docker compose down

# Rebuild and run the full stack
build:
	docker compose up -d --build
