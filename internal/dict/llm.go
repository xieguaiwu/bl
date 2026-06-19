package dict

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"bl/internal/config"
)

// LLMSource implements DictionarySource via OpenAI-compatible chat API.
// It stubs FetchURL/Parse (inherited from DictionarySource interface) and
// provides Translate() as the primary method.
type LLMSource struct {
	name       string
	provider   config.LLMProvider
	targetLang string
	sysPrompt  string
	client     *http.Client
}

// NewLLMSource creates a new LLM-based dictionary source.
// The name is used as the DictionarySource identifier.
// provider defines the API endpoint, model, and authentication.
// targetLang is the desired output language (e.g. "中文", "English", "日本語").
// sysPrompt overrides the default system prompt if non-empty.
func NewLLMSource(name string, provider config.LLMProvider, targetLang string, sysPrompt string) *LLMSource {
	if name == "" {
		name = "llm"
	}
	prompt := sysPrompt
	if prompt == "" {
		prompt = fmt.Sprintf(defaultTranslationPrompt, targetLang)
	} else if strings.Contains(prompt, "%s") {
		prompt = fmt.Sprintf(prompt, targetLang)
	} else {
		// Custom prompt without %s: ensure target language instruction is at the start.
		prompt = fmt.Sprintf("Translate the given text to %s.\n\n%s", targetLang, prompt)
	}
	return &LLMSource{
		name:       name,
		provider:   provider,
		targetLang: targetLang,
		sysPrompt:  prompt,
		client:     &http.Client{Timeout: 60 * time.Second},
	}
}

// defaultTranslationPrompt is the default system prompt used for translation.
// %s is replaced with the target language.
const defaultTranslationPrompt = `You are a professional translator and linguist. Translate the given text to %s.

RULES:
1. Return ONLY a JSON object — no markdown, no code fences, no extra text.
2. Use this exact JSON structure. Only include fields that are relevant; set irrelevant fields to empty string or empty array:
{
  "translations": ["primary translation", "alternative (if applicable)"],
  "pronunciation": "phonetic or pinyin",
  "part_of_speech": "noun / verb / adjective / adverb / preposition / etc.",
  "gender": "masculine / feminine / neuter (for nouns, if applicable)",
  "plural": "plural form (for countable nouns)",
  "comparative": "comparative form (for adjectives/adverbs)",
  "superlative": "superlative form (for adjectives/adverbs)",
  "examples": [
    {"en": "example in original language", "zh": "translated example"}
  ]
}
3. If the word has multiple meanings, include up to 3 in the translations array.
4. Provide exactly 5 example sentences in vivid, concrete scenes with clear actions and imagery. Each must depict a specific situation the reader can visualize.
5. For nouns in inflected languages (German, French, etc.), always provide gender and plural if applicable.
6. For adjectives, provide comparative and superlative forms if applicable.
7. Set any field that is not relevant or unknown to empty string (or empty array for examples).`

// Name returns a composite source identifier that includes provider and target language
// to avoid cache key collisions when switching providers or target languages.
func (s *LLMSource) Name() string {
	return fmt.Sprintf("llm:%s:%s", s.provider.Name, s.targetLang)
}

// FetchURL stubs the DictionarySource interface. LLM source does not use URL fetching.
func (s *LLMSource) FetchURL(word string) string { return "" }

// Parse stubs the DictionarySource interface. LLM source does not use HTML parsing.
func (s *LLMSource) Parse(word string, html string) (*TranslationData, error) {
	return nil, fmt.Errorf("LLMSource: Parse not supported, use Translate()")
}

// resolveAPIKey resolves the API key from the provider config.
// Supports literal values and "env:VAR_NAME" references.
func (s *LLMSource) resolveAPIKey() string {
	key := s.provider.APIKey
	if strings.HasPrefix(key, "env:") {
		return os.Getenv(strings.TrimPrefix(key, "env:"))
	}
	return key
}

