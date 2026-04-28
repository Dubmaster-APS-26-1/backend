package input

import (
	"bytes"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

type YouTube struct{}

func New() *YouTube {
	return &YouTube{}
}

func (y *YouTube) Download(url string) string {
	slog.Info("[YOUTUBE] Downloading video with 'yt-dlp'", "url", url)

	path, _ := filepath.Abs("pipe/")
	outputPath := path + "/video.mp4"

	cmd := exec.Command("yt-dlp", "-f", "best", "-o", outputPath, url)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		slog.Error("[YOUTUBE] yt-dlp failed", "stderr", stderr.String())
		os.Stdout.WriteString("[YOUTUBE] yt-dlp stderr:\n\t" + stderr.String())
		panic(err)
	}

	slog.Info("[YOUTUBE] Download finished", "url", url)
	return outputPath
}