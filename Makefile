.PHONY: infra-up infra-down setup build run dev clean admin admin-dev admin-build admin-start

BINARY := bin/memory-server
ADMIN_BINARY := bin/memory-admin
COMPOSE := docker compose -f docker/docker-compose.yml -p claude-memory
ENV := QDRANT_HOST=localhost QDRANT_PORT=6334 OLLAMA_URL=http://localhost:11434

infra-up:
	$(COMPOSE) up -d

infra-down:
	$(COMPOSE) down

setup: infra-up
	./scripts/setup.sh

build:
	go build -o $(BINARY) ./cmd/memory-server

run: build
	$(ENV) ./$(BINARY)

dev:
	$(ENV) go run ./cmd/memory-server

clean:
	rm -f $(BINARY) $(ADMIN_BINARY)
	$(COMPOSE) down -v

# Admin UI
admin:
	$(ENV) go run ./cmd/memory-admin

admin-dev:
	$(ENV) go run ./cmd/memory-admin &
	cd web && pnpm dev

admin-build:
	cd web && pnpm build
	go build -o $(ADMIN_BINARY) ./cmd/memory-admin

admin-start: admin-build
	$(ENV) ./$(ADMIN_BINARY)
