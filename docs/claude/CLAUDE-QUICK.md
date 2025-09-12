# CLAUDE-QUICK.md

## 快速開發指南

### 當前狀態
- **Consumer v2.0**: 性能優化 + 代碼重構 ✅ 已完成 (2025-09-12)
- **Router v2.0**: 統一路由管理系統 ✅ 已完成 (2025-09-09)
- **Core v1.0**: 核心身份管理系統 ✅ 已完成 (2025-09-02)
- **當前階段**: Consumer 性能驗證與生產部署準備 🔄 進行中
- **下一里程碑**: 監控告警配置與操作手冊編寫

### 快速命令

#### 開發環境
```bash
# 啟動所有服務
docker-compose up -d --build

# 啟動單一服務 (本地開發)
go run main.go web       # Web API服務 :8080
go run main.go consumer  # 高性能KDS消費者服務 (v2.0)
go run main.go worker    # 背景Worker服務

# Consumer v2.0 性能測試
go test ./test/consumer_performance_test.go -v

# 檢視Swagger API文檔
# http://localhost:8080/swagger/index.html

# 檢視健康狀態
# http://localhost:8080/health
```

#### 測試命令
```bash
# 運行所有測試
go test ./...

# 運行覆蓋率測試
go test -cover ./...

# Consumer v2.0 性能基準測試 ✨ NEW
go test ./test/consumer_performance_test.go -v
go test ./internal/infrastructure/kds/benchmark_test.go -bench=.

# 運行特定模組測試
go test ./internal/adapter/outbound/repository/merchant/...
go test ./internal/adapter/outbound/repository/player/...
go test ./internal/adapter/inbound/handler/api/...
```

#### 資料庫操作
```bash
# 應用遷移
./migrate.sh apply

# 檢查遷移狀態
./migrate.sh status

# 生成新遷移
./migrate.sh gen <migration_name>

# Atlas 遷移命令
atlas migrate apply --env local
atlas migrate diff <migration_name> --env local
```

#### 代碼生成
```bash
# 更新Swagger文檔
swag init

# 重新生成依賴注入
wire ./internal/di
```

### 重要檔案位置

#### Clean Architecture 組織 ✨ **v1.0 Core**

##### Domain Layer (核心領域)
- `internal/domain/entity/` - 領域實體
  - `merchant.go` - 商戶實體
  - `player.go` - 玩家實體
  - `manager.go` - 管理員實體
  - `level.go` - 等級實體
  - `tag.go` - 標籤實體
  - `player_tag.go` - 玩家標籤關聯實體
- `internal/domain/ports/inbound/` - 入站介面 (Use Case 介面)
- `internal/domain/ports/outbound/` - 出站介面 (Repository, Service 介面)
  - `repository/` - Repository 介面定義
- `internal/domain/dto/` - 資料傳輸物件
- `internal/domain/consts/` - 領域常數
- `internal/domain/errmsg/` - 錯誤定義
- `internal/domain/event/` - KDS 事件結構定義

##### Adapter Layer (適配器層)
**Inbound Adapters (入站適配器)**
- `internal/adapter/inbound/handler/` - 處理器
  - `api/` - HTTP API 處理器 (Merchant, Player, Manager API)
  - `worker/` - Worker 處理器 (背景任務處理)
- `internal/adapter/inbound/router/` ✨ **NEW v2.0 統一路由管理系統**
  - `router_manager.go` - **統一路由管理器**
  - `api_router.go` - API路由組件
  - `swagger_router.go` - Swagger路由組件  
  - `health_router.go` - 健康檢查路由
- `internal/adapter/inbound/middleware/` - HTTP 中間件

**Outbound Adapters (出站適配器)**
- `internal/adapter/outbound/repository/` - 資料庫操作 (按業務分組)
  - `merchant/` - 商戶相關 Repository
  - `player/` - 玩家相關 Repository (含 Tag 管理)
  - `manager/` - 管理員相關 Repository
  - `level/` - 等級管理 Repository
  - `tag/` - 標籤管理 Repository

##### Infrastructure Layer (基礎設施層)
- `internal/infrastructure/database/mysql/` - MySQL 資料庫連接
- `internal/infrastructure/cache/redis/` - Redis 快取與佇列管理 (支援批次操作)
- `internal/infrastructure/kds/` ✨ **v2.0 高性能架構** - AWS Kinesis 整合
  - `consumer.go` - 批次處理主邏輯 (100 records/batch)
  - `backoff_strategy.go` - 智能退避策略組件
  - `benchmark_test.go` - 性能基準測試
