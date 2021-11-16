package machine

import (
	"context"
	"encoding/base64"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

type GroupReceiver struct {
	Machine *Machine
}

func NewGroupReceiver() *GroupReceiver {
	return nil
}

func (r *GroupReceiver) ProcessOutput(ctx context.Context, out *mtg.Output) {
	op, err := r.parseOperation(out.Memo)
	if err != nil {
		logger.Verbosef("parseOperation(%s) => %s", out.Memo, err)
		return
	}
	switch op.Purpose {
	case encoding.OperationPurposeAddProcess:
		r.Machine.AddProcess(out.Sender, op.Platform, op.Address, out)
	case encoding.OperationPurposeGroupEvent:
		r.Machine.WriteGroupEvent(op.Process, out, op.Extra)
	}
}

func (r *GroupReceiver) ProcessCollectibleOutput(context.Context, *mtg.CollectibleOutput) {
}

func (r *GroupReceiver) parseOperation(memo string) (*encoding.Operation, error) {
	b, err := base64.RawURLEncoding.DecodeString(memo)
	if err != nil {
		return nil, err
	}
	return encoding.DecodeOperation(b)
}
