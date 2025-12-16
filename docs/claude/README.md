# Claude Code 文檔目錄

## 目錄結構

本目錄包含 Fat Identity Cat 專案的 Claude Code 工作文檔，用於指導 AI 助手進行開發工作。

```
docs/claude/
├── README.md                    # 本文件 - 目錄導航
├── CLAUDE-CURRENT.md           # 當前任務狀態
├── CLAUDE-QUICK.md             # 快速開發指南
├── archieve/                   # 歷史文檔歸檔
│   └── 2025-09/
│       └── consumer-refactor/  # Consumer 重構完整歷史
├── audit/                      # 專案稽核報告工作
│   └── AUDIT_REPORT_v1.0.md
└── refactor/                   # 重構工作指南
    ├── README.md               # 重構目錄說明
    └── REFACTOR_BEST_PRACTICES.md # 重構最佳實踐
```

## 🚀 快速開始

### 新手入門
1. **[CLAUDE-QUICK.md](./CLAUDE-QUICK.md)** - 快速開發指南
   - 當前系統狀態
   - 快速命令參考
   - API 測試示例
   - 故障排除技巧

### 了解當前狀態
2. **[CLAUDE-CURRENT.md](./CLAUDE-CURRENT.md)** - 當前任務狀態
   - 最新完成任務
   - 進行中任務
   - 測試計劃
   - 成功標準

## 📚 專業文檔

### 重構工作
- **[refactor/](./refactor/)** - 重構工作指南
  - 基於成功經驗的最佳實踐
  - 技術實施策略
  - 常見陷阱避免方法

### 歷史歸檔
- **[archieve/2025-09/consumer-refactor/](archieve/2025/2025-09/consumer-refactor/)** - Consumer 重構完整歷史
  - 三階段重構過程
  - 技術決策記錄
  - 性能驗證結果
  - 經驗教訓總結

## 🎯 專案現狀

### ✅ 已完成的重要里程碑

**Player Sync Deadlock Optimization v7.0 (2025-12-03)**
- 🛡️ Repository 層 MySQL deadlock retry 機制
- 📉 Deadlock 錯誤率 0.81% → <0.05% (95%+ 改善)
- 🏗️ 三層防護架構 (Asynq + Redis + MySQL)
- 🎯 零業務邏輯影響，完美層級實作
- 💯 系統穩定性大幅提升

**Agent Synchronization v6.0 (2025-11-05)**
- 🤖 完整代理身份管理系統
- 🔄 雙向KDS事件同步
- 🧪 KDS測試API集成
- 🛡️ 商戶隔離與類型安全
- 📊 架構優化與Clean Architecture合規

**Domain Model v5.0 標準化 (2025-10-22)**
- 🏗️ 統一領域模型調用方式
- 🔒 實體封裝與 getter 方法
- 🛠️ 時間感知建構子
- ✅ 實體驗證集成
- 🔄 向後兼容性維護

**Security v3.0 中間件系統 (2025-10-20)**
- 🔐 環境感知 CORS 配置
- 🛡️ API Key 認證與商戶隔離
- 🔍 OpenTelemetry 分散式追蹤
- 🔧 DSN 密碼洩露修復

**Consumer v2.0 + 代碼重構 (2025-09-12)**
- 🚀 性能提升 3.3 倍 (10,000+ records/sec)
- 📊 錯誤率 <0.23%
- 🏗️ 代碼品質大幅提升
- 📋 完整文檔歸檔

**Router v2.0 統一管理 (2025-09-09)**
- 🔧 RouterManager 集中式路由協調
- 🧩 組件化路由設計
- 🛡️ 中間件統一管理
- 🔄 向後兼容性維護

**Core v1.0 身份管理 (2025-09-02)**
- 🏛️ Clean Architecture 實作
- 🔄 事件驅動架構
- 💾 身份實體管理
- 🔗 KDS 整合

### 🔄 當前階段
**系統穩定性優化完成，企業級可靠性達成**
- Player Sync Deadlock v7.0 Repository 層 deadlock 防護完成
- Agent Synchronization v6.0 代理身份管理完成
- Redis Cache v6.0 四階段功能增強完成
- Domain Model v5.0 標準化已完成
- 安全中間件系統已部署
- Consumer 性能優化穩定運行
- 系統整體架構現代化與穩定性達成

## 🛠️ 技術棧概覽

- **架構**: Clean Architecture + Hexagonal Pattern
- **語言**: Go 1.18+
- **數據庫**: MySQL (GORM)
- **緩存**: Redis (批次操作優化)
- **消息隊列**: AWS Kinesis Data Streams
- **監控**: OpenTelemetry 分散式追蹤
- **部署**: Docker + Kubernetes (Helm)

## 📈 關鍵性能指標

### Consumer Service (v2.0)
- **吞吐量**: 10,000+ records/sec
- **錯誤率**: <0.23%
- **處理延遲**: ~991μs 平均
- **批次效率**: 100 records/batch

### Player Sync Stability (v7.0) 🆕
- **Deadlock 錯誤率**: 0.81% → <0.05% (95%+ 改善)
- **最終成功率**: 99.95%+ (含重試)
- **系統穩定性**: Repository 層三重防護
- **部分成功問題**: 完全解決

### System Overall
- **API 響應時間**: <100ms (目標)
- **測試覆蓋率**: >85%
- **代碼品質**: 零 linter 警告
- **部署方式**: 多服務 (Web/Consumer/Worker)
- **系統可靠性**: 企業級 (99.95%+ 穩定度)

## 🔍 查找信息指南

### 想要了解...

**系統整體架構** → `CLAUDE-QUICK.md` - Clean Architecture 章節

**Consumer 重構詳情** → `archieve/2025-09/consumer-refactor/CONSUMER_REFACTOR_FINAL_SUMMARY.md`

**當前開發任務** → `CLAUDE-CURRENT.md` - 進行中任務章節

**性能優化經驗** → `refactor/REFACTOR_BEST_PRACTICES.md`

**API 使用方法** → `CLAUDE-QUICK.md` - API 快速測試章節

**故障排除** → `CLAUDE-QUICK.md` - 除錯技巧章節

## 📝 文檔維護

- **維護責任**: Claude Code Agent
- **更新頻率**: 隨專案進度即時更新
- **歸檔策略**: 重要里程碑完成後歸檔到 `archieve/`
- **版本控制**: 所有文檔納入 Git 版本控制

---

**專案**: Fat Identity Cat - 身份管理微服務  
**最後更新**: 2025-12-03  
**文檔版本**: v7.0 (系統穩定性優化完成)  
**維護狀態**: ✅ 積極維護中