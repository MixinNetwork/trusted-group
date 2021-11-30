package machine

import (
	"context"
	"encoding/base64"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

func (m *Machine) ProcessOutput(ctx context.Context, out *mtg.Output) {
	op, err := parseOperation(out.Memo)
	if err != nil {
		logger.Verbosef("parseOperation(%s) => %s", out.Memo, err)
		return
	}
	switch op.Purpose {
	case encoding.OperationPurposeAddProcess:
		m.AddProcess(ctx, op.Process, op.Platform, op.Address, out, op.Extra)
	case encoding.OperationPurposeGroupEvent:
		m.WriteGroupEvent(op.Process, out, op.Extra)
	}
}

func (m *Machine) ProcessCollectibleOutput(context.Context, *mtg.CollectibleOutput) {
}

func parseOperation(memo string) (*encoding.Operation, error) {
	b, err := base64.RawURLEncoding.DecodeString(memo)
	if err != nil {
		return nil, err
	}
	return encoding.DecodeOperation(b)
}
