package ordermanager

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestQueue(t *testing.T) {
	queue := NewPriorityQueueMap(5)
	// add order id 101 price 1.2
	queue.Add("101", decimal.NewFromFloat(1.2), "a", "1")
	// add order id 102 price 1.3
	queue.Add("102", decimal.NewFromFloat(1.3), "b", "2")
	// add order id 100 price 1.1
	queue.Add("99", decimal.NewFromFloat(1.1), "c", "3")
	queue.Add("98", decimal.NewFromFloat(1.1), "d", "4")
	// add order id 100 price 1.1
	queue.Add("100", decimal.NewFromFloat(1.1), "e", "5")
	// add order id 102 price 1.3
	queue.Add("103", decimal.NewFromFloat(1.4), "f", "6")
	// add order id 102 price 1.3
	queue.Add("104", decimal.NewFromFloat(1.5), "g", "7")

	id, price := queue.GetMin()
	assert.Equal(t, id, "99")
	assert.Equal(t, price, decimal.NewFromFloat(1.1))

	id, price = queue.GetMax()
	assert.Equal(t, id, "104")
	assert.Equal(t, price, decimal.NewFromFloat(1.5))
	queue.Remove("104")
	id, price = queue.GetMin()
	assert.Equal(t, id, "99")
	assert.Equal(t, price, decimal.NewFromFloat(1.1))

	id, price = queue.GetMax()
	assert.Equal(t, id, "102")
	assert.Equal(t, price, decimal.NewFromFloat(1.3))
}

func TestQueueRemoveMaker(t *testing.T) {
	queue := NewPriorityQueueMap(5)
	// add order id 101 price 1.2
	queue.Add("101", decimal.NewFromFloat(1.2), "a", "1")
	// add order id 102 price 1.3
	queue.Add("102", decimal.NewFromFloat(1.3), "b", "2")
	// add order id 102 price 1.3
	queue.Add("103", decimal.NewFromFloat(1.0), "A", "3")

	// add order id 100 price 1.1
	queue.RemoveMakerOrders("a", "1")

	id, price := queue.GetMin()
	assert.Equal(t, id, "103")
	assert.Equal(t, price, decimal.NewFromFloat(1.0))
	queue.RemoveMakerOrders("b", "2")
	queue.RemoveMakerOrders("a", "3")

	id, price = queue.GetMin()
	assert.Equal(t, id, "")
	assert.Equal(t, price, decimal.Zero)
}

func TestPoint(t *testing.T) {
	infos := make(map[string]*collectionTradeInfo)
	infos["a"] = &collectionTradeInfo{
		floorPrice: decimal.NewFromFloat(1.1),
	}

	info := infos["a"]
	setNewFloorPrice(info)
	fmt.Println(info)
}

func setNewFloorPrice(info *collectionTradeInfo) {
	info.floorPrice = decimal.NewFromFloat(1.2)
}
