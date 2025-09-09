# CLAUDE-CURRENT.md

## 當前任務階段：統一路由管理系統完成與核心功能優化
統一路由管理系統重構完成，系統進入身份管理核心功能優化階段

### 最新完成任務
- [x] ✅ **v2.0 統一路由管理系統** (2025-09-09)
  - [x] RouterManager 集中式路由協調實作
  - [x] 組件化路由設計 (API, Swagger, Health)
  - [x] 中間件統一管理與配置
  - [x] TracingMiddleware 整合與優化
  - [x] 環境感知配置 (開發 vs 生產環境)
  - [x] 向後相容性保持，維持原 RegisterRoutes() 方法

- [x] ✅ **v1.0 核心身份管理系統** (2025-09-02)
  - [x] 完整 Clean Architecture 實作
  - [x] 六角架構 (Hexagonal Architecture) 
  - [x] 身份實體管理 (Merchant, Player, Manager)
  - [x] Level 和 Tag 系統實作
  - [x] PlayerTag 多對多關聯管理
  - [x] AWS Kinesis Data Streams (KDS) 整合
  - [x] 事件驅動架構實作
  - [x] 背景任務處理 (Redis Queue)

### 當前重點
1. **路由系統驗證**: 測試新的統一路由管理系統穩定性
2. **身份同步功能**: 驗證 KDS 事件處理與身份資料同步
3. **Player Activity 追蹤**: last_active_at 欄位更新功能測試
4. **API 整合測試**: 確保 Swagger UI 和身份管理 API 正常運作
5. **多服務架構測試**: 驗證 Web/Consumer/Worker 三服務協作
6. **效能監控**: 利用分散式追蹤進行身份管理性能分析

### 進行中任務

#### 統一路由管理系統驗證
- [ ] **RouterManager 功能測試**
  - [x] SetupRoutersWithMiddleware() 方法實作
  - [x] 中間件統一配置管理
  - [x] 組件化路由設計完成
  - [ ] 路由組件獨立性測試
  - [ ] 中間件隔離性驗證

- [ ] **路由組件測試**
  - [x] API Router 組件測試 (身份管理 API)
  - [x] Swagger Router 組件測試
  - [x] Health Router 組件測試  
  - [ ] 路由間依賴關係驗證
  - [ ] TracingMiddleware 整合測試

#### 身份管理核心功能驗證
- [ ] **Merchant 身份管理**
  - [x] Merchant CRUD 操作實作
  - [x] API Key 管理功能
  - [ ] Global ID 查詢功能測試
  - [ ] Merchant 隔離性驗證

- [ ] **Player 身份管理**
  - [x] Player CRUD 操作實作
  - [x] Level 分級系統整合
  - [x] Tag 標籤系統整合
  - [x] PlayerTag 多對多關聯
  - [x] last_active_at 活動追蹤實作
  - [ ] Player 活動更新 API 測試
  - [ ] 標籤查詢與篩選功能驗證

- [ ] **Manager 身份管理**
  - [x] Manager CRUD 操作實作
  - [ ] Manager 權限管理驗證
  - [ ] Global ID 查詢功能測試

#### 事件驅動架構驗證
- [ ] **KDS 事件處理**
  - [x] AWS Kinesis Data Streams 整合
  - [x] Consumer 服務實作
  - [ ] 事件處理流程測試
  - [ ] 事件序列化/反序列化驗證
  - [ ] 錯誤恢復機制測試

- [ ] **背景任務處理**
  - [x] Redis Queue 整合
  - [x] Worker 服務實作
  - [ ] 任務佇列處理驗證
  - [ ] 並行處理效能測試
  - [ ] 任務失敗重試機制測試

#### 多服務架構測試
- [ ] **服務啟動穩定性**
  - [ ] Web 服務 (HTTP API) 啟動測試
  - [ ] Consumer 服務 (KDS) 啟動測試  
  - [ ] Worker 服務 (背景處理) 啟動測試
  - [ ] 依賴服務 (Redis/MySQL) 連接驗證

- [ ] **服務間通信測試**
  - [ ] Web -> Redis Queue 流程測試
  - [ ] Consumer -> Redis Queue 流程測試
  - [ ] Worker <- Redis Queue 流程測試
  - [ ] 分散式追蹤整合驗證

### 測試階段任務清單

#### 1. 功能測試 (Functional Testing)
- [ ] **身份管理 API 端點測試**
  - [ ] Merchant 管理 API 測試
  - [ ] Player 管理 API 測試 (包含 last_active_at 更新)
  - [ ] Manager 管理 API 測試
  - [ ] Level 管理 API 測試
  - [ ] Tag 管理 API 測試

