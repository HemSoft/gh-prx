package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	gh "github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

type atmOptions struct {
	org            string
	limit          int
	authored       bool
	reviewRequired bool
	json           bool
}

func runAtm(args []string, stdout io.Writer, stderr io.Writer) error {
	options, err := parseAtmOptions(args, stderr)
	if err != nil {
		if errors.Is(err, errHelpDisplayed) {
			return nil
		}
		return err
	}

	return executeAtm(options, stdout)
}

func parseAtmOptions(args []string, stderr io.Writer) (atmOptions, error) {
	options := atmOptions{limit: 30}

	flags := flag.NewFlagSet("atm", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		writeAtmUsage(stderr)
	}

	flags.StringVar(&options.org, "org", "", "Organization to search (default: inferred from current repo)")
	flags.StringVar(&options.org, "o", "", "Organization to search (default: inferred from current repo)")
	flags.IntVar(&options.limit, "limit", 30, "Maximum number of pull requests to fetch")
	flags.IntVar(&options.limit, "L", 30, "Maximum number of pull requests to fetch")
	flags.BoolVar(&options.authored, "authored", false, "Show PRs you authored (default: show PRs needing your review)")
	flags.BoolVar(&options.authored, "a", false, "Show PRs you authored (default: show PRs needing your review)")
	flags.BoolVar(&options.reviewRequired, "review-required", false, "Show only PRs where your review is directly requested")
	flags.BoolVar(&options.reviewRequired, "r", false, "Show only PRs where your review is directly requested")
	flags.BoolVar(&options.json, "json", false, "Output enriched JSON instead of a table")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return options, errHelpDisplayed
		}
		return options, err
	}

	if flags.NArg() > 0 {
		return options, fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), ", "))
	}

	if options.limit < 1 {
		return options, errors.New("limit must be greater than zero")
	}

	return options, nil
}

func writeAtmUsage(w io.Writer) {
	fmt.Fprint(w, atmUsage)
}

const atmUsage = `Usage:
  gh prx atm [flags]

Show open pull requests across an organization that need your attention.
By default, shows PRs needing your review. Use --authored for PRs you created.

Flags:
  -o, --org string        Organization to search (default: inferred from current repo)
  -L, --limit int         Maximum number of pull requests to fetch (default 30)
  -a, --authored          Show PRs you authored instead of PRs needing review
  -r, --review-required   Show only PRs where your review is directly requested
      --json              Output enriched JSON instead of a table
`

func executeAtm(options atmOptions, stdout io.Writer) error {
	org, err := resolveAtmOrg(options.org)
	if err != nil {
		return fmt.Errorf("cannot determine organization: %w", err)
	}

	login, err := resolveCurrentUser()
	if err != nil {
		return fmt.Errorf("cannot determine current user: %w", err)
	}

	var nodes []atmPullRequestNode

	if options.authored || options.reviewRequired {
		searchQuery := buildAtmSearchQuery(org, login, options.reviewRequired)
		query := buildAtmGraphQLQuery(searchQuery, options.limit)
		stdoutBuf, _, execErr := gh.Exec("api", "graphql", "-f", fmt.Sprintf("query=%s", query))
		if execErr != nil {
			return fmt.Errorf("GraphQL search failed: %w", execErr)
		}
		nodes, err = parseAtmSearchResponse(stdoutBuf.Bytes())
		if err != nil {
			return err
		}
	} else {
		queries := buildAtmNeedsReviewQueries(org, login)
		query := buildAtmMultiSearchQuery(queries, options.limit)
		stdoutBuf, _, execErr := gh.Exec("api", "graphql", "-f", fmt.Sprintf("query=%s", query))
		if execErr != nil {
			return fmt.Errorf("GraphQL search failed: %w", execErr)
		}
		nodes, err = parseAtmMultiSearchResponse(stdoutBuf.Bytes())
		if err != nil {
			return err
		}
		filtered := make([]atmPullRequestNode, 0, len(nodes))
		for _, n := range nodes {
			if !userApprovedPR(n, login) {
				filtered = append(filtered, n)
			}
		}
		nodes = filtered
	}

	now := time.Now().UTC()
	rendered := make([]displayPullRequest, 0, len(nodes))
	for _, node := range nodes {
		rendered = append(rendered, mapAtmNode(node, now))
	}

	if options.json {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(rendered)
	}

	return renderAtmTable(stdout, org, login, options, rendered)
}

func resolveAtmOrg(orgOverride string) (string, error) {
	if orgOverride != "" {
		return orgOverride, nil
	}
	owner, _, err := resolveRepo("")
	if err != nil {
		return "", fmt.Errorf("not in a repository; use --org to specify organization")
	}
	return owner, nil
}

func resolveCurrentUser() (string, error) {
	stdout, _, err := gh.Exec("api", "user", "--jq", ".login")
	if err != nil {
		return "", err
	}
	login := strings.TrimSpace(stdout.String())
	if login == "" {
		return "", fmt.Errorf("empty login returned")
	}
	return login, nil
}

