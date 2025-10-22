# 領域貧血模型重構計劃

**日期**: 2025-10-21  
**重構類型**: 貧血領域模型 → 豐富領域模型  
**目標**: `internal/domain/entity`  
**狀態**: 規劃階段

## 1. 問題分析

目前專案採用貧血領域模型（Anemic Domain Model）。`internal/domain/entity` 中的實體主要是具有公共欄位且無業務邏輯的資料結構。所有業務邏輯，包括物件建立、驗證和狀態轉換，都位於用例層（`internal/application/usecase`）。

這會導致：
- **低內聚性**: 業務邏輯與其操作的資料分離
- **高耦合性**: 用例與實體的內部結構緊密耦合
- **隱藏的業務邏輯**: 業務規則分散在各個用例中，難以尋找、理解和重用
- **降低可維護性**: 業務規則變更可能需要修改多個用例

## 2. 重構目標

目標是將貧血領域模型轉換為豐富領域模型，通過在領域實體內封裝業務邏輯來實現。這將涉及：

- **封裝資料**: 將實體欄位設為私有，以控制存取和修改
- **引入行為**: 為實體添加方法來執行業務操作和強制不變性
- **建立建構子**: 使用建構函數確保實體在有效狀態下建立
- **提高內聚性並降低耦合性**: 在領域實體內結合資料和行為

## 3. 重構計劃（範例：`Player` 實體）

本計劃將以 `Player` 實體為主要範例。相同原則可應用於其他實體，如 `Merchant`、`Manager` 等。

### 步驟 1: 封裝 `Player` 欄位

**檔案**: `internal/domain/entity/entity.go`

1. **將欄位設為私有**: 將 `Player` 結構的欄位從公共（如 `Account`）改為私有（如 `account`）
2. **提供 getter 方法**: 為需要從實體外部存取的欄位建立公共 getter 方法（如 `Account()`）

**重構前:**
```go
type Player struct {
    ID             uint64
    Account        string
    // ...
}
```

**重構後:**
```go
type Player struct {
    id             uint64
    account        string
    // ...
}

func (p *Player) ID() uint64 { return p.id }
func (p *Player) Account() string { return p.account }
// ... 其他 getter 方法
```

### 步驟 2: 建立 `Player` 建構子

1. **建立 `NewPlayer` 函數**: 此函數負責建立有效狀態的新 `Player`
2. **移動建立邏輯**: 將 `APIKey` 生成和其他初始化邏輯從 `PlayerUseCase.SyncPlayer` 移至 `NewPlayer` 建構子

**重構前（在 `PlayerUseCase.SyncPlayer` 中）:**
```go
player := entity.Player{
    // ...
    APIKey:         uuid.New().String(),
    // ...
}
```

**重構後（在 `internal/domain/entity/entity.go` 中）:**
```go
func NewPlayer(merchantID uint64, globalPlayerID string, account string /* 其他必要欄位 */) *Player {
    return &Player{
        merchantID:     merchantID,
        globalPlayerID: globalPlayerID,
        apiKey:         uuid.New().String(),
        account:        account,
        createdAt:      time.Now(),
        updatedAt:      time.Now(),
        // ... 其他欄位初始化
    }
}
```

### 步驟 3: 為 `Player` 實體添加行為

1. **建立 `UpdateLastActive` 方法**: 將更新玩家最後活動時間的邏輯從 `PlayerUseCase.UpdatePlayerLastActive` 移至 `Player` 結構的方法

**重構前（在 `PlayerUseCase.UpdatePlayerLastActive` 中）:**
```go
now := time.Now()
player.LastActiveAt = &now
player.UpdatedAt = now
```

**重構後（在 `internal/domain/entity/entity.go` 中）:**
```go
func (p *Player) UpdateLastActive() {
    now := time.Now()
    p.lastActiveAt = &now
    p.updatedAt = now
}
```

2. **建立其他方法**: 識別用例中可移至 `Player` 實體的其他業務邏輯，如 `ChangeLevel`、`Deactivate`、`AddTag`、`RemoveTag` 等

**補充建議的方法:**
```go
// 等級管理
func (p *Player) ChangeLevel(newLevelID uint64) error {
    if newLevelID == 0 {
        return errors.New("invalid level ID")
    }
    p.levelID = newLevelID
    p.updatedAt = time.Now()
    return nil
}

// 狀態管理
func (p *Player) Activate() {
    p.isActive = true
    p.updatedAt = time.Now()
}

func (p *Player) Deactivate() {
    p.isActive = false
    p.updatedAt = time.Now()
}

// 標籤管理（業務邏輯驗證）
func (p *Player) CanAddTag(tagID uint64) bool {
    // 業務規則：檢查是否已有此標籤、標籤數量限制等
    return true // 簡化示例
}

// 驗證方法
func (p *Player) IsValid() error {
    if p.account == "" {
        return errors.New("player account cannot be empty")
    }
    if p.merchantID == 0 {
        return errors.New("player must belong to a merchant")
    }
    return nil
}
```

### 步驟 4: 重構 `PlayerUseCase`

1. **更新 `PlayerUseCase` 使用新的實體方法**: 用對 `Player` 實體新方法的呼叫替換直接欄位存取和邏輯

**重構前:**
```go
// in PlayerUseCase.UpdatePlayerLastActive
now := time.Now()
player.LastActiveAt = &now
player.UpdatedAt = now
if err := u.playerRepo.Update(ctx, player); err != nil {
    // ...
}
```

