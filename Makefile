.PHONY: all build run clean api display run-display test lint

# 项目名称
APP_NAME := news_agent

# 输出目录
OUTPUT_DIR := output

# Go 源文件入口
SRC := ./app/domain_radar/cmd/domain_radar
DISPLAY_SRC := ./app/display/cmd/server

all: build display

build:
	@echo "正在编译项目..."
	@mkdir -p $(OUTPUT_DIR)
	@go build -o $(OUTPUT_DIR)/$(APP_NAME) $(SRC)
	@if [ -d "configs" ]; then \
		mkdir -p $(OUTPUT_DIR)/configs; \
		cp configs/config.yaml $(OUTPUT_DIR)/configs/; \
	fi
	@echo "编译完成，输出目录: $(OUTPUT_DIR)"

run: build
	@echo "正在运行项目..."
	@./$(OUTPUT_DIR)/$(APP_NAME) -config configs/config.yaml

api:
	@echo "正在生成 API 代码..."
	@buf generate
	@echo "API 代码生成完成"

display:
	@echo "Building Display Service..."
	@mkdir -p $(OUTPUT_DIR)
	@go build -o $(OUTPUT_DIR)/display $(DISPLAY_SRC)
	@echo "Build complete: $(OUTPUT_DIR)/display"

run-display: display
	@echo "Running Display Service..."
	@$(OUTPUT_DIR)/display -conf app/display/configs/config.yaml

test:
	@echo "正在运行测试..."
	@go test ./... -cover

lint:
	@echo "正在进行静态检查..."
	@go vet ./...

clean:
	@echo "正在清理..."
	@rm -rf $(OUTPUT_DIR)
	@rm -f index.html
	@rm -f app.log
	@echo "清理完成"
