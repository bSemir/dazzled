SHELL := /bin/sh
GO ?= go
PNPM ?= pnpm

.PHONY: setup docker-up docker-down dev frontend-dev backend-dev migrate test test-backend test-frontend typecheck vet build lint clean

setup:
	@command -v $(PNPM) >/dev/null 2>&1 || { echo "pnpm is required for local development (with Node.js 22+). Install pnpm 12.3.4, or use 'make docker-up' to run with Docker Compose."; exit 127; }
	@command -v $(GO) >/dev/null 2>&1 || { echo "Go 1.26.10 is required for local development. Install Go, or use 'make docker-up' to run with Docker Compose."; exit 127; }
	cd frontend && $(PNPM) install --frozen-lockfile
	cd backend && $(GO) mod download

docker-up:
	docker compose up --build -d
	@echo "Dazzled is starting at http://localhost:8080"

docker-down:
	docker compose down

dev:
	@echo "Run 'make backend-dev' and 'make frontend-dev' in separate terminals."

frontend-dev:
	set -a; [ ! -f .env ] || . ./.env; set +a; cd frontend && $(PNPM) dev

backend-dev:
	set -a; [ ! -f .env ] || . ./.env; set +a; cd backend && DATABASE_PATH="$${DATABASE_PATH:-../data/dazzled.db}" WEB_DIR="$${WEB_DIR:-../frontend/out}" $(GO) run ./cmd/server

migrate:
	set -a; [ ! -f .env ] || . ./.env; set +a; cd backend && DATABASE_PATH="$${DATABASE_PATH:-../data/dazzled.db}" $(GO) run ./cmd/server --migrate-only

test: test-backend test-frontend typecheck

test-backend:
	cd backend && $(GO) test -race ./...

test-frontend:
	cd frontend && $(PNPM) test

typecheck:
	cd frontend && $(PNPM) typecheck

lint:
	cd backend && test -z "$$($(GO) fmt ./...)"
	cd backend && $(GO) vet ./...
	cd frontend && $(PNPM) lint
	cd frontend && $(PNPM) typecheck

vet:
	cd backend && $(GO) vet ./...

build:
	cd frontend && $(PNPM) build
	cd backend && CGO_ENABLED=1 $(GO) build -trimpath -ldflags='-s -w' -o ../dist/dazzled ./cmd/server

clean:
	rm -rf frontend/.next frontend/out frontend/tsconfig.tsbuildinfo dist