**重構後:**
```go
// in PlayerUseCase.UpdatePlayerLastActive
player, err := u.playerRepo.FindByID(ctx, id)
if err != nil {
    return err
}
player.UpdateLastActive()
if err := u.playerRepo.Update(ctx, player); err != nil {
    // ...
}
```

## 4. 其他實體的通用重構計劃

以下步驟也應該應用於其他實體（`Merchant`、`Manager`、`Level`、`Tag`）：

### 4.1 `Merchant` 實體建議

**重點業務邏輯:**
```go
func NewMerchant(globalMerchantID, name string) *Merchant {
    return &Merchant{
        globalMerchantID: globalMerchantID,
        name:            name,
        apiKey:          uuid.New().String(),
        isActive:        true,
        createdAt:       time.Now(),
        updatedAt:       time.Now(),
    }
}

func (m *Merchant) RegenerateAPIKey() {
    m.apiKey = uuid.New().String()
    m.updatedAt = time.Now()
}

func (m *Merchant) UpdateName(newName string) error {
    if newName == "" {
        return errors.New("merchant name cannot be empty")
    }
    m.name = newName
    m.updatedAt = time.Now()
    return nil
}
```

### 4.2 `Tag` 實體建議

**標籤邏輯封裝:**
```go
func NewTag(merchantID uint64, name, description string) *Tag {
    return &Tag{
        merchantID:  merchantID,
        name:        name,
        description: description,
        isActive:    true,
        createdAt:   time.Now(),
        updatedAt:   time.Now(),
    }
}

func (t *Tag) UpdateDescription(newDesc string) {
    t.description = newDesc
    t.updatedAt = time.Now()
}

func (t *Tag) IsValidForMerchant(merchantID uint64) bool {
    return t.merchantID == merchantID && t.isActive
}
```

### 4.3 通用步驟

1. **識別業務邏輯**: 分析每個實體的用例，識別可移至實體本身的業務邏輯
2. **封裝欄位**: 將實體欄位設為私有並提供 getter 方法
3. **建立建構子**: 為每個實體實作 `New...` 函數確保有效建立
4. **添加行為**: 在實體上建立方法來封裝業務邏輯
5. **重構用例**: 更新用例以使用新的、更豐富的領域實體

## 5. 實作策略與風險管控

### 5.1 分階段實作

**第一階段: Player 實體（高優先級）**
- 封裝 Player 欄位和添加 getter 方法
- 實作 NewPlayer 建構子
- 添加核心業務方法（UpdateLastActive, ChangeLevel）
- 更新相關用例和測試

**第二階段: Merchant 實體（中優先級）**
- 類似 Player 的重構步驟
- 重點關注 API Key 管理邏輯

**第三階段: 其他實體（低優先級）**
- Manager, Level, Tag 實體重構
- 標籤關聯邏輯重構

### 5.2 向後兼容性考量

**過渡期間策略:**
```go
// 在重構期間保持舊的公共欄位
type Player struct {
    // 新的私有欄位
    id      uint64
    account string
    
    // 向後兼容的公共欄位（標記為 deprecated）
    ID      uint64 `json:"id" deprecated:"use ID() method instead"`
    Account string `json:"account" deprecated:"use Account() method instead"`
}

// 同步方法確保資料一致性
func (p *Player) syncFields() {
    p.ID = p.id
    p.Account = p.account
}
```

### 5.3 測試策略

**單元測試重點:**
```go
// 測試實體行為
func TestPlayer_UpdateLastActive(t *testing.T) {
    player := NewPlayer(1, "global123", "test@example.com")
    oldUpdatedAt := player.UpdatedAt()
    
    time.Sleep(1 * time.Millisecond)
    player.UpdateLastActive()
    
    assert.NotNil(t, player.LastActiveAt())
    assert.True(t, player.UpdatedAt().After(oldUpdatedAt))
}

// 測試業務規則
func TestPlayer_ChangeLevel_Validation(t *testing.T) {
    player := NewPlayer(1, "global123", "test@example.com")
    
    err := player.ChangeLevel(0)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid level ID")
}
```

## 6. 預期效益

### 6.1 程式碼品質提升
- **更好的組織**: 程式碼將更有組織性、內聚性且易於理解
- **提高可維護性**: 業務邏輯將集中在領域實體中，使其更容易修改和維護
- **減少錯誤**: 封裝和不變性將有助於防止建立無效的實體狀態

### 6.2 架構清晰度
- **更好的可測試性**: 領域邏輯可以獨立測試，無需複雜的用例設定
- **更清晰的架構**: 領域層和應用層之間的關注點分離將更加明確
- **業務邏輯集中化**: 業務規則不再分散，容易找到和理解

### 6.3 開發效率
- **減少重複代碼**: 常用的業務邏輯封裝在實體中，避免在多個用例中重複
- **更快的開發速度**: 新功能開發時可直接使用實體的業務方法
- **更容易的程式碼審查**: 業務邏輯集中，程式碼審查更有效率

## 7. 風險評估與緩解

### 7.1 低風險
- **現有測試覆蓋**: 重構過程中現有測試可以確保功能不變
- **漸進式重構**: 分階段實作降低風險

### 7.2 中風險
- **資料庫映射**: GORM 標籤和序列化可能需要調整
- **API 回應**: JSON 序列化需要確保向後兼容

### 7.3 緩解策略
- **充分測試**: 每個階段完成後執行完整的測試套件
- **向後兼容**: 保持公共 API 不變，內部逐步重構
- **文件更新**: 及時更新 API 文件和開發指南

---

**注意**: 此重構將維持完全的向後兼容性，同時實現豐富領域模型的架構目標。分階段的方法確保最小風險並允許每個步驟的增量驗證。