- `internal/infrastructure/queue/` - 背景任務佇列管理
- `internal/infrastructure/tracing/` - OpenTelemetry 分散式追蹤
- `internal/infrastructure/config/` - 系統配置管理
- `internal/infrastructure/logger/` - 結構化日誌

#### 配置檔案
- `internal/di/wire.go` - 依賴注入配置 (使用 Google Wire)
- `docker-compose.yml` - 本地開發環境
- `CLAUDE.md` - 專案指引文件

### 文檔結構
```
docs/claude/
├── CLAUDE-QUICK.md      # 此文件 - 快速參考
├── CLAUDE-CURRENT.md    # 當前任務狀態
└── refactor/            # 重構工作模板目錄 (即將新增)
```

### API快速測試

#### 健康檢查
```bash
curl -X GET http://localhost:8080/health
```

#### Merchant 身份管理
```bash
# 查詢 Merchant (by ID)
curl -X GET http://localhost:8080/api/v1/merchants/1 \
  -H "API-Key: YOUR_BASE64_ENCODED_API_KEY"

# 查詢 Merchant (by Global ID)  
curl -X GET http://localhost:8080/api/v1/merchants/global/MERCHANT_GLOBAL_ID \
  -H "API-Key: YOUR_BASE64_ENCODED_API_KEY"
```

#### Player 身份管理
```bash
# 查詢 Player (by ID)
curl -X GET http://localhost:8080/api/v1/players/1 \
  -H "API-Key: YOUR_BASE64_ENCODED_API_KEY"

# 查詢 Player (by Global ID)
curl -X GET http://localhost:8080/api/v1/players/global/PLAYER_GLOBAL_ID \
  -H "API-Key: YOUR_BASE64_ENCODED_API_KEY"

# 更新 Player 活動時間
curl -X PUT http://localhost:8080/api/v1/players/1/active \
  -H "API-Key: YOUR_BASE64_ENCODED_API_KEY" \
  -H "Content-Type: application/json"
```

#### Manager 身份管理
```bash
# 查詢 Manager (by ID)
curl -X GET http://localhost:8080/api/v1/managers/1 \
  -H "API-Key: YOUR_BASE64_ENCODED_API_KEY"

# 查詢 Manager (by Global ID)
curl -X GET http://localhost:8080/api/v1/managers/global/MANAGER_GLOBAL_ID \
  -H "API-Key: YOUR_BASE64_ENCODED_API_KEY"
```

#### Swagger API測試
- 訪問: http://localhost:8080/swagger/index.html
- 點擊 "Authorize" 按鈕
- 輸入 base64 編碼的 API Key

### 除錯技巧

#### 日誌檢視
```bash
# 查看服務日誌
go run main.go web       # 前景執行，直接查看日誌
go run main.go consumer  
go run main.go worker

# Docker Compose 日誌
docker-compose logs -f fat-identity-cat-web
docker-compose logs -f fat-identity-cat-consumer
docker-compose logs -f fat-identity-cat-worker

# 查看Redis佇列狀態
docker exec -it redis redis-cli
127.0.0.1:6379> KEYS *
127.0.0.1:6379> LLEN asynq:default
```

#### 分散式追蹤 ✨ **NEW**
```bash
# OpenTelemetry 追蹤查看
# 配置追蹤端點後，可在追蹤系統中查看請求鏈路
# 追蹤身份管理 API 調用過程
```

#### 資料庫檢查
```bash
# 連接MySQL
docker exec -it mysql mysql -u root -p

# 身份管理常用查詢
SELECT * FROM merchants ORDER BY created_at DESC LIMIT 10;
SELECT * FROM players ORDER BY last_active_at DESC LIMIT 10;
SELECT * FROM managers ORDER BY created_at DESC LIMIT 10;

# Player Tag 關聯查詢
SELECT p.*, pt.tag_id, t.name as tag_name 
FROM players p 
LEFT JOIN player_tags pt ON p.id = pt.player_id 
LEFT JOIN tags t ON pt.tag_id = t.id 
WHERE p.id = 1;

# Level 查詢
SELECT l.*, COUNT(p.id) as player_count 
FROM levels l 
LEFT JOIN players p ON l.id = p.level_id 
GROUP BY l.id;
```

### 常見問題

#### 編譯問題
```bash
# 清理模組快取
go clean -modcache
go mod tidy

# 重新生成 Wire
wire ./internal/di

# 檢查 import 路徑
go mod graph | grep fat-identity-cat
```

