package utils

import (
	"crypto/subtle"
	"time"
)

// ValidateAPIKeyConstantTime 使用常數時間比較防止時序攻擊
// 採用加強版安全實現：
// 1. 長度預檢查避免不必要的計算
// 2. 完整遍歷所有key避免提前退出的時序洩露
// 3. 失敗時額外延遲進一步混淆時序特徵
func ValidateAPIKeyConstantTime(providedKey string, validKeys map[string]string) (string, bool) {
	var merchantID string
	found := false

	// 遍歷所有有效key，即使找到匹配也不提前退出
	for key, id := range validKeys {
		// 長度預檢查，避免對不同長度的key進行costly的常數時間比較
		if len(providedKey) == len(key) {
			if subtle.ConstantTimeCompare([]byte(providedKey), []byte(key)) == 1 {
				merchantID = id
				found = true
				// 不使用return，繼續遍歷剩餘key以避免時序洩露
			}
		}
	}

	// 驗證失敗時加入固定延遲，進一步混淆時序特徵
	if !found {
		time.Sleep(time.Millisecond * 2)
	}

	return merchantID, found
}
