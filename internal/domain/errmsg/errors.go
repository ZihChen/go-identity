package errmsg

import "errors"

var (
	ErrRepoMerchantNotFound        = errors.New("repo merchant not found")
	ErrRepoDeleteMerchantNotFound  = errors.New("repo delete merchant not found")
	ErrRepoManagerNotFound         = errors.New("repo manager not found")
	ErrRepoDeleteManagerNotFound   = errors.New("repo delete manager not found")
	ErrRepoPlayerNotFound          = errors.New("repo player not found")
	ErrRepoDeletePlayerNotFound    = errors.New("repo delete player not found")
	ErrRepoLevelNotFound           = errors.New("repo level not found")
	ErrRepoAgentNotFound           = errors.New("repo agent not found")
	ErrRepoFailedTaskEventNotFound = errors.New("repo failed task event not found")
	ErrInvalidEntity               = errors.New("invalid entity")
	ErrUnknownEventType            = errors.New("unknown event type")

	// Merchant entity validation errors
	ErrMerchantGlobalIDEmpty    = errors.New("merchant global ID cannot be empty")
	ErrMerchantNameEmpty        = errors.New("merchant name cannot be empty")
	ErrMerchantDisplayNameEmpty = errors.New("merchant display name cannot be empty")
	ErrMerchantAPIKeyEmpty      = errors.New("merchant API key cannot be empty")

	// Player entity validation errors
	ErrPlayerGlobalIDEmpty  = errors.New("player global ID cannot be empty")
	ErrPlayerAccountEmpty   = errors.New("player account cannot be empty")
	ErrPlayerNoMerchant     = errors.New("player must belong to a merchant")
	ErrPlayerInvalidLevelID = errors.New("invalid level ID")

	// Manager entity validation errors
	ErrManagerGlobalIDEmpty = errors.New("manager global ID cannot be empty")
	ErrManagerAccountEmpty  = errors.New("manager account cannot be empty")
	ErrManagerNoMerchant    = errors.New("manager must belong to a merchant")

	// Tag entity validation errors
	ErrTagGlobalIDEmpty = errors.New("tag global ID cannot be empty")
	ErrTagNameEmpty     = errors.New("tag name cannot be empty")
	ErrTagNoMerchant    = errors.New("tag must belong to a merchant")

	// Level entity validation errors
	ErrLevelGlobalIDEmpty = errors.New("level global player level ID cannot be empty")
	ErrLevelNameEmpty     = errors.New("level name cannot be empty")
	ErrLevelNoMerchant    = errors.New("level must belong to a merchant")

	// Agent entity validation errors
	ErrAgentGlobalIDEmpty = errors.New("agent global ID cannot be empty")
	ErrAgentAccountEmpty  = errors.New("agent account cannot be empty")
	ErrAgentNoMerchant    = errors.New("agent merchant ID cannot be zero")
)