#### 測試失敗
```bash
# 重置測試環境
docker-compose down
docker-compose up -d
sleep 10  # 等待服務啟動

# 檢查依賴服務
docker-compose ps

# 重新運行測試
go test ./...
```

#### KDS 事件問題
```bash
# 檢查 AWS 配置
aws configure list

# 檢查 KDS Stream
aws kinesis list-streams

# Consumer 服務日誌
go run main.go consumer  # 查看事件處理日誌
```

### 統一路由管理系統新功能 ✨ v2.0

#### RouterManager 集中管理
```go
// 使用新的統一路由管理器
routerManager := router.NewRouterManager(httpHandler)
routerManager.SetupRoutersWithMiddleware(ginEngine, config)
```

#### 組件化路由架構
- **RouterManager**: 中央協調器，統一管理所有路由組件
- **API Router**: 身份管理 API 路由 (含認證中間件)
- **Swagger Router**: API 文檔路由 (不含認證中間件)
- **Health Router**: 健康檢查路由 (不含認證中間件)

#### 中間件統一管理
- **TracingMiddleware**: 分散式追蹤整合
- **AuthenticationMiddleware**: API Key 認證 (僅 API 路由)
- **CORS配置**: 環境感知配置

#### 向後相容性
- 保留原有 `RegisterRoutes()` 方法
- 漸進式升級路徑
- 不影響現有程式碼運作

### Clean Architecture 實現 ✨ v1.0

#### Hexagonal Architecture 模式
```go
// Domain 定義介面，Adapters 實作
// Domain -> Application -> Adapters -> Infrastructure

// Inbound Port (Use Case 介面)
type MerchantUseCase interface {
    GetMerchantByID(ctx context.Context, id uint64) (*entity.Merchant, error)
    GetMerchantByGlobalID(ctx context.Context, globalID string) (*entity.Merchant, error)
}

// Outbound Port (Repository 介面)  
type MerchantRepository interface {
    FindByID(ctx context.Context, id uint64) (*entity.Merchant, error)
    FindByGlobalID(ctx context.Context, globalID string) (*entity.Merchant, error)
    // ...
}
```

#### Repository 按業務組織
- **清晰分離**: merchant/, player/, manager/, level/, tag/
- **單一職責**: 每個 Repository 負責特定業務領域
- **依賴方向**: Domain <- Application <- Adapters <- Infrastructure

#### 身份管理核心實體
- **Merchant**: 商戶身份，含 API Key 管理和 Global ID
- **Player**: 玩家身份，支援 Level 分級和 Tag 標籤
- **Manager**: 管理員身份，用於商戶管理
- **Level**: 玩家等級系統
- **Tag**: 玩家標籤系統  
- **PlayerTag**: Player 和 Tag 的多對多關聯

### 事件驅動架構 ✨ v2.0 高性能版

#### AWS Kinesis Data Streams 整合
```bash
# 啟動高性能 Consumer 服務處理 KDS 事件
go run main.go consumer  # 自動使用 v2.0 批次處理

# 事件處理流程: KDS -> 批次處理 -> Redis Pipeline -> Worker
# 性能指標: 10,000+ records/sec, <0.23% error rate
```

#### Consumer v2.0 核心特性
- **批次處理引擎**: 100 records/batch 高效處理
- **Worker Pool**: 10 並行 goroutines
- **Redis 批次操作**: MGet 去重 + Pipeline 標記
- **BackoffManager**: 智能退避策略
- **動態 Panic Recovery**: 4KB-1MB stack buffer

#### 背景任務處理
```bash
# 啟動 Worker 服務處理背景任務
go run main.go worker

# Redis 佇列管理
docker exec -it redis redis-cli
127.0.0.1:6379> LLEN asynq:default  # 查看待處理任務數量
```

#### 身份同步功能
- **事件觸發**: 身份變更時發送 KDS 事件
- **異步處理**: Consumer 接收事件並入佇列
- **背景同步**: Worker 處理身份同步任務
- **錯誤恢復**: 失敗任務重試機制

---
**專案**: Fat Identity Cat - 身份管理微服務  
**架構**: Clean Architecture + 高性能事件處理 + 多服務協作  
**核心功能**: Merchant/Player/Manager 身份管理、Level/Tag 系統、高性能 KDS Consumer  
**Consumer 性能**: 10,000+ records/sec (3.3倍提升), <0.23% 錯誤率  
**更新日期**: 2025-09-12  
**版本**: Consumer v2.0 + Router v2.0 + Core v1.0  
**用途**: 日常開發快速參考