package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBuildURL(t *testing.T) {
	c := NewClient("https://example.com/api/v1", http.DefaultClient)

	tests := []struct {
		name string
		opts SearchOptions
		want string
	}{
		{
			name: "basic search",
			opts: SearchOptions{Query: "go", Tags: "story"},
			want: "https://example.com/api/v1/search?query=go&tags=story",
		},
		{
			name: "sort by date",
			opts: SearchOptions{Query: "rust", Tags: "story", SortByDate: true},
			want: "https://example.com/api/v1/search_by_date?query=rust&tags=story",
		},
		{
			name: "with limit",
			opts: SearchOptions{Query: "test", Tags: "story", NumResults: 5},
			want: "https://example.com/api/v1/search?hitsPerPage=5&query=test&tags=story",
		},
		{
			name: "with min points",
			opts: SearchOptions{Query: "go", Tags: "story", MinPoints: 100},
			want: "https://example.com/api/v1/search?numericFilters=points%3E%3D100&query=go&tags=story",
		},
		{
			name: "with page",
			opts: SearchOptions{Query: "go", Tags: "story", Page: 2},
			want: "https://example.com/api/v1/search?page=2&query=go&tags=story",
		},
		{
			name: "url only",
			opts: SearchOptions{Query: "example.com", Tags: "story", URLOnly: true},
			want: "https://example.com/api/v1/search?query=example.com&restrictSearchableAttributes=url&tags=story",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.BuildURL(tt.opts)
			if got != tt.want {
				t.Errorf("BuildURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildURLWithAfterTime(t *testing.T) {
	c := NewClient("https://example.com/api/v1", http.DefaultClient)
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	opts := SearchOptions{Query: "test", Tags: "story", AfterTime: &ts}
	got := c.BuildURL(opts)
	if got == "" {
		t.Fatal("BuildURL returned empty string")
	}
	if !contains(got, "numericFilters=") {
		t.Errorf("expected numericFilters in URL, got %q", got)
	}
}

func TestBuildURLDateWindow(t *testing.T) {
	c := NewClient("https://example.com/api/v1", http.DefaultClient)
	after := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	got := c.BuildURL(SearchOptions{AfterTime: &after, BeforeTime: &before})
	want := "https://example.com/api/v1/search?numericFilters=created_at_i%3E1767225600%2Ccreated_at_i%3C1769904000"
	if got != want {
		t.Errorf("BuildURL() = %q, want %q", got, want)
	}
}

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(SearchResult{
			Hits:   []Hit{{ObjectID: "1", Title: "Test", Author: "user"}},
			NbHits: 1,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())
	result, err := c.Search(SearchOptions{Query: "test"})
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(result.Hits) != 1 {
		t.Errorf("expected 1 hit, got %d", len(result.Hits))
	}
	if result.Hits[0].Title != "Test" {
		t.Errorf("expected title 'Test', got %q", result.Hits[0].Title)
	}
}

func TestGetItem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/items/123" {
			json.NewEncoder(w).Encode(Item{ID: 123, Title: "Story", Author: "pg"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())

	item, err := c.GetItem("123")
	if err != nil {
		t.Fatalf("GetItem() error: %v", err)
	}
	if item.Title != "Story" {
		t.Errorf("expected title 'Story', got %q", item.Title)
	}

	_, err = c.GetItem("999")
	if err == nil {
		t.Error("expected error for missing item")
	}
}

func TestGetUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/pg" {
			json.NewEncoder(w).Encode(User{Username: "pg", Karma: 157316})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())

	user, err := c.GetUser("pg")
	if err != nil {
		t.Fatalf("GetUser() error: %v", err)
	}
	if user.Username != "pg" {
		t.Errorf("expected username 'pg', got %q", user.Username)
	}

	_, err = c.GetUser("nonexistent_user_xyz")
	if err == nil {
		t.Error("expected error for missing user")
	}
}

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"paragraph tags", "<p>hello</p><p>world</p>", "hello\nworld"},
		{"bold and italic", "<b>bold</b> and <i>italic</i>", "bold and italic"},
		{"code blocks", "<pre><code>fmt.Println()</code></pre>", "```\nfmt.Println()\n```"},
		{"inline code", "use <code>go run</code> here", "use `go run` here"},
		{"html entities", "this &amp; that &lt; those", "this & that < those"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanHTML(tt.input)
			if got != tt.want {
				t.Errorf("CleanHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
