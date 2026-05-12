package controller

import (
	"dubmaster/singleton"

	"github.com/gofiber/fiber/v3"
)

func Me(c fiber.Ctx) error {
	email := c.Locals("user.email").(string)
	user, err := singleton.UserService.GetByEmail(email)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(
		fiber.Map{"name": user.Name, "email": user.Email},
	)
}