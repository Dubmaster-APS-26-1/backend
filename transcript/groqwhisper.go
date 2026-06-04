package transcript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"dubmaster/config"
)

type GroqWhisper struct {
	apiKey   string
	model    string
	language string
}

type GroqOption func(*GroqWhisper)

func NewGroqWhisper(opts ...GroqOption) *GroqWhisper {
	g := &GroqWhisper{
		apiKey:   config.GroqApiKey,
		model:    "whisper-large-v3",
		language: "en",
	}
	for _, o := range opts {
		o(g)
	}
	return g
}

func WithGroqModel(model string) GroqOption {
	return func(g *GroqWhisper) { g.model = model }
}

func WithGroqLanguage(lang string) GroqOption {
	return func(g *GroqWhisper) { g.language = lang }
}

func (g *GroqWhisper) Transcript(fPath string) (string, error) {
	result, err := g.TranscriptDetailed(fPath)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

func (g *GroqWhisper) TranscriptDetailed(fPath string) (*TranscriptResult, error) {
	slog.Info("[GROQ] Starting transcription", "file", fPath)

	file, err := os.Open(fPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", fPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	_ = writer.WriteField("model", g.model)
	_ = writer.WriteField("language", g.language)
	_ = writer.WriteField("response_format", "verbose_json")

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/audio/transcriptions", &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("[GROQ] API error", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("groq API error: %d - %s", resp.StatusCode, string(body))
	}

	var groqResp struct {
		Text     string `json:"text"`
		Segments []struct {
			Start float64 `json:"start"`
			End   float64 `json:"end"`
			Text  string  `json:"text"`
		} `json:"segments"`
	}

	if err := json.Unmarshal(body, &groqResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	slog.Info("[GROQ] Transcription completed", "text_length", len(groqResp.Text), "segments", len(groqResp.Segments))

	result := &TranscriptResult{Text: groqResp.Text}
	for _, seg := range groqResp.Segments {
		result.Segments = append(result.Segments, Segment{
			Start: seg.Start,
			End:   seg.End,
			Text:  seg.Text,
		})
	}

	return result, nil
}
