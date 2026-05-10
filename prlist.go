package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	gh "github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/repository"
)

const jsonFields = "number,title,author,state,isDraft,reviewDecision,statusCheckRollup,updatedAt,headRefName,baseRefName,url"

type listOptions struct {
	repo      string
	limit     int
	state     string
	author    string
	assignee  string
	app       string
	base      string
	head      string
	search    string
	draftOnly bool
	web       bool
	json      bool
	labels    stringSliceFlag
}

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type pullRequest struct {
	Number            int         `json:"number"`
	Title             string      `json:"title"`
	State             string      `json:"state"`
	IsDraft           bool        `json:"isDraft"`
	ReviewDecision    string      `json:"reviewDecision"`
	StatusCheckRollup []checkItem `json:"statusCheckRollup"`
	UpdatedAt         time.Time   `json:"updatedAt"`
	HeadRefName       string      `json:"headRefName"`
	BaseRefName       string      `json:"baseRefName"`
	URL               string      `json:"url"`
	Author            *author     `json:"author"`
}

type author struct {
	Login string `json:"login"`
}

// checkItem represents a single entry in the statusCheckRollup array.
// CheckRun items use Status+Conclusion; StatusContext items use State.
type checkItem struct {
	Typename   string `json:"__typename"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	State      string `json:"state"`
}

type displayPullRequest struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Author  string `json:"author"`
	State   string `json:"state"`
	Review  string `json:"review"`
	Checks  string `json:"checks"`
	Branch  string `json:"branch"`
	Updated string `json:"updated"`
	URL     string `json:"url"`
}

func defaultListOptions() listOptions {
	return listOptions{
		limit: 30,
		state: "open",
	}
}

func executeList(options listOptions, stdout io.Writer) error {
	arguments := buildListArgs(options)
	commandOutput, commandError, err := gh.Exec(arguments...)
	if err != nil {
		if commandError.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(commandError.String()))
		}

		return err
	}

	if options.web {
		return nil
	}

	var pullRequests []pullRequest
	if err := json.Unmarshal(commandOutput.Bytes(), &pullRequests); err != nil {
		return fmt.Errorf("decode gh pr list output: %w", err)
	}

	rendered := make([]displayPullRequest, 0, len(pullRequests))
	now := time.Now().UTC()
	for _, pullRequest := range pullRequests {
		rendered = append(rendered, buildDisplayPullRequest(pullRequest, now))
	}

	if options.json {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(rendered)
	}

	return renderTable(stdout, options, rendered)
}

func buildListArgs(options listOptions) []string {
	arguments := []string{"pr", "list"}

	if options.web {
		arguments = append(arguments, "--web")
	} else {
		arguments = append(arguments, "--json", jsonFields)
	}

	if options.repo != "" {
		arguments = append(arguments, "--repo", options.repo)
	}

	arguments = append(arguments, "--limit", fmt.Sprintf("%d", options.limit))
	arguments = append(arguments, "--state", options.state)

	if options.author != "" {
		arguments = append(arguments, "--author", options.author)
	}

	if options.assignee != "" {
		arguments = append(arguments, "--assignee", options.assignee)
	}

	if options.app != "" {
		arguments = append(arguments, "--app", options.app)
	}

	if options.base != "" {
		arguments = append(arguments, "--base", options.base)
	}

	if options.head != "" {
		arguments = append(arguments, "--head", options.head)
	}

	if options.search != "" {
		arguments = append(arguments, "--search", options.search)
	}

	if options.draftOnly {
		arguments = append(arguments, "--draft")
	}

	for _, label := range options.labels {
		arguments = append(arguments, "--label", label)
	}

	return arguments
}

func buildDisplayPullRequest(pullRequest pullRequest, now time.Time) displayPullRequest {
	authorName := "-"
	if pullRequest.Author != nil && pullRequest.Author.Login != "" {
		authorName = pullRequest.Author.Login
	}

	return displayPullRequest{
		Number:  pullRequest.Number,
		Title:   trimTitle(pullRequest.Title, 56),
		Author:  authorName,
		State:   normalizeState(pullRequest.State, pullRequest.IsDraft),
		Review:  normalizeReviewDecision(pullRequest.ReviewDecision),
		Checks:  normalizeCheckState(pullRequest.StatusCheckRollup),
		Branch:  formatBranch(pullRequest.HeadRefName, pullRequest.BaseRefName),
		Updated: formatRelativeTime(pullRequest.UpdatedAt, now),
		URL:     pullRequest.URL,
	}
}

func renderTable(stdout io.Writer, options listOptions, pullRequests []displayPullRequest) error {
	if len(pullRequests) == 0 {
		fmt.Fprintln(stdout, "No pull requests found.")
		return nil
	}

	if repoLabel := resolveRepoLabel(options.repo); repoLabel != "" {
		fmt.Fprintf(stdout, "Pull requests for %s\n\n", repoLabel)
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "#\tTitle\tAuthor\tState\tReview\tChecks\tBranch\tUpdated")
	for _, pullRequest := range pullRequests {
		fmt.Fprintf(
			writer,
			"#%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			pullRequest.Number,
			pullRequest.Title,
			pullRequest.Author,
			pullRequest.State,
			pullRequest.Review,
			pullRequest.Checks,
			pullRequest.Branch,
			pullRequest.Updated,
		)
	}

	return writer.Flush()
}

func resolveRepoLabel(repoOverride string) string {
	if repoOverride != "" {
		return repoOverride
	}

	repo, err := repository.Current()
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
}

func normalizeState(state string, isDraft bool) string {
	if isDraft {
		return "draft"
	}

	switch strings.ToUpper(state) {
	case "OPEN":
		return "open"
	case "CLOSED":
		return "closed"
	case "MERGED":
		return "merged"
	default:
		if state == "" {
			return "-"
		}

		return strings.ToLower(state)
	}
}

func normalizeReviewDecision(reviewDecision string) string {
	switch strings.ToUpper(reviewDecision) {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "changes"
	case "REVIEW_REQUIRED":
		return "review"
	case "":
		return "-"
	default:
		return strings.ToLower(reviewDecision)
	}
}

func normalizeCheckState(items []checkItem) string {
	if len(items) == 0 {
		return "-"
	}

	hasFail := false
	hasPending := false
	for _, item := range items {
		switch {
		case item.Typename == "StatusContext":
			switch strings.ToUpper(item.State) {
			case "ERROR", "FAILURE":
				hasFail = true
			case "EXPECTED", "PENDING":
				hasPending = true
			}
		default: // CheckRun
			switch strings.ToUpper(item.Conclusion) {
			case "FAILURE", "TIMED_OUT", "STARTUP_FAILURE", "ACTION_REQUIRED":
				hasFail = true
			case "":
				// No conclusion yet — still running
				hasPending = true
			}
			if strings.ToUpper(item.Status) != "COMPLETED" {
				hasPending = true
			}
		}
	}

	switch {
	case hasFail:
		return "fail"
	case hasPending:
		return "pending"
	default:
		return "pass"
	}
}

func formatBranch(head string, base string) string {
	switch {
	case head == "" && base == "":
		return "-"
	case base == "":
		return head
	case head == "":
		return base
	default:
		return fmt.Sprintf("%s -> %s", head, base)
	}
}

func formatRelativeTime(updatedAt time.Time, now time.Time) string {
	if updatedAt.IsZero() {
		return "-"
	}

	if now.Before(updatedAt) {
		return "0m"
	}

	age := now.Sub(updatedAt)
	switch {
	case age < time.Minute:
		return fmt.Sprintf("%ds", int(age.Seconds()))
	case age < time.Hour:
		return fmt.Sprintf("%dm", int(age.Minutes()))
	case age < 24*time.Hour:
		return fmt.Sprintf("%dh", int(age.Hours()))
	case age < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(age.Hours()/24))
	case age < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(age.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy", int(age.Hours()/(24*365)))
	}
}

func trimTitle(title string, limit int) string {
	title = strings.TrimSpace(title)
	if limit <= 0 || len(title) <= limit {
		return title
	}

	if limit <= 3 {
		return title[:limit]
	}

	return title[:limit-3] + "..."
}
