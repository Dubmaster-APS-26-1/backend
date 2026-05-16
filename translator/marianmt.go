package translator

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type MarianMT struct{}

func (mt MarianMT) Translate(text string) (string, error) {
	results, err := mt.TranslateBatch([]string{text})
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", fmt.Errorf("translation returned empty result")
	}
	return results[0], nil
}

func (mt MarianMT) TranslateBatch(texts []string) ([]string, error) {
	slog.Info("[TRANSLATE] Generating batch translation", "count", len(texts))

	scriptPath, err := filepath.Abs("scripts/translator_marianmt.py")
	if err != nil {
		return nil, err
	}

	pipeDir, _ := filepath.Abs("pipe/")
	os.MkdirAll(pipeDir, 0755)

	inputPath := filepath.Join(pipeDir, uuid.NewString()+"_input.txt")
	outputPath := filepath.Join(pipeDir, uuid.NewString()+"_output.txt")

	defer func() {
		os.Remove(inputPath)
		os.Remove(outputPath)
	}()

	inputContent := strings.Join(texts, "\n")
	if err := os.WriteFile(inputPath, []byte(inputContent), 0644); err != nil {
		return nil, err
	}

	pythonPath := "./.venv/bin/python"
	if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
		pythonPath = "python3"
	}

	cmd := exec.Command(pythonPath, scriptPath, "--input", inputPath, "--output", outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		slog.Error("[TRANSLATE] Error: " + err.Error() + "\n" + stderr.String())
		return nil, err
	}

	b, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, err
	}

	if len(b) == 0 {
		return nil, fmt.Errorf("translation result is empty")
	}

	results := strings.Split(strings.TrimSpace(string(b)), "\n")
	slog.Info("[TRANSLATE] Batch translation completed", "count", len(results))
	return results, nil
}
