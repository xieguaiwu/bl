package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"bl/internal/dict"
)

const (
	ansiReset   = "\033[0m"
	ansiGreen   = "\033[32m"
	ansiMagenta = "\033[35m"
	ansiCyan    = "\033[36m"
	ansiYellow  = "\033[33m"
	ansiBrightBlack = "\033[90m"
)

func renderExamples(b *strings.Builder, examples []dict.Example, colored bool) {
	fmt.Fprintln(b, "# Examples")
	for _, ex := range examples {
		if colored {
			fmt.Fprintf(b, "* %s%s%s\n", ansiGreen, ex.En, ansiReset)
			fmt.Fprintf(b, "  %s%s%s\n", ansiMagenta, ex.Zh, ansiReset)
		} else {
			fmt.Fprintf(b, "* %s\n", ex.En)
			fmt.Fprintf(b, "  %s\n", ex.Zh)
		}
	}
	fmt.Fprintln(b)
}

func RenderChinese(result *dict.ToChinese, colored bool) string {
	var b strings.Builder

	if result.Pronunciation.Uk != "" || result.Pronunciation.Us != "" {
		fmt.Fprintln(&b, "# Pronunciation")
		if result.Pronunciation.Uk != "" {
			if colored {
				fmt.Fprintf(&b, "英：[%s%s%s]\n", ansiGreen, result.Pronunciation.Uk, ansiReset)
			} else {
				fmt.Fprintf(&b, "英：[%s]\n", result.Pronunciation.Uk)
			}
		}
		if result.Pronunciation.Us != "" {
			if colored {
				fmt.Fprintf(&b, "美：[%s%s%s]\n", ansiGreen, result.Pronunciation.Us, ansiReset)
			} else {
				fmt.Fprintf(&b, "美：[%s]\n", result.Pronunciation.Us)
			}
		}
		fmt.Fprintln(&b)
	}

	if len(result.Meanings) > 0 {
		fmt.Fprintln(&b, "# Meanings")
		for _, m := range result.Meanings {
			if m.PartOfSpeech != "" {
				fmt.Fprintf(&b, "[%s]\n", m.PartOfSpeech)
			}
			for _, def := range m.Definitions {
				if colored {
					fmt.Fprintf(&b, "* %s%s%s\n", ansiGreen, def, ansiReset)
				} else {
					fmt.Fprintf(&b, "* %s\n", def)
				}
			}
			fmt.Fprintln(&b)
		}
	}

	if len(result.Examples) > 0 {
		renderExamples(&b, result.Examples, colored)
	}

	return strings.TrimRight(b.String(), "\n")
}

func RenderEnglish(result *dict.ToEnglish, colored bool) string {
	var b strings.Builder

	if len(result.Meanings) > 0 {
		fmt.Fprintln(&b, "# Meanings")
		for _, m := range result.Meanings {
			if colored {
				fmt.Fprintf(&b, "* %s%s%s\n", ansiGreen, m, ansiReset)
			} else {
				fmt.Fprintf(&b, "* %s\n", m)
			}
		}
		fmt.Fprintln(&b)
	}

	if len(result.Examples) > 0 {
		renderExamples(&b, result.Examples, colored)
	}

	return strings.TrimRight(b.String(), "\n")
}

