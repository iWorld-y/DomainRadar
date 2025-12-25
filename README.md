# DomainRadar (领域雷达)

DomainRadar 是一个智能化的领域情报分析工具。它利用先进的 LLM 技术和 Tavily 搜索引擎，为您自动追踪、分析并总结特定领域的最新动态，生成深度洞察报告。

旨在帮助开发者、研究人员和行业关注者从海量信息中解放出来，快速掌握核心趋势。

## ✨ 核心特性

- **🎯 多领域精准监控**：支持自定义配置多个感兴趣的领域（如 AIGC, Web3, Cloud Native 等），针对性获取信息。
- **🔍 智能搜索增强**：集成 [Tavily API](https://tavily.com/)，自动过滤低质量内容，专注获取高价值的新闻与文章。
- **🧠 深度 AI 分析**：
  - **领域综述**：自动生成核心动态与热点话题总结。
  - **趋势洞察**：基于新闻分析未来的技术或市场走向。
  - **关键事件**：提取每日重要事件列表。
  - **热度评分**：量化领域活跃度。
- **👤 个性化战略解读**：结合用户画像（User Persona），跨领域交叉分析，提供定制化的机遇挖掘、风险预警及行动建议。
- **📊 精美 HTML 报告**：一键生成包含图文排版的每日早报 (`index.html`)，阅读体验极佳。
- **🛡️ 稳健的工程设计**：内置并发控制与限流机制，优雅处理 API 限制；模块化设计，易于扩展。

## 🛠️ 技术栈

- **语言**: Go (Golang)
- **搜索**: Tavily API
- **LLM**: OpenAI Compatible API (支持 GPT-4, DeepSeek, Claude 等兼容接口)
- **框架**: CloudWeGo Eino (Model Component)

## 🚀 快速开始

### 前置要求

- macOS / Linux / Windows
- Go 1.21+
- Docker (用于启动 PostgreSQL 数据库)
- Tavily API Key ([获取地址](https://tavily.com/))
- LLM API Key (OpenAI 或其他兼容服务商)

### 1. 克隆项目

```bash
git clone https://github.com/iWorld-y/domain_radar.git
cd domain_radar
```

### 2. 启动数据库

项目使用 PostgreSQL 存储分析报告，可以通过 Docker 快速启动：

```bash
docker-compose up -d
```

### 3. 配置环境

拷贝示例配置文件并进行修改：

```bash
mkdir -p configs
cp config.yaml.example configs/config.yaml
```

编辑 `configs/config.yaml`，填入您的 API Key 和数据库配置：

```yaml
llm:
  base_url: "https://api.openai.com/v1" # 或其他兼容服务的 Base URL
  api_key: "your_llm_api_key"
  model: "gpt-4-turbo"

tavily_api_key: "tvly-xxxxxxxxxxxx"

user_persona: "资深全栈工程师，关注架构设计与 AI 落地"

domains:
  - "Artificial Intelligence"
  - "Cloud Computing"

db:
  host: "localhost"
  port: 5432
  user: "user"       # 需与 docker-compose.yaml 中的配置一致
  password: "password"
  name: "domain_radar"
```

### 4. 编译与运行

项目提供了 `Makefile` 以简化操作：

```bash
# 编译并运行情报收集分析工具
make run
```

运行完成后，将在 `output` 目录下生成 `index.html` 文件。您可以直接用浏览器打开查看生成的领域雷达早报。

```bash
open output/index.html
```

### 5. 启动展示服务 (可选)

如果您希望通过 Web 界面查看历史报告，可以启动展示服务：

```bash
make run-display
```

默认访问 `http://localhost:8001`。

### 6. 清理

```bash
make clean
```

## 📂 项目结构

```text
.
├── Makefile                # 构建管理
├── README.md               # 项目文档
├── app/
│   ├── display/            # 展示服务 (Web UI)
│   └── domain_radar/       # 情报分析核心服务
├── configs/                # 运行时配置文件目录
├── proto/                  # Protobuf 定义
├── config.yaml.example     # 配置文件示例
├── docker-compose.yaml     # 基础设施 (PostgreSQL)
└── output/                 # 编译产物与报告输出
```

## 🤝 贡献

欢迎提交 Issue 或 Pull Request。在提交代码前，请确保代码风格符合 Go 语言规范，并已通过本地测试。

## 📄 许可证

本项目采用 [MIT License](LICENSE) 开源。
