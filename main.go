package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bl/internal/config"
	"bl/internal/dict"
	"bl/internal/render"
)

var userAgent = "bl/" + "0.1.0"

type runConfig struct {
	text       string
	json       bool
	oneliner   bool
	noCache    bool
	source     string
	german     bool
	offline    bool
	online     bool
	updateDict bool
	dictStatus bool
	genConfig  bool
	modeFlag   string // "auto", "offline", "online" — sets config and exits
}

func parseFlags() runConfig {
	var cfg runConfig
	flag.BoolVar(&cfg.json, "json", false, "output using JSON")
	flag.BoolVar(&cfg.json, "j", false, "shorthand for --json")
	flag.BoolVar(&cfg.oneliner, "oneliner", false, "single-line compact output")
	flag.BoolVar(&cfg.oneliner, "o", false, "shorthand for --oneliner")
	flag.BoolVar(&cfg.noCache, "no-cache", false, "disable translation cache")
	flag.StringVar(&cfg.source, "source", "youdao", "dictionary source (youdao, woerter-net)")
	flag.StringVar(&cfg.source, "s", "youdao", "shorthand for --source")
	flag.BoolVar(&cfg.german, "german", false, "use German dictionary (woerter-net)")
	flag.BoolVar(&cfg.german, "g", false, "shorthand for --german")
	flag.BoolVar(&cfg.offline, "offline", false, "offline mode: only use local dictionaries, no network")
	flag.BoolVar(&cfg.online, "online", false, "online mode: skip offline dictionary, fetch from network")
	flag.BoolVar(&cfg.updateDict, "update-dict", false, "download and install offline dictionaries")
	flag.BoolVar(&cfg.dictStatus, "dict-status", false, "show offline dictionary status")
	flag.BoolVar(&cfg.genConfig, "generate-config", false, "generate default config file")
	flag.StringVar(&cfg.modeFlag, "mode", "", "set default mode (auto/offline/online) and save to config")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: bl [flags] <text>\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nConfig:\n")
		fmt.Fprintf(os.Stderr, "  Config file: ~/.config/bl/config.json\n")
		fmt.Fprintf(os.Stderr, "  Use --mode offline  to permanently switch to offline mode\n")
		fmt.Fprintf(os.Stderr, "  Use --mode online   to permanently switch to online mode\n")
		fmt.Fprintf(os.Stderr, "  Use --mode auto     to restore default (try offline, then online)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  bl hello                    Youdao EN<->ZH (default)\n")
		fmt.Fprintf(os.Stderr, "  bl -g Haus                  German dictionary\n")
		fmt.Fprintf(os.Stderr, "  bl -j hello                 JSON output\n")
		fmt.Fprintf(os.Stderr, "  bl -o hello                 single-line output\n")
		fmt.Fprintf(os.Stderr, "  bl -g -j Haus               German + JSON\n")
		fmt.Fprintf(os.Stderr, "  bl --offline hello          offline mode only\n")
		fmt.Fprintf(os.Stderr, "  bl --online hello           skip offline dict\n")
		fmt.Fprintf(os.Stderr, "  bl --mode offline           set offline as default\n")
		fmt.Fprintf(os.Stderr, "  bl --update-dict            download offline dictionaries\n")
		fmt.Fprintf(os.Stderr, "  bl --dict-status            show offline dictionary status\n")
	}
	flag.Parse()

	if cfg.german {
		cfg.source = "woerter-net"
	}

	cfg.text = strings.Join(flag.Args(), " ")
	return cfg
}

func outputFmt(cfg runConfig) dict.Format {
	if cfg.json {
		return dict.FormatJSON
	}
	if cfg.oneliner {
		return dict.FormatOneliner
	}
	return dict.FormatMarkdown
}

