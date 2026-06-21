package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const beolingusURL = "https://ftp.tu-chemnitz.de/pub/Local/urz/ding/de-en-devel/de-en.txt.gz"

type deEnEntry struct {
	Word        string   `json:"word"`
	Type        string   `json:"type"`
	Definitions []string `json:"definitions"`
	Gender      string   `json:"gender,omitempty"`
	Article     string   `json:"article,omitempty"`
	WordType    string   `json:"word_type,omitempty"`
}

func main() {
	output := flag.String("output", "", "Output JSONL path (required)")
	dictURL := flag.String("url", beolingusURL, "Beolingus de-en.txt.gz URL")
	flag.Parse()

	if *output == "" {
		fmt.Fprintln(os.Stderr, "Usage: go run scripts/build_de_en/ -output data.jsonl")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "create output dir: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Downloading Beolingus dictionary...")
	body, err := downloadDict(*dictURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "download failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Download complete, parsing...")

	entries, err := parseBeolingus(body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Parsed %d entries\n", len(entries))

	out, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	for _, entry := range entries {
		if err := enc.Encode(entry); err != nil {
			fmt.Fprintf(os.Stderr, "write entry %q: %v\n", entry.Word, err)
		}
	}
	fmt.Printf("Wrote %d entries to %s\n", len(entries), *output)
}

func downloadDict(url string) ([]byte, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var reader io.Reader
	if strings.HasSuffix(url, ".gz") {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer gr.Close()
		reader = gr
	} else {
		reader = resp.Body
	}

	return io.ReadAll(reader)
}

// Beolingus metadata format: {m}, {f}, {n}, {adj}, {v}, {pl}, etc.
var metaRe = regexp.MustCompile(`\{([^}]+)\}`)

var posMap = map[string]string{
	"adj": "adjective",
	"v":   "verb",
	"v.":  "verb",
	"vt":  "verb",
	"vi":  "verb",
}

var genderMap = map[string]string{
	"m": "masculine",
	"f": "feminine",
	"n": "neuter",
}

var articleMap = map[string]string{
	"masculine": "der",
	"feminine":  "die",
	"neuter":    "das",
}

func parseBeolingus(data []byte) ([]*deEnEntry, error) {
	entriesByWord := make(map[string]*deEnEntry)
	var order []string

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 2*1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		entry := parseLine(line)
		if entry == nil {
			continue
		}

		if existing, ok := entriesByWord[entry.Word]; ok {
			// Merge definitions from duplicate lines, capping at 20.
			seenDef := make(map[string]bool, len(existing.Definitions))
			for _, d := range existing.Definitions {
				seenDef[strings.ToLower(d)] = true
			}
			for _, d := range entry.Definitions {
				if !seenDef[strings.ToLower(d)] && len(existing.Definitions) < 20 {
					existing.Definitions = append(existing.Definitions, d)
					seenDef[strings.ToLower(d)] = true
				}
			}
			continue
		}

		entriesByWord[entry.Word] = entry
		order = append(order, entry.Word)
	}

	entries := make([]*deEnEntry, 0, len(order))
	for _, w := range order {
		entries = append(entries, entriesByWord[w])
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan at line %d: %w", lineNum, err)
	}

	return entries, nil
}

func parseLine(line string) *deEnEntry {
	parts := strings.SplitN(line, "::", 2)
	if len(parts) != 2 {
		return nil
	}

	germanSide := strings.TrimSpace(parts[0])
	englishSide := strings.TrimSpace(parts[1])

	if germanSide == "" || englishSide == "" {
		return nil
	}

	word := extractWord(germanSide)
	if word == "" || len([]rune(word)) < 2 {
		return nil
	}

	if strings.Contains(word, " ") {
		return nil
	}

	entry := &deEnEntry{
		Word: word,
		Type: "german",
	}

	entry.Gender, entry.Article = extractGender(germanSide)
	entry.WordType = extractWordType(germanSide, word)
	entry.Definitions = extractDefinitions(englishSide)

	if len(entry.Definitions) == 0 {
		return nil
	}

	return entry
}

func extractWord(germanSide string) string {
	primary := germanSide
	if idx := strings.Index(germanSide, "|"); idx > 0 {
		primary = strings.TrimSpace(germanSide[:idx])
	}

	cleaned := metaRe.ReplaceAllString(primary, " ")
	cleaned = regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(cleaned, " ")
	cleaned = strings.ReplaceAll(cleaned, "&nbsp;", " ")
	cleaned = strings.ReplaceAll(cleaned, "&amp;", "&")

	fields := strings.Fields(cleaned)
	if len(fields) == 0 {
		return ""
	}

	return strings.TrimRight(fields[0], ";\u00a0")
}

func extractGender(germanSide string) (string, string) {
	matches := metaRe.FindAllStringSubmatch(germanSide, -1)
	for _, m := range matches {
		content := strings.TrimSpace(m[1])
		base := strings.TrimRight(content, "0123456789., ")
		if gender, ok := genderMap[base]; ok {
			return gender, articleMap[gender]
		}
	}
	return "", ""
}

func extractWordType(germanSide, word string) string {
	matches := metaRe.FindAllStringSubmatch(germanSide, -1)
	for _, m := range matches {
		content := strings.TrimSpace(m[1])
		if pos, ok := posMap[content]; ok {
			return pos
		}
	}
	return guessWordType(word)
}

func extractDefinitions(englishSide string) []string {
	var defs []string
	groups := strings.Split(englishSide, "|")
	for _, group := range groups {
		group = strings.TrimSpace(group)
		group = metaRe.ReplaceAllString(group, " ")
		group = regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(group, " ")
		group = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(group, " ")
		group = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(group, " ")
		group = strings.Join(strings.Fields(group), " ")
		if group == "" {
			continue
		}
		items := strings.Split(group, ";")
		for _, item := range items {
			item = strings.TrimSpace(item)
			item = strings.Trim(item, ".,;:!?")
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if len(item) < 2 || len(item) > 200 {
				continue
			}
			dup := false
			for _, d := range defs {
				if strings.EqualFold(d, item) {
					dup = true
					break
				}
			}
			if !dup {
				defs = append(defs, item)
			}
		}
	}
	if len(defs) > 20 {
		defs = defs[:20]
	}
	return defs
}

// Duplicates heuristic from internal/dict/german.go (unexported).
func guessWordType(word string) string {
	lower := strings.ToLower(word)
	switch {
	case strings.HasSuffix(lower, "ieren"),
		strings.HasSuffix(lower, "eln"),
		strings.HasSuffix(lower, "ern"):
		return "verb"
	case strings.HasSuffix(lower, "en") && !isUpper(word):
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

func isUpper(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}
