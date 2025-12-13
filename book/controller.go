package book

import (
	"net/http"
	"order-book/order"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Response struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func BindOrderBookRouter(r fiber.Router, book Book) {
	r.Delete("/order-book/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			c.Status(http.StatusBadRequest)
			return c.JSON(&Response{
				Message: "ID is required",
				Data:    nil,
			})
		}
		err := book.RemoveOrder(id)
		if err == ErrOrderNotFound {
			c.Status(http.StatusNotFound)
			return c.JSON(&Response{
				Message: "The order not found",
				Data:    nil,
			})
		}

		c.Status(http.StatusOK)
		return c.JSON(&Response{
			Message: "Order removed successfully",
			Data:    nil,
		})
	})
	r.Post("/add-order", func(c *fiber.Ctx) error {
		var order order.Order
		if err := c.BodyParser(&order); err != nil {
			return err
		}
		order.CreatedAt = time.Now()
		order.ID = uuid.NewString()

		book.AddOrder(order)
		resp := &Response{
			Message: "Order Submitted Succesfully",
			Data:    nil,
		}
		c.Status(http.StatusAccepted)
		return c.JSON(resp)

	})

	r.Get("/order-book/:pair_id", func(c *fiber.Ctx) error {
		pairId := c.Params("pair_id")
		if pairId == "" {
			c.Status(http.StatusBadRequest)
			return c.JSON(&Response{
				Message: "Pair ID is required",
				Data:    nil,
			})
		}

		asks, bids := book.GetAllOrders(pairId)
		asksJson := make(map[string][]order.Order, len(asks))
		bidsJson := make(map[string][]order.Order, len(bids))

		for k, v := range asks {
			asksJson[strconv.FormatFloat(k, 'f', -1, 64)] = v
		}

		for k, v := range bids {
			bidsJson[strconv.FormatFloat(k, 'f', -1, 64)] = v
		}

		c.Status(http.StatusOK)
		return c.JSON(&Response{
			Message: "",
			Data: map[string]any{
				"asks": asksJson,
				"bids": bidsJson,
			},
		})

	})

}
