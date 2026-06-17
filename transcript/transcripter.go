package transcript

import (
	"log"

	"dubmaster/config"
)

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type TranscriptResult struct {
	Text     string
	Segments []Segment
}

type Transcripter interface {
	Transcript(audioPath string) (string, error)
	TranscriptDetailed(audioPath string) (*TranscriptResult, error)
}

func NewTranscripter() Transcripter {
	if config.GroqApiKey != "" {
		log.Println("[SINGLETON] Using Groq Whisper for transcription")
		return NewGroqWhisper()
	}
	log.Println("[SINGLETON] Using local Whisper for transcription")
	return NewWhisper()
}
