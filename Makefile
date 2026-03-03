.PHONY: build test clean run docker-build docker-run lint help

# Имя бинарника
BINARY_NAME=projectT

# Версия (из git tag или default)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Go параметры
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Путь к main.go
CMD_PATH=./cmd/main.go

help: ## Показать эту справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Собрать приложение для текущей ОС
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(CMD_PATH)
	@echo "Build complete: $(BINARY_NAME)"

build-windows: ## Собрать для Windows
	@echo "Building for Windows..."
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME).exe $(CMD_PATH)

build-linux: ## Собрать для Linux
	@echo "Building for Linux..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(CMD_PATH)

test: ## Запустить тесты
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...

test-coverage: ## Запустить тесты с покрытием и создать отчёт
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Запустить линтер
	@echo "Running linter..."
	golangci-lint run ./...

clean: ## Очистить артефакты сборки
	@echo "Cleaning..."
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe coverage.out coverage.html
	rm -rf dist/
	@echo "Clean complete"

run: ## Запустить приложение
	@echo "Running $(BINARY_NAME)..."
	$(GO) run $(LDFLAGS) $(CMD_PATH)

docker-build: ## Собрать Docker образ
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

docker-run: ## Запустить Docker контейнер
	@echo "Running Docker container..."
	docker run --rm -it $(BINARY_NAME):latest

docker-clean: ## Удалить Docker образы
	@echo "Cleaning Docker images..."
	docker rmi $(BINARY_NAME):latest 2>/dev/null || true

install-deps: ## Установить зависимости для сборки (Linux)
	@echo "Installing build dependencies..."
	sudo apt-get update
	sudo apt-get install -y libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev

mod-tidy: ## Привести go.mod в порядок
	@echo "Running go mod tidy..."
	$(GO) mod tidy

mod-download: ## Скачать все зависимости
	@echo "Downloading dependencies..."
	$(GO) mod download

release: ## Создать релиз через GoReleaser
	@echo "Creating release..."
	goreleaser release --clean

release-snapshot: ## Создать snapshot релиз (без публикации)
	@echo "Creating snapshot release..."
	goreleaser release --snapshot --clean

version: ## Показать версию
	@echo "Version: $(VERSION)"
