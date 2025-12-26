package main

import (
	"order-book/book"
	"order-book/db"
	"order-book/order"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

func main() {
	dbpool, err := db.Connect()
	if err != nil {
		panic(err)
	}
	orderHistoryRepo := order.NewOrderRepository(dbpool)
	orderBook := book.NewBook(orderHistoryRepo)

	app := fiber.New()
	app.Use(logger.New())

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Send([]byte("Working..."))
	})

	book.BindOrderBookRouter(app, orderBook)

	app.Listen(":5000")
}
