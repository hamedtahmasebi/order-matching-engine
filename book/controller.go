package book

import (
	"errors"
	"net/http"
	"order-book/logger"
	"order-book/order"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

var (
	ErrFieldRequired = errors.New("ErrFieldRequired")
	ErrInvalidData   = errors.New("ErrInvalidData")
)

type Response struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
	Error   error
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
		orderId, err := strconv.Atoi(id)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return c.JSON(&Response{
				Message: "Invalid ID",
				Data:    nil,
			})
		}

		err = book.CancellOrder(orderId)
		if err == ErrOrderNotFound {
			c.Status(http.StatusNotFound)
			return c.JSON(&Response{
				Message: "The order not found",
				Data:    nil,
			})
		}

		c.Status(http.StatusOK)
		return c.JSON(&Response{
			Message: "Order cancelled successfully",
			Data:    nil,
		})
	})
	r.Post("/add-order", func(c *fiber.Ctx) error {
		var order order.Order
		if err := c.BodyParser(&order); err != nil {
			return err
		}
		order.CreatedAt = time.Now()

		book.AddOrder(order)
		resp := &Response{
			Message: "Order Submitted Succesfully",
			Data:    nil,
		}
		c.Status(http.StatusAccepted)
		return c.JSON(resp)

	})

	r.Get("/ws/order-book/:pair_id", websocket.New(func(c *websocket.Conn) {
		defer func() {
			c.Close()
		}()

		pairId := c.Params("pair_id")
		if pairId == "" {
			c.WriteJSON(&Response{
				Error:   ErrFieldRequired,
				Message: "Please provide a pair_id",
				Data:    nil,
			})
			c.Close()
		}

		size, err := strconv.Atoi(c.Query("size"))
		if err != nil {
			c.WriteJSON(&Response{
				Error:   ErrInvalidData,
				Message: "Size should be a number",
			})
		}

		offset, err := strconv.Atoi(c.Query("offset"))
		if err != nil {
			c.WriteJSON(&Response{
				Error:   ErrInvalidData,
				Message: "Offset should be a number",
			})
		}

		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()
		go func() {
			defer c.Close()
			for {
				_, _, err := c.ReadMessage()
				if err != nil {
					return
				}
			}
		}()

		for t := range ticker.C {
			err := c.WriteControl(websocket.PingMessage, []byte("Ping message"), time.Now().Add(5*time.Second))
			if err != nil {
				logger.Error("Closing ws connection", map[string]any{
					"err": err.Error(),
				})
				break
			}
			asks, bids := book.GetOrders(pairId, size, offset)
			err = c.WriteJSON(map[string]any{
				"asks": asks,
				"bids": bids,
				"time": t,
			})
			if err != nil {
				logger.Error("Error while sending orders through ws", map[string]any{
					"err":     err,
					"pair_id": pairId,
					"size":    size,
					"offset":  offset,
				})
				c.Close()
				return
			}
		}

	}))

	// r.Get("/order-book/:pair_id", func(c *fiber.Ctx) error {
	// 	pairId := c.Params("pair_id")
	// 	if pairId == "" {
	// 		c.Status(http.StatusBadRequest)
	// 		return c.JSON(&Response{
	// 			Message: "Pair ID is required",
	// 			Data:    nil,
	// 		})
	// 	}

	// 	asks, bids := book.GetAllOrders(pairId)
	// 	asksJson := make(map[string][]order.Order, len(asks))
	// 	bidsJson := make(map[string][]order.Order, len(bids))

	// 	for k, v := range asks {
	// 		asksJson[strconv.FormatFloat(k, 'f', -1, 64)] = v
	// 	}

	// 	for k, v := range bids {
	// 		bidsJson[strconv.FormatFloat(k, 'f', -1, 64)] = v
	// 	}

	// 	c.Status(http.StatusOK)
	// 	return c.JSON(&Response{
	// 		Message: "",
	// 		Data: map[string]any{
	// 			"asks": asksJson,
	// 			"bids": bidsJson,
	// 		},
	// 	})

	// })

}