func RenderGermanEntry(entry *dict.GermanEntry, colored bool) string {
	var b strings.Builder

	if colored {
		fmt.Fprintf(&b, "%s# German Dictionary%s\n\n", ansiBrightBlack, ansiReset)
	} else {
		fmt.Fprintf(&b, "# German Dictionary\n\n")
	}

	if colored {
		if entry.Article != "" {
			fmt.Fprintf(&b, "%s%s%s %s%s%s", ansiCyan, entry.Article, ansiReset, ansiGreen, entry.Word, ansiReset)
		} else {
			fmt.Fprintf(&b, "%s%s%s", ansiGreen, entry.Word, ansiReset)
		}
		if entry.CefrLevel != "" {
			fmt.Fprintf(&b, "  [%s%s%s]", ansiYellow, entry.CefrLevel, ansiReset)
		}
	} else {
		if entry.Article != "" {
			fmt.Fprintf(&b, "%s %s", entry.Article, entry.Word)
		} else {
			fmt.Fprint(&b, entry.Word)
		}
		if entry.CefrLevel != "" {
			fmt.Fprintf(&b, "  [%s]", entry.CefrLevel)
		}
	}
	fmt.Fprintln(&b)

	if entry.WordType != "" {
		if colored {
			fmt.Fprintf(&b, "%s# Type%s %s%s%s", ansiBrightBlack, ansiReset, ansiCyan, entry.WordType, ansiReset)
			if entry.Gender != "" {
				fmt.Fprintf(&b, " | %s%s%s", ansiYellow, entry.Gender, ansiReset)
			}
		} else {
			fmt.Fprintf(&b, "# Type %s", entry.WordType)
			if entry.Gender != "" {
				fmt.Fprintf(&b, " (%s)", entry.Gender)
			}
		}
		fmt.Fprintln(&b)
	}

	if entry.Phonetic != "" {
		if colored {
			fmt.Fprintf(&b, "%s# IPA%s  /%s%s%s/\n", ansiBrightBlack, ansiReset, ansiGreen, entry.Phonetic, ansiReset)
		} else {
			fmt.Fprintf(&b, "# IPA /%s/\n", entry.Phonetic)
		}
	}

	if len(entry.Definitions) > 0 {
		fmt.Fprintln(&b, "# Translations")
		for _, def := range entry.Definitions {
			if colored {
				fmt.Fprintf(&b, "* %s%s%s\n", ansiGreen, def, ansiReset)
			} else {
				fmt.Fprintf(&b, "* %s\n", def)
			}
		}
	}

	if len(entry.Examples) > 0 {
		fmt.Fprintln(&b, "# Examples")
		for _, ex := range entry.Examples {
			if colored {
				fmt.Fprintf(&b, "* %s%s%s\n", ansiGreen, ex[0], ansiReset)
				fmt.Fprintf(&b, "  %s%s%s\n", ansiMagenta, ex[1], ansiReset)
			} else {
				fmt.Fprintf(&b, "* %s\n", ex[0])
				fmt.Fprintf(&b, "  %s\n", ex[1])
			}
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func RenderOneliner(data *dict.TranslationData) string {
	switch data.Type {
	case dict.TypeToChinese:
		if data.ToChinese != nil {
			return renderChineseOneliner(data.ToChinese)
		}
	case dict.TypeToEnglish:
		if data.ToEnglish != nil {
			return renderEnglishOneliner(data.ToEnglish)
		}
	case dict.TypeGerman:
		if data.German != nil {
			return renderGermanOneliner(data.German)
		}
	}
	return ""
}

func renderChineseOneliner(r *dict.ToChinese) string {
	var parts []string
	for _, m := range r.Meanings {
		prefix := ""
		if m.PartOfSpeech != "" {
			prefix = "[" + m.PartOfSpeech + "] "
		}
		parts = append(parts, prefix+strings.Join(m.Definitions, ", "))
	}
	return strings.Join(parts, "; ")
}

func renderEnglishOneliner(r *dict.ToEnglish) string {
	return strings.Join(r.Meanings, "; ")
}

func renderGermanOneliner(e *dict.GermanEntry) string {
	var tag string
	if e.WordType != "" && e.Gender != "" {
		tag = fmt.Sprintf(" [%s/%s]", e.WordType, e.Gender)
	} else if e.WordType != "" {
		tag = fmt.Sprintf(" [%s]", e.WordType)
	} else if e.Gender != "" {
		tag = fmt.Sprintf(" [%s]", e.Gender)
	}
	defs := strings.Join(e.Definitions, ", ")
	return e.Word + tag + ": " + defs
}

func RenderTranslation(data *dict.TranslationData, fmt_ dict.Format, colored bool) string {
	switch fmt_ {
	case dict.FormatJSON:
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Sprintf("json error: %v", err)
		}
		return string(b)
	case dict.FormatOneliner:
		return RenderOneliner(data)
	default:
		switch data.Type {
		case dict.TypeToChinese:
			if data.ToChinese != nil {
				return RenderChinese(data.ToChinese, colored)
			}
		case dict.TypeToEnglish:
			if data.ToEnglish != nil {
				return RenderEnglish(data.ToEnglish, colored)
			}
		case dict.TypeGerman:
			if data.German != nil {
				return RenderGermanEntry(data.German, colored)
			}
		}
		return ""
	}
}
