package infrastructure

import "time"

// DistributedMutex 代表一個分散式互斥鎖，不依賴 redsync 具體型別。
type DistributedMutex interface {
	Lock() error
	Unlock() (bool, error)
}

// LockOptions 配置分散式鎖的取得選項。
type LockOptions struct {
	Expiry     time.Duration
	Tries      int
	RetryDelay time.Duration
}

// DistributedLockService 提供分散式鎖的抽象，供 domain/application 層使用。
// 由 infrastructure/cache/redis/lock.go 的 RedisLockService 實作。
type DistributedLockService interface {
	// GetLock 取得一個簡單的分散式鎖（使用預設重試策略）。
	GetLock(key string, ttl time.Duration) (DistributedMutex, error)
	// GetLockWithOptions 取得一個帶自訂選項的分散式鎖。
	GetLockWithOptions(key string, opts LockOptions) (DistributedMutex, error)
}
