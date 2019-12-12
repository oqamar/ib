package ib

import (
	"fmt"
)

// SingleAccountManager tracks the account's values and portfolio.
type OrderManager struct {
	*AbstractManager
	allOrders map[int64]*OrderInfo
	newOrders map[int64]*OrderInfo
}

type OrderInfo struct {
	OpenOrder     *OpenOrder
	OrderStatus   *OrderStatus
	ExecutionData []*ExecutionData
}

var allIDs, newIDs []int64

// NewOrderManager .
func NewOrderManager(e *Engine) (*OrderManager, error) {
	am, err := NewAbstractManager(e)
	if err != nil {
		return nil, err
	}

	s := &OrderManager{
		AbstractManager: am,
		allOrders:       make(map[int64]*OrderInfo),
		newOrders:       make(map[int64]*OrderInfo),
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
		if r.Code == 202 {
			fmt.Printf("canceled # %d: %d -- %s\n", r.id, r.Code, r.Error())
			//cancel <- true
			return UpdateFalse, nil
		}
		return UpdateTrue, r.Error()
	case *OpenOrder:
		id := r.Order.OrderID
		aoi := o.allOrders[id]
		aoi.OpenOrder = r
		noi := o.newOrders[id]
		noi.OpenOrder = r
		return UpdateTrue, nil
	case *OrderStatus:
		id := r.id
		aoi := o.allOrders[id]
		aoi.OrderStatus = r
		noi := o.newOrders[id]
		noi.OrderStatus = r
		return UpdateTrue, nil
	case *ExecutionData:
		id := r.id
		aoi := o.allOrders[id]
		aoi.ExecutionData = append(aoi.ExecutionData, r)
		noi := o.newOrders[id]
		noi.ExecutionData = append(noi.ExecutionData, r)
		return UpdateTrue, nil
	default:
		return UpdateTrue, fmt.Errorf("Unexpected type %T: %v", r, r)
	}
}

func (o *OrderManager) preDestroy() {
	for _, id := range allIDs {
		o.eng.Unsubscribe(o.rc, id)
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
	o.allOrders[req.id] = &OrderInfo{}
	o.newOrders[req.id] = &OrderInfo{}
	allIDs = append(allIDs, req.id)
	newIDs = append(newIDs, req.id)
	return nil
}

func (o *OrderManager) CancelOrder(req *CancelOrder) error {
	o.eng.Unsubscribe(o.rc, req.id)
	return o.eng.Send(req)
}

func (o *OrderManager) NewData() []*OrderInfo {
	o.rwm.Lock()
	defer o.rwm.Unlock()
	var ois []*OrderInfo
	for _, id := range newIDs {
		ois = append(ois, o.newOrders[id])
	}
	o.newOrders = map[int64]*OrderInfo{}
	newIDs = []int64{}
	return ois
}

func (o *OrderManager) AllData() []*OrderInfo {
	o.rwm.RLock()
	defer o.rwm.RUnlock()
	var ois []*OrderInfo
	for _, id := range allIDs {
		if o.allOrders[id] != nil {
			ois = append(ois, o.allOrders[id])
		}
	}
	return ois
}
