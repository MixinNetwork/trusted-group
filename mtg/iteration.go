package mtg

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

func (grp *Group) AddNode(id string, threshold int, epoch uint64) error {
	ir := &Iteration{
		Action:    IterationActionAdd,
		NodeId:    id,
		Threshold: threshold,
		CreatedAt: epoch,
	}
	return grp.store.WriteIteration(ir)
}

func (grp *Group) RemoveNode(id string, threshold int, epoch uint64) error {
	ir := &Iteration{
		Action:    IterationActionRemove,
		NodeId:    id,
		Threshold: threshold,
		CreatedAt: epoch,
	}
	return grp.store.WriteIteration(ir)
}

func (grp *Group) ListActiveNodes() ([]string, int, uint64, error) {
	irs, err := grp.store.ListIterations()
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
