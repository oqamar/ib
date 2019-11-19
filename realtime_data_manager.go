package ib

import "fmt"

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
	switch r.(type) {
	case *ErrorMessage:
		r := r.(*ErrorMessage)
		if r.SeverityWarning() {
			return UpdateFalse, nil
		}
		return UpdateFalse, r.Error()
	case *RealtimeBars:
		hd := r.(*RealtimeBars)
		m.rtData = hd
		return UpdateTrue, nil
	}
	return UpdateFalse, fmt.Errorf("Unexpected type %v", r)
}

func (m *RealtimeDataManager) preDestroy() {
	m.eng.Unsubscribe(m.rc, m.request.id)
}

// Items .
func (m *RealtimeDataManager) Item() *RealtimeBars {
	m.rwm.RLock()
	defer m.rwm.RUnlock()
	return m.rtData
}
