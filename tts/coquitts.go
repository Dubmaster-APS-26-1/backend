package tts

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CoquiTTS struct {
	modelName   string
	languageIdx string
}

type Option func(*CoquiTTS)

func NewCoquiTTS(opts ...Option) *CoquiTTS {
	cq := new(CoquiTTS)
	cq.modelName = "tts_models/multilingual/multi-dataset/xtts_v2"
	cq.languageIdx = "pt"
	for _, o := range opts {
		o(cq)
	}
	return cq
}

func WithModelName(modelName string) Option {
	return func(ct *CoquiTTS) { ct.modelName = modelName }
}

func WithLanguageIdx(languageIdx string) Option {
	return func(ct *CoquiTTS) { ct.languageIdx = languageIdx }
}

func (c *CoquiTTS) Speech(message, speakerWavPath string) (string, error) {
	slog.Info("[TTS] Generating speech now")

	t := time.Now()
	outFile := fmt.Sprintf("%d-%d-%d-%s-tts.wav", t.Day(), t.Month(), t.Year(), uuid.New())
	outPath, err := filepath.Abs("pipe/" + outFile)
	if err != nil {
		slog.Error("[TTS] Error in getting absolute outPath: " + err.Error())
		return "", err
	}

	pipeDir, _ := filepath.Abs("pipe/")
	os.MkdirAll(pipeDir, 0755)

	text := strings.ReplaceAll(message, ".", `\n`)
	slog.Info("[TTS] Model Name: " + c.modelName)
	slog.Info("[TTS] Text length: " + fmt.Sprintf("%d", len(text)))
	slog.Info("[TTS] Speaker Wav Path: " + speakerWavPath)
	slog.Info("[TTS] Out Path: " + outPath)
	slog.Info("[TTS] Language Index: " + c.languageIdx)

	ttsPath := "./.venv/bin/tts"
	if _, err := os.Stat(ttsPath); os.IsNotExist(err) {
		ttsPath = "tts"
	}

	cmd := exec.Command(ttsPath,
		"--model_name", c.modelName,
		"--text", text,
		"--speaker_wav", speakerWavPath,
		"--out_path", outPath,
		"--language_idx", c.languageIdx)

	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	cmd.Stdin = strings.NewReader("y\n")

	err = cmd.Run()
	if err != nil {
		slog.Error("[TTS] Error in running 'cmd.Run': " + err.Error() + "\n\t\t\tCLI Error Output: " + stderr.String())
		return "", err
	}

	slog.Info("[TTS] Speech generated with success. Output path: " + outPath)
	return outPath, nil
}
