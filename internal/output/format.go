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
	text := api.CleanHTML(hit.CommentText)
	if len(text) > 200 {
		text = text[:200] + "..."
	}
	text = strings.ReplaceAll(text, "\n", " ")

	storyTitle := hit.StoryTitle
	if storyTitle == "" {
		storyTitle = fmt.Sprintf("story:%d", hit.StoryID)
	}
	fmt.Fprintf(w, "%d. @%s  %s  on: %s\n", n, hit.Author, ago, storyTitle)
	fmt.Fprintf(w, "   %s\n", text)
	fmt.Fprintf(w, "   HN: https://news.ycombinator.com/item?id=%s\n", hit.ObjectID)
}

// FormatFrontPage prints front page stories in compact plaintext.
func FormatFrontPage(w io.Writer, result *api.SearchResult) {
	if len(result.Hits) == 0 {
		fmt.Fprintln(w, "No stories on front page.")
		return
	}

	fmt.Fprintf(w, "Front page (%d stories):\n", len(result.Hits))
	for i, hit := range result.Hits {
		formatStory(w, i+1, hit)
	}
}

// FormatItem prints a story with its comments.
func FormatItem(w io.Writer, item *api.Item, expand bool) {
	pts := 0
	if item.Points != nil {
		pts = *item.Points
	}
	ago := timeAgo(item.CreatedAt)

	fmt.Fprintf(w, "%s\n", item.Title)
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
	if expand {
		printComments(w, item.Children, 0, -1)
	} else {
		printComments(w, item.Children, 0, 3)
	}
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

func printComments(w io.Writer, children []api.Item, depth int, maxDepth int) {
	if maxDepth >= 0 && depth > maxDepth {
		return
	}

	indent := strings.Repeat("  ", depth)
	for _, c := range children {
		if c.Author == "" && c.Text == "" {
			continue
		}
		ago := timeAgo(c.CreatedAt)
		fmt.Fprintf(w, "%s@%s  %s\n", indent, c.Author, ago)
		text := api.CleanHTML(c.Text)
		for _, line := range strings.Split(text, "\n") {
			fmt.Fprintf(w, "%s  %s\n", indent, line)
		}

		if len(c.Children) > 0 {
			if maxDepth >= 0 && depth+1 > maxDepth {
				collapsed := countComments(c.Children)
				if collapsed > 0 {
					fmt.Fprintf(w, "%s  [%d more replies]\n", indent, collapsed)
				}
			} else {
				printComments(w, c.Children, depth+1, maxDepth)
			}
		}
	}
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
