.PHONY: all build run clean

# 项目名称
APP_NAME := news_agent

# 输出目录
OUTPUT_DIR := output

# Go 源文件入口
SRC := ./app/domain_radar/cmd/domain_radar

all: build

build:
	@echo "正在编译项目..."
	@mkdir -p $(OUTPUT_DIR)
	@go build -o $(OUTPUT_DIR)/$(APP_NAME) $(SRC)
	@mkdir -p $(OUTPUT_DIR)/configs
	@cp configs/config.yaml $(OUTPUT_DIR)/configs/
	@echo "编译完成，输出目录: $(OUTPUT_DIR)"

run: build
	@echo "正在运行项目..."
	@cd $(OUTPUT_DIR) && ./$(APP_NAME)

clean:
	@echo "正在清理..."
	@rm -rf $(OUTPUT_DIR)
	@rm -f index.html
	@rm -f app.log
	@echo "清理完成"
