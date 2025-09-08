package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
)

type TagUseCase interface {
	SyncPlayerTag(
		ctx context.Context,
		data []event.TagData,
		globalMerchantID, globalPlayerID string,
	) error
	SyncTag(ctx context.Context, data *event.TagSyncEvent) error
}
