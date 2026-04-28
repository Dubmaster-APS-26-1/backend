package controller

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"dubmaster/singleton"

	"github.com/gofiber/fiber/v3"
)

func SendVideo(c fiber.Ctx) error {
	slog.Info("[VIDEO] Sending video")
	id := c.Locals("user.sso_id").(string)

	videosPath, _ := filepath.Abs("videos/")
	userVideosPath := videosPath + "/" + id
	outputPath := userVideosPath + ".mp4"

	err := os.MkdirAll(videosPath, 0755)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	uploadType := c.FormValue("uploadType")

	switch uploadType {
	case "YOUTUBE":
		url := c.FormValue("url")
		downloadedPath := singleton.Downloader.Download(url)

		src, err := os.Open(downloadedPath)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		defer src.Close()

		dst, err := os.Create(outputPath)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		defer dst.Close()

		io.Copy(dst, src)

		slog.Info("[VIDEO] Processing video (5s delay)")
		time.Sleep(5 * time.Second)

		os.Remove(downloadedPath)
	case "FILE":
		fileHeader, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		fileInputPath := userVideosPath + "_input" + ".mp4"
		c.SaveFile(fileHeader, fileInputPath)

		slog.Info("[VIDEO] Processing video (5s delay)")
		time.Sleep(5 * time.Second)

		src, err := os.Open(fileInputPath)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		defer src.Close()

		dst, err := os.Create(outputPath)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		defer dst.Close()

		io.Copy(dst, src)
		os.Remove(fileInputPath)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"videoUrl": "http://localhost:4000/api/videos/" + id + ".mp4",
	})
}

func GetVideo(c fiber.Ctx) error {
	filename := c.Params("filename")
	videosPath, _ := filepath.Abs("videos/")
	userVideosPath := videosPath + "/" + filename

	if _, err := os.Stat(userVideosPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Vídeo não encontrado",
		})
	}

	c.Set("Content-Type", "video/mp4")
	c.Set("Accept-Ranges", "bytes")

	return c.SendFile(userVideosPath)
}