package consts

type contextKey string

const (
	TraceIDKey contextKey = "trace_id"
	SpanIDKey  contextKey = "span_id"
	EventIDKey contextKey = "event_id"
)

const (
	ShardMutexRedisKey    = "kds:shard:mutex:%s:%s"
	SyncPlayerTagRedisKey = "worker:sync:play_tag:%d"

	RedisMerchantGlobalIDKey    = "merchant:global_id:%s"
	RedisPlayerGlobalIDKey      = "player:global_id:%s"
	RedisPlayerLevelGlobalIDKey = "player_level:global_id:%s"
)
