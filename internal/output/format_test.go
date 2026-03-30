package output

import (
	"strings"
	"testing"
	"time"

	"github.com/voska/hn-cli/internal/api"
)

func TestFormatSearchResults(t *testing.T) {
	result := &api.SearchResult{
		NbHits: 1,
		Hits: []api.Hit{
			{
				ObjectID:    "12345",
				Title:       "Test Story",
				URL:         "https://example.com",
				Author:      "testuser",
				Points:      42,
				NumComments: 10,
				CreatedAt:   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			},
		},
	}

	var buf strings.Builder
	FormatSearchResults(&buf, result, false)
	out := buf.String()

	if !strings.Contains(out, "1 results:") {
		t.Errorf("expected '1 results:' header, got %q", out)
	}
	if !strings.Contains(out, "[42pts 10c]") {
		t.Errorf("expected points/comments, got %q", out)
	}
	if !strings.Contains(out, "Test Story") {
		t.Errorf("expected title in output, got %q", out)
	}
	if !strings.Contains(out, "@testuser") {
		t.Errorf("expected author in output, got %q", out)
	}
	if !strings.Contains(out, "https://example.com") {
		t.Errorf("expected URL in output, got %q", out)
	}
	if !strings.Contains(out, "item?id=12345") {
		t.Errorf("expected HN link in output, got %q", out)
	}
}

func TestFormatUser(t *testing.T) {
	user := &api.User{
		Username: "pg",
		Karma:    157316,
		About:    "Bug fixer.",
	}

	var buf strings.Builder
	FormatUser(&buf, user)
	out := buf.String()

	if !strings.Contains(out, "@pg") {
		t.Errorf("expected @pg, got %q", out)
	}
	if !strings.Contains(out, "karma:157316") {
		t.Errorf("expected karma, got %q", out)
	}
	if !strings.Contains(out, "Bug fixer.") {
		t.Errorf("expected about text, got %q", out)
	}
}

func TestFormatEmptyResults(t *testing.T) {
	result := &api.SearchResult{Hits: []api.Hit{}}
	var buf strings.Builder
	FormatSearchResults(&buf, result, false)
	if !strings.Contains(buf.String(), "No results.") {
		t.Error("expected 'No results.' for empty result set")
	}
}

func TestTimeAgo(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"just now", time.Now().Add(-10 * time.Second).Format(time.RFC3339), "just now"},
		{"minutes ago", time.Now().Add(-30 * time.Minute).Format(time.RFC3339), "30m ago"},
		{"hours ago", time.Now().Add(-5 * time.Hour).Format(time.RFC3339), "5h ago"},
		{"days ago", time.Now().Add(-3 * 24 * time.Hour).Format(time.RFC3339), "3d ago"},
		{"invalid returns raw", "not-a-date", "not-a-date"},
		{"millisecond format", time.Now().UTC().Add(-1 * time.Hour).Format("2006-01-02T15:04:05.000Z"), "1h ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeAgo(tt.input)
			if got != tt.want {
				t.Errorf("timeAgo(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
