package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ─── Main UI ──────────────────────────────────────────────────────

// InteractiveConfig opens an interactive terminal UI that lets the user
// view and modify LLM provider/model configuration.
// Changes are saved to ~/.config/bl/config.json on exit.
func InteractiveConfig(cfg *Config) error {
	scanner := bufio.NewScanner(os.Stdin)

	// Quick validation: ensure at least one provider exists
	if len(cfg.LLM.Providers) == 0 {
		cfg.LLM.Providers = DefaultLLMConfig().Providers
	}

	// Ensure the default provider actually exists in the provider list
	defaultValid := false
	for i := range cfg.LLM.Providers {
		if cfg.LLM.Providers[i].Name == cfg.LLM.Provider {
			defaultValid = true
			break
		}
	}
	if !defaultValid && len(cfg.LLM.Providers) > 0 {
		cfg.LLM.Provider = cfg.LLM.Providers[0].Name
	}

	for {
		printConfigScreen(cfg)

		fmt.Print("\n  > ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "q":
			if err := Save(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "  ✗ Save failed: %v\n", err)
				return err
			}
			fmt.Println("  ✓ Config saved.")
			return nil

		case "x":
			fmt.Println("  Exiting without saving.")
			return nil

		case "e":
			cfg.LLM.Enabled = !cfg.LLM.Enabled

		case "t":
			promptString(scanner, "Target language", &cfg.LLM.TargetLang)

		case "s":
			promptString(scanner,
				"Source language (empty = auto-detect, e.g. French, German)",
				&cfg.LLM.SourceLang)

		case "m":
			changeProviderModel(cfg, scanner)

		case "p":
			editProvider(cfg, scanner)

		case "a":
			addProvider(cfg, scanner)

		case "d":
			deleteProvider(cfg, scanner)

		default:
			if idx, err := strconv.Atoi(input); err == nil && idx >= 1 && idx <= len(cfg.LLM.Providers) {
				cfg.LLM.Provider = cfg.LLM.Providers[idx-1].Name
			}
		}
	}

	return nil
}

// ─── Screen Rendering ─────────────────────────────────────────────

func printConfigScreen(cfg *Config) {
	fmt.Print("\033[H\033[2J") // clear screen (ANSI)
	fmt.Println(strings.Repeat("─", 56))
	fmt.Println("  📖 bl  LLM Configuration")
	fmt.Println(strings.Repeat("─", 56))

	// ── Status summary ──
	status := "🔴 disabled"
	if cfg.LLM.Enabled {
		status = "🟢 enabled "
	}
	fmt.Printf("  LLM:          %s\n", status)

	// Find the default provider's model
	defaultModel := ""
	hasBaseURL := false
	for i := range cfg.LLM.Providers {
		if cfg.LLM.Providers[i].Name == cfg.LLM.Provider {
			if cfg.LLM.Providers[i].Model != "" {
				defaultModel = cfg.LLM.Providers[i].Model
			} else {
				defaultModel = "(not configured)"
			}
			hasBaseURL = cfg.LLM.Providers[i].BaseURL != ""
			break
		}
	}
	fmt.Printf("  Provider:     %s\n", cfg.LLM.Provider)
	fmt.Printf("  Model:        %s\n", defaultModel)
	if !hasBaseURL && cfg.LLM.Provider != "" {
		fmt.Printf("  ⚠  The default provider has no base_url. Translation will fail.\n")
	}

	targetLabel := cfg.LLM.TargetLang
	if targetLabel == "" {
		targetLabel = "(not set)"
	}
	srcLabel := cfg.LLM.SourceLang
	if srcLabel == "" {
		srcLabel = "(auto-detect)"
	}
	fmt.Printf("  Target lang:  %s\n", targetLabel)
	fmt.Printf("  Source lang:  %s\n", srcLabel)

	// ── Provider list ──
	fmt.Println(strings.Repeat("─", 56))
	fmt.Println("  Providers:")
	if len(cfg.LLM.Providers) == 0 {
		fmt.Println("    (none configured)")
	} else {
		for i, p := range cfg.LLM.Providers {
			mark := " "
			if p.Name == cfg.LLM.Provider {
				mark = "→"
			}
			model := p.Model
			if model == "" {
				model = "(not configured)"
			}
			url := p.BaseURL
			if url == "" {
				url = "(no base URL)"
			}
			keyHint := ""
			if p.APIKey == "" {
				keyHint = "  ⚠ no key"
			} else if strings.HasPrefix(p.APIKey, "env:") {
				keyHint = "  env:" + p.APIKey[4:]
			}
			fmt.Printf("  [%d] %s %-14s %s%s\n", i+1, mark, p.Name, model, keyHint)
			if url != "(no base URL)" || p.Name == cfg.LLM.Provider {
				fmt.Printf("      %s\n", url)
			}
		}
	}

	// ── Commands ──
	fmt.Println(strings.Repeat("─", 56))
	fmt.Println("  Commands:")
	fmt.Println("    [1-N]  Select default provider    [m] Change model")
	fmt.Println("    [e]    Toggle LLM on/off          [t] Target language")
	fmt.Println("    [s]    Source language             [p] Edit provider")
	fmt.Println("    [a]    Add provider                [d] Delete provider")
	fmt.Println("    [q]    Save and exit               [x] Exit without saving")
}