- [ ] **身份同步系統測試**
  - [ ] KDS 事件觸發機制驗證
  - [ ] 身份資料同步邏輯測試
  - [ ] 錯誤恢復機制測試

- [ ] **認證中間件測試**
  - [ ] API Key 驗證測試
  - [ ] Merchant 權限隔離測試
  - [ ] 無效請求處理測試

#### 2. 整合測試 (Integration Testing)
- [ ] **資料庫整合**
  - [ ] Repository 層完整性測試
  - [ ] 事務處理驗證
  - [ ] 身份資料一致性檢查
  - [ ] PlayerTag 關聯查詢測試

- [ ] **Redis 整合**
  - [ ] 任務佇列操作測試
  - [ ] 快取機制驗證 (如有實作)
  - [ ] 分散式處理測試

- [ ] **服務間整合**
  - [ ] Web -> Consumer -> Worker 流程測試
  - [ ] KDS -> Consumer -> Queue 流程測試
  - [ ] 身份資料同步端到端測試

#### 3. 性能測試 (Performance Testing)
- [ ] **併發處理測試**
  - [ ] 身份查詢併發測試
  - [ ] KDS 事件併發處理驗證
  - [ ] Worker 併發數量調優

- [ ] **負載測試**
  - [ ] 身份管理 API 負載測試
  - [ ] 資料庫連線池測試
  - [ ] Redis 連線數測試

#### 4. 安全測試 (Security Testing)
- [ ] **API 安全**
  - [ ] API Key 認證繞過嘗試測試
  - [ ] SQL 注入防護測試
  - [ ] Merchant 資料隔離測試

- [ ] **資料安全**
  - [ ] 敏感身份資料洩露檢查
  - [ ] 權限提升測試
  - [ ] 身份資料篡改防護測試

#### 5. 可靠性測試 (Reliability Testing)
- [ ] **故障恢復**
  - [ ] MySQL 連線中斷恢復
  - [ ] Redis 服務中斷恢復
  - [ ] KDS 服務中斷恢復

- [ ] **資料完整性**
  - [ ] 身份同步失敗處理
  - [ ] 重複事件防護
  - [ ] 身份資料準確性驗證

### 測試環境配置

#### 本地開發環境
```bash
# 啟動特定服務進行測試
go run main.go web       # Web 服務 :8080
go run main.go consumer  # KDS Consumer 服務
go run main.go worker    # 背景 Worker 服務

# Docker Compose 完整環境
docker-compose up -d --build

# 運行完整測試套件
go test ./...

# 運行覆蓋率測試
go test -cover ./...
```

#### 測試資料準備
- [ ] 建立測試用 Merchant 資料 (含 API Key)
- [ ] 建立測試用 Player 資料 (含 Level 和 Tag 關聯)
- [ ] 建立測試用 Manager 資料
- [ ] 準備各種狀態的身份測試資料

### 測試工具與框架
- **單元測試**: Go 內建 testing 框架
- **Mock**: testify/mock 或 GoMock
- **API 測試**: HTTP 測試客戶端
- **負載測試**: 待選擇工具 (如 wrk, hey, 或 k6)
- **事件測試**: AWS Kinesis Local 或 Mock

### 成功標準
- [ ] 所有單元測試通過率 100%
- [ ] 整合測試通過率 100%
- [ ] 身份管理 API 響應時間 < 100ms (95th percentile)
- [ ] KDS 事件處理延遲 < 500ms
- [ ] 測試覆蓋率 > 80%

### 已知問題與待解決項目
- [ ] 確認 OpenTelemetry 分散式追蹤配置
- [ ] 優化身份查詢效能 (索引優化)
- [ ] 完善 KDS 事件錯誤處理機制
- [ ] PlayerTag 關聯查詢效能調優

### 下一階段規劃
測試完成後將進入功能擴展階段：
1. 身份管理進階功能 (批次操作、匯入匯出)
2. 身份關聯管理 (Player-Merchant 關係)
3. 身份歷史記錄與版本管理
4. 身份資料分析與統計功能
5. 生產環境部署準備

---
**專案**: Fat Identity Cat - 身份管理微服務  
**架構**: Clean Architecture + 事件驅動 + 多服務  
**核心功能**: Merchant/Player/Manager 身份管理、Level/Tag 系統、KDS 同步  
**更新日期**: 2025-09-09  
**版本**: v2.0 (統一路由管理系統)