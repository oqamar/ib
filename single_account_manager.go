package ib

import (
	"fmt"
)

// SingleAccountManager tracks the account's values and portfolio.
type SingleAccountManager struct {
	AbstractManager
	id        int64
	values    map[AccountValueKey]AccountValue
	portfolio map[PortfolioValueKey]PortfolioValue
}

// NewSingleAccountManager .
func NewSingleAccountManager(e *Engine) (*SingleAccountManager, error) {
	am, err := NewAbstractManager(e)
	if err != nil {
		return nil, err
	}

	s := &SingleAccountManager{
		AbstractManager: *am,
		id:              UnmatchedReplyID,
		values:          map[AccountValueKey]AccountValue{},
		portfolio:       map[PortfolioValueKey]PortfolioValue{},
	}

	go s.startMainLoop(s.preLoop, s.receive, s.preDestroy)
	return s, nil
}

func (s *SingleAccountManager) preLoop() error {
	s.eng.Subscribe(s.rc, s.id)

	return s.eng.Send(&RequestAccountUpdates{})
}

func (s *SingleAccountManager) receive(r Reply) (UpdateStatus, error) {
	switch r := r.(type) {
	case *ErrorMessage:
		if r.SeverityWarning() {
			return UpdateFalse, nil
		}
		return UpdateFalse, r.Error()
	case *NextValidID:
		return UpdateFalse, nil
	case *AccountUpdateTime:
		return UpdateFalse, nil
	case *AccountValue:
		s.rwm.Lock()
		defer s.rwm.Unlock()
		s.values[r.Key] = *r
		return UpdateTrue, nil
	case *PortfolioValue:
		s.rwm.Lock()
		defer s.rwm.Unlock()
		s.portfolio[r.Key] = *r
		return UpdateTrue, nil
	}
	return UpdateFalse, fmt.Errorf("Unexpected type %v", r)
}

func (s *SingleAccountManager) preDestroy() {
	s.eng.Unsubscribe(s.rc, s.id)
	req := &RequestAccountUpdates{}
	req.Subscribe = false
	s.eng.Send(req)
}

// Values returns the most recent snapshot of account information.
func (s *SingleAccountManager) Values() map[AccountValueKey]AccountValue {
	s.rwm.RLock()
	defer s.rwm.RUnlock()
	return s.values
}

// Portfolio returns the most recent snapshot of account portfolio.
func (s *SingleAccountManager) Portfolio() map[PortfolioValueKey]PortfolioValue {
	s.rwm.RLock()
	defer s.rwm.RUnlock()
	return s.portfolio
}