// ─── Sub-menus ────────────────────────────────────────────────────

// changeProviderModel lets the user pick a provider and change its model ID.
func changeProviderModel(cfg *Config, scanner *bufio.Scanner) {
	fmt.Println(strings.Repeat("─", 56))
	fmt.Println("  📦 Change Model")
	fmt.Println(strings.Repeat("─", 56))
	for i, p := range cfg.LLM.Providers {
		model := p.Model
		if model == "" {
			model = "(not configured)"
		}
		mark := " "
		if p.Name == cfg.LLM.Provider {
			mark = "→"
		}
		fmt.Printf("  [%d] %s %-14s %s\n", i+1, mark, p.Name, model)
	}
	fmt.Print("\n  Select provider number: ")
	if !scanner.Scan() {
		return
	}
	input := strings.TrimSpace(scanner.Text())
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(cfg.LLM.Providers) {
		fmt.Println("  Invalid selection.")
		pressEnter(scanner)
		return
	}

	p := &cfg.LLM.Providers[idx-1]
	fmt.Printf("  Current model: %s\n", p.Model)
	fmt.Print("  New model ID (Enter to keep): ")
	if !scanner.Scan() {
		return
	}
	newModel := strings.TrimSpace(scanner.Text())
	if newModel != "" {
		p.Model = newModel
		fmt.Printf("  ✓ Model updated to: %s\n", p.Model)
	} else {
		fmt.Println("  No change.")
	}
	pressEnter(scanner)
}

// editProvider lets the user modify all fields of an existing provider.
func editProvider(cfg *Config, scanner *bufio.Scanner) {
	fmt.Println(strings.Repeat("─", 56))
	fmt.Println("  ✏️  Edit Provider")
	fmt.Println(strings.Repeat("─", 56))
	for i, p := range cfg.LLM.Providers {
		mark := " "
		if p.Name == cfg.LLM.Provider {
			mark = "→"
		}
		fmt.Printf("  [%d] %s %s\n", i+1, mark, p.Name)
	}
	fmt.Print("\n  Select provider number: ")
	if !scanner.Scan() {
		return
	}
	input := strings.TrimSpace(scanner.Text())
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(cfg.LLM.Providers) {
		fmt.Println("  Invalid selection.")
		pressEnter(scanner)
		return
	}

	p := &cfg.LLM.Providers[idx-1]
	fmt.Println("  (Enter = keep current value)")

	promptString(scanner, fmt.Sprintf("  Name [%s]", p.Name), &p.Name)
	promptString(scanner, fmt.Sprintf("  Base URL [%s]", p.BaseURL), &p.BaseURL)
	promptString(scanner, fmt.Sprintf("  Model [%s]", p.Model), &p.Model)
	promptString(scanner, fmt.Sprintf("  API key [%s]", maskKey(p.APIKey)), &p.APIKey)

	fmt.Println("  ✓ Provider updated.")
	pressEnter(scanner)
}

