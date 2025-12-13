package book

import (
	"order-book/logger"
	"order-book/order"
	"slices"
	"sync"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
)

type Book interface {
	AddOrder(o order.Order)
	GetAllOrders(pairId string) (
		ask map[float64][]order.Order,
		bid map[float64][]order.Order,
	)

	insertOrder(o order.Order)
	matchOrder(o order.Order) (matchedOrders []order.Order, amountLeft float64)

	// utils
	getTreeFor(pairId string, orderType order.OrderType) *redblacktree.Tree
}

type BookImpl struct {
	mu                     sync.RWMutex
	askTreesMap            map[string]*redblacktree.Tree
	bidTreesMap            map[string]*redblacktree.Tree
	orderProcessingChannel chan order.Order
}

func (b *BookImpl) insertOrder(o order.Order) {
	b.mu.Lock()
	defer b.mu.Unlock()

	tree := b.getTreeFor(o.PairID, o.Type)

	node := tree.GetNode(o.Price)
	if node == nil {
		tree.Put(o.Price, &order.OrderList{List: []order.Order{o}})
		logger.Debug("order inserted at new price level", map[string]interface{}{
			"order_id": o.ID,
			"pair_id":  o.PairID,
			"price":    o.Price,
			"amount":   o.Amount,
		})
		return
	}

	orderList := node.Value.(*order.OrderList).List
	orderList = append(orderList, o)
	slices.SortFunc(orderList, func(a, b order.Order) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return 0
	})
	node.Value.(*order.OrderList).List = orderList
	logger.Debug("order inserted at existing price level", map[string]interface{}{
		"order_id":        o.ID,
		"pair_id":         o.PairID,
		"price":           o.Price,
		"amount":          o.Amount,
		"orders_at_price": len(orderList),
	})
}

func (b *BookImpl) matchOrder(o order.Order) (matchedOrders []order.Order, amountLeft float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var treeType order.OrderType
	if o.Type == order.ASK {
		treeType = order.BID
	} else {
		treeType = order.ASK
	}

	tree := b.getTreeFor(o.PairID, treeType)
	priceMatchedOrdersNode := tree.GetNode(o.Price)
	if priceMatchedOrdersNode == nil {
		logger.Debug("no matching orders with this price found", map[string]interface{}{
			"order_id": o.ID,
			"pair_id":  o.PairID,
			"price":    o.Price,
		})
		return
	}

	amountLeft = o.Amount
	matchedOrders = make([]order.Order, 0)
	ordersList := priceMatchedOrdersNode.Value.(*order.OrderList).List

	for idx := 0; idx < len(ordersList) && amountLeft > 0; {
		// NOTE: A larger existing order causes a break -> No need to increase idx
		// 		 A smaller existing order causes a delete -> a shift in the array -> idx now points to the next element automatically -> no need to increase idx
		// 		 This is more like a FIFO stack instead of an array
		if ordersList[idx].Amount > amountLeft {
			matched := ordersList[idx]
			matched.Amount = amountLeft
			ordersList[idx].Amount -= amountLeft
			matchedOrders = append(matchedOrders, matched)
			amountLeft = 0
			break
		}
		if ordersList[idx].Amount <= amountLeft {
			matchedOrders = append(matchedOrders, ordersList[idx])
			amountLeft -= ordersList[idx].Amount
			ordersList = slices.Delete(ordersList, idx, idx+1)
			continue
		}
	}
	priceMatchedOrdersNode.Value.(*order.OrderList).List = ordersList

	if len(matchedOrders) > 0 {
		logger.Info("order matched", map[string]interface{}{
			"order_id":      o.ID,
			"pair_id":       o.PairID,
			"matched_count": len(matchedOrders),
			"amount_left":   amountLeft,
		})
	}

	return
}

func (b *BookImpl) AddOrder(o order.Order) {
	logger.Info("order received", map[string]interface{}{
		"order_id": o.ID,
		"pair_id":  o.PairID,
		"type":     o.Type,
		"price":    o.Price,
		"amount":   o.Amount,
	})
	b.orderProcessingChannel <- o
}

func (b *BookImpl) GetAllOrders(pairId string) (
	ask map[float64][]order.Order,
	bid map[float64][]order.Order,
) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	pairAskTree := b.askTreesMap[pairId]
	pairBidTree := b.bidTreesMap[pairId]

	ask = make(map[float64][]order.Order)
	bid = make(map[float64][]order.Order)

	askCount := 0
	bidCount := 0

	if pairAskTree != nil {
		it := pairAskTree.Iterator()
		for it.Next() != false {
			orderList := it.Value().(*order.OrderList).List
			ask[it.Key().(float64)] = orderList
			askCount += len(orderList)
		}
	}

	if pairBidTree != nil {
		it := pairBidTree.Iterator()
		for it.Next() != false {
			orderList := it.Value().(*order.OrderList).List
			bid[it.Key().(float64)] = orderList
			bidCount += len(orderList)
		}
	}

	logger.Debug("retrieved all orders", map[string]any{
		"pair_id":   pairId,
		"ask_count": askCount,
		"bid_count": bidCount,
	})

	return
}

func (b *BookImpl) getTreeFor(pairId string, orderType order.OrderType) *redblacktree.Tree {
	if orderType == order.ASK {
		tree := b.askTreesMap[pairId]
		if tree == nil {
			tree = redblacktree.NewWith(utils.Float64Comparator)
			b.askTreesMap[pairId] = tree
			logger.Debug("created new ask tree", map[string]any{
				"pair_id": pairId,
			})
		}
		return tree
	}
	if orderType == order.BID {
		tree := b.bidTreesMap[pairId]
		if tree == nil {
			tree = redblacktree.NewWith(utils.Float64Comparator)
			b.bidTreesMap[pairId] = tree
			logger.Debug("created new bid tree", map[string]any{
				"pair_id": pairId,
			})
		}
		return tree
	}
	return nil
}

func NewBook() Book {
	logger.Info("order book initialized")

	b := BookImpl{
		askTreesMap:            make(map[string]*redblacktree.Tree, 0),
		bidTreesMap:            make(map[string]*redblacktree.Tree, 0),
		orderProcessingChannel: make(chan order.Order),
	}

	go func() {
		for o := range b.orderProcessingChannel {
			matchedOrders, amountLeft := b.matchOrder(o)
			if len(matchedOrders) == 0 {
				b.insertOrder(o)
				continue
			}
			if amountLeft > 0 {
				o.Amount = amountLeft
				b.insertOrder(o)
				logger.Info("order partially matched", map[string]any{
					"order_id":  o.ID,
					"pair_id":   o.PairID,
					"remaining": amountLeft,
				})
				continue
			}
			logger.Info("order fully matched", map[string]any{
				"order_id": o.ID,
				"pair_id":  o.PairID,
			})
		}
	}()

	return &b
}
