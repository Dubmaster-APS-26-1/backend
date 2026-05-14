package media

import (
	"bytes"
	"log/slog"
	"os/exec"
	"regexp"
	"strings"
)

type FfmpegWrapper struct{}

func (w *FfmpegWrapper) ExtractAudio(inputVideoPath, outputAudioPath string) error {
	slog.Info("[MEDIA] Extracting audio from video", "input", inputVideoPath)

	args := []string{"-i", inputVideoPath, "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", outputAudioPath, "-y"}
	cmd := exec.Command("ffmpeg", args...)

	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	err := cmd.Run()
	if err != nil {
		slog.Error("[MEDIA] Error extracting audio: " + err.Error())
		if msg, ok := handleFfmpegLog(stdErr.String()); ok {
			slog.Error("[MEDIA] " + msg)
		}
		return err
	}

	slog.Info("[MEDIA] Successfully extracted audio", "output", outputAudioPath)
	return nil
}

func handleFfmpegLog(log string) (string, bool) {
	r, _ := regexp.Compile(`Error.*`)
	matches := r.FindAllString(log, -1)
	if len(matches) == 0 {
		return "", false
	}
	return strings.Join(matches, "\n"), true
}
