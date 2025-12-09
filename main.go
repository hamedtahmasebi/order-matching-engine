package main

import (
	"order-book/book"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	orderBook := book.NewBook()

	app := fiber.New()
	app.Use(logger.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Send([]byte("Working..."))
	})
	book.BindOrderBookRouter(app, orderBook)

	app.Listen(":5000")
}
