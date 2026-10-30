package cmd

import (
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

// ErrNotFound signals a missing resource (exit code 5).
type ErrNotFound struct{ msg string }

func (e ErrNotFound) Error() string { return e.msg }

// ErrRetryable signals a transient failure (exit code 8).
type ErrRetryable struct{ msg string }

func (e ErrRetryable) Error() string { return e.msg }

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
	switch err.(type) {
	case ErrNoResults:
		return 3
	case ErrNotFound:
		return 5
	case ErrRetryable:
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
	searchPage     int
	searchAfter    string
	searchBefore   string
	searchMinPts   int
	searchTag      string
	searchBy       string
	searchURL      bool
)

// parseDate reads a YYYY-MM-DD flag value, or returns nil when it is unset.
func parseDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q (use YYYY-MM-DD)", value)
	}
	return &t, nil
}

var searchCmd = &cobra.Command{
	Use:     "search [QUERY]",
	Aliases: []string{"s"},
	Short:   "Search HN stories or comments",
	Long: `Search HN stories or comments. The query is optional, so filters alone can
browse: hn search --tag show_hn --min-points 100 --sort date`,
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		tags := []string{"story"}
		if searchComments {
			tags = []string{"comment"}
		}
		if searchTag != "" {
			tags = append(tags, searchTag)
		}
		if searchBy != "" {
			tags = append(tags, "author_"+searchBy)
		}

		after, err := parseDate(searchAfter)
		if err != nil {
			return err
		}
		before, err := parseDate(searchBefore)
		if err != nil {
			return err
		}

		result, err := client().Search(api.SearchOptions{
			Query:      query,
			Tags:       strings.Join(tags, ","),
			SortByDate: searchSort == "date",
			NumResults: searchNum,
			Page:       searchPage,
			URLOnly:    searchURL,
			AfterTime:  after,
			BeforeTime: before,
			MinPoints:  searchMinPts,
		})
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
	searchCmd.Flags().IntVar(&searchPage, "page", 0, "Result page, starting at 0")
	searchCmd.Flags().StringVar(&searchAfter, "after", "", "Only results after date (YYYY-MM-DD)")
	searchCmd.Flags().StringVar(&searchBefore, "before", "", "Only results before date (YYYY-MM-DD)")
	searchCmd.Flags().StringVar(&searchTag, "tag", "", "Only this Algolia tag: show_hn, ask_hn, front_page")
	searchCmd.Flags().StringVar(&searchBy, "by", "", "Only this author")
	searchCmd.Flags().BoolVar(&searchURL, "url", false, "Match the query against story URLs only")
	searchCmd.Flags().IntVar(&searchMinPts, "min-points", 0, "Minimum points filter")
}

// --- front ---

var frontNum int

var frontCmd = &cobra.Command{
	Use:     "front",
	Aliases: []string{"f"},
	Short:   "Current front page stories",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := client().Search(api.SearchOptions{
			Tags:       "front_page",
			NumResults: frontNum,
		})
		if err != nil {
			return err
		}

		if jsonOutput {
			return output.JSON(result)
		}
		output.FormatFrontPage(os.Stdout, result)
		return nil
	},
}

func init() {
	frontCmd.Flags().IntVarP(&frontNum, "num", "n", 30, "Number of stories")
}

// --- read ---

var readExpand bool

var readCmd = &cobra.Command{
	Use:     "read ID",
	Aliases: []string{"r"},
	Short:   "Read a story with comments",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		item, err := client().GetItem(args[0])
		if err != nil {
			return ErrNotFound{msg: err.Error()}
		}

		if jsonOutput {
			return output.JSON(item)
		}
		output.FormatItem(os.Stdout, item, readExpand)
		return nil
	},
}

func init() {
	readCmd.Flags().BoolVar(&readExpand, "expand", false, "Show all comments expanded")
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
			return ErrNotFound{msg: err.Error()}
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
				return output.JSON(map[string]any{"status": "error", "error": err.Error()})
			}
			return ErrRetryable{msg: fmt.Sprintf("HN API: DOWN (%v)", err)}
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
