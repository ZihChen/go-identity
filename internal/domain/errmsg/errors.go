package errmsg

import "errors"

var (
	ErrRepoMerchantNotFound       = errors.New("repo merchant not found")
	ErrRepoDeleteMerchantNotFound = errors.New("repo delete merchant not found")
	ErrRepoManagerNotFound        = errors.New("repo manager not found")
	ErrRepoDeleteManagerNotFound  = errors.New("repo delete manager not found")
	ErrRepoPlayerNotFound         = errors.New("repo player not found")
	ErrRepoDeletePlayerNotFound   = errors.New("repo delete player not found")
	ErrRepoLevelNotFound          = errors.New("repo level not found")
	ErrRepoAgentNotFound          = errors.New("repo agent not found")
	ErrRepoDeleteAgentNotFound    = errors.New("repo delete agent not found")
	ErrUnknownEventType           = errors.New("unknown event type")
)
