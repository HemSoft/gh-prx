package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	gh "github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

const jsonFields = "number,title,author,state,isDraft,reviewDecision,statusCheckRollup,updatedAt,headRefName,baseRefName,url,latestReviews"

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
	LatestReviews     []review    `json:"latestReviews"`
}

type author struct {
	Login string `json:"login"`
}

type review struct {
	State  string  `json:"state"`
	Author *author `json:"author"`
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
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	State    string `json:"state"`
	Review   string `json:"review"`
	Approvals int   `json:"approvals"`
	Checks   string `json:"checks"`
	Comments string `json:"comments"`
	AIReview string `json:"aiReview"`
	Branch   string `json:"branch"`
	Updated  string `json:"updated"`
	URL      string `json:"url"`
}

type tableCell struct {
	text   string // plain text for width calculation
	styled string // styled text for display (may contain ANSI codes)
}

type tableStyler struct {
	output *termenv.Output
}

func newTableStyler(w io.Writer, colorEnabled bool) tableStyler {
	profile := termenv.Ascii
	if colorEnabled {
		profile = termenv.ANSI
	}
	output := termenv.NewOutput(w, termenv.WithProfile(profile))
	return tableStyler{output: output}
}

func (s tableStyler) colored(text string, color termenv.ANSIColor) tableCell {
	return tableCell{
		text:   text,
		styled: s.output.String(text).Foreground(color).String(),
	}
}

func (s tableStyler) dim(text string) tableCell {
	return tableCell{
		text:   text,
		styled: s.output.String(text).Faint().String(),
	}
}

func (s tableStyler) plain(text string) tableCell {
	return tableCell{text: text, styled: text}
}

func (s tableStyler) numberCell(number int) tableCell {
	text := fmt.Sprintf("#%d", number)
	return s.colored(text, termenv.ANSIGreen)
}

func (s tableStyler) stateCell(state string) tableCell {
	switch state {
	case "open":
		return s.colored(state, termenv.ANSIGreen)
	case "draft":
		return s.colored(state, termenv.ANSIYellow)
	case "closed":
		return s.colored(state, termenv.ANSIRed)
	case "merged":
		return s.colored(state, termenv.ANSIMagenta)
	default:
		return s.plain(state)
	}
}

func (s tableStyler) reviewCell(review string) tableCell {
	switch review {
	case "approved":
		return s.colored(review, termenv.ANSIGreen)
	case "changes":
		return s.colored(review, termenv.ANSIRed)
	case "review":
		return s.colored(review, termenv.ANSIYellow)
	default:
		return s.plain(review)
	}
}

func (s tableStyler) checksCell(checks string) tableCell {
	switch checks {
	case "pass":
		return s.colored(checks, termenv.ANSIGreen)
	case "fail":
		return s.colored(checks, termenv.ANSIRed)
	case "pending":
		return s.colored(checks, termenv.ANSIYellow)
	default:
		return s.plain(checks)
	}
}

func (s tableStyler) branchCell(branch string) tableCell {
	return s.colored(branch, termenv.ANSICyan)
}

func (s tableStyler) approvalCell(count int) tableCell {
	text := fmt.Sprintf("%d", count)
	if count > 0 {
		return s.colored(text, termenv.ANSIGreen)
	}
	return s.dim(text)
}

func (s tableStyler) commentsCell(comments string) tableCell {
	if comments == "-" || comments == "?" {
		return s.plain(comments)
	}
	parts := strings.SplitN(comments, "/", 2)
	if len(parts) == 2 && parts[0] == parts[1] {
		return s.colored(comments, termenv.ANSIGreen)
	}
	if len(parts) == 2 && parts[0] == "0" {
		return s.colored(comments, termenv.ANSIRed)
	}
	return s.colored(comments, termenv.ANSIYellow)
}

func (s tableStyler) aiReviewCell(aiReview string) tableCell {
	switch aiReview {
	case "pass":
		return s.colored(aiReview, termenv.ANSIGreen)
	case "fail":
		return s.colored(aiReview, termenv.ANSIRed)
	default:
		return s.plain(aiReview)
	}
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

	// Fetch supplemental PR data via GraphQL (best-effort)
	supplemental := make(map[int]prSupplementalInfo)
	supplementalFailed := false
	if owner, name, err := resolveRepo(options.repo); err == nil {
		numbers := make([]int, len(pullRequests))
		for i, pr := range pullRequests {
			numbers[i] = pr.Number
		}
		if fetched, err := fetchPRSupplemental(owner, name, numbers); err == nil {
			supplemental = fetched
		} else {
			supplementalFailed = true
		}
	} else {
		supplementalFailed = true
	}

	for _, pullRequest := range pullRequests {
		dp := buildDisplayPullRequest(pullRequest, now)
		if supplementalFailed {
			dp.Comments = "?"
			dp.AIReview = "?"
		} else {
			info := supplemental[pullRequest.Number]
			dp.Comments = formatComments(info.Threads)
			dp.AIReview = info.AIReview
			if dp.AIReview == "" {
				dp.AIReview = "-"
			}
			if info.ChecksOverride != "" {
				dp.Checks = info.ChecksOverride
			}
		}
		rendered = append(rendered, dp)
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
		Number:    pullRequest.Number,
		Title:     trimTitle(pullRequest.Title, 56),
		Author:    authorName,
		State:     normalizeState(pullRequest.State, pullRequest.IsDraft),
		Review:    normalizeReviewDecision(pullRequest.ReviewDecision),
		Approvals: countApprovals(pullRequest.LatestReviews),
		Checks:    normalizeCheckState(pullRequest.StatusCheckRollup),
		Comments:  "-",
		AIReview:  "-",
		Branch:    formatBranch(pullRequest.HeadRefName),
		Updated:   formatRelativeTime(pullRequest.UpdatedAt, now),
		URL:       pullRequest.URL,
	}
}