func buildAtmSearchQuery(org, login string, reviewRequired bool) string {
	if reviewRequired {
		return fmt.Sprintf("is:pr is:open review-requested:%s org:%s", login, org)
	}
	return fmt.Sprintf("is:pr is:open author:%s org:%s", login, org)
}

func buildAtmNeedsReviewQueries(org, login string) []string {
	return []string{
		fmt.Sprintf("is:pr is:open review-requested:%s org:%s", login, org),
		fmt.Sprintf("is:pr is:open assignee:%s org:%s -author:%s", login, org, login),
		fmt.Sprintf("is:pr is:open reviewed-by:%s org:%s", login, org),
	}
}

const atmPRFieldsFragment = `
        number
        title
        author { login }
        state
        isDraft
        reviewDecision
        updatedAt
        headRefName
        baseRefName
        url
        repository { nameWithOwner }
        commits(last: 1) {
          nodes {
            commit {
              statusCheckRollup {
                contexts(first: 100) {
                  nodes {
                    __typename
                    ... on CheckRun { name status conclusion }
                    ... on StatusContext { context state }
                  }
                }
              }
            }
          }
        }
        latestReviews(first: 50) {
          nodes {
            state
            author { login }
            comments { totalCount }
          }
        }
        reviewThreads(first: 100) {
          totalCount
          nodes {
            isResolved
            comments(first: 1) {
              nodes {
                author { login __typename }
              }
            }
          }
        }
        approvedReviews: reviews(states: [APPROVED], last: 50) {
          nodes {
            author { login }
          }
        }`

func buildAtmGraphQLQuery(searchQuery string, limit int) string {
	return fmt.Sprintf(`{
  search(query: %q, type: ISSUE, first: %d) {
    nodes {
      ... on PullRequest {%s
      }
    }
  }
}`, searchQuery, limit, atmPRFieldsFragment)
}

func buildAtmMultiSearchQuery(queries []string, limit int) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	for i, q := range queries {
		sb.WriteString(fmt.Sprintf(`  q%d: search(query: %q, type: ISSUE, first: %d) {
    nodes {
      ... on PullRequest {%s
      }
    }
  }
`, i, q, limit, atmPRFieldsFragment))
	}
	sb.WriteString("}")
	return sb.String()
}

// atmPullRequestNode represents a PR returned from the GraphQL search query.
type atmPullRequestNode struct {
	Number         int       `json:"number"`
	Title          string    `json:"title"`
	Author         *author   `json:"author"`
	State          string    `json:"state"`
	IsDraft        bool      `json:"isDraft"`
	ReviewDecision string    `json:"reviewDecision"`
	UpdatedAt      time.Time `json:"updatedAt"`
	HeadRefName    string    `json:"headRefName"`
	BaseRefName    string    `json:"baseRefName"`
	URL            string    `json:"url"`
	Repository     struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					Contexts struct {
						Nodes []checkItem `json:"nodes"`
					} `json:"contexts"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
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
	ReviewThreads struct {
		TotalCount int `json:"totalCount"`
		Nodes      []struct {
			IsResolved bool `json:"isResolved"`
			Comments   struct {
				Nodes []struct {
					Author struct {
						Login    string `json:"login"`
						Typename string `json:"__typename"`
					} `json:"author"`
				} `json:"nodes"`
			} `json:"comments"`
		} `json:"nodes"`
	} `json:"reviewThreads"`
	ApprovedReviews struct {
		Nodes []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
		} `json:"nodes"`
	} `json:"approvedReviews"`
}

func parseAtmSearchResponse(data []byte) ([]atmPullRequestNode, error) {
	var resp struct {
		Data struct {
			Search struct {
				Nodes []atmPullRequestNode `json:"nodes"`
			} `json:"search"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode GraphQL response: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", resp.Errors[0].Message)
	}
	return resp.Data.Search.Nodes, nil
}

func parseAtmMultiSearchResponse(data []byte) ([]atmPullRequestNode, error) {
	var raw struct {
		Data   map[string]json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode GraphQL response: %w", err)
	}
	if len(raw.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", raw.Errors[0].Message)
	}

	seen := make(map[string]bool)
	var result []atmPullRequestNode

	for i := 0; ; i++ {
		key := fmt.Sprintf("q%d", i)
		v, ok := raw.Data[key]
		if !ok {
			break
		}
		var search struct {
			Nodes []atmPullRequestNode `json:"nodes"`
		}
		if err := json.Unmarshal(v, &search); err != nil {
			return nil, fmt.Errorf("failed to parse query q%d: %w", i, err)
		}
		for _, node := range search.Nodes {
			prKey := fmt.Sprintf("%s#%d", node.Repository.NameWithOwner, node.Number)
			if !seen[prKey] {
				seen[prKey] = true
				result = append(result, node)
			}
		}
	}

	return result, nil
}

func userApprovedPR(node atmPullRequestNode, login string) bool {
	for _, r := range node.LatestReviews.Nodes {
		if strings.EqualFold(r.Author.Login, login) && strings.EqualFold(r.State, "APPROVED") {
			return true
		}
	}
	return false
}

