package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// testClient points both APIs at srv.
func testClient(srv *httptest.Server) *Client {
	c := NewClient(srv.URL, srv.Client())
	c.firebaseURL = srv.URL
	return c
}

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
			want: "https://example.com/api/v1/search?numericFilters=points%3E100&query=go&tags=story",
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

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(SearchResult{
			Hits:   []Hit{{ObjectID: "1", Title: "Test", Author: "user"}},
			NbHits: 1,
		})
	}))
	defer srv.Close()

	c := testClient(srv)
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

	c := testClient(srv)

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
		if r.URL.Path == "/user/pg.json" {
			w.Write([]byte(`{"id":"pg","karma":157316,"about":"Bug fixer."}`))
			return
		}
		w.Write([]byte("null"))
	}))
	defer srv.Close()

	c := testClient(srv)

	user, err := c.GetUser("pg")
	if err != nil {
		t.Fatalf("GetUser() error: %v", err)
	}
	if user.Username != "pg" || user.Karma != 157316 || user.About != "Bug fixer." {
		t.Errorf("unexpected user: %+v", user)
	}

	_, err = c.GetUser("nonexistent_user_xyz")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing user, got %v", err)
	}
}

func TestErrorClassification(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   error
	}{
		{"not found", http.StatusNotFound, ErrNotFound},
		{"rate limited", http.StatusTooManyRequests, ErrRetryable},
		{"server error", http.StatusBadGateway, ErrRetryable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			c := testClient(srv)
			if _, err := c.GetItem("1"); !errors.Is(err, tt.want) {
				t.Errorf("GetItem() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestNetworkFailureIsRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	c := NewClient(url, http.DefaultClient)
	c.firebaseURL = url
	if _, err := c.GetItem("1"); !errors.Is(err, ErrRetryable) {
		t.Errorf("GetItem() error = %v, want ErrRetryable", err)
	}
	if _, err := c.Search(SearchOptions{Query: "go"}); !errors.Is(err, ErrRetryable) {
		t.Errorf("Search() error = %v, want ErrRetryable", err)
	}
}

func TestClientErrorIsNotRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := testClient(srv)
	if _, err := c.Search(SearchOptions{Query: "go"}); err == nil || errors.Is(err, ErrRetryable) {
		t.Errorf("Search() error = %v, want a non-retryable error", err)
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
