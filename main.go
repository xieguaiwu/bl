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

var userAgent = "bl/" + "0.2.0"

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

	// LLM translation flags
	llm          bool
	llmProvider  string
	llmModel     string
	llmKey       string
	targetLang   string
	sourceLang   string
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
	// LLM flags
	flag.BoolVar(&cfg.llm, "llm", false, "use LLM-based translation (requires API key in config or env)")
	flag.StringVar(&cfg.llmProvider, "llm-provider", "", "LLM provider name (nemotron, bigpickle, opencode, custom)")
	flag.StringVar(&cfg.llmModel, "llm-model", "", "LLM model ID (overrides provider default)")
	flag.StringVar(&cfg.llmKey, "llm-key", "", "API key for LLM provider (overrides config/env)")
	flag.StringVar(&cfg.targetLang, "to-lang", "", "target language (e.g. 中文, English, 日本語)")
	flag.StringVar(&cfg.sourceLang, "from-lang", "", "source language (auto-detect if empty; specify for ambiguous words)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: bl [flags] <text>\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nConfig:\n")
		fmt.Fprintf(os.Stderr, "  Config file: ~/.config/bl/config.json\n")
		fmt.Fprintf(os.Stderr, "  Use --mode offline  to permanently switch to offline mode\n")
		fmt.Fprintf(os.Stderr, "  Use --mode online   to permanently switch to online mode\n")
		fmt.Fprintf(os.Stderr, "  Use --mode auto     to restore default (try offline, then online)\n")
		fmt.Fprintf(os.Stderr, "\nLLM Translation (grammar: gender/plural/comparative for inflected languages):\n")
		fmt.Fprintf(os.Stderr, "  bl --llm hello                          Default provider+model\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --llm-provider openrouter hello       Named provider\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --llm-model google/gemma-4-31b-it:free hello   Specific model\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --to-lang 日本語 hello                Translate to Japanese\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --to-lang English Haus                 German → English\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --from-lang French Raisonnement         Specify source lang for ambiguous words\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --from-lang German --to-lang English Handy     German 'Handy' = mobile phone\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --llm-key \"$API_KEY\" hello            Inline API key\n")
		fmt.Fprintf(os.Stderr, "\nLocal config (.blrc in current dir, overrides global config):\n")
		fmt.Fprintf(os.Stderr, "  echo '{\"provider\":\"openrouter\"}' > .blrc\n")
		fmt.Fprintf(os.Stderr, "  echo '{\"provider\":\"openrouter\",\"model\":\"...\",\"target_lang\":\"Français\"}' > .blrc\n")
		fmt.Fprintf(os.Stderr, "  echo '{\"base_url\":\"https://...\",\"model\":\"...\",\"api_key\":\"env:X\"}' > .blrc\n")
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
		fmt.Fprintf(os.Stderr, "  bl --dict-status            show offline + LLM config status\n")
		fmt.Fprintf(os.Stderr, "  bl --llm hello              LLM translation (default provider)\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --to-lang 日本語 konnichiwa   LLM to Japanese\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --to-lang English Haus           German noun → English\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --llm-model google/gemma-4-31b-it:free hello   Pick model\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --from-lang French Raisonnement          French 'reasoning'\n")
		fmt.Fprintf(os.Stderr, "  bl --llm --from-lang German Handy                German 'mobile phone'\n")
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
		} else {
			fmt.Println("Config file already exists, not overwriting.")
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
		dictStatusCmd(cfg)
		return
	}

	// Load local .blrc if present (project-specific LLM overrides)
	if lrc, err := loadLocalRC(); err == nil && lrc != nil {
		lrc.applyTo(cfg)
	}

	// Determine effective mode: CLI flag > env var > config file > default.
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

	// Determine source: LLM or traditional
	useLLM := rc.llm || cfg.LLM.Enabled

	// Resolve target and source languages (needed by both LLM and traditional paths)
	targetLang := rc.targetLang
	if targetLang == "" {
		targetLang = cfg.LLM.TargetLang
	}
	sourceLang := rc.sourceLang
	if sourceLang == "" {
		sourceLang = cfg.LLM.SourceLang
	}

	var source dict.DictionarySource

	if useLLM {
		// Resolve LLM provider
		providerName := rc.llmProvider
		if providerName == "" {
			providerName = cfg.LLM.Provider
		}
		providers := cfg.LLM.Providers
		provider := findProvider(providers, providerName)
		if provider == nil {
			fmt.Fprintf(os.Stderr, "error: LLM provider %q not found in config\n", providerName)
			fmt.Fprintf(os.Stderr, "  Available providers: ")
			for i, p := range providers {
				if i > 0 {
					fmt.Fprintf(os.Stderr, ", ")
				}
				fmt.Fprintf(os.Stderr, "%s", p.Name)
			}
			fmt.Fprintln(os.Stderr)
			os.Exit(1)
		}

		// Apply CLI overrides
		if rc.llmModel != "" {
			provider.Model = rc.llmModel
		}
		if rc.llmKey != "" {
			provider.APIKey = rc.llmKey
		}

		source = dict.NewLLMSource("llm", *provider, targetLang, sourceLang, cfg.LLM.SystemPrompt)
	} else {
		source = dict.NewSourceByName(rc.source)
		if source == nil {
			fmt.Fprintf(os.Stderr, "unknown source: %s (use youdao or woerter-net)\n", rc.source)
			os.Exit(1)
		}
	}

	dbPath := cachePath(rc.noCache)

	var offlineDict *dict.OfflineDictionary
	onlyOffline := false
	// Offline mode only works with traditional sources, not LLM
	if !useLLM {
		switch mode {
		case config.ModeOffline:
			onlyOffline = true
			od, err := openOfflineDict(source.Name(), rc.text)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			offlineDict = od
		case config.ModeAuto:
			od, err := openOfflineDict(source.Name(), rc.text)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: offline dict not available, falling back to online: %v\n", err)
			} else {
				offlineDict = od
			}
		case config.ModeOnline:
			// No offline dictionary.
		}
	}

	outfmt := outputFmt(rc)

	if useLLM {
		llmQuery(cfg, rc, outfmt, targetLang, sourceLang)
		return
	}

	client, err := dict.NewRdictWithOffline(source, dbPath, offlineDict, onlyOffline)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

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