func main() {
	rc := parseFlags()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: config load: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// --generate-config: create default config file
	if rc.genConfig {
		created, err := config.GenerateConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if created {
			path, _ := config.ConfigPath()
			fmt.Printf("Created default config at %s\n", path)
		}
		return
	}

	// --mode: set default mode and save to config
	if rc.modeFlag != "" {
		if !config.IsValidMode(rc.modeFlag) {
			fmt.Fprintf(os.Stderr, "error: invalid mode %q (use: auto, offline, online)\n", rc.modeFlag)
			os.Exit(1)
		}
		cfg.Mode = config.Mode(rc.modeFlag)
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "error: save config: %v\n", err)
			os.Exit(1)
		}
		path, _ := config.ConfigPath()
		fmt.Printf("Set default mode to %q in %s\n", cfg.Mode, path)
		return
	}

	// --update-dict: download offline dictionaries
	if rc.updateDict {
		updateDictCmd()
		return
	}

	// --dict-status: show offline dictionary info
	if rc.dictStatus {
		dictStatusCmd()
		return
	}

	// Determine effective mode: CLI flag > env var > config file > default.
	// "auto" tries offline first, falls back to online.
	// "offline" uses only offline dictionary.
	// "online" skips offline dictionary entirely.
	mode := config.ModeAuto
	switch {
	case rc.offline:
		mode = config.ModeOffline
	case rc.online:
		mode = config.ModeOnline
	default:
		envMode := os.Getenv("BL_MODE")
		if envMode != "" && config.IsValidMode(envMode) {
			mode = config.Mode(envMode)
		} else {
			mode = cfg.Mode
		}
	}

	source := dict.NewSourceByName(rc.source)
	if source == nil {
		fmt.Fprintf(os.Stderr, "unknown source: %s (use youdao or woerter-net)\n", rc.source)
		os.Exit(1)
	}

	dbPath := cachePath(rc.noCache)

	var offlineDict *dict.OfflineDictionary
	onlyOffline := false
	switch mode {
	case config.ModeOffline:
		onlyOffline = true
		fallthrough
	case config.ModeAuto:
		// Open offline dict for both offline and auto modes.
		// In auto mode, onlyOffline=false so GetResults falls through to cache→online on miss.
		od, err := openOfflineDict(source.Name(), rc.text)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		offlineDict = od
	case config.ModeOnline:
		// No offline dictionary.
	}

	client, err := dict.NewRdictWithOffline(source, dbPath, offlineDict, onlyOffline)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	outfmt := outputFmt(rc)

	if rc.text != "" {
		output(client, rc.text, outfmt)
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

	interactiveMode(client, outfmt, source.Name())
}

func openOfflineDict(sourceName, text string) (*dict.OfflineDictionary, error) {
	dir, err := dict.DictDir()
	if err != nil {
		return nil, err
	}
	lang := dict.LangForSource(sourceName, text)
	od, err := dict.NewOfflineDict(dir, lang)
	if err != nil {
		return nil, fmt.Errorf("open offline %s dict: %w", lang, err)
	}
	if od == nil {
		return nil, fmt.Errorf("offline dictionary not found for %s\n  run 'bl --update-dict' to download, or build one with 'go run scripts/build_dict/'", lang)
	}
	return od, nil
}

func updateDictCmd() {
	baseURL := os.Getenv("BL_DICT_URL")
	if baseURL == "" {
		fmt.Fprintf(os.Stderr, `error: BL_DICT_URL environment variable not set

Set BL_DICT_URL to the base URL where dictionary files are hosted.
The URL pattern is: {BL_DICT_URL}/{lang}.db  (lang: de-en, en-zh, zh-en)

Example:
  export BL_DICT_URL=https://github.com/yourname/bl-dicts/releases/download/v1

Or build your own dictionaries from word lists:
  go run scripts/build_dict/ -lang de-en -input words.jsonl -output ~/.config/bl/dict/de-en.db
`)
		os.Exit(1)
	}

	dir, err := dict.EnsureDictDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	pairs := []string{"de-en", "en-zh", "zh-en"}
	for _, lang := range pairs {
		url := baseURL + "/" + lang + ".db"
		dest := filepath.Join(dir, lang+".db")
		fmt.Printf("Downloading %s to %s ...\n", lang, dest)
		if err := downloadFile(dest, url); err != nil {
			fmt.Fprintf(os.Stderr, "  failed: %v\n", err)
			continue
		}
		fi, _ := os.Stat(dest)
		fmt.Printf("  done (%d bytes)\n", fi.Size())
	}
	fmt.Println("Update complete.")
}

func dictStatusCmd() {
	dir, err := dict.DictDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	pairs := []string{"de-en", "en-zh", "zh-en"}
	anyFound := false
	for _, lang := range pairs {
		od, err := dict.NewOfflineDict(dir, lang)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: error opening: %v\n", lang, err)
			continue
		}
		if od == nil {
			fmt.Printf("  %s: not installed  (run 'bl --update-dict' to download)\n", lang)
			continue
		}
		entries, size, err := od.Stats()
		od.Close()
		if err != nil {
			fmt.Printf("  %s:  stats error: %v\n", lang, err)
		} else {
			fmt.Printf("  %s:  (%d entries, %d bytes)\n", lang, entries, size)
		}
		anyFound = true
	}
	if !anyFound {
		fmt.Println("No offline dictionaries installed.")
		fmt.Println("Run 'bl --update-dict' to download, or build one with 'go run scripts/build_dict/'")
	}
	// Show current config mode
	cfg, _ := config.Load()
	path, _ := config.ConfigPath()
	fmt.Printf("\nDefault mode: %s\n", cfg.Mode)
	fmt.Printf("Config file: %s\n", path)
	fmt.Printf("Env override: BL_MODE=%s\n", os.Getenv("BL_MODE"))
}

func downloadFile(dest, url string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	dlClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := dlClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(dest)
		return fmt.Errorf("write file: %w", err)
	}
	if written == 0 {
		os.Remove(dest)
		return fmt.Errorf("downloaded empty file")
	}
	return nil
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

func interactiveMode(client *dict.Rdict, fmt_ dict.Format, sourceName string) {
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
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
}
