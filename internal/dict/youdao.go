package dict

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type YoudaoSource struct {
	BaseURL string
}

func NewYoudaoSource(baseURL string) *YoudaoSource {
	return &YoudaoSource{BaseURL: baseURL}
}

func (s *YoudaoSource) Name() string { return "youdao" }

func (s *YoudaoSource) FetchURL(word string) string {
	return fmt.Sprintf("%s/result?word=%s&lang=en", s.BaseURL, url.QueryEscape(word))
}

func (s *YoudaoSource) Parse(word string, html string) (*TranslationData, error) {
	if IsCJK(word) {
		return parseToEnglish(word, html)
	}
	return parseToChinese(word, html)
}

func parseToChinese(inputText string, html string) (*TranslationData, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	body := doc.Find(".search_result-dict")
	if body.Length() == 0 {
		return nil, fmt.Errorf("no .search_result-dict found")
	}

	result := &ToChinese{InputText: inputText}

	body.Find(".phone_con .per-phone").Each(func(_ int, container *goquery.Selection) {
		phonetic := container.Find(".phonetic")
		label := container.Find(".label")
		if phonetic.Length() == 0 || label.Length() == 0 {
			return
		}
		text := strings.Trim(phonetic.Text(), "/ ")
		if text == "" {
			return
		}
		labelText := strings.TrimSpace(label.Text())
		if strings.Contains(labelText, "英") {
			result.Pronunciation.Uk = text
		} else if strings.Contains(labelText, "美") {
			result.Pronunciation.Us = text
		}
	})

	body.Find(".trans-container .basic .word-exp").Each(func(_ int, s *goquery.Selection) {
		m := Meaning{}

		if pos := s.Find(".pos").First(); pos.Length() > 0 {
			m.PartOfSpeech = strings.TrimSpace(pos.Text())
		}

		if trans := s.Find(".trans").First(); trans.Length() > 0 {
			raw := strings.TrimSpace(trans.Text())
			for _, def := range strings.Split(raw, "\uff1b") {
				def = strings.TrimSpace(def)
				if def != "" {
					m.Definitions = append(m.Definitions, def)
				}
			}
		}

		if m.PartOfSpeech != "" || len(m.Definitions) > 0 {
			result.Meanings = append(result.Meanings, m)
		}
	})

	result.Examples = parseExamples(body)

	if len(result.Examples) == 0 && len(result.Meanings) == 0 &&
		result.Pronunciation.Uk == "" && result.Pronunciation.Us == "" {
		return nil, &NoTranslationResults{word: inputText}
	}

	return &TranslationData{Type: TypeToChinese, ToChinese: result}, nil
}

func parseToEnglish(inputText string, html string) (*TranslationData, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	body := doc.Find(".search_result-dict")
	if body.Length() == 0 {
		return nil, fmt.Errorf("no .search_result-dict found")
	}

	result := &ToEnglish{InputText: inputText}

	body.Find(".trans-container .basic .col2 .point").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			result.Meanings = append(result.Meanings, text)
		}
	})

	result.Examples = parseExamples(body)

	if len(result.Examples) == 0 && len(result.Meanings) == 0 {
		return nil, &NoTranslationResults{word: inputText}
	}

	return &TranslationData{Type: TypeToEnglish, ToEnglish: result}, nil
}

func parseExamples(body *goquery.Selection) []Example {
	var examples []Example
	body.Find(".trans-container .mcols-layout .col2").Each(func(_ int, s *goquery.Selection) {
		ex := Example{}
		if en := s.Find(".sen-eng").First(); en.Length() > 0 {
			ex.En = strings.TrimSpace(en.Text())
		}
		if zh := s.Find(".sen-ch").First(); zh.Length() > 0 {
			ex.Zh = strings.TrimSpace(zh.Text())
		}
		if ex.En != "" || ex.Zh != "" {
			examples = append(examples, ex)
		}
	})
	return examples
}
