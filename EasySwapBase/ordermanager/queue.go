package ordermanager

import (
	"sort"
	"strings"

	"github.com/shopspring/decimal"
)

type Entry struct {
	orderID  string          //order ID
	priority decimal.Decimal // price
	maker    string
	tokenID  string
}

type PriorityQueue []*Entry

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority.LessThan(pq[j].priority)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

type PriorityQueueMap struct {
	pq     PriorityQueue
	orders map[string]*Entry
	maxLen int
}

func NewPriorityQueueMap(maxLen int) *PriorityQueueMap {
	return &PriorityQueueMap{
		orders: make(map[string]*Entry),
		maxLen: maxLen,
	}
}

func (pqm *PriorityQueueMap) Len() int {
	return len(pqm.orders)
}

func (pqm *PriorityQueueMap) Add(orderID string, price decimal.Decimal, maker, tokenID string) {
	if len(pqm.pq) > pqm.maxLen {
		delete(pqm.orders, pqm.pq[len(pqm.pq)-1].orderID)
		pqm.pq = pqm.pq[0 : len(pqm.pq)-1]
	}

	entry := &Entry{orderID: orderID, priority: price, maker: strings.ToLower(maker), tokenID: strings.ToLower(tokenID)}
	pqm.pq = append(pqm.pq, entry)
	sort.Sort(pqm.pq)
	pqm.orders[orderID] = entry
}

func (pqm *PriorityQueueMap) GetMin() (string, decimal.Decimal) {
	if len(pqm.pq) == 0 {
		return "", decimal.Zero
	}
	entry := pqm.pq[0]
	return entry.orderID, entry.priority
}

func (pqm *PriorityQueueMap) GetMax() (string, decimal.Decimal) {
	if len(pqm.pq) == 0 {
		return "", decimal.Zero
	}
	entry := pqm.pq[len(pqm.pq)-1]
	return entry.orderID, entry.priority
}

func (pqm *PriorityQueueMap) Remove(orderID string) {
	_, ok := pqm.orders[orderID]
	if !ok {
		return
	}

	var newPQ []*Entry
	for i, v := range pqm.pq {
		if v.orderID != orderID {
			newPQ = append(newPQ, pqm.pq[i])
		}
	}

	pqm.pq = newPQ
	delete(pqm.orders, orderID)
}

func (pqm *PriorityQueueMap) RemoveMakerOrders(maker, tokenID string) {
	maker = strings.ToLower(maker)
	tokenID = strings.ToLower(tokenID)
	var newPQ []*Entry

	for i, v := range pqm.pq {
		if v.maker == maker && v.tokenID == tokenID {
			delete(pqm.orders, v.orderID)
		} else {
			newPQ = append(newPQ, pqm.pq[i])
		}
	}

	pqm.pq = newPQ
}
