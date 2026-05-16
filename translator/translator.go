package translator

type Translator interface {
	Translate(text string) (string, error)
	TranslateBatch(texts []string) ([]string, error)
}
