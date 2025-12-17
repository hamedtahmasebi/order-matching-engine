package order

import "time"

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

type Order struct {
	Price     float64   `json:"price"`
	Amount    float64   `json:"amount"`
	PairID    string    `json:"pair_id"`
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	CreatedAt time.Time `json:"created_at"`
	Type      OrderType `json:"type"`
}
