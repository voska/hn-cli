package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
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

func TestFeed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/topstories.json":
			w.Write([]byte("[30,10,20,40,50]"))
		case "/search":
			if got := r.URL.Query().Get("tags"); got != "(story,job),(story_30,story_10,story_20,story_40)" {
				t.Errorf("tags = %q", got)
			}
			json.NewEncoder(w).Encode(SearchResult{Hits: []Hit{
				{ObjectID: "10", Title: "ten"},
				{ObjectID: "30", Title: "thirty"},
			}})
		case "/item/20.json":
			w.Write([]byte(`{"id":20,"type":"job","title":"twenty","by":"yc","score":1,"time":1700000000}`))
		case "/item/40.json":
			w.Write([]byte(`{"id":40,"dead":true,"title":"forty"}`))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := testClient(srv)
	result, err := c.Feed("top", 4)
	if err != nil {
		t.Fatalf("Feed() error: %v", err)
	}

	var titles []string
	for _, hit := range result.Hits {
		titles = append(titles, hit.Title)
	}
	if want := []string{"thirty", "ten", "twenty"}; !slices.Equal(titles, want) {
		t.Errorf("titles = %v, want %v", titles, want)
	}
	if result.Hits[2].CreatedAt != "2023-11-14T22:13:20Z" {
		t.Errorf("CreatedAt = %q", result.Hits[2].CreatedAt)
	}
}

func TestFeedUnknown(t *testing.T) {
	c := NewClient("", http.DefaultClient)
	if _, err := c.Feed("nope", 1); err == nil {
		t.Error("expected error for unknown feed")
	}
}

func TestRankComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/item/1.json":
			w.Write([]byte(`{"id":1,"kids":[4,2]}`))
		case "/item/4.json":
			w.Write([]byte(`{"id":4,"kids":[6,5]}`))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	item := &Item{ID: 1, Children: []Item{
		{ID: 2, Children: []Item{{ID: 8}, {ID: 7}}},
		{ID: 3},
		{ID: 4, Children: []Item{{ID: 5}, {ID: 6}}},
	}}

	c := testClient(srv)
	if err := c.RankComments(item, 1); err != nil {
		t.Fatalf("RankComments() error: %v", err)
	}

	ids := func(items []Item) []int {
		var out []int
		for _, it := range items {
			out = append(out, it.ID)
		}
		return out
	}
	if got, want := ids(item.Children), []int{4, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("comments = %v, want %v", got, want)
	}
	if got, want := ids(item.Children[0].Children), []int{6, 5}; !slices.Equal(got, want) {
		t.Errorf("ranked replies = %v, want %v", got, want)
	}
	if got, want := ids(item.Children[1].Children), []int{8, 7}; !slices.Equal(got, want) {
		t.Errorf("replies past n = %v, want %v", got, want)
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
