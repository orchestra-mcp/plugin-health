// Package tools contains all tool handler implementations for the health plugin.
package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/plugin-health/internal/storage"
)

// ToolHandler is the function signature for all tool handlers.
type ToolHandler = func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error)

// newEntryID generates a short random entry ID with the given prefix, e.g. "wtr-a1b2c3".
func newEntryID(prefix string) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

// todayPrefix returns today's date as "2006-01-02" for filtering entries by date.
func todayPrefix() string {
	return time.Now().UTC().Format("2006-01-02")
}

// parseTime attempts to parse a time string in HH:MM format, returning the hour
// and minute. Returns 0, 0 if parsing fails.
func parseTime(s string) (hour, minute int) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	h, m := 0, 0
	for _, c := range parts[0] {
		if c < '0' || c > '9' {
			return 0, 0
		}
		h = h*10 + int(c-'0')
	}
	for _, c := range parts[1] {
		if c < '0' || c > '9' {
			return 0, 0
		}
		m = m*10 + int(c-'0')
	}
	return h, m
}

// nowHourMinute returns the current UTC hour and minute.
func nowHourMinute() (int, int) {
	now := time.Now().UTC()
	return now.Hour(), now.Minute()
}

// minutesSinceMidnight converts hours and minutes to total minutes since midnight.
func minutesSinceMidnight(h, m int) int {
	return h*60 + m
}

// containsIgnoreCase checks if haystack contains needle (case-insensitive).
func containsIgnoreCase(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

// loadTodayEntries reads all entries from a sub-directory and filters to today's date.
func loadTodayEntries(ctx context.Context, store *storage.DataStorage, projectSlug, subDir string) ([]map[string]any, error) {
	entries, err := store.ListEntries(ctx, projectSlug, subDir)
	if err != nil {
		return nil, err
	}

	today := todayPrefix()
	var results []map[string]any

	for _, entry := range entries {
		entryID := storage.EntryIDFromPath(entry.Path)
		meta, _, _, err := store.ReadEntry(ctx, projectSlug, subDir, entryID)
		if err != nil {
			continue
		}
		m := meta.AsMap()

		// Filter by today's date using created_at field.
		createdAt, _ := m["created_at"].(string)
		if !strings.HasPrefix(createdAt, today) {
			continue
		}

		results = append(results, m)
	}

	return results, nil
}

// Food safety blacklists for various conditions.
var conditionBlacklists = map[string][]string{
	"gout":        {"red meat", "organ meat", "shellfish", "beer", "beans", "lentils"},
	"gerd":        {"citrus", "tomato", "chocolate", "caffeine", "alcohol", "fried food", "onion", "garlic"},
	"ibs":         {"dairy", "beans", "onion", "garlic", "wheat", "artificial sweeteners"},
	"fatty_liver": {"added sugar", "honey", "white flour", "fried food", "alcohol"},
}

// Food alternatives suggestions.
var conditionAlternatives = map[string][]string{
	"gout":        {"chicken", "turkey", "tofu", "low-fat dairy", "cherries", "citrus fruits"},
	"gerd":        {"bananas", "melon", "oatmeal", "ginger tea", "lean protein", "green vegetables"},
	"ibs":         {"rice", "oats", "lean meats", "bananas", "blueberries", "carrots"},
	"fatty_liver": {"whole grains", "leafy greens", "fish", "nuts", "olive oil", "berries"},
}
