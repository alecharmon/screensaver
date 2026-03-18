package screensaver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	zenQuotesTodayBaseURL  = "https://zenquotes.io/api/today"
	zenQuotesRandomBaseURL = "https://zenquotes.io/api/random"
)

type Quote struct {
	Text   string
	Author string
}

type quoteCache struct {
	Date   string `json:"date"`
	Quote  string `json:"quote"`
	Author string `json:"author"`
}

type quoteFetcher func(context.Context, string) (Quote, error)

func loadQuoteCmd(now time.Time) tea.Cmd {
	return func() tea.Msg {
		cachePath, err := defaultQuoteCachePath()
		if err != nil {
			return quoteLoadedMsg{Err: err}
		}

		q, err := loadDailyQuote(now, cachePath, os.Getenv("ZENQUOTES_API_KEY"), fetchQuoteFromAPI)
		return quoteLoadedMsg{Quote: q, Err: err}
	}
}

func loadRandomQuoteCmd() tea.Cmd {
	return func() tea.Msg {
		cachePath, pathErr := defaultQuoteCachePath()
		if pathErr != nil {
			return quoteLoadedMsg{Err: pathErr}
		}

		q, err := loadRandomQuoteAndCache(time.Now(), cachePath, os.Getenv("ZENQUOTES_API_KEY"), fetchRandomQuoteFromAPI)
		return quoteLoadedMsg{Quote: q, Err: err}
	}
}

func loadDailyQuote(now time.Time, cachePath, apiKey string, fetch quoteFetcher) (Quote, error) {
	today := now.Format("2006-01-02")

	cached, cacheErr := readQuoteCache(cachePath)
	if cacheErr == nil && cached.Date == today && cached.Quote != "" {
		return Quote{Text: cached.Quote, Author: cached.Author}, nil
	}

	fetched, fetchErr := fetch(context.Background(), apiKey)
	if fetchErr != nil {
		if cacheErr == nil && cached.Quote != "" {
			return Quote{Text: cached.Quote, Author: cached.Author}, nil
		}
		return Quote{}, fetchErr
	}

	err := writeQuoteCache(cachePath, quoteCache{
		Date:   today,
		Quote:  fetched.Text,
		Author: fetched.Author,
	})
	if err != nil {
		return fetched, err
	}
	return fetched, nil
}

func fetchQuoteFromAPI(ctx context.Context, apiKey string) (Quote, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zenQuotesTodayURL(apiKey), nil)
	if err != nil {
		return Quote{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Quote{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Quote{}, fmt.Errorf("quote API returned %s", resp.Status)
	}

	var payload []struct {
		Quote  string `json:"q"`
		Author string `json:"a"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Quote{}, err
	}
	if len(payload) == 0 || payload[0].Quote == "" {
		return Quote{}, errors.New("quote API returned empty payload")
	}

	return Quote{Text: payload[0].Quote, Author: payload[0].Author}, nil
}

func loadRandomQuoteAndCache(now time.Time, cachePath, apiKey string, fetch quoteFetcher) (Quote, error) {
	q, err := fetch(context.Background(), apiKey)
	if err != nil {
		return Quote{}, err
	}

	writeErr := writeQuoteCache(cachePath, quoteCache{
		Date:   now.Format("2006-01-02"),
		Quote:  q.Text,
		Author: q.Author,
	})
	if writeErr != nil {
		return q, writeErr
	}
	return q, nil
}

func fetchRandomQuoteFromAPI(ctx context.Context, apiKey string) (Quote, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zenQuotesRandomURL(apiKey), nil)
	if err != nil {
		return Quote{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Quote{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Quote{}, fmt.Errorf("quote API returned %s", resp.Status)
	}

	var payload []struct {
		Quote  string `json:"q"`
		Author string `json:"a"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Quote{}, err
	}
	if len(payload) == 0 || payload[0].Quote == "" {
		return Quote{}, errors.New("quote API returned empty payload")
	}

	return Quote{Text: payload[0].Quote, Author: payload[0].Author}, nil
}

func zenQuotesTodayURL(apiKey string) string {
	if apiKey == "" {
		return zenQuotesTodayBaseURL
	}
	return zenQuotesTodayBaseURL + "/" + apiKey
}

func zenQuotesRandomURL(apiKey string) string {
	if apiKey == "" {
		return zenQuotesRandomBaseURL
	}
	return zenQuotesRandomBaseURL + "/" + apiKey
}

func defaultQuoteCachePath() (string, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheRoot, "screensaver", "quote_of_day.json"), nil
}

func readQuoteCache(path string) (quoteCache, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return quoteCache{}, err
	}
	var c quoteCache
	if err := json.Unmarshal(b, &c); err != nil {
		return quoteCache{}, err
	}
	return c, nil
}

func writeQuoteCache(path string, c quoteCache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
