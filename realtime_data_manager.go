package ib

// RealtimeDataManager .
type RealtimeDataManager struct {
	*AbstractManager
	request RequestRealTimeBars
	rtData  *RealtimeBars
}

// NewRealtimeDataManager creates a new RealtimeDataManager for the given data request.
func NewRealtimeDataManager(e *Engine, request RequestRealTimeBars) (*RealtimeDataManager, error) {
	am, err := NewAbstractManager(e)
	if err != nil {
		return nil, err
	}

	request.id = e.NextRequestID()
	m := &RealtimeDataManager{
		AbstractManager: am,
		request:         request,
	}

	go m.startMainLoop(m.preLoop, m.receive, m.preDestroy)
	return m, nil
}

func (m *RealtimeDataManager) preLoop() error {
	m.eng.Subscribe(m.rc, m.request.id)
	return m.eng.Send(&m.request)
}

func (m *RealtimeDataManager) receive(r Reply) (UpdateStatus, error) {
	switch r := r.(type) {
	//case *ErrorMessage:
	//	if r.SeverityWarning() {
	//		return UpdateFalse, nil
	//	}
	//	return UpdateTrue, r.Error()
	case *RealtimeBars:
		m.rtData = r
		return UpdateTrue, nil
	}
	return UpdateFalse, nil
}

func (m *RealtimeDataManager) preDestroy() {
	req := CancelRealTimeBars{
		id: m.request.id,
	}
	m.eng.Send(&req)
	m.eng.Unsubscribe(m.rc, m.request.id)
}

// Items .
func (m *RealtimeDataManager) Item() *RealtimeBars {
	m.rwm.RLock()
	defer m.rwm.RUnlock()
	return m.rtData
}
