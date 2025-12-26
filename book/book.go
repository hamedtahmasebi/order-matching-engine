package book

import (
	"errors"
	"order-book/logger"
	"order-book/order"
	"slices"
	"sync"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
)

var ErrOrderNotFound = errors.New("Order not found")

type Book interface {
	AddOrder(o order.Order)
	GetOrders(pairId string, size int, offset int) (
		ask []order.Order,
		bid []order.Order,
	)
	CancellOrder(id int) error
}

type OrderMetadata struct {
	PairId string
	Price  float64
	Type   order.OrderType
	Amount float64
	ID     int
}

type BookImpl struct {
	mu                     sync.RWMutex
	askTreesMap            map[string]*redblacktree.Tree
	bidTreesMap            map[string]*redblacktree.Tree
	orderProcessingChannel chan order.Order
	orderRepo              order.OrderRepo
}

func (b *BookImpl) insertOrder(o order.Order) {
	createdOrder, err := b.orderRepo.CreateOrder(o.PairID, o.Price, o.Amount, o.AccountID, o.Type)
	if err != nil {
		logger.Error("failed to add order history event", map[string]any{
			"error": err,
		})
	}
	if err != nil {
		logger.Error("failed to create order", map[string]any{
			"order_id": o.ID,
			"pair_id":  o.PairID,
			"price":    o.Price,
			"amount":   o.Amount,
			"error":    err,
		})
		return
	}
	o.ID = createdOrder.ID
	b.mu.Lock()
	defer b.mu.Unlock()

	tree := b.getTreeFor(o.PairID, o.Type)
	if tree == nil {
		tree = b.genTreeFor(o.PairID, o.Type)
	}

	node := tree.GetNode(o.Price)
	if node == nil {
		tree.Put(o.Price, &order.OrderList{List: []order.Order{o}})
		logger.Debug("order inserted at new price level", map[string]any{
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

type MatchResult struct {
	targetOrder  order.Order
	match_status string
}

func (b *BookImpl) matchOrder(o order.Order) (matchResults []MatchResult, amountLeft float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var treeType order.OrderType
	if o.Type == order.ASK {
		treeType = order.BID
	} else {
		treeType = order.ASK
	}

	tree := b.getTreeFor(o.PairID, treeType)
	if tree == nil {
		tree = b.genTreeFor(o.PairID, treeType)
	}
	priceMatchedOrdersNode := tree.GetNode(o.Price)
	if priceMatchedOrdersNode == nil {
		logger.Debug("no matching orders with this price found", map[string]any{
			"order_id": o.ID,
			"pair_id":  o.PairID,
			"price":    o.Price,
		})
		return
	}

	amountLeft = o.Amount
	ordersList := priceMatchedOrdersNode.Value.(*order.OrderList).List

	for idx := 0; idx < len(ordersList) && amountLeft > 0; {
		// Skip user's previous orders
		if ordersList[idx].AccountID == o.AccountID {
			idx++
			continue
		}
		// NOTE: A larger existing order causes a break -> No need to increase idx
		// 		 A smaller existing order causes a delete -> a shift in the array -> idx now points to the next element automatically -> no need to increase idx
		// 		 This is more like a FIFO stack instead of an array
		if ordersList[idx].Amount > amountLeft {
			matched := ordersList[idx]
			matched.Amount = amountLeft
			ordersList[idx].Amount -= amountLeft
			matchResults = append(matchResults, MatchResult{targetOrder: matched, match_status: "partial"})
			amountLeft = 0
			break
		}
		if ordersList[idx].Amount <= amountLeft {
			matched := ordersList[idx]
			matchResults = append(matchResults, MatchResult{targetOrder: matched, match_status: "full"})
			amountLeft -= matched.Amount
			ordersList = slices.Delete(ordersList, idx, idx+1)
			continue
		}
	}
	priceMatchedOrdersNode.Value.(*order.OrderList).List = ordersList

	if len(matchResults) > 0 {
		logger.Info("order matched", map[string]any{
			"order_id":      o.ID,
			"pair_id":       o.PairID,
			"matched_count": len(matchResults),
			"amount_left":   amountLeft,
		})
	}

	return
}

func (b *BookImpl) CancellOrder(id int) error {
	logger.Info("Searching for order", map[string]any{
		"order_id": id,
	})

	foundOrder, err := b.orderRepo.GetOrderByID(id)
	if err != nil {
		logger.Error("Order not found in the index")
		return err
	}

	tree := b.getTreeFor(foundOrder.PairID, foundOrder.Type)
	if tree == nil {
		logger.Error("Order tree not found")
		return ErrOrderNotFound
	}
	node := tree.GetNode(foundOrder.Price)
	if node == nil {
		return ErrOrderNotFound
	}

	orders := node.Value.(*order.OrderList).List
	for idx, o := range orders {
		if o.ID == foundOrder.ID {
			logger.Debug("order removed", map[string]any{
				"order_id": o.ID,
				"pair_id":  o.PairID,
				"type":     o.Type,
				"price":    o.Price,
				"amount":   o.Amount,
			})
			orders := slices.Delete(orders, idx, idx+1)
			node.Value.(*order.OrderList).List = orders
			err = b.orderRepo.AddEvent(order.OrderHistoryEvent{
				Name:    "ORDER_CANCELLED",
				OrderId: foundOrder.ID,
			})
			if len(orders) == 0 {
				tree.Remove(node.Key)
			}
			return nil
		}
	}
	return ErrOrderNotFound
}

func (b *BookImpl) AddOrder(o order.Order) {
	logger.Info("order received", map[string]any{
		"order_id": o.ID,
		"pair_id":  o.PairID,
		"type":     o.Type,
		"price":    o.Price,
		"amount":   o.Amount,
	})
	b.orderProcessingChannel <- o
}

func (b *BookImpl) GetOrders(pairId string, size int, offset int) (
	ask []order.Order,
	bid []order.Order,
) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	pairAskTree := b.askTreesMap[pairId]
	pairBidTree := b.bidTreesMap[pairId]

	if pairAskTree != nil {
		it := pairAskTree.Iterator()
		it.End() // because we want the highest ask first, we need to start from the end backwards
		for i := 0; i < offset && it.Prev() != false; i++ {
		}

		for i := 0; i < size && it.Prev() != false; i++ {
			orderList := it.Value().(*order.OrderList).List
			ask = append(ask, orderList...)
		}
	}

	if pairBidTree != nil {
		it := pairBidTree.Iterator()

		for i := 0; i < offset && it.Next() != false; i++ {
		}

		for i := 0; i < size && it.Next() != false; i++ {
			orderList := it.Value().(*order.OrderList).List
			bid = append(bid, orderList...)
		}
	}

	logger.Debug("retrieved all orders", map[string]any{
		"pair_id":   pairId,
		"ask_count": len(ask),
		"bid_count": len(bid),
	})

	return
}

func (b *BookImpl) getTreeFor(pairId string, orderType order.OrderType) *redblacktree.Tree {
	if orderType == order.ASK {
		tree := b.askTreesMap[pairId]
		return tree
	}
	if orderType == order.BID {
		tree := b.bidTreesMap[pairId]
		return tree
	}
	return nil
}

func (b *BookImpl) genTreeFor(pairId string, orderType order.OrderType) *redblacktree.Tree {
	tree := redblacktree.NewWith(utils.Float64Comparator)
	if orderType == order.ASK {
		logger.Debug("created new ask tree", map[string]any{
			"pair_id": pairId,
		})
		b.askTreesMap[pairId] = tree
	} else {
		logger.Debug("created new bid tree", map[string]any{
			"pair_id": pairId,
		})
		b.bidTreesMap[pairId] = tree
	}
	return tree
}

func NewBook(orderRepo order.OrderRepo) Book {
	logger.Info("order book initialized")

	b := BookImpl{
		askTreesMap:            make(map[string]*redblacktree.Tree, 0),
		bidTreesMap:            make(map[string]*redblacktree.Tree, 0),
		orderProcessingChannel: make(chan order.Order),
		orderRepo:              orderRepo,
	}

	go func() {
		for o := range b.orderProcessingChannel {
			matchedResults, amountLeft := b.matchOrder(o)
			b.insertOrder(o)

			for _, matchedResult := range matchedResults {
				b.orderRepo.AddEvent(order.OrderHistoryEvent{
					Name:    "TARGET_HIT",
					OrderId: matchedResult.targetOrder.ID,
					Metadata: map[string]any{
						"matching_order_id": o.ID,
					},
				})
			}

			if amountLeft > 0 {
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
