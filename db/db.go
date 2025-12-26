package db

import (
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func Connect() (*sqlx.DB, error) {
	dsn := "postgres://postgres:postgres@localhost:5432/order_book?sslmode=disable"
	pool, e := sqlx.Open("postgres", dsn)
	if e != nil {
		return nil, e
	}
	pool.SetMaxOpenConns(10)
	pool.SetConnMaxLifetime(5 * time.Minute)

	pool.SetMaxIdleConns(5)
	pool.SetConnMaxIdleTime(10 * time.Minute)
	return pool, e
}
