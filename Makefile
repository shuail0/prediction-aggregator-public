.PHONY: help build test clean install lint fmt build-hedge build-arbitrage

BIN_DIR := bin

help: ## 显示帮助信息
	@echo "Prediction Aggregator - Makefile Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

install: ## 安装依赖
	go mod download
	go mod tidy

build: ## 编译所有策略
	@echo "🔨 Building..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/01_polymarket_hedge ./strategies/farm/01_polymarket_hedge/
	go build -o $(BIN_DIR)/01_updown_arbitrage ./strategies/trading/01_updown_arbitrage/
	@echo "✅ Build complete: $(BIN_DIR)/"

build-hedge: ## 编译 polymarket 对刷策略
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/01_polymarket_hedge ./strategies/farm/01_polymarket_hedge/

build-arbitrage: ## 编译 Up/Down 套利策略
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/01_updown_arbitrage ./strategies/trading/01_updown_arbitrage/

test: ## 运行测试
	@echo "🧪 Running tests..."
	go test -v -race -cover ./...

lint: ## 代码检查
	@echo "🔍 Linting code..."
	golangci-lint run ./...

fmt: ## 格式化代码
	@echo "✨ Formatting code..."
	go fmt ./...

clean: ## 清理构建文件
	@echo "🧹 Cleaning..."
	rm -rf $(BIN_DIR)/
	go clean

.DEFAULT_GOAL := help
