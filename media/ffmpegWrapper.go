package media

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type FfmpegWrapper struct{}

const (
	ffmpegOutRegex       = `\[out#[0-9].*`
	ffmpegErrorRegex     = "Error.*"
	ffprobeDurationRegex = `\b\d{2}:\d{2}:\d{2}\.\d{2}\b`
)

func (w *FfmpegWrapper) ExtractAudio(inputVideoPath, outputAudioPath string) error {
	slog.Info("[MEDIA] Extracting audio from video", "input", inputVideoPath)

	args := []string{"-i", inputVideoPath, "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", outputAudioPath, "-y"}
	cmd := exec.Command("ffmpeg", args...)

	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	err := cmd.Run()
	if err != nil {
		slog.Error("[MEDIA] Error extracting audio: " + err.Error())
		if msg, ok := handleFfmpegLog(stdErr.String(), ffmpegErrorRegex); ok {
			slog.Error("[MEDIA] " + msg)
		}
		return err
	}

	slog.Info("[MEDIA] Successfully extracted audio", "output", outputAudioPath)
	return nil
}

func (w *FfmpegWrapper) Unmerge(inputVideoPath, outputVideoPath, outputAudioPath string) error {
	inputValue := inputVideoPath
	if abs, err := filepath.Abs(inputVideoPath); err == nil {
		inputValue = abs
	}

	slog.Info("[MEDIA] Start unmerge audio and video from " + inputValue)

	args := []string{"-i", inputVideoPath, "-an", "-c:v", "copy", outputVideoPath, "-vn", "-c:a", "copy", outputAudioPath, "-y"}
	cmd := exec.Command("ffmpeg", args...)

	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	err := cmd.Run()
	if err != nil {
		slog.Error("[MEDIA] Error on unmerge files: " + err.Error())
		if msg, ok := handleFfmpegLog(stdErr.String(), ffmpegErrorRegex); ok {
			slog.Error("[MEDIA] Ffmpeg error message:\n" + msg)
		}
		return err
	}

	if msg, ok := handleFfmpegLog(stdErr.String(), ffmpegOutRegex); ok {
		slog.Info("[MEDIA] ffmpeg output message: \n" + msg)
	}

	slog.Info("[MEDIA] Successfully unmerge audio and video")
	return err
}

func (w *FfmpegWrapper) Merge(inputVideoPath, inputAudioPath, outputVideoPath string) error {
	slog.Info("[MEDIA] Start merge video and audio files")

	speedUpAudioRatio, err := getAudioSpeedRatio(inputVideoPath, inputAudioPath)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	atempo := fmt.Sprintf("atempo=%f", speedUpAudioRatio)

	args := []string{"-i", inputAudioPath, "-i", inputVideoPath, "-filter:a", atempo, outputVideoPath, "-y"}
	cmd := exec.Command("ffmpeg", args...)

	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	err = cmd.Run()
	if err != nil {
		slog.Error("[MEDIA] Error on merge files: " + err.Error())
		if msg, ok := handleFfmpegLog(stdErr.String(), ffmpegErrorRegex); ok {
			slog.Error("[MEDIA] Ffmpeg error message:\n" + msg)
		}
		return err
	}

	if msg, ok := handleFfmpegLog(stdErr.String(), ffmpegOutRegex); ok {
		slog.Info("[MEDIA] ffmpeg output message:\n" + msg)
	}

	absOutPath, outPathErr := filepath.Abs(outputVideoPath)
	outValue := absOutPath
	if outPathErr != nil {
		outValue = outputVideoPath
	}

	slog.Info("[MEDIA] Successfully merged files into path " + outValue)
	return err
}

func (w *FfmpegWrapper) MergeSubtitle(inputVideoPath, inputSubtitlePath, outputVideoPath string) (string, error) {
	slog.Info("[MEDIA] Merge subtitles to video")

	subtitles := fmt.Sprintf("subtitles=%s", inputSubtitlePath)
	args := []string{"-i", inputVideoPath, "-vf", subtitles, outputVideoPath, "-y"}

	cmd := exec.Command("ffmpeg", args...)

	var stdErr bytes.Buffer
	var stdOut bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	err := cmd.Run()
	if err != nil {
		slog.Error("[MEDIA] Error on merge subtitle: " + err.Error())
		if msg, ok := handleFfmpegLog(stdErr.String(), ffmpegErrorRegex); ok {
			slog.Error("[MEDIA] " + msg)
		}
	}
	slog.Info("[MEDIA] Successfully merge subtitle to video")
	return outputVideoPath, nil
}

func (w *FfmpegWrapper) ConvertToWav(inputAudioPath, wavAudioPath string) error {
	args := []string{"-i", inputAudioPath, "-f", "wav", wavAudioPath}
	cmd := exec.Command("ffmpeg", args...)

	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	err := cmd.Run()
	if err != nil {
		slog.Error("[MEDIA] Error on convert to wav: " + err.Error())
		if msg, ok := handleFfmpegLog(stdErr.String(), ffmpegErrorRegex); ok {
			slog.Error("[MEDIA] " + msg)
		}
		return err
	}

	slog.Info("[MEDIA] Successfully converted to wav on path " + wavAudioPath)
	return nil
}

func (w *FfmpegWrapper) ConcatAudio(audioFiles []string, outputPath string) (string, error) {
	slog.Info("[MEDIA] Concatenating audio files")

	listFile := outputPath + ".list.txt"
	var sb strings.Builder
	for _, f := range audioFiles {
		sb.WriteString(fmt.Sprintf("file '%s'\n", f))
	}
	if err := os.WriteFile(listFile, []byte(sb.String()), 0644); err != nil {
		return "", err
	}
	defer os.Remove(listFile)

	args := []string{"-f", "concat", "-safe", "0", "-i", listFile, "-c", "copy", outputPath, "-y"}
	cmd := exec.Command("ffmpeg", args...)

	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	if err := cmd.Run(); err != nil {
		slog.Error("[MEDIA] Error concatenating audio: " + err.Error())
		return "", err
	}

	slog.Info("[MEDIA] Successfully concatenated audio to " + outputPath)
	return outputPath, nil
}

func handleFfmpegLog(log string, exp string) (string, bool) {
	r, _ := regexp.Compile(exp)
	matches := r.FindAllString(log, -1)
	if len(matches) == 0 {
		return "", false
	}
	return strings.Join(matches, "\n"), true
}

func getAudioSpeedRatio(vPath, aPath string) (float64, error) {
	videoDuration, err := getMediaDuration(vPath)
	if err != nil {
		return 0, err
	}
	audioDuration, err := getMediaDuration(aPath)
	if err != nil {
		return 0, err
	}
	return audioDuration / videoDuration, nil
}

func getMediaDuration(fPath string) (float64, error) {
	args := []string{"-i", fPath}
	cmd := exec.Command("ffprobe", args...)

	var stdOut bytes.Buffer
	var stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	err := cmd.Run()
	if err != nil {
		return 0, errors.New(stdErr.String())
	}

	outText := stdErr.String()
	r := regexp.MustCompile(ffprobeDurationRegex)
	matches := r.FindAllString(outText, -1)

	if len(matches) == 0 {
		return 0, errors.New("Failed to retrieve duration of file: " + fPath)
	}
	multiplier := []float64{3600, 60, 1}
	duration := strings.Split(matches[0], ":")

	var secondsDuration float64
	for i := 0; i < 3; i++ {
		sTime := duration[i]
		currentDuration, _ := strconv.ParseFloat(sTime, 32)
		secondsDuration += currentDuration * multiplier[i]
	}

	return secondsDuration, nil
}
