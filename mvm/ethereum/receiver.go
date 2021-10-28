package ethereum

import (
	"context"

	"github.com/MixinNetwork/nfo/mtg"
)

type GroupReceiver struct {
}

func NewGroupReceiver() *GroupReceiver {
	return nil
}

func (r *GroupReceiver) ProcessOutput(context.Context, *mtg.Output) {
}

func (r *GroupReceiver) ProcessCollectibleOutput(context.Context, *mtg.CollectibleOutput) {
}
