package api

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://hn.algolia.com/api/v1"

var httpClient = &http.Client{Timeout: 15 * time.Second}

// SearchResult represents the Algolia search response.
type SearchResult struct {
	Hits      []Hit  `json:"hits"`
	NbHits    int    `json:"nbHits"`
	NbPages   int    `json:"nbPages"`
	Page      int    `json:"page"`
	Query     string `json:"query"`
	HitsPerPage int  `json:"hitsPerPage"`
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
	ID        int     `json:"id"`
	Author    string  `json:"author"`
	Title     string  `json:"title"`
	URL       string  `json:"url"`
	Text      string  `json:"text"`
	Points    *int    `json:"points"`
	Type      string  `json:"type"`
	CreatedAt string  `json:"created_at"`
	ParentID  *int    `json:"parent_id"`
	StoryID   int     `json:"story_id"`
	Children  []Item  `json:"children"`
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
	Tags       string // e.g. "story", "comment", "front_page"
	SortByDate bool
	NumResults int
	AfterTime  *time.Time
	MinPoints  int
}

// Search queries the HN Algolia search API.
func Search(opts SearchOptions) (*SearchResult, error) {
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

	reqURL := fmt.Sprintf("%s%s?%s", baseURL, endpoint, params.Encode())
	resp, err := httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	return &result, nil
}

// GetItem fetches a single item (story/comment) with its children.
func GetItem(id string) (*Item, error) {
	reqURL := fmt.Sprintf("%s/items/%s", baseURL, id)
	resp, err := httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("item %s not found", id)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned %d", resp.StatusCode)
	}

	var item Item
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	return &item, nil
}

// GetUser fetches a user profile.
func GetUser(username string) (*User, error) {
	reqURL := fmt.Sprintf("%s/users/%s", baseURL, url.PathEscape(username))
	resp, err := httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user %q not found", username)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned %d", resp.StatusCode)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	return &user, nil
}

// HealthCheck pings the API and returns latency.
func HealthCheck() (time.Duration, error) {
	start := time.Now()
	resp, err := httpClient.Get(baseURL + "/search?query=test&hitsPerPage=0")
	if err != nil {
		return 0, fmt.Errorf("api unreachable: %w", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		return elapsed, fmt.Errorf("api returned %d", resp.StatusCode)
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
