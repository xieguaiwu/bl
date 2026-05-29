package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"bl/internal/dict"
)

type inputEntry struct {
	Word        string      `json:"word"`
	Type        string      `json:"type"`
	Definitions []string    `json:"definitions"`
	Examples    [][2]string `json:"examples,omitempty"`
	// German-specific
	Gender    string `json:"gender,omitempty"`
	Article   string `json:"article,omitempty"`
	Phonetic  string `json:"phonetic,omitempty"`
	WordType  string `json:"word_type,omitempty"`
	CefrLevel string `json:"cefr_level,omitempty"`
	// EN-ZH
	Pronunciation struct {
		Uk string `json:"uk,omitempty"`
		Us string `json:"us,omitempty"`
	} `json:"pronunciation,omitempty"`
	PartOfSpeech string `json:"part_of_speech,omitempty"`
	// ZH-EN
	Meanings []string `json:"meanings,omitempty"`
}

func main() {
	inputFile := flag.String("input", "", "Path to JSONL input file (required)")
	lang := flag.String("lang", "", "Language pair: de-en, en-zh, zh-en (required)")
	output := flag.String("output", "", "Output path for .db file (required)")
	flag.Parse()

	if *inputFile == "" || *lang == "" || *output == "" {
		fmt.Fprintf(os.Stderr, "Usage: go run scripts/build_dict/main.go -input <file.jsonl> -lang <pair> -output <path.db>\n")
		fmt.Fprintf(os.Stderr, "  lang: de-en, en-zh, zh-en\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Validate lang
	switch *lang {
	case "de-en", "en-zh", "zh-en":
	default:
		fmt.Fprintf(os.Stderr, "invalid lang %q: must be de-en, en-zh, or zh-en\n", *lang)
		os.Exit(1)
	}

	f, err := os.Open(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open input: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	entries := make(map[string]*dict.TranslationData)
	var lineNum int
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var ie inputEntry
		if err := json.Unmarshal([]byte(line), &ie); err != nil {
			fmt.Fprintf(os.Stderr, "line %d: parse error: %v\n", lineNum, err)
			continue
		}
		if ie.Word == "" {
			fmt.Fprintf(os.Stderr, "line %d: missing 'word'\n", lineNum)
			continue
		}

		td := buildTranslationData(*lang, &ie)
		if td != nil {
			entries[ie.Word] = td
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Parsed %d entries from %s\n", len(entries), *inputFile)

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "no valid entries found, exiting")
		os.Exit(1)
	}

	if err := dict.CreateOfflineDict(*output, entries); err != nil {
		fmt.Fprintf(os.Stderr, "create dict: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created offline dictionary: %s (%d entries)\n", *output, len(entries))
}

func buildTranslationData(lang string, ie *inputEntry) *dict.TranslationData {
	switch lang {
	case "de-en":
		return &dict.TranslationData{
			Type: dict.TypeGerman,
			German: &dict.GermanEntry{
				Word:        ie.Word,
				Gender:      ie.Gender,
				Article:     ie.Article,
				Phonetic:    ie.Phonetic,
				CefrLevel:   ie.CefrLevel,
				WordType:    ie.WordType,
				Definitions: ie.Definitions,
				Examples:    ie.Examples,
			},
		}
	case "en-zh":
		meanings := make([]dict.Meaning, 0)
		if ie.PartOfSpeech != "" || len(ie.Definitions) > 0 {
			meanings = append(meanings, dict.Meaning{
				PartOfSpeech: ie.PartOfSpeech,
				Definitions:  ie.Definitions,
			})
		}
		examples := make([]dict.Example, 0)
		for _, ex := range ie.Examples {
			examples = append(examples, dict.Example{En: ex[0], Zh: ex[1]})
		}
		return &dict.TranslationData{
			Type: dict.TypeToChinese,
			ToChinese: &dict.ToChinese{
				InputText: ie.Word,
				Pronunciation: dict.Pronunciation{
					Uk: ie.Pronunciation.Uk,
					Us: ie.Pronunciation.Us,
				},
				Meanings: meanings,
				Examples: examples,
			},
		}
	case "zh-en":
		examples := make([]dict.Example, 0)
		for _, ex := range ie.Examples {
			examples = append(examples, dict.Example{En: ex[0], Zh: ex[1]})
		}
		return &dict.TranslationData{
			Type: dict.TypeToEnglish,
			ToEnglish: &dict.ToEnglish{
				InputText: ie.Word,
				Meanings:  ie.Definitions,
				Examples:  examples,
			},
		}
	}
	return nil
}
