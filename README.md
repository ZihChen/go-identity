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
# 使用 Docker Compose
docker-compose up -d fat_identity_web

# 或直接使用 Go 命令
go run main.go web --port 8080
```

Web 服務使用統一的 RouterManager 管理所有路由和中間件：
- 自動配置追蹤中間件
- 統一管理 API、健康檢查、Swagger 路由
- 支援優雅關機和資源清理

### Consumer 服務

啟動高性能 Consumer 服務以從 KDS 消費事件（已完成 v2.0 性能優化）：

```bash
# 使用 Docker Compose
docker-compose up -d fat-identity-consumer

# 或直接使用 Go 命令
go run main.go consumer
```

#### Consumer 性能特性 ✅ (v2.0 + 代碼重構完成)

- **批次處理引擎**：每批次處理 100 條記錄，顯著提升吞吐量
- **並行處理**：10 個 Worker goroutine 並行處理事件
- **Redis 批次操作**：使用 MGet 和 Pipeline 減少網絡開銷
- **智能退避策略**：BackoffManager 組件提供自適應錯誤恢復
- **動態 Panic 恢復**：4KB-1MB 動態 stack buffer 分配
- **代碼重構優化**：函數分解、組件隔離、統一日誌記錄

**性能指標**：
- 吞吐量：10,000+ records/sec（3.3倍提升）
- 錯誤率：<0.23%
- 平均延遲：~991μs/record

### Worker 服務

啟動 Worker 服務以處理隊列中的任務：

```bash
# 使用 Docker Compose
docker-compose up -d fat-identity-worker

# 或直接使用 Go 命令
go run main.go worker
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

### HTTP 服務器配置

```
SERVER_HTTP_DOMAIN=                  # HTTP 服務器域名（選用）
SERVER_READ_TIMEOUT=30s              # 讀取請求超時時間（預設：30秒）
SERVER_WRITE_TIMEOUT=30s             # 寫入響應超時時間（預設：30秒）
SERVER_IDLE_TIMEOUT=120s             # 空閒連接超時時間（預設：120秒）
```

### 數據庫配置

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=identity_cat
DB_OPTIONS=
DB_MAX_IDLE=25              # 最大閒置連線數（預設：25）
DB_MAX_OPEN=100             # 最大開啟連線數（預設：100）
DB_TIMEOUT=5s               # 連接超時時間（預設：5s）
DB_MAX_LIFETIME=1h          # 連線最大生命週期（預設：1小時）
DB_MAX_IDLE_TIME=30m        # 連線最大空閒時間（預設：30分鐘）
```

### Redis 配置

```
REDIS_DOMAIN=localhost
REDIS_PORT=6379
REDIS_PWD=
REDIS_DB=0
REDIS_POOL_SIZE=20            # 連線池大小（預設：20）
REDIS_MIN_IDLE_CONNS=5        # 最小空閒連線數（預設：5）
REDIS_MAX_RETRIES=3           # 最大重試次數（預設：3）
REDIS_DIAL_TIMEOUT=5s         # 連線超時（預設：5秒）
REDIS_READ_TIMEOUT=3s         # 讀取超時（預設：3秒）
REDIS_WRITE_TIMEOUT=3s        # 寫入超時（預設：3秒）
REDIS_POOL_TIMEOUT=4s         # 連線池等待超時（預設：4秒）
REDIS_IDLE_TIMEOUT=5m         # 連線最大空閒時間（預設：5分鐘）
REDIS_MAX_CONN_AGE=30m        # 連線最大生命週期（預設：30分鐘）
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

### Consumer 性能配置 ✨ (新增)

```
# Consumer 批次處理配置
CONSUMER_BATCH_SIZE=100              # 每批次記錄數
CONSUMER_WORKER_POOL_SIZE=10         # 並行 worker 數量
CONSUMER_WORKER_BUFFER_SIZE=200      # Worker 通道緩衝大小
CONSUMER_KDS_RECORD_LIMIT=1000       # KDS GetRecords 限制

# Consumer 退避策略配置
CONSUMER_MIN_BACKOFF=500ms           # 最小退避時間
CONSUMER_MAX_BACKOFF=5s              # 最大退避時間

# Consumer 監控配置
CONSUMER_ENABLE_PANIC_RECOVERY=true  # 啟用 Panic 恢復
CONSUMER_MAX_RECOVERY_ATTEMPTS=3     # 最大恢復嘗試次數
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
│   │   ├── inbound/        # 入站適配器
│   │   │   ├── handler/    # HTTP 和 Worker 處理器
│   │   │   │   ├── api/    # API 處理器
│   │   │   │   └── worker/ # Worker 處理器
│   │   │   ├── middleware/ # HTTP 中間件 (統一管理)
│   │   │   └── router/     # 路由管理器 (統一封裝)
│   │   └── outbound/       # 出站適配器
│   │       └── repository/ # 資料庫操作實作層
│   ├── di/                 # 依賴注入
│   ├── domain/             # 領域層 (定義接口、參數、結構體)
│   │   ├── consts/         # 常數定義
│   │   ├── dto/            # 資料轉換結構體定義 (handler <-> usecase)
│   │   ├── entity/         # 領域層結構體定義 (usecase <-> repository)
│   │   ├── errmsg/         # error message定義
│   │   ├── event/          # 事件結構體定義
│   │   └── ports/
│   │       ├── inbound/    # 入站端口接口 (用例接口)
│   │       └── outbound/
│   │           ├── infrastructure/ # 出站端口接口 (基礎設施接口)
│   │           ├── repository/ # 出站端口接口 (存儲庫接口)
│   │           └── service/    # 出站端口接口 (服務接口)
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

- **Web 服務**：提供 HTTP API 用於管理身份（統一路由管理器架構）
- **Consumer 服務**：高性能批次處理 KDS 事件（v2.0 優化完成）
  - 批次處理引擎（100 records/batch）
  - 並行 Worker Pool（10 goroutines）
  - Redis 批次操作優化
  - BackoffManager 智能退避策略
- **Worker 服務**：處理 Redis 隊列中的任務並更新數據庫

### 架構設計

該項目遵循清晰的架構分層：

- **領域層(Domain)**：包含業務邏輯和實體
- **用例層(Usecase)**：實現業務用例
- **適配器層(Adaptor)**：連接用例和基礎設施
  - **入站適配器(Inbound)**：處理來自外部的請求
    - **Handler**：分為 API 和 Worker 兩個模組，處理不同類型的請求
    - **Middleware**：統一管理 HTTP 中間件，包含追蹤、認證等功能
    - **Router**：路由管理器統一封裝所有路由配置，支援自動中間件配置
  - **出站適配器(Outbound)**：連接外部服務和資源
- **基礎設施層(Infra)**：提供技術實現

### 最新架構特點

1. **統一路由管理**：RouterManager 統一封裝所有路由配置，提供兩種使用方式：
   - `SetupRoutersWithMiddleware`：自動配置中間件並註冊路由
   - `RegisterRoutes`：僅註冊路由（向後兼容）

2. **中間件集中管理**：所有中間件統一在 `internal/adapter/inbound/middleware` 目錄管理

3. **處理器模組化**：Handler 分為 API 和 Worker 兩個獨立模組，各自負責不同的業務處理

4. **清晰的責任分離**：入站和出站適配器明確分離，提高代碼可維護性
