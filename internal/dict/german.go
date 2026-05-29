package dict

import (
	"fmt"
	"net/url"
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
		strings.HasSuffix(lower, "reich"),
		strings.HasSuffix(lower, "iv"),
		strings.HasSuffix(lower, "al"),
		strings.HasSuffix(lower, "ell"),
		strings.HasSuffix(lower, "ant"),
		strings.HasSuffix(lower, "ent"),
		strings.HasSuffix(lower, "istisch"),
		strings.HasSuffix(lower, "abel"),
		strings.HasSuffix(lower, "ibel"):
		return "adjective"
	default:
		return "noun"
	}
}

func (s *WoerterNetSource) FetchURL(word string) string {
	path := guessWordType(word)
	word = url.PathEscape(word)
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

	parseEnglishTranslations(doc, entry)
	extractBodyMeta(doc, entry)

	if entry.WordType == "" {
		if title := doc.Find("title").First(); title.Length() > 0 {
			t := strings.ToLower(title.Text())
			switch {
			case strings.Contains(t, "conjugation"):
				entry.WordType = "verb"
			case strings.Contains(t, "declension of adjective"), strings.Contains(t, "declension adjective"):
				entry.WordType = "adjective"
			case strings.Contains(t, "declension"):
				entry.WordType = "noun"
			}
		}
	}

	doc.Find("[itemprop=\"mainEntity\"]").Each(func(_ int, faq *goquery.Selection) {
		answer := strings.TrimSpace(faq.Find("[itemprop=\"acceptedAnswer\"] p").First().Text())
		if answer == "" {
			return
		}
		aLower := strings.ToLower(answer)

		if entry.CefrLevel == "" {
			for _, level := range []string{"A1", "A2", "B1", "B2", "C1", "C2"} {
				if strings.Contains(aLower, strings.ToLower(level)) {
					entry.CefrLevel = level
					break
				}
			}
		}

		if entry.Gender == "" {
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
		}
	})

	bodyText := doc.Find("body").First().Text()
	runes := []rune(bodyText)
	if len(runes) > 3000 {
		runes = runes[:3000]
	}
	head := string(runes)
	if ipa := extractIPA(head); ipa != "" {
		entry.Phonetic = ipa
	}

	if len(entry.Definitions) == 0 {
		parseEnglishFromVStck(doc, entry)
	}

	exCount := 0
	doc.Find("ul.rLstGt li").Each(func(_ int, li *goquery.Selection) {
		if exCount >= 3 {
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
			exCount++
		}
	})

	if entry.WordType == "" && entry.CefrLevel == "" &&
		entry.Phonetic == "" && len(entry.Definitions) == 0 &&
		len(entry.Examples) == 0 {
		return nil, &NoTranslationResults{word: word}
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

func extractBodyMeta(doc *goquery.Document, entry *GermanEntry) {
	doc.Find("p.rInf").Each(func(_ int, p *goquery.Selection) {
		p.Find("span").Each(func(_ int, s *goquery.Selection) {
			title, exists := s.Attr("title")
			if !exists {
				return
			}
			title = strings.ToLower(title)
			text := strings.TrimSpace(s.Text())
			switch {
			case title == "noun" || title == "verb" || title == "adjective":
				if entry.WordType == "" {
					entry.WordType = text
				}
			case strings.HasPrefix(title, "gender"):
				if entry.Gender == "" {
					parts := strings.SplitN(title, " ", 2)
					if len(parts) == 2 {
						entry.Gender = parts[1]
					}
				}
			case strings.HasPrefix(title, "vocabulary certificate"):
				if entry.CefrLevel == "" {
					entry.CefrLevel = strings.ToUpper(text)
				}
			}
		})
	})
	if entry.Article == "" && entry.Gender != "" {
		switch entry.Gender {
		case "masculine":
			entry.Article = "der"
		case "feminine":
			entry.Article = "die"
		case "neutral", "neuter":
			entry.Article = "das"
		}
	}
}

func parseEnglishTranslations(doc *goquery.Document, entry *GermanEntry) {
	// Tier 1: comma-separated English in vStckKrz <span lang="en">
	if enSpan := doc.Find("#vStckKrz [lang=\"en\"]").First(); enSpan.Length() > 0 {
		text := enSpan.Text()
		text = strings.ReplaceAll(text, "\u00a0", " ")
		text = strings.TrimSpace(text)
		if text != "" {
			for _, t := range splitEnglishTranslations(text) {
				if t != "" {
					entry.Definitions = append(entry.Definitions, t)
				}
			}
		}
	}
	if len(entry.Definitions) == 0 {
		// Tier 2: individual English definitions in <dd lang="en">
		doc.Find("dd[lang=\"en\"]").Each(func(_ int, s *goquery.Selection) {
			text := s.Text()
			text = strings.ReplaceAll(text, "\u00a0", " ")
			text = strings.TrimSpace(text)
			if text != "" {
				for _, t := range splitEnglishTranslations(text) {
					if t != "" {
						entry.Definitions = append(entry.Definitions, t)
					}
				}
			}
		})
	}
	if len(entry.Definitions) == 0 {
		// Tier 3: any <span> with lang=en (broader, catches edge cases)
		doc.Find("span[lang=\"en\"]").Each(func(_ int, s *goquery.Selection) {
			text := s.Text()
			text = strings.ReplaceAll(text, "\u00a0", " ")
			text = strings.TrimSpace(text)
			if text != "" {
				for _, t := range splitEnglishTranslations(text) {
					if t != "" {
						entry.Definitions = append(entry.Definitions, t)
					}
				}
			}
		})
	}
}

func parseEnglishFromVStck(doc *goquery.Document, entry *GermanEntry) {
	vStck := doc.Find("#vStckKrz")
	if vStck.Length() == 0 {
		return
	}
	// Look for English text after known patterns: en.svg flag, "English" alt text
	vStck.Find("img[alt=\"English\" i]").Each(func(_ int, img *goquery.Selection) {
		if len(entry.Definitions) > 0 {
			return
		}
		parent := img.Parent()
		if parent.Length() == 0 {
			parent = img.ParentsFiltered("span").First()
		}
		if parent.Length() > 0 {
			text := strings.TrimSpace(parent.Text())
			text = strings.ReplaceAll(text, "\u00a0", " ")
			if text != "" {
				for _, t := range splitEnglishTranslations(text) {
					if t != "" {
						entry.Definitions = append(entry.Definitions, t)
					}
				}
			}
		}
	})
}

func splitEnglishTranslations(text string) []string {
	var result []string
	parts := strings.Split(text, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func fallbackGermanDefinitions(doc *goquery.Document, entry *GermanEntry) {
	doc.Find("i").Each(func(_ int, el *goquery.Selection) {
		if len(entry.Definitions) >= 8 {
			return
		}
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
		}
	})
}
