package transcript

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var availableModels = [...]string{"turbo", "tiny", "base", "small", "medium", "large"}

const errRegex = `Skipping.*|Error.*`

type Whisper struct {
	model    string
	language string
}

type Option func(*Whisper)

func NewWhisper(options ...Option) *Whisper {
	w := new(Whisper)
	w.model = "turbo"
	w.language = "English"
	for _, option := range options {
		option(w)
	}
	return w
}

func WithModel(model string) Option {
	return func(w *Whisper) {
		for _, m := range availableModels {
			if model == m {
				w.model = model
				return
			}
		}
		w.model = "turbo"
	}
}

func WithLanguage(language string) Option {
	return func(w *Whisper) {
		w.language = language
	}
}

func (w *Whisper) Transcript(audioPath string) (string, error) {
	slog.Info("[TRANSCRIPT] Start transcribing audio")

	outputDir := filepath.Dir(audioPath)
	baseName := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	args := []string{
		audioPath,
		"--language", w.language,
		"--model", w.model,
		"--output_dir", outputDir,
		"--output_format", "srt",
	}
	cmd := exec.Command("whisper", args...)

	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	cmd.Run()

	errReg := regexp.MustCompile(errRegex)
	errMsgs := errReg.FindAllString(stdErr.String(), -1)
	if len(errMsgs) > 0 {
		slog.Error("[TRANSCRIPT] Whisper errors: " + strings.Join(errMsgs, ","))
		return "", errors.New(strings.Join(errMsgs, ","))
	}

	srtPath := filepath.Join(outputDir, baseName+".srt")
	content, err := os.ReadFile(srtPath)
	if err != nil {
		slog.Error("[TRANSCRIPT] Error reading SRT file: " + err.Error())
		return "", err
	}

	os.Remove(srtPath)

	slog.Info("[TRANSCRIPT] Successfully transcribed audio")
	return string(content), nil
}

func (w *Whisper) TranscriptDetailed(audioPath string) (*TranscriptResult, error) {
	text, err := w.Transcript(audioPath)
	if err != nil {
		return nil, err
	}
	return &TranscriptResult{Text: text}, nil
}
