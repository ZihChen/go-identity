package redis

import (
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
)

// Compile-time interface checks
var _ infrastructure.DistributedLockService = (*RedisLockService)(nil)
var _ infrastructure.DistributedMutex = (*redsync.Mutex)(nil)

// RedisLockService 實作 infrastructure.DistributedLockService，使用 redsync。
type RedisLockService struct {
	manager *Manager
}

// NewRedisLockService 建立 RedisLockService 實例。
func NewRedisLockService(manager *Manager) *RedisLockService {
	return &RedisLockService{manager: manager}
}

func (s *RedisLockService) GetLock(key string, ttl time.Duration) (infrastructure.DistributedMutex, error) {
	rs, err := s.manager.GetRedsync()
	if err != nil {
		return nil, err
	}
	return rs.NewMutex(key, redsync.WithExpiry(ttl)), nil
}

func (s *RedisLockService) GetLockWithOptions(key string, opts infrastructure.LockOptions) (infrastructure.DistributedMutex, error) {
	rs, err := s.manager.GetRedsync()
	if err != nil {
		return nil, err
	}
	rsOpts := []redsync.Option{redsync.WithExpiry(opts.Expiry)}
	if opts.Tries > 0 {
		rsOpts = append(rsOpts, redsync.WithTries(opts.Tries))
	}
	if opts.RetryDelay > 0 {
		rsOpts = append(rsOpts, redsync.WithRetryDelay(opts.RetryDelay))
	}
	return rs.NewMutex(key, rsOpts...), nil
}
