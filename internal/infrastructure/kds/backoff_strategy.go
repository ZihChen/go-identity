package kds

import "time"

// BackoffManager 管理自適應退避策略
// 根據KDS記錄獲取情況動態調整請求頻率，避免過度頻繁請求造成資源浪費
// 同時在出現錯誤時實施退避策略，提高系統穩定性
type BackoffManager struct {
	// currentBackoff 當前退避時間
	currentBackoff time.Duration

	// minBackoff 最小退避時間，防止退避時間過短
	minBackoff time.Duration

	// maxBackoff 最大退避時間，防止退避時間過長影響處理效率
	maxBackoff time.Duration
}

// NewBackoffManager 創建退避管理器（公共構造函數，用於測試）
func NewBackoffManager(minBackoff, maxBackoff time.Duration) *BackoffManager {
	return &BackoffManager{
		currentBackoff: minBackoff,
		minBackoff:     minBackoff,
		maxBackoff:     maxBackoff,
	}
}

// createBackoffManager 創建退避管理器（內部使用）
func (k *KDSService) createBackoffManager() *BackoffManager {
	return NewBackoffManager(k.config.Consumer.MinBackoff, k.config.Consumer.MaxBackoff)
}

// IncreaseBackoff 增加退避時間
// 當出現錯誤時使用，以 1.5 倍的速率增加退避時間
func (b *BackoffManager) IncreaseBackoff() {
	b.currentBackoff = time.Duration(float64(b.currentBackoff) * 1.5)
	if b.currentBackoff > b.maxBackoff {
		b.currentBackoff = b.maxBackoff
	}
}

// DecreaseBackoff 減少退避時間
// 當無錯誤時使用，以 0.8 倍的速率減少退避時間
func (b *BackoffManager) DecreaseBackoff() {
	b.currentBackoff = time.Duration(float64(b.currentBackoff) * 0.8)
	if b.currentBackoff < b.minBackoff {
		b.currentBackoff = b.minBackoff
	}
}

// AdjustBackoffByRecordCount 根據記錄數量調整退避時間
// 當無記錄時增加退避時間，有記錄時減少退避時間
func (b *BackoffManager) AdjustBackoffByRecordCount(recordCount int) {
	if recordCount == 0 {
		b.currentBackoff = time.Duration(float64(b.currentBackoff) * 1.2)
		if b.currentBackoff > b.maxBackoff {
			b.currentBackoff = b.maxBackoff
		}
	} else {
		b.currentBackoff = time.Duration(float64(b.currentBackoff) * 0.8)
		if b.currentBackoff < b.minBackoff {
			b.currentBackoff = b.minBackoff
		}
	}
}

// GetCurrentBackoff 獲取當前退避時間
func (b *BackoffManager) GetCurrentBackoff() time.Duration {
	return b.currentBackoff
}
