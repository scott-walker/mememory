.PHONY: infra-up infra-down setup build run dev clean admin admin-dev admin-build admin-start cli

BINARY := bin/mememory-server
ADMIN_BINARY := bin/mememory-admin
CLI_BINARY := bin/mememory
COMPOSE := docker compose -f docker/docker-compose.yml -p mememory
# For local dev only. Production uses .env via mememory setup.
ENV := DATABASE_URL=postgres://mememory:mememory@localhost:5432/mememory?sslmode=disable OLLAMA_URL=http://localhost:11434

infra-up:
	$(COMPOSE) up -d

infra-down:
	$(COMPOSE) down

setup: infra-up
	@echo "Waiting for Ollama to be healthy..."
	@for i in $$(seq 1 30); do \
		status=$$(docker inspect --format '{{.State.Health.Status}}' mememory-ollama 2>/dev/null); \
		[ "$$status" = "healthy" ] && break; \
		[ $$i -eq 30 ] && echo "ERROR: Ollama not healthy after 60s" && exit 1; \
		sleep 2; \
	done
	docker exec mememory-ollama ollama pull nomic-embed-text
	@echo "Setup complete."

build:
	go build -o $(BINARY) ./cmd/mememory-server

cli:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(CLI_BINARY) ./cmd/mememory

run: build
	$(ENV) ./$(BINARY)

dev:
	$(ENV) go run ./cmd/mememory-server

clean:
	rm -f $(BINARY) $(ADMIN_BINARY) $(CLI_BINARY)
	$(COMPOSE) down

# Admin UI
admin:
	$(ENV) go run ./cmd/mememory-admin

admin-dev:
	$(ENV) go run ./cmd/mememory-admin &
	cd web && pnpm dev

admin-build:
	cd web && pnpm build
	go build -o $(ADMIN_BINARY) ./cmd/mememory-admin

admin-start: admin-build
	$(ENV) ./$(ADMIN_BINARY)
