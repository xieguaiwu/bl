package dict

import "strings"

type DictionarySource interface {
	Name() string
	FetchURL(word string) string
	Parse(word string, html string) (*TranslationData, error)
}

// NewSourceByName creates a DictionarySource from a name string.
func NewSourceByName(name string) DictionarySource {
	switch strings.ToLower(name) {
	case "woerter-net":
		return NewWoerterNetSource("https://www.verbformen.com")
	default:
		return NewYoudaoSource("https://m.youdao.com")
	}
}
