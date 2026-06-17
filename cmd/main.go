package main

import (
	"log"

	"dubmaster/auth"
	"dubmaster/config"
	"dubmaster/controller"
	"dubmaster/database"
	"dubmaster/middleware"
	"dubmaster/pipeline"
	"dubmaster/singleton"
	"dubmaster/transcript"
	"dubmaster/ws"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func init() {
	config.LoadEnvironment()
	singleton.Transcripter = transcript.NewTranscripter()
}

func main() {
	err := database.ConnectDatabase()
	if err != nil {
		log.Fatal(err)
	}

	singleton.Queue = pipeline.NewQueue(
		singleton.VideoService,
		singleton.MediaHandler,
		singleton.Transcripter,
		singleton.Translator,
		singleton.TTS,
		singleton.Subtitler,
		singleton.Hub,
	)
	singleton.Queue.Start(3)

	app := fiber.New(fiber.Config{
		BodyLimit: 500 * 1024 * 1024,
	})

	app.Use(cors.New())

	app.Get("/api/auth/callback", auth.Callback)

	sseHandler := ws.NewSSEHandler(singleton.Hub, func(token string) (string, error) {
		userInfo, err := auth.GetUserInfoByAccessToken(token)
		if err != nil {
			return "", err
		}
		ssoID, _ := userInfo["id"].(string)
		return ssoID, nil
	})
	app.Get("/api/events", sseHandler)

	privateRoute := app.Group("/api", middleware.Authentication)

	privateRoute.Get("/me", controller.Me, middleware.DefaultAuthorization)
	privateRoute.Post("/videos", controller.SendVideo, middleware.DefaultAuthorization)
	privateRoute.Get("/videos", controller.ListVideos, middleware.DefaultAuthorization)
	privateRoute.Get("/videos/:id", controller.GetVideoById, middleware.DefaultAuthorization)
	privateRoute.Get("/videos/:id/file", controller.GetVideoFile, middleware.DefaultAuthorization)
	privateRoute.Get("/videos/:id/dubbed", controller.GetDubbedVideoFile, middleware.DefaultAuthorization)
	privateRoute.Get("/videos/:id/transcription", controller.GetTranscription, middleware.DefaultAuthorization)
	privateRoute.Put("/videos/:id/transcription", controller.UpdateTranscription, middleware.DefaultAuthorization)
	privateRoute.Post("/videos/:id/translate", controller.TranslateVideo, middleware.DefaultAuthorization)
	privateRoute.Post("/videos/:id/dub", controller.DubVideo, middleware.DefaultAuthorization)

	log.Fatal(app.Listen(":4000"))
}
