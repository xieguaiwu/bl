package dict

import (
	"strings"
)

// DictionarySource is the interface for dictionary data sources.
// For HTML-scraping sources, implement FetchURL + Parse.
// For API-based sources (LLM), implement Name() and provide Translate() separately.
type DictionarySource interface {
	Name() string
	FetchURL(word string) string
	Parse(word string, html string) (*TranslationData, error)
}

// NewSourceByName creates a DictionarySource from a name string.
// For LLM translation, use NewLLMSource directly instead.
func NewSourceByName(name string) DictionarySource {
	switch strings.ToLower(name) {
	case "youdao":
		return NewYoudaoSource("https://m.youdao.com")
	case "woerter-net":
		return NewWoerterNetSource("https://www.verbformen.com")
	default:
		return nil
	}
}
