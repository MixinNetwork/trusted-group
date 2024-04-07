package mtg

import (
	"context"
	"database/sql"

	"github.com/MixinNetwork/mixin/logger"
)

const (
	ActionStateInitial = 10
	ActionStateDone    = 11
)

type Action struct {
	UTXOID    string
	CreatedAt uint64
	State     int
}

var actionCols = []string{"utxo_id", "created_at", "state"}

func (a *Action) values() []any {
	return []any{a.UTXOID, a.CreatedAt, a.State}
}

func actionFromRow(row *sql.Row) (*Action, error) {
	var a Action
	err := row.Scan(&a.UTXOID, &a.CreatedAt, &a.State)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

// actions queue is all the utxos ordered by their created time
// this queue can only be queried after all utxos are drained from api
func (grp *Group) handleActionsQueue(ctx context.Context) error {
	outputs, err := grp.store.ListActions(ctx, 16)
	if err != nil {
		return err
	}
	for _, out := range outputs {
		for _, wkr := range grp.workers {
			var handled bool
			switch out.Type {
			case OutputTypeSafeOutput:
				handled = wkr.ProcessOutput(ctx, out.Output())
			default:
				panic(out.Type)
			}
			if handled {
				break
			}
		}
		grp.writeAction(ctx, out, ActionStateDone)
	}
	return nil
}

func (grp *Group) writeAction(ctx context.Context, out *UnifiedOutput, state int) {
	logger.Verbosef("Group.writeAction(%v, %d)", out, state)
	err := grp.store.WriteAction(ctx, &Action{
		UTXOID:    out.OutputId,
		CreatedAt: out.Sequence,
		State:     state,
	})
	if err != nil {
		panic(err)
	}
}
