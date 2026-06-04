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
	"strings"
	"time"

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

	slog.Info("[GROQ] Initialized", "model", g.model, "language", g.language, "api_key_set", g.apiKey != "", "api_key_len", len(g.apiKey))
	if g.apiKey == "" {
		slog.Error("[GROQ] API key is empty! Set GROQ_API_KEY in .env")
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
	if len(result.Segments) == 0 {
		return result.Text, nil
	}
	return buildSRTFromSegments(result.Segments), nil
}

func buildSRTFromSegments(segments []Segment) string {
	var sb strings.Builder
	for i, seg := range segments {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("%d\n%s --> %s\n%s",
			i+1,
			formatSRTTime(seg.Start),
			formatSRTTime(seg.End),
			strings.TrimSpace(seg.Text),
		))
	}
	return sb.String()
}

func formatSRTTime(seconds float64) string {
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	s := int(seconds) % 60
	ms := int((seconds - float64(int(seconds))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

func (g *GroqWhisper) TranscriptDetailed(fPath string) (*TranscriptResult, error) {
	slog.Info("[GROQ] Starting transcription", "file", fPath)

	fileInfo, err := os.Stat(fPath)
	if err != nil {
		slog.Error("[GROQ] File stat failed", "file", fPath, "error", err)
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	slog.Info("[GROQ] File info", "name", fileInfo.Name(), "size_bytes", fileInfo.Size())

	file, err := os.Open(fPath)
	if err != nil {
		slog.Error("[GROQ] Failed to open file", "file", fPath, "error", err)
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", fPath)
	if err != nil {
		slog.Error("[GROQ] Failed to create form file", "error", err)
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	written, err := io.Copy(part, file)
	if err != nil {
		slog.Error("[GROQ] Failed to copy file to form", "error", err)
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}
	slog.Info("[GROQ] File copied to multipart", "bytes_written", written)

	_ = writer.WriteField("model", g.model)
	_ = writer.WriteField("language", g.language)
	_ = writer.WriteField("response_format", "verbose_json")

	if err := writer.Close(); err != nil {
		slog.Error("[GROQ] Failed to close multipart writer", "error", err)
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	slog.Info("[GROQ] Multipart body created", "content_type", writer.FormDataContentType(), "body_size", requestBody.Len())

	url := "https://api.groq.com/openai/v1/audio/transcriptions"
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		slog.Error("[GROQ] Failed to create HTTP request", "url", url, "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	slog.Info("[GROQ] Sending request", "url", url, "model", g.model, "language", g.language, "has_auth_header", g.apiKey != "")

	startTime := time.Now()
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	elapsed := time.Since(startTime)

	if err != nil {
		slog.Error("[GROQ] HTTP request failed", "elapsed", elapsed.String(), "error", err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("[GROQ] Response received", "status", resp.StatusCode, "elapsed", elapsed.String())

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("[GROQ] Failed to read response body", "error", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	slog.Info("[GROQ] Response body", "size", len(body), "body_preview", truncateString(string(body), 500))

	if resp.StatusCode != http.StatusOK {
		slog.Error("[GROQ] API returned non-200", "status", resp.StatusCode, "body", string(body))
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
		slog.Error("[GROQ] Failed to parse JSON response", "error", err, "raw_body", truncateString(string(body), 300))
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	slog.Info("[GROQ] Transcription completed", "text_length", len(groqResp.Text), "segments", len(groqResp.Segments), "elapsed_total", time.Since(startTime).String())
	if len(groqResp.Text) > 0 {
		slog.Info("[GROQ] Text preview", "text", truncateString(groqResp.Text, 200))
	}

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

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
