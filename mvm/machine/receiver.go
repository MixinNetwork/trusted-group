package machine

import (
	"context"

	"github.com/MixinNetwork/nfo/mtg"
)

const (
	OperationPurposeUnknown       = "UNKNOWN"
	OperationPurposeAddProcess    = "PROCESS:ADD"
	OperationPurposeCreditProcess = "PROCESS:CREDIT"
	OperationPurposeGroupEvent    = "EVENT"
)

type Operation struct {
	Purpose string

	Platform string
	Address  string
}

type GroupReceiver struct {
	Machine *Machine
}

func NewGroupReceiver() *GroupReceiver {
	return nil
}

func (r *GroupReceiver) ProcessOutput(ctx context.Context, out *mtg.Output) {
	op := r.parseOperation(out.Memo)
	switch op.Purpose {
	case OperationPurposeAddProcess:
		r.Machine.AddProcess(out.Sender, op.Platform, op.Address, out)
	case OperationPurposeGroupEvent:
		r.Machine.WriteGroupEvent(out)
	}
}

func (r *GroupReceiver) ProcessCollectibleOutput(context.Context, *mtg.CollectibleOutput) {
}

func (r *GroupReceiver) parseOperation(memo string) *Operation {
	return &Operation{
		Purpose: OperationPurposeUnknown,
	}
}
