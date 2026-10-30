package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// BaseURL is the default HN Algolia API endpoint.
	BaseURL = "https://hn.algolia.com/api/v1"
	// FirebaseURL is the default official HN API endpoint.
	FirebaseURL = "https://hacker-news.firebaseio.com/v0"
)

const (
	// feedChunk is how many story IDs fit in one Algolia tag filter.
	feedChunk = 100
	// maxConcurrent caps parallel Firebase requests.
	maxConcurrent = 10
)

// feedLists maps feed names to Firebase story lists.
var feedLists = map[string]string{
	"top":  "topstories",
	"new":  "newstories",
	"best": "beststories",
	"ask":  "askstories",
	"show": "showstories",
	"job":  "jobstories",
}

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

// firebaseItem is an item from the Firebase API.
type firebaseItem struct {
	ID          int    `json:"id"`
	By          string `json:"by"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
	Descendants int    `json:"descendants"`
	Time        int64  `json:"time"`
	Dead        bool   `json:"dead"`
	Deleted     bool   `json:"deleted"`
	Kids        []int  `json:"kids"`
}

func (f firebaseItem) hit() Hit {
	return Hit{
		ObjectID:    strconv.Itoa(f.ID),
		Title:       f.Title,
		URL:         f.URL,
		Author:      f.By,
		Points:      f.Score,
		NumComments: f.Descendants,
		CreatedAt:   time.Unix(f.Time, 0).UTC().Format(time.RFC3339),
		CreatedAtI:  f.Time,
	}
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

// FeedNames returns the valid feed names, sorted.
func FeedNames() []string {
	names := make([]string, 0, len(feedLists))
	for name := range feedLists {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Feed returns the first n stories of a feed in site rank order. The order
// comes from Firebase and the details from Algolia, with single Firebase
// lookups for what Algolia lacks: job posts and stories not indexed yet.
func (c *Client) Feed(feed string, n int) (*SearchResult, error) {
	list, ok := feedLists[feed]
	if !ok {
		return nil, fmt.Errorf("unknown feed %q (use %s)", feed, strings.Join(FeedNames(), ", "))
	}

	var ids []int
	if err := c.getJSON(fmt.Sprintf("%s/%s.json", c.firebaseURL, list), &ids); err != nil {
		return nil, fmt.Errorf("feed %s: %w", feed, err)
	}
	if n > 0 && n < len(ids) {
		ids = ids[:n]
	}

	found := make(map[string]Hit, len(ids))
	for start := 0; start < len(ids); start += feedChunk {
		chunk := ids[start:min(start+feedChunk, len(ids))]
		tags := make([]string, len(chunk))
		for i, id := range chunk {
			tags[i] = fmt.Sprintf("story_%d", id)
		}
		result, err := c.Search(SearchOptions{
			Tags:       "(story,job),(" + strings.Join(tags, ",") + ")",
			NumResults: len(chunk),
		})
		if err != nil {
			return nil, err
		}
		for _, hit := range result.Hits {
			found[hit.ObjectID] = hit
		}
	}

	if err := c.fillMissing(ids, found); err != nil {
		return nil, err
	}

	hits := make([]Hit, 0, len(ids))
	for _, id := range ids {
		if hit, ok := found[strconv.Itoa(id)]; ok {
			hits = append(hits, hit)
		}
	}
	return &SearchResult{Hits: hits, NbHits: len(hits), HitsPerPage: len(ids)}, nil
}

// fillMissing adds to found the ids it lacks, looked up on Firebase. Dead and
// deleted items stay out.
func (c *Client) fillMissing(ids []int, found map[string]Hit) error {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		slots    = make(chan struct{}, maxConcurrent)
	)
	for _, id := range ids {
		if _, ok := found[strconv.Itoa(id)]; ok {
			continue
		}
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-slots }()

			fb, err := c.firebaseItem(id)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				if firstErr == nil {
					firstErr = err
				}
			case fb != nil && !fb.Dead && !fb.Deleted:
				found[strconv.Itoa(id)] = fb.hit()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

// firebaseItem fetches one item from the Firebase API. It returns nil when the
// item does not exist.
func (c *Client) firebaseItem(id int) (*firebaseItem, error) {
	var fb *firebaseItem
	if err := c.getJSON(fmt.Sprintf("%s/item/%d.json", c.firebaseURL, id), &fb); err != nil {
		return nil, fmt.Errorf("item %d: %w", id, err)
	}
	return fb, nil
}

// RankComments sorts item's comments into site rank order, and the replies of
// the first n comments too. Algolia returns comments oldest first, and only
// Firebase knows the ranking. Deeper replies stay oldest first.
func (c *Client) RankComments(item *Item, n int) error {
	if err := c.rankChildren(item); err != nil {
		return err
	}

	children := item.Children
	if n <= 0 {
		return nil
	}
	if n < len(children) {
		children = children[:n]
	}

	var (
		wg    sync.WaitGroup
		slots = make(chan struct{}, maxConcurrent)
	)
	for i := range children {
		if len(children[i].Children) < 2 {
			continue
		}
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			// Replies are only reordered, so a failed lookup costs nothing.
			_ = c.rankChildren(&children[i])
		}()
	}
	wg.Wait()
	return nil
}

func (c *Client) rankChildren(item *Item) error {
	fb, err := c.firebaseItem(item.ID)
	if err != nil {
		return err
	}
	if fb == nil {
		return fmt.Errorf("item %d: %w", item.ID, ErrNotFound)
	}

	rank := make(map[int]int, len(fb.Kids))
	for i, id := range fb.Kids {
		rank[id] = i
	}
	position := func(id int) int {
		if r, ok := rank[id]; ok {
			return r
		}
		return len(fb.Kids)
	}
	sort.SliceStable(item.Children, func(i, j int) bool {
		return position(item.Children[i].ID) < position(item.Children[j].ID)
	})
	return nil
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