// findProvider looks up a provider by name in the slice.
func findProvider(providers []config.LLMProvider, name string) *config.LLMProvider {
	for i := range providers {
		if providers[i].Name == name {
			return &providers[i]
		}
	}
	return nil
}

// LocalRC is a project-local .blrc config that overrides LLM settings.
// Place a .blrc file in the current directory to quickly switch provider/model.
type LocalRC struct {
	Provider   string `json:"provider,omitempty"`    // references a named provider in global config
	Model      string `json:"model,omitempty"`       // model ID override
	TargetLang string `json:"target_lang,omitempty"`  // target language override
	SourceLang string `json:"source_lang,omitempty"`  // source language override
	BaseURL    string `json:"base_url,omitempty"`     // ad-hoc base URL (if not using a named provider)
	APIKey     string `json:"api_key,omitempty"`      // ad-hoc API key or "env:VAR"
}

// applyTo applies the local RC overrides to the given config.
func (lrc *LocalRC) applyTo(cfg *config.Config) {
	if lrc.Provider != "" {
		cfg.LLM.Provider = lrc.Provider
	}
	if lrc.TargetLang != "" {
		cfg.LLM.TargetLang = lrc.TargetLang
	}
	if lrc.SourceLang != "" {
		cfg.LLM.SourceLang = lrc.SourceLang
	}
	if lrc.Model != "" || lrc.BaseURL != "" || lrc.APIKey != "" {
		// If a named provider is referenced and model is specified, update that provider's model.
		provider := findProvider(cfg.LLM.Providers, lrc.Provider)
		if provider != nil {
			if lrc.Model != "" {
				provider.Model = lrc.Model
			}
			if lrc.BaseURL != "" {
				provider.BaseURL = lrc.BaseURL
			}
			if lrc.APIKey != "" {
				provider.APIKey = lrc.APIKey
			}
		} else if lrc.BaseURL != "" || lrc.Model != "" {
			// If no matching named provider but we have enough info, create an ad-hoc provider.
			name := lrc.Provider
			if name == "" {
				name = "rc"
			}
			adHoc := config.LLMProvider{
				Name:    name,
				BaseURL: lrc.BaseURL,
				Model:   lrc.Model,
				APIKey:  lrc.APIKey,
			}
			if adHoc.BaseURL == "" {
				adHoc.BaseURL = "https://openrouter.ai/api/v1"
			}
			if adHoc.Model == "" {
				adHoc.Model = "google/gemma-4-31b-it:free"
			}
			cfg.LLM.Provider = name
			cfg.LLM.Providers = append(cfg.LLM.Providers, adHoc)
		}
	}
	cfg.LLM.Enabled = true
}

// loadLocalRC looks for a .blrc file in the current directory.
// Returns nil without error if no .blrc is found.
func loadLocalRC() (*LocalRC, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(cwd, ".blrc")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read .blrc: %w", err)
	}
	var rc LocalRC
	if err := json.Unmarshal(data, &rc); err != nil {
		return nil, fmt.Errorf("parse .blrc: %w", err)
	}
	return &rc, nil
}

