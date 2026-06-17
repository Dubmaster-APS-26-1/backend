package input

type Downloader interface {
	Download(url, outputPath string) string
}