// addProvider prompts for all fields and appends a new provider.
func addProvider(cfg *Config, scanner *bufio.Scanner) {
	fmt.Println(strings.Repeat("─", 56))
	fmt.Println("  ➕ Add New Provider")
	fmt.Println(strings.Repeat("─", 56))

	var p LLMProvider
	fmt.Print("  Name: ")
	if !scanner.Scan() {
		return
	}
	p.Name = strings.TrimSpace(scanner.Text())
	if p.Name == "" {
		fmt.Println("  Name is required. Cancelled.")
		pressEnter(scanner)
		return
	}

	// Check for duplicate
	for _, existing := range cfg.LLM.Providers {
		if existing.Name == p.Name {
			fmt.Printf("  Provider %q already exists. Cancelled.\n", p.Name)
			pressEnter(scanner)
			return
		}
	}

	fmt.Print("  Base URL: ")
	if scanner.Scan() {
		p.BaseURL = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("  Model: ")
	if scanner.Scan() {
		p.Model = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("  API key (literal or env:VAR_NAME): ")
	if scanner.Scan() {
		p.APIKey = strings.TrimSpace(scanner.Text())
	}

	cfg.LLM.Providers = append(cfg.LLM.Providers, p)
	fmt.Printf("  ✓ Provider %q added.\n", p.Name)
	pressEnter(scanner)
}

// deleteProvider removes a provider after confirmation.
// Prevents deleting the last remaining provider.
func deleteProvider(cfg *Config, scanner *bufio.Scanner) {
	fmt.Println(strings.Repeat("─", 56))
	fmt.Println("  🗑  Delete Provider")
	fmt.Println(strings.Repeat("─", 56))

	if len(cfg.LLM.Providers) <= 1 {
		fmt.Println("  At least one provider is required. Cannot delete.")
		pressEnter(scanner)
		return
	}

	for i, p := range cfg.LLM.Providers {
		mark := " "
		if p.Name == cfg.LLM.Provider {
			mark = "→"
		}
		fmt.Printf("  [%d] %s %s\n", i+1, mark, p.Name)
	}
	fmt.Print("\n  Select provider to delete: ")
	if !scanner.Scan() {
		return
	}
	input := strings.TrimSpace(scanner.Text())
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(cfg.LLM.Providers) {
		fmt.Println("  Invalid selection.")
		pressEnter(scanner)
		return
	}

	p := cfg.LLM.Providers[idx-1]
	fmt.Printf("  Delete provider %q? (y/N): ", p.Name)
	if !scanner.Scan() {
		return
	}
	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("  Cancelled.")
		pressEnter(scanner)
		return
	}

	// If deleting the current default, switch to the first remaining provider
	if p.Name == cfg.LLM.Provider {
		switchTo := cfg.LLM.Providers[0].Name
		// If the first item is the one being deleted, use the second
		if switchTo == p.Name && len(cfg.LLM.Providers) > 1 {
			switchTo = cfg.LLM.Providers[1].Name
		}
		cfg.LLM.Provider = switchTo
		fmt.Printf("  Default switched to %q.\n", switchTo)
	}

	cfg.LLM.Providers = append(cfg.LLM.Providers[:idx-1], cfg.LLM.Providers[idx:]...)
	fmt.Printf("  ✓ Provider %q deleted.\n", p.Name)
	pressEnter(scanner)
}

// ─── Helpers ──────────────────────────────────────────────────────

// promptString prompts for a string value. If the user enters non-empty text,
// it updates the target. If empty, the target is left unchanged.
func promptString(scanner *bufio.Scanner, prompt string, target *string) {
	fmt.Printf("  %s: ", prompt)
	if !scanner.Scan() {
		return
	}
	val := strings.TrimSpace(scanner.Text())
	if val != "" {
		*target = val
	}
}

// pressEnter waits for the user to press Enter.
func pressEnter(scanner *bufio.Scanner) {
	fmt.Print("  Press Enter to continue... ")
	scanner.Scan()
}

// maskKey partially hides an API key for display, preserving "env:" prefix.
func maskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if strings.HasPrefix(key, "env:") {
		return key
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + "****" + key[len(key)-4:]
}