// llmQuery handles LLM translation with automatic provider fallback.
// It tries providers in order starting from the configured default.
// If a different provider succeeds, it saves the working one as new default.
func llmQuery(cfg *config.Config, rc runConfig, outfmt dict.Format, targetLang, sourceLang string) {
	providers := cfg.LLM.Providers
	startIdx := -1
	for i, p := range providers {
		if p.Name == cfg.LLM.Provider {
			startIdx = i
			break
		}
	}
	if startIdx < 0 {
		startIdx = 0
	}

	// Determine the text to translate.
	text := rc.text
	if text == "" {
		// Try pipe mode.
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			input, err := io.ReadAll(os.Stdin)
			if err == nil {
				text = strings.TrimSpace(string(input))
			}
		}
	}

	cp := cachePath(rc.noCache)

	for i := 0; i < len(providers); i++ {
		idx := (startIdx + i) % len(providers)
		p := &providers[idx]

		// Apply CLI model/key overrides only to the initially requested provider.
		model := p.Model
		apiKey := p.APIKey
		if idx == startIdx {
			if rc.llmModel != "" {
				model = rc.llmModel
			}
			if rc.llmKey != "" {
				apiKey = rc.llmKey
			}
		}

		if p.BaseURL == "" || model == "" {
			continue
		}

		workingProvider := *p
		workingProvider.Model = model
		workingProvider.APIKey = apiKey

		source := dict.NewLLMSource("llm", workingProvider, targetLang, sourceLang, cfg.LLM.SystemPrompt)
		client, err := dict.NewRdict(source, cp)
		if err != nil {
			continue
		}

		if text != "" {
			fmt.Fprintf(os.Stderr, "[%s / %s]\n", p.Name, model)
			err = llmDoQuery(client, text, outfmt)
			client.Close()
			if err == nil {
				if idx != startIdx {
					cfg.LLM.Provider = p.Name
					if err := config.Save(cfg); err == nil {
						fmt.Fprintf(os.Stderr, "\n  (auto-switched to \"%s\" for future queries)\n", p.Name)
					}
				}
				return
			}
			if idx == startIdx {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				if !isAPIError(err) {
					// Input-related error (bad word, no translation) — don't fallback.
					return
				}
				fmt.Fprintf(os.Stderr, "  (falling back to next provider...)\n")
			}
		} else {
			// Interactive mode — no provider fallback, use the primary provider.
			interactiveMode(client, outfmt, source.Name())
			client.Close()
			return
		}
	}

	fmt.Fprintf(os.Stderr, "All %d LLM providers failed.\n", len(providers))
}

func llmDoQuery(client *dict.Rdict, text string, outfmt dict.Format) error {
	result, err := client.GetResults(text)
	if err != nil {
		return err
	}

	if outfmt == dict.FormatJSON {
		out, err := json.MarshalIndent(&result.Data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	if outfmt == dict.FormatOneliner {
		fmt.Println(render.RenderOneliner(&result.Data))
		return nil
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
	return nil
}

// isAPIError checks if an error is caused by the API (retryable)
// vs. bad user input (not worth retrying with another provider).
func isAPIError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	// API-level errors: these warrant trying another provider.
	apiPatterns := []string{
		"LLM API error",
		"API request",
		"rate limit",
		"timeout",
		"EOF",
		"connection refused",
		"no such host",
		"tls",
		"HTTP 429",
		"HTTP 5",
		"HTTP 503",
		"HTTP 502",
		"HTTP 500",
		"temporarily rate-limited",
		"Provider returned error",
	}
	for _, p := range apiPatterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
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
		fi, err := os.Stat(dest)
		if err != nil {
			fmt.Printf("  done (unknown size: %v)\n", err)
		} else {
			fmt.Printf("  done (%d bytes)\n", fi.Size())
		}
	}
	fmt.Println("Update complete.")
}

func dictStatusCmd(cfg *config.Config) {
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
	path, _ := config.ConfigPath()
	fmt.Printf("\nDefault mode: %s\n", cfg.Mode)
	fmt.Printf("LLM translation: %v\n", cfg.LLM.Enabled)
	fmt.Printf("LLM provider: %s\n", cfg.LLM.Provider)
	fmt.Printf("LLM target lang: %s\n", cfg.LLM.TargetLang)
	if cfg.LLM.SourceLang != "" {
		fmt.Printf("LLM source lang: %s\n", cfg.LLM.SourceLang)
	}
	fmt.Printf("Config file: %s\n", path)
	fmt.Printf("Env override: BL_MODE=%s\n", os.Getenv("BL_MODE"))

	// Show .blrc status
	if lrc, err := loadLocalRC(); err == nil && lrc != nil {
		fmt.Printf("\n.blrc active (local override):\n")
		if lrc.Provider != "" {
			fmt.Printf("  provider: %s\n", lrc.Provider)
		}
		if lrc.Model != "" {
			fmt.Printf("  model: %s\n", lrc.Model)
		}
		if lrc.TargetLang != "" {
			fmt.Printf("  target_lang: %s\n", lrc.TargetLang)
		}
		if lrc.SourceLang != "" {
			fmt.Printf("  source_lang: %s\n", lrc.SourceLang)
		}
		if lrc.BaseURL != "" {
			fmt.Printf("  base_url: %s\n", lrc.BaseURL)
		}
	} else {
		fmt.Printf("\n.blrc: not found (place in current dir for project-specific LLM config)\n")
	}
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
		out, err := json.MarshalIndent(&result.Data, "", "  ")
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
