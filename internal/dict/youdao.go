package dict

import (
	"fmt"
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
	return fmt.Sprintf("%s/result?word=%s&lang=en", s.BaseURL, word)
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

	body.Find(".phone_con .per-phone .phonetic").Each(func(i int, s *goquery.Selection) {
		if i >= 2 {
			return
		}
		text := strings.Trim(s.Text(), "/ ")
		if text == "" {
			return
		}
		if i == 0 {
			result.Pronunciation.Uk = text
		} else {
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

	body.Find(".trans-container .mcols-layout .col2").Each(func(_ int, s *goquery.Selection) {
		ex := Example{}
		if en := s.Find(".sen-eng").First(); en.Length() > 0 {
			ex.En = strings.TrimSpace(en.Text())
		}
		if zh := s.Find(".sen-ch").First(); zh.Length() > 0 {
			ex.Zh = strings.TrimSpace(zh.Text())
		}
		if ex.En != "" || ex.Zh != "" {
			result.Examples = append(result.Examples, ex)
		}
	})

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

	body.Find(".trans-container .mcols-layout .col2").Each(func(_ int, s *goquery.Selection) {
		ex := Example{}
		if en := s.Find(".sen-eng").First(); en.Length() > 0 {
			ex.En = strings.TrimSpace(en.Text())
		}
		if zh := s.Find(".sen-ch").First(); zh.Length() > 0 {
			ex.Zh = strings.TrimSpace(zh.Text())
		}
		if ex.En != "" || ex.Zh != "" {
			result.Examples = append(result.Examples, ex)
		}
	})

	if len(result.Examples) == 0 && len(result.Meanings) == 0 {
		return nil, &NoTranslationResults{word: inputText}
	}

	return &TranslationData{Type: TypeToEnglish, ToEnglish: result}, nil
}
