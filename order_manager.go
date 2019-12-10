package ib

import (
	"fmt"
)

type orderValue struct {
	seen  bool
	Order Order
}

// SingleAccountManager tracks the account's values and portfolio.
type OrderManager struct {
	*AbstractManager
	orders map[int64]*OrderInfo
	unread map[int64]bool
}

type OrderInfo struct {
	Order         *Order
	OrderStatus   *OrderStatus
	ExecutionData *ExecutionData
}

// NewOrderManager .
func NewOrderManager(e *Engine) (*OrderManager, error) {
	am, err := NewAbstractManager(e)
	if err != nil {
		return nil, err
	}

	s := &OrderManager{
		AbstractManager: am,
		orders:          make(map[int64]*OrderInfo),
		unread:          make(map[int64]bool),
	}

	go s.startMainLoop(s.preLoop, s.receive, s.preDestroy)
	return s, nil
}

func (o *OrderManager) preLoop() error {
	return nil
}

func (o *OrderManager) receive(r Reply) (UpdateStatus, error) {
	switch r := r.(type) {
	case *ErrorMessage:
		if r.SeverityWarning() {
			return UpdateFalse, nil
		}
		return UpdateTrue, r.Error()
	case *OpenOrder:
		oi := o.orders[r.Order.OrderID]
		if oi == nil {
			oi = &OrderInfo{}
		}
		oi.Order = &r.Order
		o.unread[r.Order.OrderID] = true
		return UpdateTrue, nil
	case *OrderStatus:
		oi := o.orders[r.id]
		if oi == nil {
			oi = &OrderInfo{}
		}
		oi.OrderStatus = r
		o.unread[r.id] = true
		return UpdateTrue, nil
	case *ExecutionData:
		oi := o.orders[r.id]
		if oi == nil {
			oi = &OrderInfo{}
		}
		oi.ExecutionData = r
		o.unread[r.id] = true
		return UpdateTrue, nil
	default:
		return UpdateTrue, fmt.Errorf("Unexpected type %[1]T: %[1]v", r)
	}
}

func (o *OrderManager) preDestroy() {
	for k := range o.orders {
		o.eng.Unsubscribe(o.rc, k)
	}
}

func (o *OrderManager) SendOrder(req *PlaceOrder) error {
	o.eng.Subscribe(o.rc, req.id)
	err := o.eng.Send(req)
	if err != nil {
		return err
	}
	o.rwm.Lock()
	defer o.rwm.Unlock()
	o.orders[req.id] = nil
	return nil
}

func (o *OrderManager) NewData() []*OrderInfo {
	o.rwm.Lock()
	defer o.rwm.Unlock()
	var ois []*OrderInfo
	var seen []int64
	for k := range o.unread {
		ois = append(ois, o.orders[k])
		seen = append(seen, k)
	}
	for _, id := range seen {
		delete(o.unread, id)
	}
	return ois
}

func (o *OrderManager) AllData() []*OrderInfo {
	o.rwm.RLock()
	defer o.rwm.RUnlock()
	var ois []*OrderInfo
	for k := range o.orders {
		ois = append(ois, o.orders[k])
	}
	return ois
}
