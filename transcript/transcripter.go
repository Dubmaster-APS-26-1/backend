package transcript

type Transcripter interface {
	Transcript(audioPath string) (string, error)
}
