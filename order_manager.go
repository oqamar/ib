package ib

import (
	"fmt"
	"sync"
)

// SingleAccountManager tracks the account's values and portfolio.
type OrderManager struct {
	*AbstractManager
	orders      []*OrderInfo
	orderErrors chan bool
	orderError  string
	oRwm        *sync.RWMutex
}

type OrderInfo struct {
	ID            int64
	OpenOrder     *OpenOrder
	OrderStatus   *OrderStatus
	ExecutionData *ExecutionData
}

var hidx int

// NewOrderManager .
func NewOrderManager(e *Engine) (*OrderManager, error) {
	am, err := NewAbstractManager(e)
	if err != nil {
		return nil, err
	}

	s := &OrderManager{
		AbstractManager: am,
		orderErrors:     make(chan bool),
		oRwm:            &sync.RWMutex{},
	}

	go s.startMainLoop(s.preLoop, s.receive, s.preDestroy)
	return s, nil
}

func (o *OrderManager) preLoop() error {
	o.eng.SubscribeAll(o.rc)
	return nil
}

func (o *OrderManager) receive(r Reply) (UpdateStatus, error) {
	switch r := r.(type) {
	case *ErrorMessage:
		if r.SeverityWarning() {
			return UpdateFalse, nil
		}
		o.oRwm.Lock()
		o.orderError = fmt.Sprintf("order # %d: %d -- %s", r.id, r.Code, r.Error())
		o.oRwm.Unlock()
		o.orderErrors <- true
		return UpdateFalse, nil
	case *OpenOrder:
		id := r.ID()
		o.orders = append(o.orders, &OrderInfo{ID: id, OpenOrder: r})
		return UpdateTrue, nil
	case *OrderStatus:
		id := r.ID()
		o.orders = append(o.orders, &OrderInfo{ID: id, OrderStatus: r})
		return UpdateTrue, nil
	case *ExecutionData:
		id := r.ID()
		o.orders = append(o.orders, &OrderInfo{ID: id, ExecutionData: r})
		return UpdateTrue, nil
	default:
		o.oRwm.Lock()
		o.orderError = fmt.Sprintf("Unexpected type %T: %v", r, r)
		o.oRwm.Unlock()
		o.orderErrors <- true
		return UpdateFalse, nil
	}
}

func (o *OrderManager) preDestroy() {
	done := map[int64]bool{}
	for _, oi := range o.orders {
		if !done[oi.ID] {
			o.eng.Unsubscribe(o.rc, oi.ID)
			done[oi.ID] = true
		}
	}
}

func (o *OrderManager) ErrorRefresh() <-chan bool {
	return o.orderErrors
}

func (o *OrderManager) OrderError() string {
	o.oRwm.Lock()
	defer o.oRwm.Unlock()
	return o.orderError
}

func (o *OrderManager) SendOrder(reqs []*PlaceOrder) error {
	for _, req := range reqs {
		o.eng.Subscribe(o.rc, req.id)
		err := o.eng.Send(req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *OrderManager) CancelOrder(req *CancelOrder) error {
	return o.eng.Send(req)
}

func (o *OrderManager) NewData() []*OrderInfo {
	o.rwm.Lock()
	defer o.rwm.Unlock()
	var ois []*OrderInfo
	nhidx := len(o.orders)
	for i := hidx; i < nhidx; i++ {
		ois = append(ois, o.orders[i])
	}
	hidx = nhidx
	return ois
}

func (o *OrderManager) AllData() []*OrderInfo {
	o.rwm.RLock()
	defer o.rwm.RUnlock()
	return o.orders
}
