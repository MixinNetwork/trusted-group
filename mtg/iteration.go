package mtg

import (
	"context"
	"database/sql"
)

const (
	IterationActionAdd    = 11
	IterationActionRemove = 12
)

// a node joins or leaves the group with an iteration
// this is for the evolution mechanism of MTG
// TODO not implemented yet
type Iteration struct {
	Action    int
	NodeId    string
	Threshold int
	CreatedAt uint64
}

var iterationCols = []string{"action", "node_id", "threshold", "created_at"}

func (i *Iteration) values() []any {
	return []any{i.Action, i.NodeId, i.Threshold, i.CreatedAt}
}

func iterationFromRow(row Row) (*Iteration, error) {
	var i Iteration
	err := row.Scan(&i.Action, &i.NodeId, &i.Threshold, &i.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &i, err
}

func (grp *Group) AddNode(ctx context.Context, id string, threshold int, epoch uint64) error {
	ir := &Iteration{
		Action:    IterationActionAdd,
		NodeId:    id,
		Threshold: threshold,
		CreatedAt: epoch,
	}
	return grp.store.WriteIteration(ctx, ir)
}

func (grp *Group) ListActiveNodes(ctx context.Context) ([]string, int, uint64, error) {
	irs, err := grp.store.ListIterations(ctx)
	var actives []string
	for _, ir := range irs {
		if ir.Action == IterationActionAdd {
			actives = append(actives, ir.NodeId)
		}
	}
	if err != nil || len(actives) == 0 {
		return nil, 0, 0, err
	}
	last := irs[len(irs)-1]
	return actives, last.Threshold, last.CreatedAt, nil
}
