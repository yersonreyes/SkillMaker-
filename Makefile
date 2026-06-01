# Makefile — orquestador del monorepo Angular + Go
.DEFAULT_GOAL := help
SHELL := /bin/bash

## ---------- Help ----------
.PHONY: help
help: ## Muestra esta ayuda
	@grep -E '^[a-zA-Z_:-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}'

## ---------- Setup ----------
.PHONY: install
install: backend-install frontend-install ## Instala dependencias de ambos workspaces

.PHONY: backend-install
backend-install: ## Descarga dependencias Go
	cd backend && go mod download

.PHONY: frontend-install
frontend-install: ## Instala dependencias npm del frontend
	cd frontend && npm install --legacy-peer-deps

## ---------- Dev ----------
.PHONY: dev
dev: ## Levanta backend (Air) + frontend (ng serve) en paralelo
	@command -v concurrently >/dev/null 2>&1 || npm i -g concurrently
	concurrently --names "BACKEND,FRONTEND" --prefix-colors "cyan,magenta" \
	  "make backend-dev" "make frontend-dev"

.PHONY: backend-dev
backend-dev: ## Backend con hot reload (Air)
	cd backend && air

.PHONY: frontend-dev
frontend-dev: ## Frontend en modo desarrollo (ng serve :4200)
	cd frontend && npm start

## ---------- Build ----------
.PHONY: build
build: backend-build frontend-build ## Compila ambos workspaces

.PHONY: backend-build
backend-build: ## Compila el binario de produccion (estatico, sin CGO)
	cd backend && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o dist/api ./cmd/api

.PHONY: frontend-build
frontend-build: ## Build de produccion del frontend
	cd frontend && npm run build:prod

## ---------- Tests ----------
.PHONY: test
test: backend-test frontend-test ## Ejecuta tests de ambos workspaces

.PHONY: backend-test
backend-test: ## Tests del backend con cobertura
	cd backend && go test ./... -race -coverprofile=coverage.out

.PHONY: backend-test-integration
backend-test-integration: ## Tests de integracion (testcontainers, mas lentos)
	cd backend && go test ./... -tags=integration

.PHONY: frontend-test
frontend-test: ## Tests del frontend (Vitest)
	cd frontend && npm test

## ---------- Lint ----------
.PHONY: lint
lint: backend-lint frontend-lint ## Lintea ambos workspaces

.PHONY: backend-lint
backend-lint: ## golangci-lint sobre todo el backend
	cd backend && golangci-lint run --timeout 5m

.PHONY: frontend-lint
frontend-lint: ## Angular ESLint
	cd frontend && npm run lint

## ---------- DB / Migraciones ----------
DB_URL ?= postgres://skillmaker:skillmaker@localhost:5432/skillmaker?sslmode=disable

.PHONY: db-up
db-up: ## Levanta PostgreSQL + MinIO de desarrollo
	cd backend && docker compose -f docker-compose.dev.yml up -d

.PHONY: db-down
db-down: ## Detiene PostgreSQL + MinIO de desarrollo
	cd backend && docker compose -f docker-compose.dev.yml down

.PHONY: db-reset
db-reset: ## Borra volumen de Postgres y vuelve a levantar
	cd backend && docker compose -f docker-compose.dev.yml down -v && \
	  docker compose -f docker-compose.dev.yml up -d

.PHONY: db-migrate
db-migrate: ## Aplica migraciones pendientes
	migrate -path backend/migrations -database "$(DB_URL)" up

.PHONY: db-migrate-down
db-migrate-down: ## Revierte la ultima migracion
	migrate -path backend/migrations -database "$(DB_URL)" down 1

.PHONY: db-migrate-new
db-migrate-new: ## Crea archivos .up.sql/.down.sql para una nueva migracion (name=...)
	@test -n "$(name)" || (echo "Uso: make db-migrate-new name=add_user_table"; exit 1)
	migrate create -ext sql -dir backend/migrations -seq $(name)

.PHONY: db-seed
db-seed: ## Pobla la BD con datos iniciales (ej: roles)
	cd backend && go run ./cmd/seed

## ---------- Docs ----------
.PHONY: swagger
swagger: ## Regenera el OpenAPI desde anotaciones (swaggo/swag)
	cd backend && swag init -g cmd/api/main.go -o docs

## ---------- Docker (produccion) ----------
.PHONY: docker-up
docker-up: ## Levanta los 5 servicios de produccion (postgres + minio + migrate + backend + frontend)
	docker compose up -d --build

.PHONY: docker-down
docker-down: ## Detiene los servicios de produccion
	docker compose down

.PHONY: docker-logs
docker-logs: ## Tail de logs de produccion
	docker compose logs -f
