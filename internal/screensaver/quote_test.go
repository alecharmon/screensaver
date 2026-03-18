package screensaver

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDailyQuoteUsesTodaysCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "quote.json")
	now := time.Date(2026, 3, 18, 9, 0, 0, 0, time.Local)
	err := writeQuoteCache(cachePath, quoteCache{
		Date:   now.Format("2006-01-02"),
		Quote:  "cached quote",
		Author: "cached author",
	})
	if err != nil {
		t.Fatalf("writeQuoteCache() error = %v", err)
	}

	fetchCalled := 0
	fetch := func(context.Context, string) (Quote, error) {
		fetchCalled++
		return Quote{Text: "fetched", Author: "api"}, nil
	}

	got, err := loadDailyQuote(now, cachePath, "key", fetch)
	if err != nil {
		t.Fatalf("loadDailyQuote() error = %v", err)
	}
	if fetchCalled != 0 {
		t.Fatalf("fetch called %d times, want 0", fetchCalled)
	}
	if got.Text != "cached quote" || got.Author != "cached author" {
		t.Fatalf("loadDailyQuote() = %+v, want cached quote", got)
	}
}

func TestLoadDailyQuoteRefreshesAndWritesCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "quote.json")
	now := time.Date(2026, 3, 18, 9, 0, 0, 0, time.Local)

	fetchCalled := 0
	fetch := func(context.Context, string) (Quote, error) {
		fetchCalled++
		return Quote{Text: "fresh quote", Author: "api"}, nil
	}

	got, err := loadDailyQuote(now, cachePath, "key", fetch)
	if err != nil {
		t.Fatalf("loadDailyQuote() error = %v", err)
	}
	if fetchCalled != 1 {
		t.Fatalf("fetch called %d times, want 1", fetchCalled)
	}
	if got.Text != "fresh quote" || got.Author != "api" {
		t.Fatalf("loadDailyQuote() = %+v, want fetched quote", got)
	}

	stored, err := readQuoteCache(cachePath)
	if err != nil {
		t.Fatalf("readQuoteCache() error = %v", err)
	}
	if stored.Date != now.Format("2006-01-02") || stored.Quote != "fresh quote" || stored.Author != "api" {
		t.Fatalf("stored cache = %+v, want today's fetched quote", stored)
	}
}

func TestLoadDailyQuoteFallsBackToStaleCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "quote.json")
	now := time.Date(2026, 3, 18, 9, 0, 0, 0, time.Local)
	err := writeQuoteCache(cachePath, quoteCache{
		Date:   "2026-03-17",
		Quote:  "stale quote",
		Author: "stale author",
	})
	if err != nil {
		t.Fatalf("writeQuoteCache() error = %v", err)
	}

	fetch := func(context.Context, string) (Quote, error) {
		return Quote{}, errors.New("network down")
	}

	got, err := loadDailyQuote(now, cachePath, "key", fetch)
	if err != nil {
		t.Fatalf("loadDailyQuote() error = %v", err)
	}
	if got.Text != "stale quote" || got.Author != "stale author" {
		t.Fatalf("loadDailyQuote() = %+v, want stale quote fallback", got)
	}
}

func TestZenQuotesTodayURLWithoutKey(t *testing.T) {
	got := zenQuotesTodayURL("")
	if got != "https://zenquotes.io/api/today" {
		t.Fatalf("zenQuotesTodayURL(\"\") = %q, want %q", got, "https://zenquotes.io/api/today")
	}
}

func TestZenQuotesTodayURLWithKey(t *testing.T) {
	got := zenQuotesTodayURL("abc123")
	if got != "https://zenquotes.io/api/today/abc123" {
		t.Fatalf("zenQuotesTodayURL(\"abc123\") = %q, want %q", got, "https://zenquotes.io/api/today/abc123")
	}
}

func TestLoadRandomQuoteAndCacheWritesTodaysCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "quote.json")
	now := time.Date(2026, 3, 18, 14, 0, 0, 0, time.Local)

	fetchCalled := 0
	fetch := func(context.Context, string) (Quote, error) {
		fetchCalled++
		return Quote{Text: "random quote", Author: "random author"}, nil
	}

	got, err := loadRandomQuoteAndCache(now, cachePath, "key", fetch)
	if err != nil {
		t.Fatalf("loadRandomQuoteAndCache() error = %v", err)
	}
	if fetchCalled != 1 {
		t.Fatalf("fetch called %d times, want 1", fetchCalled)
	}
	if got.Text != "random quote" || got.Author != "random author" {
		t.Fatalf("loadRandomQuoteAndCache() = %+v, want fetched random quote", got)
	}

	stored, err := readQuoteCache(cachePath)
	if err != nil {
		t.Fatalf("readQuoteCache() error = %v", err)
	}
	if stored.Date != now.Format("2006-01-02") || stored.Quote != "random quote" || stored.Author != "random author" {
		t.Fatalf("stored cache = %+v, want today's random quote", stored)
	}
}
