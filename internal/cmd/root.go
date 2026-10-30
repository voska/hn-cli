package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/voska/hn-cli/internal/api"
	"github.com/voska/hn-cli/internal/output"
)

var (
	appVersion = "dev"
	appCommit  = ""
	appDate    = ""

	jsonOutput bool
	hnClient   *api.Client
)

// ErrNoResults signals an empty result set (exit code 3).
type ErrNoResults struct{}

func (e ErrNoResults) Error() string { return "no results" }

func SetVersion(v, c, d string) {
	appVersion = v
	appCommit = c
	appDate = d
	rootCmd.Version = v
}

var rootCmd = &cobra.Command{
	Use:           "hn",
	Short:         "Hacker News CLI - search, read, and browse HN",
	Long:          "Agent-friendly CLI for Hacker News via the Algolia API.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "JSON output")

	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(frontCmd)
	rootCmd.AddCommand(readCmd)
	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}

// ExitCode maps sentinel errors to process exit codes.
func ExitCode(err error) int {
	switch {
	case errors.As(err, &ErrNoResults{}):
		return 3
	case errors.Is(err, api.ErrNotFound):
		return 5
	case errors.Is(err, api.ErrRetryable):
		return 8
	default:
		return 1
	}
}

func client() *api.Client {
	if hnClient == nil {
		hnClient = api.DefaultClient()
	}
	return hnClient
}

// --- search ---

var (
	searchComments bool
	searchSort     string
	searchNum      int
	searchAfter    string
	searchMinPts   int
)

var searchCmd = &cobra.Command{
	Use:     "search QUERY",
	Aliases: []string{"s"},
	Short:   "Search HN stories or comments",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		tags := "story"
		if searchComments {
			tags = "comment"
		}

		opts := api.SearchOptions{
			Query:      query,
			Tags:       tags,
			SortByDate: searchSort == "date",
			NumResults: searchNum,
			MinPoints:  searchMinPts,
		}

		if searchAfter != "" {
			t, err := time.Parse("2006-01-02", searchAfter)
			if err != nil {
				return fmt.Errorf("invalid date %q (use YYYY-MM-DD)", searchAfter)
			}
			opts.AfterTime = &t
		}

		result, err := client().Search(opts)
		if err != nil {
			return err
		}

		if len(result.Hits) == 0 {
			return ErrNoResults{}
		}

		if jsonOutput {
			return output.JSON(result)
		}
		output.FormatSearchResults(os.Stdout, result, searchComments)
		return nil
	},
}

func init() {
	searchCmd.Flags().BoolVar(&searchComments, "comments", false, "Search comments instead of stories")
	searchCmd.Flags().StringVar(&searchSort, "sort", "relevance", "Sort order: relevance or date")
	searchCmd.Flags().IntVarP(&searchNum, "num", "n", 20, "Number of results")
	searchCmd.Flags().StringVar(&searchAfter, "after", "", "Only results after date (YYYY-MM-DD)")
	searchCmd.Flags().IntVar(&searchMinPts, "min-points", 0, "Minimum points filter")
}

// --- front ---

var (
	frontNum  int
	frontFeed string
)

var frontCmd = &cobra.Command{
	Use:     "front",
	Aliases: []string{"f"},
	Short:   "Stories in site rank order",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := client().Feed(frontFeed, frontNum)
		if err != nil {
			return err
		}

		if len(result.Hits) == 0 {
			return ErrNoResults{}
		}

		if jsonOutput {
			return output.JSON(result)
		}
		output.FormatFeed(os.Stdout, result, frontFeed)
		return nil
	},
}

func init() {
	frontCmd.Flags().IntVarP(&frontNum, "num", "n", 30, "Number of stories")
	frontCmd.Flags().StringVar(&frontFeed, "feed", "top", "Feed: "+strings.Join(api.FeedNames(), ", "))
}

// --- read ---

var (
	readNum      int
	readReplies  int
	readDepth    int
	readMaxChars int
	readExpand   bool
)

var readCmd = &cobra.Command{
	Use:     "read ID",
	Aliases: []string{"r"},
	Short:   "Read a story or comment with its top comments",
	Long: `Read a story or comment with its comments in site rank order. The output is
capped to fit an agent's context, and names what it leaves out. Comment IDs
open a collapsed thread: hn read COMMENT_ID.

--json returns the whole tree in rank order and ignores the caps.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		item, err := client().GetItem(args[0])
		if err != nil {
			return err
		}

		opts := output.ReadOptions{Num: readNum, Replies: readReplies, Depth: readDepth, MaxChars: readMaxChars}
		if readExpand {
			opts = output.ReadOptions{Replies: -1, Depth: -1}
		}

		if err := client().RankComments(item, opts.Num); err != nil {
			fmt.Fprintf(os.Stderr, "warning: comments are oldest first: %v\n", err)
		}

		if jsonOutput {
			return output.JSON(item)
		}
		output.FormatItem(os.Stdout, item, opts)
		return nil
	},
}

func init() {
	readCmd.Flags().IntVarP(&readNum, "num", "n", 20, "Top comments to show, 0 for all")
	readCmd.Flags().IntVar(&readReplies, "replies", 3, "Replies to show per comment")
	readCmd.Flags().IntVar(&readDepth, "depth", 1, "Levels of replies to show")
	readCmd.Flags().IntVar(&readMaxChars, "max-chars", 800, "Characters per comment, 0 for no cap")
	readCmd.Flags().BoolVar(&readExpand, "expand", false, "Show every comment in full")
}

// --- user ---

var userCmd = &cobra.Command{
	Use:     "user USERNAME",
	Aliases: []string{"u"},
	Short:   "User profile and stats",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		user, err := client().GetUser(args[0])
		if err != nil {
			return err
		}

		if jsonOutput {
			return output.JSON(user)
		}
		output.FormatUser(os.Stdout, user)
		return nil
	},
}

// --- status ---

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "API health check",
	RunE: func(cmd *cobra.Command, args []string) error {
		latency, err := client().HealthCheck()
		if err != nil {
			if jsonOutput {
				if jsonErr := output.JSON(map[string]any{"status": "error", "error": err.Error()}); jsonErr != nil {
					return jsonErr
				}
			}
			return fmt.Errorf("HN API: DOWN (%w)", err)
		}

		if jsonOutput {
			return output.JSON(map[string]any{
				"status":     "ok",
				"latency_ms": latency.Milliseconds(),
				"api_url":    api.BaseURL,
			})
		}
		fmt.Fprintf(os.Stdout, "HN API: OK (%dms)\n", latency.Milliseconds())
		return nil
	},
}

// --- version ---

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version info",
	Run: func(cmd *cobra.Command, args []string) {
		if jsonOutput {
			_ = output.JSON(map[string]string{
				"version": appVersion,
				"commit":  appCommit,
				"date":    appDate,
			})
			return
		}
		fmt.Printf("hn %s", appVersion)
		if appCommit != "" {
			fmt.Printf(" (%s)", appCommit)
		}
		if appDate != "" {
			fmt.Printf(" built %s", appDate)
		}
		fmt.Println()
	},
}
