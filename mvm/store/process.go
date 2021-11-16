package store

import "github.com/MixinNetwork/trusted-group/mvm/machine"

func (bs *BadgerStore) ReadEngineGroupEventsOffset(pid string) (uint64, error) {
	panic(0)
}

func (bs *BadgerStore) WriteEngineGroupEventsOffset(pid string, offset uint64) error {
	panic(0)
}

func (bs *BadgerStore) ListProcesses() ([]*machine.Process, error) {
	panic(0)
}

func (bs *BadgerStore) WriteProcess(p *machine.Process) error {
	panic(0)
}
