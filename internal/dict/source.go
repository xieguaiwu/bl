package dict

type DictionarySource interface {
	Name() string
	FetchURL(word string) string
	Parse(word string, html string) (*TranslationData, error)
}
