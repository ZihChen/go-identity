package queue

import (
	"time"
)

// WorkerConfig 工作器配置結構
type WorkerConfig struct {
	Concurrency     int                     // 並發工作器數量
	QueuePriorities map[string]int          // 佇列優先級配置
	TaskTimeout     time.Duration           // 任務超時時間
	MaxRetries      int                     // 最大重試次數
	RetryDelay      func(int) time.Duration // 重試延遲函數
}

// getDevConfig 開發環境配置
// 針對 64Mi memory, 100m-500m CPU 的資源限制進行優化
func getDevConfig() WorkerConfig {
	return WorkerConfig{
		Concurrency: 3, // 減少併發以避免記憶體超限
		QueuePriorities: map[string]int{
			"default":  2, // 降低優先級數量
			"critical": 3,
		},
		TaskTimeout: 15 * time.Second, // 縮短超時避免資源卡住
		MaxRetries:  3,                // 減少重試次數
		RetryDelay: func(n int) time.Duration {
			// 線性退避策略，避免指數增長消耗資源
			return time.Duration(n) * 2 * time.Second
		},
	}
}

// getProdConfig 生產環境配置
// 針對 AWS Valkey 雙核心 4G 記憶體進行優化 (3 pods部署)
func getProdConfig() WorkerConfig {
	return WorkerConfig{
		Concurrency: 6, // 3 workers/core × 2 cores = 6，避免過度併發
		QueuePriorities: map[string]int{
			"default":  5,  // 標準優先級
			"critical": 10, // 高優先級
		},
		TaskTimeout: 60 * time.Second, // 增加超時容忍度
		MaxRetries:  5,                // 更多的重試機會
		RetryDelay: func(n int) time.Duration {
			// 指數退避策略，但設置上限
			delay := time.Duration(n*n) * time.Second
			maxDelay := 5 * time.Minute
			if delay > maxDelay {
				return maxDelay
			}
			return delay
		},
	}
}

// getDefaultConfig 默認配置
// 用於未明確指定環境的情況
func getDefaultConfig() WorkerConfig {
	return WorkerConfig{
		Concurrency: 5, // 中等併發度
		QueuePriorities: map[string]int{
			"default":  5,
			"critical": 10,
		},
		TaskTimeout: 30 * time.Second, // 中等超時時間
		MaxRetries:  3,                // 中等重試次數
		RetryDelay: func(n int) time.Duration {
			// 溫和的指數退避
			delay := time.Duration(n*n) * time.Second
			maxDelay := 2 * time.Minute
			if delay > maxDelay {
				return maxDelay
			}
			return delay
		},
	}
}

// getWorkerConfigByEnv 根據環境獲取工作器配置
func getWorkerConfigByEnv(env string) WorkerConfig {
	switch env {
	case "development", "dev":
		return getDevConfig()
	case "production", "prod":
		return getProdConfig()
	default:
		return getDefaultConfig()
	}
}