// Translate sends the word to the LLM API and returns a structured translation.
func (s *LLMSource) Translate(word string) (*TranslationData, error) {
	if s.provider.BaseURL == "" {
		return nil, fmt.Errorf("LLM provider %q: base_url is not configured", s.provider.Name)
	}
	if s.provider.Model == "" {
		return nil, fmt.Errorf("LLM provider %q: model is not configured", s.provider.Name)
	}
	apiKey := s.resolveAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("LLM provider %q: API key not resolved (check config: api_key=%q or set the referenced env var)",
			s.provider.Name, s.provider.APIKey)
	}

	// Build the OpenAI-compatible request body.
	reqBody := map[string]interface{}{
		"model": s.provider.Model,
		"messages": []map[string]string{
			{"role": "system", "content": s.sysPrompt},
			{"role": "user", "content": fmt.Sprintf("Text to translate:\n\n%s", word)},
		},
		"temperature": 0.1,
		"max_tokens":  1024,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(s.provider.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API error (HTTP %d): %s", resp.StatusCode, truncate(string(respBody), 500))
	}

	// Parse the OpenAI chat completion response.
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parse API response: %w", err)
	}
	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return nil, fmt.Errorf("LLM API error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM API returned no choices")
	}

	content := chatResp.Choices[0].Message.Content
	return s.parseTranslation(content, word)
}

// stripCodeFences removes markdown code fences (```json ... ``` or ``` ... ```)
// and inline code (`...`) from the content before JSON parsing.
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	// Remove ```json ... ``` or ``` ... ```
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		if strings.HasSuffix(s, "```") {
			s = s[:len(s)-3]
		}
		s = strings.TrimSpace(s)
	}
	// Remove inline `code` wrapping if the whole string is wrapped
	if strings.HasPrefix(s, "`") && strings.HasSuffix(s, "`") {
		s = s[1 : len(s)-1]
		s = strings.TrimSpace(s)
	}
	return s
}

// parseTranslation extracts TranslationData from the LLM's JSON response.
func (s *LLMSource) parseTranslation(content string, word string) (*TranslationData, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("LLM returned empty response for %q", word)
	}

	// Strip markdown code fences before JSON parsing.
	content = stripCodeFences(content)

	// Try direct JSON parse first.
	var t Translation
	if err := json.Unmarshal([]byte(content), &t); err == nil && len(t.Translations) > 0 {
		t.InputText = word
		return &TranslationData{Type: TypeTranslation, Translation: &t}, nil
	}

	// Try extracting JSON object(s) from arbitrary text using brace matching.
	extracted := extractJSON(content)
	if extracted != "" {
		var t2 Translation
		if err := json.Unmarshal([]byte(extracted), &t2); err == nil {
			if len(t2.Translations) > 0 {
				t2.InputText = word
				return &TranslationData{Type: TypeTranslation, Translation: &t2}, nil
			}
			// Valid JSON structure but no translations — likely an error response.
			return nil, fmt.Errorf("LLM returned unexpected JSON (no translations): %s", truncate(content, 300))
		}
	}

	// Check if content looks like an error message before falling back.
	lower := strings.ToLower(content)
	if strings.Contains(lower, "error") || strings.Contains(lower, "sorry") ||
		strings.Contains(lower, "cannot") || strings.Contains(lower, "unable") ||
		strings.Contains(lower, "rate limit") || strings.Contains(lower, "invalid") {
		return nil, fmt.Errorf("LLM returned non-translation response: %s", truncate(content, 300))
	}

	// Fallback: wrap the raw text as a single translation.
	return &TranslationData{
		Type: TypeTranslation,
		Translation: &Translation{
			InputText:    word,
			Translations: []string{content},
		},
	}, nil
}

// extractJSON finds the first top-level JSON object ({...}) within arbitrary text
// using brace-depth matching, correctly handling nested objects.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
