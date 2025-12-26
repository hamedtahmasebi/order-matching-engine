package order

import (
	"context"
	"database/sql"
	"encoding/json"
	"order-book/logger"
	repository "order-book/order/repository/gen"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sqlc-dev/pqtype"
)

type OrderList struct {
	List []Order
}

type OrderType int

const (
	ASK OrderType = iota
	BID
)

func (ot OrderType) String() string {
	return []string{"ASK", "BID"}[ot]
}

type OrderHistoryEvent struct {
	Name     string
	OrderId  int
	Metadata map[string]any
}

type Order struct {
	Price     float64   `json:"price"`
	Amount    float64   `json:"amount"`
	PairID    string    `json:"pair_id"`
	ID        int       `json:"id"`
	AccountID int       `json:"account_id"`
	CreatedAt time.Time `json:"created_at"`
	Type      OrderType `json:"type"`
}

type paginatedOrders struct {
	Orders []Order
	Total  int
}

type OrderRepo interface {
	AddEvent(ev OrderHistoryEvent) error
	GetOrders(page int, size int, accountId int) (*paginatedOrders, error)
	GetOrderByID(id int) (Order, error)
	GetOrderHistoryByID(id int) ([]OrderHistoryEvent, error)
	CreateOrder(pairID string, price float64, amount float64, accountID int, orderType OrderType) (Order, error)
}

type orderRepo struct {
	queries *repository.Queries
	dbpool  *sqlx.DB
}

func (repo *orderRepo) AddEvent(ev OrderHistoryEvent) error {
	jsonRawMsg, err := json.Marshal(ev.Metadata)
	if err != nil {
		return err
	}
	err = repo.queries.InsertOneOrderHistoryEvent(context.Background(), repository.InsertOneOrderHistoryEventParams{
		Event:    ev.Name,
		Metadata: pqtype.NullRawMessage{RawMessage: jsonRawMsg},
	})

	return err
}

func (repo *orderRepo) GetOrders(page int, size int, accountId int) (*paginatedOrders, error) {
	var offset int32
	offset = int32(max(0, page-1) * size)

	dbres, err := repo.queries.GetOrders(context.Background(), repository.GetOrdersParams{
		AccountID: sql.NullInt32{Int32: int32(accountId)},
		Offset:    offset,
		Limit:     int32(size),
	})
	if err != nil {
		return nil, err
	}
	var orders = make([]Order, len(dbres))
	for idx, ord := range dbres {
		o, err := convertOrder(ord)
		if err != nil {
			return nil, err
		}
		orders[idx] = o
	}
	return &paginatedOrders{
		Orders: orders,
		Total:  len(orders),
	}, nil
}

func (repo *orderRepo) GetOrderByID(id int) (Order, error) {
	res, err := repo.queries.GetOneById(context.Background(), int64(id))
	order, err := convertOrder(res)
	return order, err
}

func (repo *orderRepo) CreateOrder(pairID string, price float64, amount float64, accountID int, orderType OrderType) (Order, error) {
	tx, err := repo.dbpool.Begin()
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback()

	qtx := repo.queries.WithTx(tx)
	createdOrder, err := qtx.CreateOrder(context.Background(), repository.CreateOrderParams{
		PairID:    pairID,
		Price:     strconv.FormatFloat(price, 'f', -1, 64),
		Amount:    strconv.FormatFloat(amount, 'f', -1, 64),
		AccountID: sql.NullInt32{Int32: int32(accountID)},
		OrderType: int32(orderType),
	})
	err = qtx.InsertOneOrderHistoryEvent(context.Background(), repository.InsertOneOrderHistoryEventParams{
		Event:   "ORDER_CREATED",
		OrderID: sql.NullInt64{Int64: int64(createdOrder.ID)},
	})

	if err != nil {
		logger.Error("failed to insert order history event, rolling back", map[string]any{
			"order_id": createdOrder.ID,
			"error":    err,
		})
		return Order{}, err
	}

	if err := tx.Commit(); err != nil {
		logger.Error("failed to commit transaction, rolling back", map[string]any{
			"order_id": createdOrder.ID,
			"error":    err,
		})
		return Order{}, err
	}

	o, err := convertOrder(createdOrder)
	if err != nil {
		logger.Error("failed to convert order", map[string]any{
			"order_id": createdOrder.ID,
			"error":    err,
		})
		return Order{}, err
	}
	return o, err
}

func (repo *orderRepo) GetOrderHistoryByID(id int) ([]OrderHistoryEvent, error) {
	res, err := repo.GetOrderHistoryByID(id)
	return res, err
}

func NewOrderRepository(dbpool *sqlx.DB) OrderRepo {
	return &orderRepo{
		queries: repository.New(dbpool),
		dbpool:  dbpool,
	}
}

func convertOrder(ord repository.TblOrder) (res Order, err error) {
	amount, err := strconv.ParseFloat(ord.Amount, 64)
	if err != nil {
		return
	}
	price, err := strconv.ParseFloat(ord.Price, 64)
	if err != nil {
		return
	}

	res.AccountID = int(ord.AccountID.Int32)
	res.Amount = amount
	res.Price = price
	res.ID = int(ord.ID)
	res.Type = OrderType(ord.OrderType)
	res.PairID = ord.PairID
	res.CreatedAt = ord.CreatedAt.Time
	return
}
