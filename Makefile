.PHONY: build run test swagger up down migrate-up migrate-down init

APP_NAME=quotation-service
BUILD_DIR=_build
DATABASE_URL ?= $(shell grep '^DATABASE_URL=' .env | cut -d= -f2-)

up:
	@echo "Запуск инфраструктуры (БД и Workspace)..."
	docker compose up -d --wait

init: up migrate-up run

build:
	@echo "Сборка приложения внутри workspace..."
	docker compose exec workspace go build -o $(BUILD_DIR)/$(APP_NAME) cmd/app/main.go

run: build
	@echo "Запуск приложения внутри workspace..."
	docker compose exec workspace ./$(BUILD_DIR)/$(APP_NAME)

down:
	@echo "Остановка всего стека..."
	docker compose down

migrate-up:
	@echo "Применение миграций..."
	docker compose exec workspace goose -dir ./migrations postgres "$(DATABASE_URL)" up

migrate-down:
	@echo "Откат миграций..."
	docker compose exec workspace goose -dir ./migrations postgres "$(DATABASE_URL)" down

test:
	@echo "Запуск тестов в Docker..."
	docker compose exec workspace go test -v ./...

swagger:
	@echo "Генерация Swagger документации в Docker..."
	docker compose exec workspace swag init -g cmd/app/main.go --parseInternal --parseDependency --dir ./
