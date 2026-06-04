package ws

import (
	"bufio"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

type TokenValidator func(token string) (userSsoID string, err error)

func NewSSEHandler(hub *Hub, validator TokenValidator) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := c.Query("token")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token required"})
		}

		ssoID, err := validator(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")

		client := hub.Register(ssoID)
		defer hub.Unregister(client)

		return c.SendStreamWriter(func(w *bufio.Writer) {
			fmt.Fprintf(w, "data: {\"connected\":true}\n\n")
			w.Flush()
			for msg := range client.Ch {
				data, _ := hub.MarshalMessage(msg)
				fmt.Fprintf(w, "data: %s\n\n", data)
				w.Flush()
			}
		})
	}
}
