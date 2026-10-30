package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/voska/hn-cli/internal/api"
)

// FormatSearchResults prints search results in compact plaintext.
func FormatSearchResults(w io.Writer, result *api.SearchResult, isComments bool) {
	if len(result.Hits) == 0 {
		fmt.Fprintln(w, "No results.")
		return
	}

	fmt.Fprintf(w, "%d results:\n", result.NbHits)
	for i, hit := range result.Hits {
		if isComments {
			formatComment(w, i+1, hit)
		} else {
			formatStory(w, i+1, hit)
		}
	}
}

func formatStory(w io.Writer, n int, hit api.Hit) {
	ago := timeAgo(hit.CreatedAt)
	fmt.Fprintf(w, "%d. [%dpts %dc] %s  @%s  %s\n", n, hit.Points, hit.NumComments, hit.Title, hit.Author, ago)
	if hit.URL != "" {
		fmt.Fprintf(w, "   %s\n", hit.URL)
	}
	fmt.Fprintf(w, "   HN: https://news.ycombinator.com/item?id=%s\n", hit.ObjectID)
}

func formatComment(w io.Writer, n int, hit api.Hit) {
	ago := timeAgo(hit.CreatedAt)
	text := truncate(api.CleanHTML(hit.CommentText), 200)
	text = strings.ReplaceAll(text, "\n", " ")

	storyTitle := hit.StoryTitle
	if storyTitle == "" {
		storyTitle = fmt.Sprintf("story:%d", hit.StoryID)
	}
	fmt.Fprintf(w, "%d. @%s  %s  on: %s\n", n, hit.Author, ago, storyTitle)
	fmt.Fprintf(w, "   %s\n", text)
	fmt.Fprintf(w, "   HN: https://news.ycombinator.com/item?id=%s\n", hit.ObjectID)
}

// FormatFeed prints a story feed in compact plaintext.
func FormatFeed(w io.Writer, result *api.SearchResult, feed string) {
	if len(result.Hits) == 0 {
		fmt.Fprintln(w, "No stories.")
		return
	}

	title := "Front page"
	if feed != "top" {
		title = "Feed " + feed
	}
	fmt.Fprintf(w, "%s (%d stories):\n", title, len(result.Hits))
	for i, hit := range result.Hits {
		formatStory(w, i+1, hit)
	}
}

// ReadOptions caps the comments FormatItem prints.
type ReadOptions struct {
	Num      int // top comments, 0 for all
	Replies  int // replies per comment, negative for all
	Depth    int // levels of replies, negative for all
	MaxChars int // characters per comment, 0 for no cap
}

// FormatItem prints a story or comment with its comments.
func FormatItem(w io.Writer, item *api.Item, opts ReadOptions) {
	pts := 0
	if item.Points != nil {
		pts = *item.Points
	}
	ago := timeAgo(item.CreatedAt)

	if item.Title != "" {
		fmt.Fprintf(w, "%s\n", item.Title)
	}
	fmt.Fprintf(w, "@%s  %dpts  %s\n", item.Author, pts, ago)
	if item.URL != "" {
		fmt.Fprintf(w, "%s\n", item.URL)
	}
	fmt.Fprintf(w, "HN: https://news.ycombinator.com/item?id=%d\n", item.ID)

	if item.Text != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, api.CleanHTML(item.Text))
	}

	commentCount := countComments(item.Children)
	if commentCount == 0 {
		return
	}

	fmt.Fprintf(w, "\n--- %d comments ---\n", commentCount)
	limit := opts.Num
	if limit <= 0 {
		limit = -1
	}
	printComments(w, item.Children, 0, limit, opts)
}

// FormatUser prints user profile in compact plaintext.
func FormatUser(w io.Writer, user *api.User) {
	fmt.Fprintf(w, "@%s  karma:%d\n", user.Username, user.Karma)
	if user.About != "" {
		fmt.Fprintln(w, api.CleanHTML(user.About))
	}
	fmt.Fprintf(w, "https://news.ycombinator.com/user?id=%s\n", user.Username)
}

// JSON prints any value as JSON to stdout.
func JSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printComments(w io.Writer, children []api.Item, depth, limit int, opts ReadOptions) {
	indent := strings.Repeat("  ", depth)

	shown := 0
	for i, c := range children {
		if c.Author == "" && c.Text == "" {
			continue
		}
		if limit >= 0 && shown == limit {
			noun := "replies"
			if depth == 0 {
				noun = "comments"
			}
			fmt.Fprintf(w, "%s[%d more %s]\n", indent, countComments(children[i:]), noun)
			return
		}
		shown++

		ago := timeAgo(c.CreatedAt)
		fmt.Fprintf(w, "%s@%s  %s  #%d\n", indent, c.Author, ago, c.ID)
		text := truncate(api.CleanHTML(c.Text), opts.MaxChars)
		for _, line := range strings.Split(text, "\n") {
			fmt.Fprintf(w, "%s  %s\n", indent, line)
		}

		if opts.Depth >= 0 && depth >= opts.Depth {
			if collapsed := countComments(c.Children); collapsed > 0 {
				fmt.Fprintf(w, "%s  [%d more replies]\n", indent, collapsed)
			}
			continue
		}
		printComments(w, c.Children, depth+1, opts.Replies, opts)
	}
}

// truncate caps s at maxChars characters and says how many it dropped.
func truncate(s string, maxChars int) string {
	runes := []rune(s)
	if maxChars <= 0 || len(runes) <= maxChars {
		return s
	}
	return fmt.Sprintf("%s... [+%d chars]", string(runes[:maxChars]), len(runes)-maxChars)
}

func countComments(children []api.Item) int {
	count := 0
	for _, c := range children {
		if c.Author != "" || c.Text != "" {
			count++
		}
		count += countComments(c.Children)
	}
	return count
}

// timeAgo parses an HN timestamp and returns a human-readable relative time.
// The Algolia API returns RFC 3339 from search endpoints but uses a
// millisecond-precision variant (2006-01-02T15:04:05.000Z) from item endpoints,
// so both formats must be attempted.
func timeAgo(created string) string {
	t, err := time.Parse(time.RFC3339, created)
	if err != nil {
		t2, err2 := time.Parse("2006-01-02T15:04:05.000Z", created)
		if err2 != nil {
			return created
		}
		t = t2
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}
