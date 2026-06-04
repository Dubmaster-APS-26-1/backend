package transcript

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
