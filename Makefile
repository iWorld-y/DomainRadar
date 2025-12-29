.PHONY: clean api build run test lint wire

# 输出目录
OUTPUT_DIR := output

# Go 源文件入口
DISPLAY_SRC := ./app/display/cmd/server

api:
	@echo "正在生成 API 代码..."
	@buf generate
	@echo "API 代码生成完成"

wire:
	@echo "正在生成依赖注入代码..."
	@wire ./app/display/cmd/server
	@echo "依赖注入代码生成完成"

build:
	@echo "Building Display Service..."
	@mkdir -p $(OUTPUT_DIR)
	@go build -o $(OUTPUT_DIR)/display $(DISPLAY_SRC)
	@echo "Build complete: $(OUTPUT_DIR)/display"

run: build
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
	@rm -f app.log
	@echo "清理完成"
