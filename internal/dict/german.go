package dict

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type WoerterNetSource struct {
	BaseURL string
}

func NewWoerterNetSource(baseURL string) *WoerterNetSource {
	return &WoerterNetSource{BaseURL: baseURL}
}

func (s *WoerterNetSource) Name() string { return "woerter-net" }

func guessWordType(word string) string {
	lower := strings.ToLower(word)
	switch {
	case strings.HasSuffix(lower, "ieren"),
		strings.HasSuffix(lower, "eln"),
		strings.HasSuffix(lower, "ern"),
		strings.HasSuffix(lower, "en"):
		return "verb"
	case strings.HasSuffix(lower, "lich"),
		strings.HasSuffix(lower, "isch"),
		strings.HasSuffix(lower, "ig"),
		strings.HasSuffix(lower, "bar"),
		strings.HasSuffix(lower, "sam"),
		strings.HasSuffix(lower, "los"),
		strings.HasSuffix(lower, "haft"),
		strings.HasSuffix(lower, "arm"),
		strings.HasSuffix(lower, "frei"),
		strings.HasSuffix(lower, "reich"):
		return "adjective"
	default:
		return "noun"
	}
}

func (s *WoerterNetSource) FetchURL(word string) string {
	path := guessWordType(word)
	switch path {
	case "verb":
		path = fmt.Sprintf("conjugation/%s.htm", word)
	case "adjective":
		path = fmt.Sprintf("declension/adjectives/%s.htm", word)
	default:
		path = fmt.Sprintf("declension/nouns/%s.htm", word)
	}
	return fmt.Sprintf("%s/%s", s.BaseURL, path)
}

func (s *WoerterNetSource) Parse(word string, html string) (*TranslationData, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	entry := &GermanEntry{Word: word}

	if title := doc.Find("title").First(); title.Length() > 0 {
		t := strings.ToLower(title.Text())
		switch {
		case strings.Contains(t, "conjugation"):
			entry.WordType = "verb"
		case strings.Contains(t, "declension of adjective"), strings.Contains(t, "declension adjective"):
			entry.WordType = "adjective"
		case strings.Contains(t, "declension of noun"), strings.Contains(t, "declension noun"):
			entry.WordType = "noun"
		}
	}

	doc.Find("[itemprop=\"mainEntity\"]").Each(func(_ int, faq *goquery.Selection) {
		answer := strings.TrimSpace(faq.Find("[itemprop=\"acceptedAnswer\"] p").First().Text())
		if answer == "" {
			return
		}
		aLower := strings.ToLower(answer)

		for _, level := range []string{"A1", "A2", "B1", "B2", "C1", "C2"} {
			if strings.Contains(aLower, strings.ToLower(level)) {
				entry.CefrLevel = level
				break
			}
		}

		for _, candidate := range []struct {
			article string
			gender  string
		}{
			{"das", "neuter"},
			{"der", "masculine"},
			{"die", "feminine"},
		} {
			if strings.Contains(aLower, fmt.Sprintf("article is \"%s\"", candidate.article)) ||
				strings.Contains(aLower, fmt.Sprintf("\"%s\" is %s", word, candidate.gender)) {
				entry.Article = candidate.article
				entry.Gender = candidate.gender
				break
			}
		}
	})

	bodyText := doc.Find("body").First().Text()
	head := bodyText
	if len(head) > 3000 {
		head = head[:3000]
	}
	if ipa := extractIPA(head); ipa != "" {
		entry.Phonetic = ipa
	}

	doc.Find("i").Each(func(_ int, el *goquery.Selection) {
		text := strings.TrimSpace(el.Text())
		if len(text) > 10 && len(text) < 300 &&
			!strings.HasPrefix(text, "http") &&
			!strings.HasPrefix(text, "@") {
			clean := strings.TrimSpace(text)
			for _, d := range entry.Definitions {
				if d == clean {
					return
				}
			}
			entry.Definitions = append(entry.Definitions, clean)
			if len(entry.Definitions) >= 8 {
				return
			}
		}
	})

	doc.Find("ul.rLstGt li").Each(func(_ int, li *goquery.Selection) {
		if len(entry.Examples) >= 3 {
			return
		}
		allText := strings.TrimSpace(li.Text())
		english := ""
		if span := li.Find("span").Last(); span.Length() > 0 {
			english = strings.TrimSpace(span.Text())
		}
		german := allText
		if english != "" {
			german = strings.TrimSuffix(german, english)
			german = strings.TrimSpace(german)
		}
		if len(german) > 5 && len(english) > 5 {
			entry.Examples = append(entry.Examples, [2]string{german, english})
		}
	})

	if entry.WordType == "" && entry.CefrLevel == "" &&
		entry.Phonetic == "" && len(entry.Definitions) == 0 &&
		len(entry.Examples) == 0 {
		return nil, fmt.Errorf("no German dictionary data found for %q", word)
	}

	return &TranslationData{Type: TypeGerman, German: entry}, nil
}

func extractIPA(s string) string {
	inIpa := false
	var buf strings.Builder
	for _, ch := range s {
		if ch == '/' {
			if inIpa {
				candidate := buf.String()
				if len(candidate) >= 2 && len(candidate) <= 25 &&
					containsIPASymbols(candidate) {
					return candidate
				}
				buf.Reset()
			}
			inIpa = !inIpa
		} else if inIpa {
			buf.WriteRune(ch)
		}
	}
	return ""
}

func containsIPASymbols(s string) bool {
	for _, r := range s {
		switch r {
		case '\u02c8', '\u02cc', '\u02d0', '\u0283', '\u0292',
			'\u0294', '\u00e7', '\u00f8', '\u0153', '\u0289',
			'\u028a', '\u026a', '\u0259', '\u0250', '\u0281',
			'\u025b', '\u028c', '\u028d', '\u026f':
			return true
		}
	}
	return false
}
