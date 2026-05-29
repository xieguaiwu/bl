package dict

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"bl/internal/cache"
)

var userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"

type Rdict struct {
	client      *http.Client
	source      DictionarySource
	cache       *cache.Cache
	offline     *OfflineDictionary
	onlyOffline bool // when true, skip online lookup if offline misses
}

// NewRdict creates an online-only Rdict (no offline dictionary).
func NewRdict(source DictionarySource, cacheDB string) (*Rdict, error) {
	c, err := cache.New(cacheDB)
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
	}
	return &Rdict{
		client: &http.Client{Timeout: 15 * time.Second},
		source: source,
		cache:  c,
	}, nil
}

// NewRdictWithOffline creates an Rdict backed by an offline dictionary.
// When offlineSource is nil, falls back to online-only (same as NewRdict).
func NewRdictWithOffline(source DictionarySource, cacheDB string, offlineSource *OfflineDictionary, onlyOffline bool) (*Rdict, error) {
	c, err := cache.New(cacheDB)
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
	}
	return &Rdict{
		client:      &http.Client{Timeout: 15 * time.Second},
		source:      source,
		cache:       c,
		offline:     offlineSource,
		onlyOffline: onlyOffline,
	}, nil
}

func (r *Rdict) Close() error {
	var errs []error
	if err := r.cache.Close(); err != nil {
		errs = append(errs, err)
	}
	if r.offline != nil {
		if err := r.offline.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *Rdict) cacheKey(text string) string {
	return r.source.Name() + ":" + text
}

// GetResults returns translation results.
// Query chain: offline dictionary → cache → online fetch.
func (r *Rdict) GetResults(inputText string) (*FetchedResult, error) {
	// 1. Offline dictionary (fastest path)
	if r.offline != nil {
		if data, found := r.offline.Lookup(inputText); found {
			return &FetchedResult{Data: *data, IsCached: false}, nil
		}
		if r.onlyOffline {
			return nil, &OfflineUnavailable{word: inputText}
		}
	}

	// 2. Cache
	key := r.cacheKey(inputText)
	jsonStr, err := r.cache.Get(key)
	if err == nil && jsonStr != "" {
		var data TranslationData
		if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
			return &FetchedResult{Data: data, IsCached: true}, nil
		}
		_ = r.cache.Delete(key)
	}

	// 3. Online fetch
	html, err := r.fetchSourceHTML(inputText)
	if err != nil {
		return nil, err
	}

	data, err := r.source.Parse(inputText, html)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := json.Marshal(data)
	if err == nil {
		_ = r.cache.Set(key, string(jsonBytes))
	}

	return &FetchedResult{Data: *data, IsCached: false}, nil
}

func (r *Rdict) fetchSourceHTML(text string) (string, error) {
	url := r.source.FetchURL(text)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http get %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", &HttpError{
			Code:   resp.StatusCode,
			Source: r.source.Name(),
			Word:   text,
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	return string(body), nil
}
