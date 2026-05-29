package dict

import (
	"encoding/json"
	"fmt"
)

type Pronunciation struct {
	Uk string `json:"uk,omitempty"`
	Us string `json:"us,omitempty"`
}

type Example struct {
	En string `json:"en"`
	Zh string `json:"zh"`
}

type Meaning struct {
	PartOfSpeech string   `json:"part_of_speech,omitempty"`
	Definitions  []string `json:"definitions"`
}

type ToChinese struct {
	InputText     string        `json:"input_text"`
	Pronunciation Pronunciation `json:"pronunciation"`
	Meanings      []Meaning     `json:"meanings"`
	Examples      []Example     `json:"examples"`
}

type ToEnglish struct {
	InputText string    `json:"input_text"`
	Meanings  []string  `json:"meanings"`
	Examples  []Example `json:"examples"`
}

// GermanEntry holds data for a German word lookup.
// Definitions contains English translations (not German definitions) when
// scraped from the primary source (verbformen.com English pages).
type GermanEntry struct {
	Word        string      `json:"word"`
	Gender      string      `json:"gender,omitempty"`
	Article     string      `json:"article,omitempty"`
	Phonetic    string      `json:"phonetic,omitempty"`
	CefrLevel   string      `json:"cefr_level,omitempty"`
	WordType    string      `json:"word_type,omitempty"`
	Definitions []string    `json:"definitions"`
	Examples    [][2]string `json:"examples"`
}

type TranslationType int

const (
	TypeToChinese TranslationType = iota
	TypeToEnglish
	TypeGerman
)

func (t TranslationType) String() string {
	switch t {
	case TypeToChinese:
		return "to_chinese"
	case TypeToEnglish:
		return "to_english"
	case TypeGerman:
		return "german"
	}
	return "unknown"
}

// TranslationData holds exactly one variant and matches the serde-tagged
// JSON format {"type":"to_chinese","data":{...}} used by the Rust original.
type TranslationData struct {
	Type      TranslationType
	ToChinese *ToChinese
	ToEnglish *ToEnglish
	German    *GermanEntry
}

func (d *TranslationData) MarshalJSON() ([]byte, error) {
	var data interface{}
	switch d.Type {
	case TypeToChinese:
		data = d.ToChinese
	case TypeToEnglish:
		data = d.ToEnglish
	case TypeGerman:
		data = d.German
	}
	return json.Marshal(map[string]interface{}{
		"type": d.Type.String(),
		"data": data,
	})
}

func (d *TranslationData) UnmarshalJSON(b []byte) error {
	var raw struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	switch raw.Type {
	case "to_chinese":
		d.Type = TypeToChinese
		d.ToChinese = new(ToChinese)
		return json.Unmarshal(raw.Data, d.ToChinese)
	case "to_english":
		d.Type = TypeToEnglish
		d.ToEnglish = new(ToEnglish)
		return json.Unmarshal(raw.Data, d.ToEnglish)
	case "german":
		d.Type = TypeGerman
		d.German = new(GermanEntry)
		return json.Unmarshal(raw.Data, d.German)
	}
	return fmt.Errorf("unknown translation type: %s", raw.Type)
}

type Format int

const (
	FormatMarkdown Format = iota
	FormatJSON
	FormatOneliner
)

type FetchedResult struct {
	Data     TranslationData
	IsCached bool
}

type NoTranslationResults struct{ word string }

func (e *NoTranslationResults) Error() string {
	return fmt.Sprintf("no translation results for %q", e.word)
}

type HttpError struct {
	Code   int
	Source string
	Word   string
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("HTTP %d from %s while fetching %q", e.Code, e.Source, e.Word)
}

// OfflineUnavailable is returned when --offline mode is active and the word
// is not found in the offline dictionary.
type OfflineUnavailable struct{ word string }

func (e *OfflineUnavailable) Error() string {
	return fmt.Sprintf("word %q not found in offline dictionary", e.word)
}

func IsCJK(text string) bool {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}
