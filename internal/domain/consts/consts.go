package consts

type contextKey string

const (
	TraceIDKey contextKey = "trace_id"
	SpanIDKey  contextKey = "span_id"
	EventIDKey contextKey = "event_id"
)

const (
	ShardMutexRedisKey = "kds:shard:mutex:%s:%s"
)