func mapAtmNode(node atmPullRequestNode, now time.Time) displayPullRequest {
	authorName := "-"
	if node.Author != nil && node.Author.Login != "" {
		authorName = node.Author.Login
	}

	// Extract repo short name from "owner/name"
	repoName := node.Repository.NameWithOwner
	if parts := strings.SplitN(repoName, "/", 2); len(parts) == 2 {
		repoName = parts[1]
	}

	// Extract check items from nested commits structure
	var checkItems []checkItem
	if len(node.Commits.Nodes) > 0 {
		commit := node.Commits.Nodes[0].Commit
		if commit.StatusCheckRollup != nil {
			checkItems = commit.StatusCheckRollup.Contexts.Nodes
		}
	}

	// Build AI review nodes
	var aiNodes []aiReviewNode
	for _, r := range node.LatestReviews.Nodes {
		aiNodes = append(aiNodes, aiReviewNode{
			State:        r.State,
			AuthorLogin:  r.Author.Login,
			CommentCount: r.Comments.TotalCount,
		})
	}

	// Build AI review threads with author info
	var aiThreads []aiReviewThread
	for _, t := range node.ReviewThreads.Nodes {
		var login, authorType string
		if len(t.Comments.Nodes) > 0 {
			login = t.Comments.Nodes[0].Author.Login
			authorType = t.Comments.Nodes[0].Author.Typename
		}
		aiThreads = append(aiThreads, aiReviewThread{
			AuthorLogin: login,
			AuthorType:  authorType,
			IsResolved:  t.IsResolved,
		})
	}

	// Count resolved threads
	resolved := 0
	for _, t := range node.ReviewThreads.Nodes {
		if t.IsResolved {
			resolved++
		}
	}
	threads := reviewThreadInfo{
		Total:    node.ReviewThreads.TotalCount,
		Resolved: resolved,
	}

	aiReview := detectAIReview(aiNodes, aiThreads)
	if aiReview == "" {
		aiReview = "-"
	}

	// Count unique approvers from reviews(states: [APPROVED])
	approverSet := make(map[string]bool)
	for _, r := range node.ApprovedReviews.Nodes {
		if r.Author.Login != "" {
			approverSet[strings.ToLower(r.Author.Login)] = true
		}
	}

	return displayPullRequest{
		Number:    node.Number,
		Title:     trimTitle(node.Title, 42),
		Author:    authorName,
		State:     normalizeState(node.State, node.IsDraft),
		Review:    normalizeReviewDecision(node.ReviewDecision),
		Approvals: len(approverSet),
		Checks:    normalizeCheckState(checkItems),
		Comments:  formatComments(threads),
		AIReview:  aiReview,
		Branch:    formatBranch(node.HeadRefName),
		Updated:   formatRelativeTime(node.UpdatedAt, now),
		URL:       node.URL,
		Repo:      repoName,
	}
}

func countApprovalsFromNodes(nodes []aiReviewNode) int {
	count := 0
	for _, n := range nodes {
		if strings.EqualFold(n.State, "APPROVED") {
			count++
		}
	}
	return count
}

func renderAtmTable(stdout io.Writer, org, login string, options atmOptions, pullRequests []displayPullRequest) error {
	if len(pullRequests) == 0 {
		if options.authored {
			fmt.Fprintf(stdout, "No open PRs authored by %s in %s.\n", login, org)
		} else if options.reviewRequired {
			fmt.Fprintf(stdout, "No open PRs requesting review from %s in %s.\n", login, org)
		} else {
			fmt.Fprintf(stdout, "No PRs needing review from %s in %s.\n", login, org)
		}
		return nil
	}

	if options.authored {
		fmt.Fprintf(stdout, "Open PRs by %s in %s\n\n", login, org)
	} else if options.reviewRequired {
		fmt.Fprintf(stdout, "PRs requesting review from %s in %s\n\n", login, org)
	} else {
		fmt.Fprintf(stdout, "PRs needing review from %s in %s\n\n", login, org)
	}

	colorEnabled := term.FromEnv().IsColorEnabled()
	return renderAtmTableWithStyle(stdout, pullRequests, colorEnabled)
}

func renderAtmTableWithStyle(stdout io.Writer, pullRequests []displayPullRequest, colorEnabled bool) error {
	styler := newTableStyler(stdout, colorEnabled)

	headerLabels := []string{"#", "Title", "Repo", "Author", "State", "Review", "AI", "Appv", "Checks", "Cmts", "Updated"}
	headers := make([]tableCell, len(headerLabels))
	for i, label := range headerLabels {
		headers[i] = styler.dim(label)
	}

	rows := make([][]tableCell, len(pullRequests))
	for i, pr := range pullRequests {
		rows[i] = []tableCell{
			styler.numberCell(pr.Number, pr.URL),
			styler.plain(pr.Title),
			styler.colored(pr.Repo, termenv.ANSICyan),
			styler.plain(pr.Author),
			styler.stateCell(pr.State),
			styler.reviewCell(pr.Review),
			styler.aiReviewCell(pr.AIReview),
			styler.approvalCell(pr.Approvals),
			styler.checksCell(pr.Checks),
			styler.commentsCell(pr.Comments),
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
