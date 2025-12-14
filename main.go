package main

import (
	"order-book/book"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

func main() {
	orderBook := book.NewBook()

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
