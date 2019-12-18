package ib

import (
	"fmt"
)

// SingleAccountManager tracks the account's values and portfolio.
type OrderManager struct {
	*AbstractManager
	orders      []*OrderInfo
	chOrders    chan *OrderInfo
	chOrderErrs chan error
}

type OrderInfo struct {
	ID            int64
	OpenOrder     *OpenOrder
	OrderStatus   *OrderStatus
	ExecutionData *ExecutionData
}

// NewOrderManager
func NewOrderManager(e *Engine) (*OrderManager, error) {
	am, err := NewAbstractManager(e)
	if err != nil {
		return nil, err
	}

	s := &OrderManager{
		AbstractManager: am,
		chOrders:        make(chan *OrderInfo),
		chOrderErrs:     make(chan error),
	}

	go s.startMainLoop(s.preLoop, s.receive, s.preDestroy)
	return s, nil
}

func (o *OrderManager) preLoop() error {
	//o.eng.Subscribe(o.rc, UnmatchedReplyID)
	o.eng.SubscribeAll(o.rc)
	return nil
}

func (o *OrderManager) receive(r Reply) (UpdateStatus, error) {
	switch r := r.(type) {
	case *ErrorMessage:
		if r.SeverityWarning() {
			return UpdateFalse, nil
		}
		if r.Code >= 200 && r.Code <= 203 {
			o.chOrderErrs <- fmt.Errorf("order # %d: %d -- %s", r.id, r.Code, r.Error())
		} else {
			fmt.Printf("error %d: %d -- %s\n", r.id, r.Code, r.Error())
		}
		return UpdateFalse, nil
	case *OpenOrder:
		id := r.ID()
		oi := &OrderInfo{ID: id, OpenOrder: r}
		o.orders = append(o.orders, oi)
		o.chOrders <- oi
		return UpdateFalse, nil
	case *OrderStatus:
		id := r.ID()
		oi := &OrderInfo{ID: id, OrderStatus: r}
		o.orders = append(o.orders, oi)
		o.chOrders <- oi
		return UpdateFalse, nil
	case *ExecutionData:
		id := r.Exec.OrderID
		oi := &OrderInfo{ID: id, ExecutionData: r}
		o.orders = append(o.orders, oi)
		o.chOrders <- oi
		return UpdateFalse, nil
	}
	return UpdateFalse, nil
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

func (o *OrderManager) OrderRefresh() <-chan *OrderInfo {
	return o.chOrders
}

func (o *OrderManager) ErrorRefresh() <-chan error {
	return o.chOrderErrs
}

func (o *OrderManager) SendOrder(reqs []*PlaceOrder) error {
	for _, req := range reqs {
		//o.eng.Subscribe(o.rc, req.id)
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

func (o *OrderManager) AllData() []*OrderInfo {
	o.rwm.RLock()
	defer o.rwm.RUnlock()
	return o.orders
}
