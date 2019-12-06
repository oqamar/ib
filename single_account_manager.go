package ib

import (
	"fmt"
)

// SingleAccountManager tracks the account's values and portfolio.
type SingleAccountManager struct {
	*AbstractManager
	id        int64
	values    map[AccountValueKey]AccountValue
	portfolio map[PortfolioValueKey]PortfolioValue
	loaded    bool
}

// NewSingleAccountManager .
func NewSingleAccountManager(e *Engine) (*SingleAccountManager, error) {
	am, err := NewAbstractManager(e)
	if err != nil {
		return nil, err
	}

	s := &SingleAccountManager{
		AbstractManager: am,
		id:              UnmatchedReplyID,
		values:          map[AccountValueKey]AccountValue{},
		portfolio:       map[PortfolioValueKey]PortfolioValue{},
	}

	go s.startMainLoop(s.preLoop, s.receive, s.preDestroy)
	return s, nil
}

func (s *SingleAccountManager) preLoop() error {
	s.eng.Subscribe(s.rc, s.id)

	return s.eng.Send(&RequestAccountUpdates{Subscribe: true})
}

func (s *SingleAccountManager) receive(r Reply) (UpdateStatus, error) {
	switch r := r.(type) {
	case *ErrorMessage:
		if r.SeverityWarning() {
			return UpdateFalse, nil
		}
		return UpdateTrue, r.Error()
	case *AccountUpdateTime:
		if s.loaded {
			return UpdateTrue, nil
		}
		return UpdateFalse, nil
	case *AccountValue:
		s.values[r.Key] = *r
		if s.loaded {
			return UpdateTrue, nil
		}
		return UpdateFalse, nil
	case *PortfolioValue:
		s.portfolio[r.Key] = *r
		if s.loaded {
			return UpdateTrue, nil
		}
		return UpdateFalse, nil
	case *AccountDownloadEnd:
		s.loaded = true
		return UpdateTrue, nil
	}
	return UpdateTrue, fmt.Errorf("Unexpected type %v", r)
}

func (s *SingleAccountManager) preDestroy() {
	s.eng.Send(&RequestAccountUpdates{Subscribe: false})
	s.eng.Unsubscribe(s.rc, s.id)
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
