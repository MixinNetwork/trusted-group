package machine

func (m *Machine) loopSignGroupEvents() {
	for {
		m.Store.ListPendingGroupEvents(100)
	}
}