func renderTable(stdout io.Writer, options listOptions, pullRequests []displayPullRequest) error {
	if len(pullRequests) > 0 {
		if repoLabel := resolveRepoLabel(options.repo); repoLabel != "" {
			fmt.Fprintf(stdout, "Pull requests for %s\n\n", repoLabel)
		}
	}
	colorEnabled := term.FromEnv().IsColorEnabled()
	return renderTableWithStyle(stdout, options, pullRequests, colorEnabled)
}

func renderTableWithStyle(stdout io.Writer, options listOptions, pullRequests []displayPullRequest, colorEnabled bool) error {
	if len(pullRequests) == 0 {
		fmt.Fprintln(stdout, "No pull requests found.")
		return nil
	}

	styler := newTableStyler(stdout, colorEnabled)

	headerLabels := []string{"#", "Title", "Author", "State", "Review", "AI", "Appv", "Checks", "Cmts", "Branch", "Updated"}
	headers := make([]tableCell, len(headerLabels))
	for i, label := range headerLabels {
		headers[i] = styler.dim(label)
	}

	rows := make([][]tableCell, len(pullRequests))
	for i, pr := range pullRequests {
		rows[i] = []tableCell{
			styler.numberCell(pr.Number),
			styler.plain(pr.Title),
			styler.plain(pr.Author),
			styler.stateCell(pr.State),
			styler.reviewCell(pr.Review),
			styler.aiReviewCell(pr.AIReview),
			styler.approvalCell(pr.Approvals),
			styler.checksCell(pr.Checks),
			styler.commentsCell(pr.Comments),
			styler.branchCell(pr.Branch),
			styler.dim(pr.Updated),
		}
	}

	colWidths := make([]int, len(headers))
	for i, h := range headers {
		if w := runewidth.StringWidth(h.text); w > colWidths[i] {
			colWidths[i] = w
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := runewidth.StringWidth(cell.text); w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	writeRow(stdout, headers, colWidths)
	for _, row := range rows {
		writeRow(stdout, row, colWidths)
	}

	return nil
}

func writeRow(w io.Writer, cells []tableCell, widths []int) {
	for i, cell := range cells {
		fmt.Fprint(w, cell.styled)
		if i < len(cells)-1 {
			padding := widths[i] - runewidth.StringWidth(cell.text) + 2
			fmt.Fprint(w, strings.Repeat(" ", padding))
		}
	}
	fmt.Fprintln(w)
}

func resolveRepoLabel(repoOverride string) string {
	if repoOverride != "" {
		return repoOverride
	}

	owner, name, err := resolveRepo("")
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s/%s", owner, name)
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

func formatBranch(head string) string {
	if head == "" {
		return "-"
	}
	return head
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

func countApprovals(reviews []review) int {
	count := 0
	for _, r := range reviews {
		if strings.EqualFold(r.State, "APPROVED") {
			count++
		}
	}
	return count
}

type reviewThreadInfo struct {
	Total    int
	Resolved int
}

type prSupplementalInfo struct {
	Threads        reviewThreadInfo
	AIReview       string
	ChecksOverride string // aggregate commit status from GraphQL (includes expected checks)
}

// aiReviewNode holds the fields needed to detect bot reviewer status.
type aiReviewNode struct {
	State        string
	AuthorLogin  string
	CommentCount int
}

func formatComments(info reviewThreadInfo) string {
	if info.Total == 0 {
		return "-"
	}
	return fmt.Sprintf("%d/%d", info.Resolved, info.Total)
}

// Known AI reviewer logins that don't use the [bot] suffix convention.
var knownAIReviewers = map[string]bool{
	"copilot-pull-request-reviewer": true,
}

func isAIReviewer(login string) bool {
	return strings.HasSuffix(login, "[bot]") || knownAIReviewers[login]
}

func detectAIReview(nodes []aiReviewNode) string {
	hasApproval := false
	hasIssues := false
	hasBotReview := false

	for _, r := range nodes {
		if !isAIReviewer(r.AuthorLogin) {
			continue
		}
		hasBotReview = true

		switch strings.ToUpper(r.State) {
		case "APPROVED":
			hasApproval = true
		case "CHANGES_REQUESTED":
			hasIssues = true
		case "COMMENTED":
			if r.CommentCount == 0 {
				// Bot reviewed with no inline comments — implicit approval (e.g. Copilot)
				hasApproval = true
			} else {
				hasIssues = true
			}
		}
	}

	if !hasBotReview {
		return "-"
	}
	if hasIssues {
		return "fail"
	}
	if hasApproval {
		return "pass"
	}
	return "-"
}

// normalizeGraphQLCheckState maps the aggregate StatusCheckRollup.state
// (which includes branch-protection-expected checks) to display values.
func normalizeGraphQLCheckState(state string) string {
	switch strings.ToUpper(state) {
	case "SUCCESS":
		return "pass"
	case "FAILURE", "ERROR":
		return "fail"
	case "PENDING", "EXPECTED":
		return "pending"
	default:
		return ""
	}
}

func resolveRepo(repoOverride string) (string, string, error) {
	if repoOverride != "" {
		parts := strings.Split(repoOverride, "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid repo format: %s", repoOverride)
		}
		return parts[len(parts)-2], parts[len(parts)-1], nil
	}

	repo, err := repository.Current()
	if err == nil {
		return repo.Owner, repo.Name, nil
	}

	// Fall back to gh repo view for SSH aliases and non-standard remotes
	stdout, _, execErr := gh.Exec("repo", "view", "--json", "owner,name")
	if execErr != nil {
		return "", "", fmt.Errorf("repo resolution failed: %w; fallback: %v", err, execErr)
	}
	var info struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	if parseErr := json.Unmarshal(stdout.Bytes(), &info); parseErr != nil {
		return "", "", parseErr
	}
	if info.Owner.Login == "" || info.Name == "" {
		return "", "", fmt.Errorf("could not resolve repo from gh repo view")
	}
	return info.Owner.Login, info.Name, nil
}

func fetchPRSupplemental(owner, name string, prNumbers []int) (map[int]prSupplementalInfo, error) {
	if len(prNumbers) == 0 {
		return nil, nil
	}

	var queryParts []string
	for _, num := range prNumbers {
		queryParts = append(queryParts, fmt.Sprintf(
			`pr%d: pullRequest(number: %d) { number reviewThreads(first: 100) { totalCount nodes { isResolved } } latestReviews(first: 50) { nodes { state author { login } comments { totalCount } } } commits(last: 1) { nodes { commit { statusCheckRollup { state } } } } }`,
			num, num,
		))
	}

	query := fmt.Sprintf(
		`query { repository(owner: %q, name: %q) { %s } }`,
		owner, name, strings.Join(queryParts, " "),
	)

	stdout, _, err := gh.Exec("api", "graphql", "-f", fmt.Sprintf("query=%s", query))
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			Repository map[string]json.RawMessage `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, err
	}

	result := make(map[int]prSupplementalInfo)
	for _, raw := range resp.Data.Repository {
		var prData struct {
			Number        int `json:"number"`
			ReviewThreads struct {
				TotalCount int `json:"totalCount"`
				Nodes      []struct {
					IsResolved bool `json:"isResolved"`
				} `json:"nodes"`
			} `json:"reviewThreads"`
			LatestReviews struct {
				Nodes []struct {
					State  string `json:"state"`
					Author struct {
						Login string `json:"login"`
					} `json:"author"`
					Comments struct {
						TotalCount int `json:"totalCount"`
					} `json:"comments"`
				} `json:"nodes"`
			} `json:"latestReviews"`
			Commits struct {
				Nodes []struct {
					Commit struct {
						StatusCheckRollup struct {
							State string `json:"state"`
						} `json:"statusCheckRollup"`
					} `json:"commit"`
				} `json:"nodes"`
			} `json:"commits"`
		}
		if err := json.Unmarshal(raw, &prData); err != nil {
			continue
		}

		resolved := 0
		for _, node := range prData.ReviewThreads.Nodes {
			if node.IsResolved {
				resolved++
			}
		}

		var aiNodes []aiReviewNode
		for _, r := range prData.LatestReviews.Nodes {
			aiNodes = append(aiNodes, aiReviewNode{
				State:        r.State,
				AuthorLogin:  r.Author.Login,
				CommentCount: r.Comments.TotalCount,
			})
		}

		checksOverride := ""
		if len(prData.Commits.Nodes) > 0 {
			checksOverride = normalizeGraphQLCheckState(
				prData.Commits.Nodes[0].Commit.StatusCheckRollup.State,
			)
		}

		result[prData.Number] = prSupplementalInfo{
			Threads: reviewThreadInfo{
				Total:    prData.ReviewThreads.TotalCount,
				Resolved: resolved,
			},
			AIReview:       detectAIReview(aiNodes),
			ChecksOverride: checksOverride,
		}
	}

	return result, nil
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
