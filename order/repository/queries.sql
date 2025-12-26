-- name: GetOneById :one
SELECT * FROM tbl_orders WHERE id = $1;

-- name: GetOrders :many
SELECT * FROM tbl_orders WHERE account_id = $3 LIMIT $1 OFFSET $2;

-- name: InsertOneOrderHistoryEvent :exec
INSERT INTO tbl_order_history_events (event, order_id, metadata)
VALUES ($1, $2, $3);

-- name: CreateOrder :one
INSERT INTO tbl_orders (pair_id, price, amount, account_id, order_type)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetHistoryById :many
SELECT * FROM tbl_order_history_events WHERE order_id = $1 ORDER BY created_at DESC;
