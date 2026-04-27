package main

import (
	"log"
	"dubmaster/auth"
	"dubmaster/config"
	"dubmaster/controller"
	"dubmaster/database"
	"dubmaster/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func main() {
	config.LoadEnvironment()

	err := database.ConnectDatabase()
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Use(cors.New())

	app.Get("/api/auth/callback", auth.Callback)
	app.Get("/api/videos/:filename", controller.GetVideo)

	privateRoute := app.Group("/api", middleware.Authentication)

	privateRoute.Get("/me", controller.Me, middleware.DefaultAuthorization)
	privateRoute.Post("/videos", controller.SendVideo, middleware.DefaultAuthorization)

	log.Fatal(app.Listen(":4000"))
}