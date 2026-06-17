package controller

import (
	"log/slog"
	"os"
	"path/filepath"

	"dubmaster/pipeline"
	"dubmaster/singleton"
	"dubmaster/video"

	"github.com/gofiber/fiber/v3"
)

func SendVideo(c fiber.Ctx) error {
	slog.Info("[VIDEO] Uploading video")
	ssoId := c.Locals("user.sso_id").(string)

	title := c.FormValue("title")
	if title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}

	videosPath, _ := filepath.Abs("videos/")
	os.MkdirAll(videosPath, 0755)

	uploadType := c.FormValue("uploadType")

	var originalPath string
	var sourceUrl string

	switch uploadType {
	case "YOUTUBE":
		url := c.FormValue("url")
		if url == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "url is required"})
		}
		sourceUrl = url
	case "FILE":
		fileHeader, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		originalPath = filepath.Join(videosPath, ssoId+"_temp.mp4")
		c.SaveFile(fileHeader, originalPath)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid uploadType"})
	}

	v, err := singleton.VideoService.Create(title, uploadType, sourceUrl, originalPath, ssoId)
	if err != nil {
		slog.Error("[VIDEO] Error creating video record", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if uploadType == "YOUTUBE" {
		uniquePath := filepath.Join(videosPath, v.ID+"_original.mp4")
		downloadedPath := singleton.Downloader.Download(sourceUrl, uniquePath)
		if err := singleton.VideoService.UpdateOriginalPath(v.ID, downloadedPath); err != nil {
			slog.Error("[VIDEO] Error updating original path", "video_id", v.ID, "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	singleton.Queue.Enqueue(pipeline.Job{
		VideoID: v.ID,
		Type:    pipeline.JobTranscribe,
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":     v.ID,
		"status": v.Status,
	})
}

func GetVideoById(c fiber.Ctx) error {
	id := c.Params("id")
	ssoId := c.Locals("user.sso_id").(string)

	v, err := singleton.VideoService.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "video not found"})
	}
	if v.UserSsoId != ssoId {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	return c.Status(fiber.StatusOK).JSON(v)
}

func ListVideos(c fiber.Ctx) error {
	ssoId := c.Locals("user.sso_id").(string)
	videos, err := singleton.VideoService.GetByUser(ssoId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if videos == nil {
		videos = []*video.Video{}
	}
	return c.Status(fiber.StatusOK).JSON(videos)
}

func GetTranscription(c fiber.Ctx) error {
	id := c.Params("id")
	v, err := singleton.VideoService.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "video not found"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"transcription": v.Transcription,
	})
}

func UpdateTranscription(c fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		Transcription string `json:"transcription"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	err := singleton.VideoService.UpdateTranscription(id, body.Transcription)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "transcription updated"})
}

func TranslateVideo(c fiber.Ctx) error {
	id := c.Params("id")
	v, err := singleton.VideoService.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "video not found"})
	}

	if v.Transcription == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "video has no transcription"})
	}

	singleton.Queue.Enqueue(pipeline.Job{
		VideoID: id,
		Type:    pipeline.JobTranslate,
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"id":     id,
		"status": "translating",
	})
}

func DubVideo(c fiber.Ctx) error {
	id := c.Params("id")
	ssoId := c.Locals("user.sso_id").(string)

	v, err := singleton.VideoService.GetByID(id)
	if err != nil || v.UserSsoId != ssoId {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "video not found"})
	}

	if v.TranslatedTranscription == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "video has no translation"})
	}

	singleton.Queue.Enqueue(pipeline.Job{
		VideoID: id,
		Type:    pipeline.JobDub,
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"id":     id,
		"status": "generating_voice",
	})
}

func GetVideoFile(c fiber.Ctx) error {
	id := c.Params("id")
	v, err := singleton.VideoService.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "video not found"})
	}

	if _, err := os.Stat(v.OriginalPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "video file not found"})
	}

	c.Set("Content-Type", "video/mp4")
	c.Set("Accept-Ranges", "bytes")
	return c.SendFile(v.OriginalPath)
}

func GetDubbedVideoFile(c fiber.Ctx) error {
	id := c.Params("id")
	v, err := singleton.VideoService.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "video not found"})
	}

	if v.DubbedPath == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "dubbed video not available"})
	}

	if _, err := os.Stat(v.DubbedPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "dubbed video file not found"})
	}

	c.Set("Content-Type", "video/mp4")
	c.Set("Accept-Ranges", "bytes")
	return c.SendFile(v.DubbedPath)
}
