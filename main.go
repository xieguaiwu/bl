package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"bl/internal/dict"
	"bl/internal/render"
)

type config struct {
	text     string
	json     bool
	oneliner bool
	noCache  bool
	source   string
	german   bool
}

func parseFlags() config {
	var cfg config
	flag.BoolVar(&cfg.json, "json", false, "output using JSON")
	flag.BoolVar(&cfg.json, "j", false, "shorthand for --json")
	flag.BoolVar(&cfg.oneliner, "oneliner", false, "single-line compact output")
	flag.BoolVar(&cfg.oneliner, "o", false, "shorthand for --oneliner")
	flag.BoolVar(&cfg.noCache, "no-cache", false, "disable translation cache")
	flag.StringVar(&cfg.source, "source", "youdao", "dictionary source (youdao, woerter-net)")
	flag.StringVar(&cfg.source, "s", "youdao", "shorthand for --source")
	flag.BoolVar(&cfg.german, "german", false, "use German dictionary (woerter-net)")
	flag.BoolVar(&cfg.german, "g", false, "shorthand for --german")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: bl [flags] <text>\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  bl hello                  Youdao EN<->ZH (default)\n")
		fmt.Fprintf(os.Stderr, "  bl -g Haus                German dictionary\n")
		fmt.Fprintf(os.Stderr, "  bl -j hello               JSON output\n")
		fmt.Fprintf(os.Stderr, "  bl -o hello               single-line output\n")
		fmt.Fprintf(os.Stderr, "  bl -g -j Haus             German + JSON\n")
	}
	flag.Parse()

	if cfg.german {
		cfg.source = "woerter-net"
	}

	cfg.text = strings.Join(flag.Args(), " ")
	return cfg
}

func outputFmt(cfg config) dict.Format {
	if cfg.json {
		return dict.FormatJSON
	}
	if cfg.oneliner {
		return dict.FormatOneliner
	}
	return dict.FormatMarkdown
}

func main() {
	cfg := parseFlags()

	source := dict.NewSourceByName(cfg.source)
	if source == nil {
		fmt.Fprintf(os.Stderr, "unknown source: %s (use youdao or woerter-net)\n", cfg.source)
		os.Exit(1)
	}

	dbPath := cachePath(cfg.noCache)

	client, err := dict.NewRdict(source, dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	outfmt := outputFmt(cfg)

	if cfg.text != "" {
		output(client, cfg.text, outfmt)
		return
	}

	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			text := strings.TrimSpace(string(input))
			if text != "" {
				output(client, text, outfmt)
				return
			}
		}
	}

	interactiveMode(client, outfmt)
}



func cachePath(noCache bool) string {
	if noCache {
		return ""
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cacheDir, "bl", "cache.db")
}

func output(client *dict.Rdict, text string, fmt_ dict.Format) {
	result, err := client.GetResults(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}

	if fmt_ == dict.FormatJSON {
		out, err := json.MarshalIndent(result.Data, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		fmt.Println(string(out))
		return
	}

	if fmt_ == dict.FormatOneliner {
		fmt.Println(render.RenderOneliner(&result.Data))
		return
	}

	colored := isColoredOutput()
	rendered := render.RenderTranslation(&result.Data, dict.FormatMarkdown, colored)

	indented := indent(rendered, "  ")
	if result.IsCached {
		if colored {
			indented += fmt.Sprintf("\n\n  \033[90m[ %s ] From cache\033[0m", text)
		} else {
			indented += fmt.Sprintf("\n\n  [ %s ] From cache", text)
		}
	}

	fmt.Printf("\n%s\n", indented)
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func isColoredOutput() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}

func interactiveMode(client *dict.Rdict, fmt_ dict.Format) {
	colored := isColoredOutput()
	prompt := "[bl]# "
	if colored {
		prompt = "\033[32m[bl]\033[0m# "
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(prompt)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print(prompt)
			continue
		}
		output(client, line, fmt_)
		fmt.Print(prompt)
	}
}


