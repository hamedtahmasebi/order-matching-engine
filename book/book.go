package book

import (
	"order-book/order"
	"sort"
	"sync"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
)

type Book interface {
	AddOrder(o order.Order)
	MatchOrder(o order.Order) (order *order.Order, found bool)
	GetAllOrders(pairId string) (
		ask map[float64][]order.Order,
		bid map[float64][]order.Order,
	)
}

type BookImpl struct {
	mu          sync.Mutex
	askTreesMap map[string]*redblacktree.Tree
	bidTreesMap map[string]*redblacktree.Tree
}

func (b *BookImpl) AddOrder(o order.Order) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var tree *redblacktree.Tree
	if o.Type == order.ASK {
		tree = b.askTreesMap[o.PairID]
	} else {
		tree = b.bidTreesMap[o.PairID]
	}
	if tree == nil {
		tree = redblacktree.NewWith(utils.Float64Comparator)
		if o.Type == order.ASK {
			b.askTreesMap[o.PairID] = tree
		} else {
			b.bidTreesMap[o.PairID] = tree
		}
	}

	node := tree.GetNode(o.Price)
	if node == nil {
		tree.Put(o.Price, &order.OrderList{List: []order.Order{o}})
		return
	}

	orderList := node.Value.(*order.OrderList).List
	orderList = append(orderList, o)
	sort.Slice(orderList, func(i, j int) bool {
		return orderList[i].CreatedAt.Before(orderList[j].CreatedAt)
	})
	node.Value.(*order.OrderList).List = orderList
}

func (b *BookImpl) MatchOrder(o order.Order) (*order.Order, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var tree *redblacktree.Tree
	if o.Type == order.ASK {
		tree = b.bidTreesMap[o.PairID]
	} else {
		tree = b.askTreesMap[o.PairID]
	}
	if tree == nil {
		tree = redblacktree.NewWith(utils.Float64Comparator)
		if o.Type == order.ASK {
			b.askTreesMap[o.PairID] = tree
		} else {
			b.bidTreesMap[o.PairID] = tree
		}

		return nil, false
	}

	node := tree.GetNode(o.Price)
	if node == nil {
		return nil, false
	}

	orderList := node.Value.(*order.OrderList).List

	if len(orderList) == 0 {
		return nil, false
	}

	sort.Slice(orderList, func(i, j int) bool {
		return orderList[i].CreatedAt.Before(orderList[j].CreatedAt)
	})

	foundOrder := orderList[0]
	node.Value.(*order.OrderList).List = orderList[1:]
	return &foundOrder, true
}

func (b *BookImpl) GetAllOrders(pairId string) (
	ask map[float64][]order.Order,
	bid map[float64][]order.Order,
) {
	pairAskTree := b.askTreesMap[pairId]
	pairBidTree := b.bidTreesMap[pairId]

	ask = make(map[float64][]order.Order)
	bid = make(map[float64][]order.Order)
	if pairAskTree != nil {
		it := pairAskTree.Iterator()
		for it.Next() != false {
			ask[it.Key().(float64)] = it.Value().(*order.OrderList).List
		}

	}
	if pairBidTree != nil {

		it := pairBidTree.Iterator()
		for it.Next() != false {
			bid[it.Key().(float64)] = it.Value().(*order.OrderList).List
		}
	}

	return
}

func NewBook() Book {
	return &BookImpl{
		askTreesMap: make(map[string]*redblacktree.Tree, 0),
		bidTreesMap: make(map[string]*redblacktree.Tree, 0),
	}
}
