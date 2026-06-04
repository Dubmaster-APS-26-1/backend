package video

import "time"

type Video struct {
	ID                      string    `json:"id"`
	UserSsoId               string    `json:"userSsoId"`
	Title                   string    `json:"title"`
	SourceType              string    `json:"sourceType,omitempty"`
	SourceUrl               string    `json:"sourceUrl,omitempty"`
	OriginalPath            string    `json:"-"`
	DubbedPath              string    `json:"hasDubbed,omitempty"`
	Status                  string    `json:"status"`
	Transcription           string    `json:"transcription,omitempty"`
	TranslatedTranscription string    `json:"translatedTranscription,omitempty"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
}
