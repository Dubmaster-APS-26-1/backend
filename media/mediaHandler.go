package media

type MediaHandler interface {
	ExtractAudio(inputVideoPath string, outputAudioPath string) error
}
