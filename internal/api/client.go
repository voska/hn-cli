package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// BaseURL is the default HN Algolia API endpoint.
	BaseURL = "https://hn.algolia.com/api/v1"
	// FirebaseURL is the default official HN API endpoint.
	FirebaseURL = "https://hacker-news.firebaseio.com/v0"
)

var (
	// ErrNotFound means the requested item or user does not exist.
	ErrNotFound = errors.New("not found")
	// ErrRetryable means a network failure, rate limit or server error.
	ErrRetryable = errors.New("temporary failure")
)

// Client wraps an HTTP client and base URLs for the HN APIs.
type Client struct {
	baseURL     string
	firebaseURL string
	httpClient  *http.Client
}

// NewClient returns a Client configured with the given base URL and HTTP client.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:     baseURL,
		firebaseURL: FirebaseURL,
		httpClient:  httpClient,
	}
}

// DefaultClient returns a Client using the production APIs and a 15s timeout.
func DefaultClient() *Client {
	return NewClient(BaseURL, &http.Client{Timeout: 15 * time.Second})
}

// getJSON fetches reqURL and decodes the response into v.
func (c *Client) getJSON(reqURL string, v any) error {
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRetryable, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return fmt.Errorf("%w: api returned %d", ErrRetryable, resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("api returned %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}
	return nil
}

// SearchResult represents the Algolia search response.
type SearchResult struct {
	Hits        []Hit  `json:"hits"`
	NbHits      int    `json:"nbHits"`
	NbPages     int    `json:"nbPages"`
	Page        int    `json:"page"`
	Query       string `json:"query"`
	HitsPerPage int    `json:"hitsPerPage"`
}

// Hit represents a single search result.
type Hit struct {
	ObjectID    string   `json:"objectID"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Author      string   `json:"author"`
	Points      int      `json:"points"`
	NumComments int      `json:"num_comments"`
	CreatedAt   string   `json:"created_at"`
	CreatedAtI  int64    `json:"created_at_i"`
	StoryID     int      `json:"story_id"`
	StoryTitle  string   `json:"story_title"`
	StoryURL    string   `json:"story_url"`
	CommentText string   `json:"comment_text"`
	Tags        []string `json:"_tags"`
}

// Item represents a story or comment from the items API.
type Item struct {
	ID        int    `json:"id"`
	Author    string `json:"author"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Text      string `json:"text"`
	Points    *int   `json:"points"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	ParentID  *int   `json:"parent_id"`
	StoryID   int    `json:"story_id"`
	Children  []Item `json:"children"`
}

// User represents an HN user profile.
type User struct {
	Username string `json:"username"`
	About    string `json:"about"`
	Karma    int    `json:"karma"`
}

// SearchOptions configures a search request.
type SearchOptions struct {
	Query      string
	Tags       string
	SortByDate bool
	NumResults int
	AfterTime  *time.Time
	MinPoints  int
}

// BuildURL constructs the full API URL for the given search options.
func (c *Client) BuildURL(opts SearchOptions) string {
	endpoint := "/search"
	if opts.SortByDate {
		endpoint = "/search_by_date"
	}

	params := url.Values{}
	if opts.Query != "" {
		params.Set("query", opts.Query)
	}
	if opts.Tags != "" {
		params.Set("tags", opts.Tags)
	}
	if opts.NumResults > 0 {
		params.Set("hitsPerPage", strconv.Itoa(opts.NumResults))
	}

	var numericFilters []string
	if opts.AfterTime != nil {
		numericFilters = append(numericFilters, fmt.Sprintf("created_at_i>%d", opts.AfterTime.Unix()))
	}
	if opts.MinPoints > 0 {
		numericFilters = append(numericFilters, fmt.Sprintf("points>%d", opts.MinPoints))
	}
	if len(numericFilters) > 0 {
		params.Set("numericFilters", strings.Join(numericFilters, ","))
	}

	return fmt.Sprintf("%s%s?%s", c.baseURL, endpoint, params.Encode())
}

// Search queries the HN Algolia search API.
func (c *Client) Search(opts SearchOptions) (*SearchResult, error) {
	var result SearchResult
	if err := c.getJSON(c.BuildURL(opts), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetItem fetches a single item (story/comment) with its children.
func (c *Client) GetItem(id string) (*Item, error) {
	var item Item
	if err := c.getJSON(fmt.Sprintf("%s/items/%s", c.baseURL, url.PathEscape(id)), &item); err != nil {
		return nil, fmt.Errorf("item %s: %w", id, err)
	}
	return &item, nil
}

// GetUser fetches a user profile. It uses the Firebase API because Algolia
// answers 500 for unknown users, which looks like an outage.
func (c *Client) GetUser(username string) (*User, error) {
	var profile *struct {
		ID    string `json:"id"`
		About string `json:"about"`
		Karma int    `json:"karma"`
	}
	reqURL := fmt.Sprintf("%s/user/%s.json", c.firebaseURL, url.PathEscape(username))
	if err := c.getJSON(reqURL, &profile); err != nil {
		return nil, fmt.Errorf("user %s: %w", username, err)
	}
	if profile == nil {
		return nil, fmt.Errorf("user %s: %w", username, ErrNotFound)
	}
	return &User{Username: profile.ID, About: profile.About, Karma: profile.Karma}, nil
}

// HealthCheck pings the API and returns latency.
func (c *Client) HealthCheck() (time.Duration, error) {
	start := time.Now()
	resp, err := c.httpClient.Get(c.baseURL + "/search?query=test&hitsPerPage=0")
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrRetryable, err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		return elapsed, fmt.Errorf("%w: api returned %d", ErrRetryable, resp.StatusCode)
	}
	return elapsed, nil
}

// CleanHTML strips HTML entities and basic tags from HN text.
func CleanHTML(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "<p>", "\n")
	s = strings.ReplaceAll(s, "</p>", "")
	s = strings.ReplaceAll(s, "<i>", "")
	s = strings.ReplaceAll(s, "</i>", "")
	s = strings.ReplaceAll(s, "<b>", "")
	s = strings.ReplaceAll(s, "</b>", "")
	s = strings.ReplaceAll(s, "<pre><code>", "\n```\n")
	s = strings.ReplaceAll(s, "</code></pre>", "\n```\n")
	s = strings.ReplaceAll(s, "<code>", "`")
	s = strings.ReplaceAll(s, "</code>", "`")
	s = strings.ReplaceAll(s, "<a href=\"", "")
	s = strings.ReplaceAll(s, "\" rel=\"nofollow\">", " ")
	s = strings.ReplaceAll(s, "</a>", "")
	return strings.TrimSpace(s)
}
