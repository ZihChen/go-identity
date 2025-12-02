package queue

import (
	"testing"
	"time"
)

func TestGetWorkerConfigByEnv(t *testing.T) {
	tests := []struct {
		name                string
		env                 string
		expectedConcurrency int
		expectedQueues      map[string]int
		expectedMaxRetries  int
		expectedTimeout     time.Duration
		expectedPoolSize    int
		expectedDialTimeout time.Duration
	}{
		{
			name:                "Development environment",
			env:                 "development",
			expectedConcurrency: 5,
			expectedQueues: map[string]int{
				"default":  2,
				"critical": 3,
			},
			expectedMaxRetries:  3,
			expectedTimeout:     15 * time.Second,
			expectedPoolSize:    8,
			expectedDialTimeout: 3 * time.Second,
		},
		{
			name:                "Dev environment (short form)",
			env:                 "dev",
			expectedConcurrency: 5,
			expectedQueues: map[string]int{
				"default":  2,
				"critical": 3,
			},
			expectedMaxRetries:  3,
			expectedTimeout:     15 * time.Second,
			expectedPoolSize:    8,
			expectedDialTimeout: 3 * time.Second,
		},
		{
			name:                "Production environment",
			env:                 "production",
			expectedConcurrency: 8,
			expectedQueues: map[string]int{
				"default":  5,
				"critical": 10,
			},
			expectedMaxRetries:  5,
			expectedTimeout:     60 * time.Second,
			expectedPoolSize:    15,
			expectedDialTimeout: 5 * time.Second,
		},
		{
			name:                "Prod environment (short form)",
			env:                 "prod",
			expectedConcurrency: 8,
			expectedQueues: map[string]int{
				"default":  5,
				"critical": 10,
			},
			expectedMaxRetries:  5,
			expectedTimeout:     60 * time.Second,
			expectedPoolSize:    15,
			expectedDialTimeout: 5 * time.Second,
		},
		{
			name:                "Unknown environment (defaults)",
			env:                 "staging",
			expectedConcurrency: 5,
			expectedQueues: map[string]int{
				"default":  5,
				"critical": 10,
			},
			expectedMaxRetries:  3,
			expectedTimeout:     30 * time.Second,
			expectedPoolSize:    10,
			expectedDialTimeout: 4 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := getWorkerConfigByEnv(tt.env)

			// Test concurrency
			if config.Concurrency != tt.expectedConcurrency {
				t.Errorf(
					"Expected concurrency %d, got %d",
					tt.expectedConcurrency,
					config.Concurrency,
				)
			}

			// Test queue priorities
			if len(config.QueuePriorities) != len(tt.expectedQueues) {
				t.Errorf(
					"Expected %d queues, got %d",
					len(tt.expectedQueues),
					len(config.QueuePriorities),
				)
			}

			for queueName, expectedPriority := range tt.expectedQueues {
				if actualPriority, exists := config.QueuePriorities[queueName]; !exists {
					t.Errorf("Queue %s not found in configuration", queueName)
				} else if actualPriority != expectedPriority {
					t.Errorf("Queue %s: expected priority %d, got %d", queueName, expectedPriority, actualPriority)
				}
			}

			// Test max retries
			if config.MaxRetries != tt.expectedMaxRetries {
				t.Errorf(
					"Expected max retries %d, got %d",
					tt.expectedMaxRetries,
					config.MaxRetries,
				)
			}

			// Test timeout
			if config.TaskTimeout != tt.expectedTimeout {
				t.Errorf("Expected timeout %v, got %v", tt.expectedTimeout, config.TaskTimeout)
			}

			// Test retry delay function exists
			if config.RetryDelay == nil {
				t.Error("RetryDelay function should not be nil")
			}

			// Test Redis pool configuration
			if config.RedisPoolSize != tt.expectedPoolSize {
				t.Errorf("Expected pool size %d, got %d", tt.expectedPoolSize, config.RedisPoolSize)
			}

			if config.RedisDialTimeout != tt.expectedDialTimeout {
				t.Errorf(
					"Expected dial timeout %v, got %v",
					tt.expectedDialTimeout,
					config.RedisDialTimeout,
				)
			}
		})
	}
}

func TestRetryDelayFunctions(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		retryNum int
		maxDelay time.Duration
	}{
		{
			name:     "Development linear backoff",
			env:      "dev",
			retryNum: 3,
			maxDelay: 6 * time.Second, // n * 2 = 3 * 2 = 6s
		},
		{
			name:     "Production exponential backoff",
			env:      "prod",
			retryNum: 3,
			maxDelay: 9 * time.Second, // n^2 = 3^2 = 9s
		},
		{
			name:     "Production backoff with cap",
			env:      "prod",
			retryNum: 20,
			maxDelay: 5 * time.Minute, // Should be capped at 5 minutes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := getWorkerConfigByEnv(tt.env)
			delay := config.RetryDelay(tt.retryNum)

			if tt.env == "prod" && tt.retryNum == 20 {
				// For large retry numbers in prod, should be capped
				if delay != tt.maxDelay {
					t.Errorf("Expected delay to be capped at %v, got %v", tt.maxDelay, delay)
				}
			} else {
				// For normal cases, check exact calculation
				if delay != tt.maxDelay {
					t.Errorf("Expected delay %v, got %v", tt.maxDelay, delay)
				}
			}
		})
	}
}

func TestEnvironmentSpecificConfigurations(t *testing.T) {
	devConfig := getDevConfig()
	prodConfig := getProdConfig()
	defaultConfig := getDefaultConfig()

	// Dev should have lower resource usage
	if devConfig.Concurrency >= prodConfig.Concurrency {
		t.Error("Dev environment should have lower concurrency than prod")
	}

	if devConfig.TaskTimeout >= prodConfig.TaskTimeout {
		t.Error("Dev environment should have shorter timeout than prod")
	}

	if devConfig.MaxRetries > prodConfig.MaxRetries {
		t.Error("Dev environment should have fewer retries than prod")
	}

	// Default should be between dev and prod
	if defaultConfig.Concurrency < devConfig.Concurrency ||
		defaultConfig.Concurrency > prodConfig.Concurrency {
		t.Error("Default concurrency should be between dev and prod")
	}

	// Both environments should have same basic queues (default, critical)
	// No bulk queue in current implementation
	prodQueues := []string{"default", "critical"}
	devQueues := []string{"default", "critical"}

	for _, queue := range prodQueues {
		if _, exists := prodConfig.QueuePriorities[queue]; !exists {
			t.Errorf("Prod environment should have %s queue", queue)
		}
	}

	for _, queue := range devQueues {
		if _, exists := devConfig.QueuePriorities[queue]; !exists {
			t.Errorf("Dev environment should have %s queue", queue)
		}
	}
}
