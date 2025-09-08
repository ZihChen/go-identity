# 📌 Fat Identity Cat

Fat Identity Cat 是一個用於管理商戶、玩家和管理員身份的微服務，具有通過 Kinesis Data Streams (KDS) 進行同步的功能。

## 🚀 功能亮點 / 特色
  - Web 服務：提供 HTTP API
  - Consumer 服務：從 KDS 消費事件並將其排入隊列
  - Worker 服務：處理隊列中的任務

## 📦 安裝步驟

### 前置條件

- Go 1.18 或更高版本
- Docker 和 Docker Compose（用於本地開發）
- AWS 帳戶（用於 Kinesis Data Streams 和 DynamoDB）
- Redis 服務

### 使用 Docker Compose 安裝

1. Clone Repo：

```bash
git clone https://gitlab.jvdtech.dev/fatcat/fat_identity_cat.git
cd fat_identity_cat
```

2. 創建 `.env` 文件並配置環境變量（參見下面的配置部分）

3. 使用 Docker Compose 啟動服務：

```bash
docker-compose up -d
```

### 手動安裝

1. Clone Repo：

```bash
git clone https://gitlab.jvdtech.dev/fatcat/fat_identity_cat.git
cd fat_identity_cat
```

2. 安裝依賴：

```bash
go mod download
```

3. 創建 `.env` 文件並配置環境變量

4. 運行數據庫遷移：

```bash
./migrate.sh
```

## ▶️ 使用方法

Fat Identity Cat 服務有三種運行模式：

### Web 服務

啟動 Web 服務以處理 HTTP API 請求：

```bash
docker-compose up -d fat_identity_web
```

### Consumer 服務

啟動 Consumer 服務以從 KDS 消費事件：

```bash
docker-compose up -d fat-identity-consumer
```

### Worker 服務

啟動 Worker 服務以處理隊列中的任務：

```bash
docker-compose up -d fat-identity-worker
```

## 🔧 設定與環境變數

服務使用 `.env` 文件或環境變量進行配置。以下是主要的配置選項：

### 應用配置

```
APP_NAME=fat-identity-cat
APP_ENV=development
APP_PORT=8080
APP_DEBUG=true
```

### 數據庫配置

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=identity_cat
DB_OPTIONS=
DB_MAX_IDLE=10
DB_MAX_OPEN=100
DB_TIMEOUT=5s
```

### Redis 配置

```
REDIS_DOMAIN=localhost
REDIS_PORT=6379
REDIS_PWD=
REDIS_DB=0
```

### AWS 配置

```
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=ap-southeast-1
KINESIS_STREAM_ARN=arn:aws:kinesis:ap-southeast-1:123456789012:stream/identity-cat-stream
DYNAMODB_TABLE=identity-cat-checkpoints
DYNAMODB_PARTITION_KEY=shard_id
DYNAMODB_SORT_KEY=sequence_number
```

### 追踪和日誌配置

```
OPENOBSERVE_TRACE_API_ENDPOINT=https://api.openobserve.ai/api/v1/traces
OPENOBSERVE_TRACE_API_KEY=your_api_key
OPENOBSERVE_TRACE_STREAM_NAME=identity-cat-traces
OPENOBSERVE_LOGS_API_ENDPOINT=https://api.openobserve.ai/api/v1/logs
OPENOBSERVE_LOGS_USERNAME=your_username
OPENOBSERVE_LOGS_PASSWORD=your_password
```

### 事件配置

```
EVENT_MERCHANT_SYNC=merchant.sync
EVENT_PLAYER_SYNC=player.sync
EVENT_MANAGER_SYNC=manager.sync
EVENT_IDENTITY_MERCHANT_SYNC=identity.merchant.sync
EVENT_IDENTITY_PLAYER_SYNC=identity.player.sync
EVENT_IDENTITY_MANAGER_SYNC=identity.manager.sync
```

## 🧪 測試方法

### 運行所有測試

```bash
go test ./...
```

### 運行特定測試

```bash
go test ./test/database_test.go
go test ./test/redis_test.go
go test ./test/kds_connection_test.go
```

### 測試覆蓋率

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 📁 專案結構說明

```
fat_identity_cat/
├── cmd/                    # 命令行入口點
│   ├── consumer/           # KDS 消費者服務
│   ├── web/                # Web API 服務
│   └── worker/             # 任務處理服務
├── docs/                   # Swagger 文檔
├── internal/               # 內部包
│   ├── adapter/            # 適配器層 (實作層)
│   │   ├── handler/        # HTTP 和 Worker 處理器
│   │   ├── middleware/     # HTTP 中間件
│   │   ├── repository/     # 資料庫操作實作層
│   │   ├── service/        # 服務實作層
│   │   └── usecase/        # 用例實作層
│   ├── di/                 # 依賴注入
│   ├── domain/             # 領域層 (定義接口、參數、結構體)
│   │   ├── consts/         # 常數定義
│   │   ├── dto/            # 資料轉換結構體定義 (handler <-> usecase)
│   │   ├── entity/         # 領域層結構體定義 (usecase <-> repository)
│   │   ├── errmsg/         # error message定義
│   │   ├── event/          # 事件結構體定義
│   │   ├── infraport/      # 基礎設施接口
│   │   ├── repositoryport/ # 存儲庫接口
│   │   ├── serviceport/    # 服務接口
│   │   └── ports/
│   │       └── inbound/    # 入站端口接口 (用例接口)
│   └── infrastructure/     # 基礎設施層
│       ├── cache/          # 快取相關元件
│       │   └── redis/      # Redis元件
│       ├── config/         # 變數配置
│       ├── database/       # 資料庫相關元件
│       │   └── mysql/      # MySQL元件
│       ├── kds/            # Kinesis Data Streams
│       ├── logger/         # 日誌元件
│       ├── models/         # 資料庫模型
│       ├── queue/          # 任務隊列
│       └── tracing/        # 分布式追踪器
├── migrations/             # 資料庫 schema migrations檔案
├── test/                   # 集成測試
├── .env                    # 環境變量
├── .gitlab-ci.yml          # Gitlab CI 配置
├── .golangci.yml           # golangci:程式碼規範工具配置
├── atlas.hcl               # Altas:Migration 工具配置
├── docker-compose.yml      # Docker Compose 配置
└── Dockerfile
```

### 主要組件

- **Web 服務**：提供 HTTP API 用於管理身份
- **Consumer 服務**：從 KDS 消費事件並將其排入 Redis 隊列
- **Worker 服務**：處理 Redis 隊列中的任務並更新數據庫

### 架構設計

該項目遵循清晰的架構分層：

- **領域層(Domain)**：包含業務邏輯和實體
- **用例層(Usecase)**：實現業務用例
- **適配器層(Adaptor)**：連接用例和基礎設施
- **基礎設施層(Infra)**：提供技術實現
