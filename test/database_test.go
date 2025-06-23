// 檔案: tests/database_test.go
package tests

import (
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/database"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"github.com/stretchr/testify/assert"
)

func TestDBConnection(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 初始化資料庫連接
	db, err := database.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 測試資料庫連接
	sqlDB, err := db.DB.DB()
	if err != nil {
		t.Fatalf("Failed to get DB instance: %v", err)
	}

	err = sqlDB.Ping()
	assert.NoError(t, err, "Should be able to ping the database")
}

func TestMerchantCRUD(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 初始化資料庫連接
	db, err := database.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 測試資料 - 商戶
	testMerchant := &models.Merchant{
		GlobalMerchantID: "TEST-MERCHANT-" + time.Now().Format("20060102150405"),
		Name:             "Test Merchant",
		DisplayName:      "Test Merchant Display",
		APIKey:           "test-api-key-" + time.Now().Format("20060102150405"),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// 創建
	result := db.DB.Create(testMerchant)
	assert.NoError(t, result.Error, "Should create a merchant without error")
	assert.Greater(t, testMerchant.ID, uint64(0), "Should assign an ID to the merchant")

	// 讀取
	var readMerchant model.Merchant
	result = db.DB.First(&readMerchant, testMerchant.ID)
	assert.NoError(t, result.Error, "Should read the merchant without error")
	assert.Equal(t, testMerchant.Name, readMerchant.Name, "Merchant name should match")

	// 更新
	newName := "Updated Test Merchant"
	readMerchant.Name = newName
	result = db.DB.Save(&readMerchant)
	assert.NoError(t, result.Error, "Should update the merchant without error")

	// 驗證更新
	var updatedMerchant model.Merchant
	result = db.DB.First(&updatedMerchant, testMerchant.ID)
	assert.NoError(t, result.Error, "Should read the updated merchant without error")
	assert.Equal(t, newName, updatedMerchant.Name, "Merchant name should be updated")

	// 清理 - 使用軟刪除
	result = db.DB.Delete(&updatedMerchant)
	assert.NoError(t, result.Error, "Should soft-delete the merchant without error")

	// 驗證軟刪除
	var deletedMerchant model.Merchant
	result = db.DB.Unscoped().First(&deletedMerchant, testMerchant.ID)
	assert.NoError(t, result.Error, "Should find the merchant with unscoped query")
	assert.NotNil(t, deletedMerchant.DeletedAt, "DeletedAt should be set")
}

func TestPlayerCRUD(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 初始化資料庫連接
	db, err := database.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 首先創建一個商戶，作為外鍵關聯
	merchant := &model.Merchant{
		GlobalMerchantID: "TEST-MERCHANT-PLAYER-" + time.Now().Format("20060102150405"),
		Name:             "Test Merchant for Player",
		DisplayName:      "Test Merchant Display",
		APIKey:           "test-api-key-" + time.Now().Format("20060102150405"),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	db.DB.Create(merchant)

	// 測試資料 - 玩家
	email := "test@example.com"
	testPlayer := &model.Player{
		MerchantID:     merchant.ID,
		GlobalPlayerID: "TEST-PLAYER-" + time.Now().Format("20060102150405"),
		APIKey:         "test-player-key-" + time.Now().Format("20060102150405"),
		Account:        "testplayer",
		Email:          &email,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 創建
	result := db.DB.Create(testPlayer)
	assert.NoError(t, result.Error, "Should create a player without error")
	assert.Greater(t, testPlayer.ID, uint64(0), "Should assign an ID to the player")

	// 讀取
	var readPlayer model.Player
	result = db.DB.First(&readPlayer, testPlayer.ID)
	assert.NoError(t, result.Error, "Should read the player without error")
	assert.Equal(t, testPlayer.Account, readPlayer.Account, "Player account should match")

	// 更新
	newAccount := "updatedplayer"
	readPlayer.Account = newAccount
	result = db.DB.Save(&readPlayer)
	assert.NoError(t, result.Error, "Should update the player without error")

	// 驗證更新
	var updatedPlayer model.Player
	result = db.DB.First(&updatedPlayer, testPlayer.ID)
	assert.NoError(t, result.Error, "Should read the updated player without error")
	assert.Equal(t, newAccount, updatedPlayer.Account, "Player account should be updated")

	// 清理 - 使用軟刪除
	result = db.DB.Delete(&updatedPlayer)
	assert.NoError(t, result.Error, "Should soft-delete the player without error")
	result = db.DB.Delete(merchant)
	assert.NoError(t, result.Error, "Should soft-delete the merchant without error")
}

func TestManagerCRUD(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 初始化資料庫連接
	db, err := database.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 首先創建一個商戶，作為外鍵關聯
	merchant := &model.Merchant{
		GlobalMerchantID: "TEST-MERCHANT-MANAGER-" + time.Now().Format("20060102150405"),
		Name:             "Test Merchant for Manager",
		DisplayName:      "Test Merchant Display",
		APIKey:           "test-api-key-" + time.Now().Format("20060102150405"),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	db.DB.Create(merchant)

	// 測試資料 - 管理員
	email := "manager@example.com"
	testManager := &model.Manager{
		MerchantID:      merchant.ID,
		GlobalManagerID: "TEST-MANAGER-" + time.Now().Format("20060102150405"),
		Account:         "testmanager",
		Email:           &email,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// 創建
	result := db.DB.Create(testManager)
	assert.NoError(t, result.Error, "Should create a manager without error")
	assert.Greater(t, testManager.ID, uint64(0), "Should assign an ID to the manager")

	// 讀取
	var readManager model.Manager
	result = db.DB.First(&readManager, testManager.ID)
	assert.NoError(t, result.Error, "Should read the manager without error")
	assert.Equal(t, testManager.Account, readManager.Account, "Manager account should match")

	// 更新
	newAccount := "updatedmanager"
	readManager.Account = newAccount
	result = db.DB.Save(&readManager)
	assert.NoError(t, result.Error, "Should update the manager without error")

	// 驗證更新
	var updatedManager model.Manager
	result = db.DB.First(&updatedManager, testManager.ID)
	assert.NoError(t, result.Error, "Should read the updated manager without error")
	assert.Equal(t, newAccount, updatedManager.Account, "Manager account should be updated")

	// 清理 - 使用軟刪除
	result = db.DB.Delete(&updatedManager)
	assert.NoError(t, result.Error, "Should soft-delete the manager without error")
	result = db.DB.Delete(merchant)
	assert.NoError(t, result.Error, "Should soft-delete the merchant without error")
}
