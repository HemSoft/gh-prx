package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBuildListArgsIncludesFilters(t *testing.T) {
	options := listOptions{
		repo:      "HemSoft/gh-prx",
		limit:     50,
		state:     "all",
		author:    "@me",
		assignee:  "octocat",
		app:       "dependabot",
		base:      "main",
		head:      "feature/demo",
		search:    "review:required",
		draftOnly: true,
		labels:    stringSliceFlag{"bug", "urgent"},
	}

	got := buildListArgs(options)
	want := []string{
		"pr", "list",
		"--json", jsonFields,
		"--repo", "HemSoft/gh-prx",
		"--limit", "50",
		"--state", "all",
		"--author", "@me",
		"--assignee", "octocat",
		"--app", "dependabot",
		"--base", "main",
		"--head", "feature/demo",
		"--search", "review:required",
		"--draft",
		"--label", "bug",
		"--label", "urgent",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected arguments\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestBuildDisplayPullRequestNormalizesFields(t *testing.T) {
	now := time.Date(2026, 5, 10, 1, 45, 0, 0, time.UTC)
	pullRequest := pullRequest{
		Number:         42,
		Title:          "Improve the PR list view so reviews and checks are obvious at a glance",
		State:          "OPEN",
		IsDraft:        true,
		ReviewDecision: "CHANGES_REQUESTED",
		UpdatedAt:      now.Add(-2 * time.Hour),
		HeadRefName:    "feature/prx",
		BaseRefName:    "main",
		URL:            "https://github.com/HemSoft/gh-prx/pull/42",
		Author:         &author{Login: "HemSoft"},
		StatusCheckRollup: []checkItem{
			{Typename: "CheckRun", Status: "COMPLETED", Conclusion: "SUCCESS"},
		},
		LatestReviews: []review{
			{State: "APPROVED", Author: &author{Login: "reviewer1"}},
			{State: "COMMENTED", Author: &author{Login: "reviewer2"}},
			{State: "APPROVED", Author: &author{Login: "reviewer3"}},
		},
	}

	got := buildDisplayPullRequest(pullRequest, now)

	if got.State != "draft" {
		t.Fatalf("expected draft state, got %q", got.State)
	}

	if got.Review != "changes" {
		t.Fatalf("expected changes review, got %q", got.Review)
	}

	if got.Checks != "pass" {
		t.Fatalf("expected pass checks, got %q", got.Checks)
	}

	if got.Branch != "feature/prx" {
		t.Fatalf("unexpected branch column %q", got.Branch)
	}

	if got.Approvals != 2 {
		t.Fatalf("expected 2 approvals, got %d", got.Approvals)
	}

	if got.Comments != "-" {
		t.Fatalf("expected default comments '-', got %q", got.Comments)
	}

	if got.AIReview != "-" {
		t.Fatalf("expected default AIReview '-', got %q", got.AIReview)
	}

	if got.Updated != "2h" {
		t.Fatalf("unexpected updated column %q", got.Updated)
	}

	if got.Author != "HemSoft" {
		t.Fatalf("unexpected author %q", got.Author)
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Date(2026, 5, 10, 1, 45, 0, 0, time.UTC)

	testCases := []struct {
		name      string
		updatedAt time.Time
		expected  string
	}{
		{name: "seconds", updatedAt: now.Add(-30 * time.Second), expected: "30s"},
		{name: "minutes", updatedAt: now.Add(-45 * time.Minute), expected: "45m"},
		{name: "hours", updatedAt: now.Add(-3 * time.Hour), expected: "3h"},
		{name: "days", updatedAt: now.Add(-72 * time.Hour), expected: "3d"},
		{name: "months", updatedAt: now.Add(-(45 * 24 * time.Hour)), expected: "1mo"},
		{name: "years", updatedAt: now.Add(-(400 * 24 * time.Hour)), expected: "1y"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := formatRelativeTime(testCase.updatedAt, now); got != testCase.expected {
				t.Fatalf("expected %q, got %q", testCase.expected, got)
			}
		})
	}
}

func TestRenderTableNoColor(t *testing.T) {
	var buf bytes.Buffer
	prs := []displayPullRequest{
		{Number: 42, Title: "My PR", Author: "user", State: "open", Review: "approved", AIReview: "pass", Approvals: 2, Checks: "pass", Comments: "3/5", Branch: "feat", Updated: "2h"},
	}
	err := renderTableWithStyle(&buf, listOptions{}, prs, false)
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if strings.Contains(output, "\x1b[") {
		t.Fatal("expected no ANSI escape codes when color is disabled")
	}
	if !strings.Contains(output, "#42") {
		t.Fatal("expected PR number in output")
	}
	if !strings.Contains(output, "My PR") {
		t.Fatal("expected title in output")
	}
	if !strings.Contains(output, "approved") {
		t.Fatal("expected review status in output")
	}
}

func TestRenderTableWithColor(t *testing.T) {
	var buf bytes.Buffer
	prs := []displayPullRequest{
		{Number: 7, Title: "Add colors", Author: "dev", State: "open", Review: "review", AIReview: "-", Approvals: 0, Checks: "pending", Comments: "-", Branch: "color", Updated: "5m"},
	}
	err := renderTableWithStyle(&buf, listOptions{}, prs, true)
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "\x1b[") {
		t.Fatal("expected ANSI escape codes when color is enabled")
	}
	if !strings.Contains(output, "#7") {
		t.Fatal("expected PR number in output")
	}
}

func TestRenderTableAlignment(t *testing.T) {
	var buf bytes.Buffer
	prs := []displayPullRequest{
		{Number: 1, Title: "Short", Author: "a", State: "open", Review: "-", AIReview: "-", Approvals: 0, Checks: "-", Comments: "-", Branch: "x", Updated: "1h"},
		{Number: 999, Title: "Longer title here", Author: "longuser", State: "merged", Review: "approved", AIReview: "pass", Approvals: 3, Checks: "pass", Comments: "5/5", Branch: "feature/long", Updated: "30d"},
	}
	err := renderTableWithStyle(&buf, listOptions{}, prs, false)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d: %v", len(lines), lines)
	}

	// Verify header labels are present
	if !strings.Contains(lines[0], "Title") || !strings.Contains(lines[0], "Branch") {
		t.Fatal("expected header labels")
	}

	// Verify columns are aligned: the "Title" column should start at the same
	// position in header and data rows
	headerTitleIdx := strings.Index(lines[0], "Title")
	row1TitleIdx := strings.Index(lines[1], "Short")
	row2TitleIdx := strings.Index(lines[2], "Longer")
	if headerTitleIdx != row1TitleIdx || headerTitleIdx != row2TitleIdx {
		t.Fatalf("Title column misaligned: header=%d row1=%d row2=%d", headerTitleIdx, row1TitleIdx, row2TitleIdx)
	}
}

func TestRenderTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := renderTableWithStyle(&buf, listOptions{}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No pull requests found") {
		t.Fatal("expected empty message")
	}
}

func TestCountApprovals(t *testing.T) {
	tests := []struct {
		name    string
		reviews []review
		want    int
	}{
		{name: "nil", reviews: nil, want: 0},
		{name: "empty", reviews: []review{}, want: 0},
		{name: "one approved", reviews: []review{{State: "APPROVED"}}, want: 1},
		{name: "mixed", reviews: []review{
			{State: "APPROVED"},
			{State: "COMMENTED"},
			{State: "CHANGES_REQUESTED"},
			{State: "APPROVED"},
		}, want: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := countApprovals(tc.reviews); got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestFormatComments(t *testing.T) {
	tests := []struct {
		name string
		info reviewThreadInfo
		want string
	}{
		{name: "none", info: reviewThreadInfo{}, want: "-"},
		{name: "all resolved", info: reviewThreadInfo{Total: 5, Resolved: 5}, want: "5/5"},
		{name: "partial", info: reviewThreadInfo{Total: 5, Resolved: 3}, want: "3/5"},
		{name: "none resolved", info: reviewThreadInfo{Total: 3, Resolved: 0}, want: "0/3"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatComments(tc.info); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestIsAIReviewer(t *testing.T) {
	tests := []struct {
		login string
		want  bool
	}{
		{"coderabbitai[bot]", true},
		{"copilot[bot]", true},
		{"copilot-pull-request-reviewer", true},
		{"human-reviewer", false},
		{"dependabot[bot]", true},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.login, func(t *testing.T) {
			if got := isAIReviewer(tc.login); got != tc.want {
				t.Fatalf("isAIReviewer(%q) = %v, want %v", tc.login, got, tc.want)
			}
		})
	}
}

func TestExtractReportedContexts(t *testing.T) {
	items := []checkItem{
		{Typename: "CheckRun", Name: "SonarCloud Code Analysis", Status: "COMPLETED", Conclusion: "SUCCESS"},
		{Typename: "StatusContext", Context: "usergroups-api-pr", State: "SUCCESS"},
		{Typename: "CheckRun", Name: "", Status: "COMPLETED", Conclusion: "SUCCESS"}, // empty name ignored
	}
	got := extractReportedContexts(items)
	if !got["SonarCloud Code Analysis"] {
		t.Error("expected SonarCloud Code Analysis")
	}
	if !got["usergroups-api-pr"] {
		t.Error("expected usergroups-api-pr")
	}
	if len(got) != 2 {
		t.Errorf("expected 2 contexts, got %d", len(got))
	}
}

func TestExtractReportedContextsEmpty(t *testing.T) {
	got := extractReportedContexts(nil)
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestDetectAIReview(t *testing.T) {
	tests := []struct {
		name  string
		nodes []aiReviewNode
		want  string
	}{
		{name: "nil", nodes: nil, want: "-"},
		{name: "empty", nodes: []aiReviewNode{}, want: "-"},
		{name: "no bots", nodes: []aiReviewNode{
			{State: "APPROVED", AuthorLogin: "human-reviewer", CommentCount: 0},
		}, want: "-"},
		{name: "coderabbit approved", nodes: []aiReviewNode{
			{State: "APPROVED", AuthorLogin: "coderabbitai[bot]", CommentCount: 0},
		}, want: "pass"},
		{name: "copilot no comments", nodes: []aiReviewNode{
			{State: "COMMENTED", AuthorLogin: "copilot[bot]", CommentCount: 0},
		}, want: "pass"},
		{name: "copilot-pull-request-reviewer no comments", nodes: []aiReviewNode{
			{State: "COMMENTED", AuthorLogin: "copilot-pull-request-reviewer", CommentCount: 0},
		}, want: "pass"},
		{name: "bot with comments", nodes: []aiReviewNode{
			{State: "COMMENTED", AuthorLogin: "coderabbitai[bot]", CommentCount: 3},
		}, want: "fail"},
		{name: "bot changes requested", nodes: []aiReviewNode{
			{State: "CHANGES_REQUESTED", AuthorLogin: "coderabbitai[bot]", CommentCount: 5},
		}, want: "fail"},
		{name: "mixed bot approved and human", nodes: []aiReviewNode{
			{State: "APPROVED", AuthorLogin: "coderabbitai[bot]", CommentCount: 0},
			{State: "CHANGES_REQUESTED", AuthorLogin: "human-reviewer", CommentCount: 2},
		}, want: "pass"},
		{name: "issues override approval", nodes: []aiReviewNode{
			{State: "APPROVED", AuthorLogin: "coderabbitai[bot]", CommentCount: 0},
			{State: "CHANGES_REQUESTED", AuthorLogin: "copilot[bot]", CommentCount: 1},
		}, want: "fail"},
		{name: "dismissed bot review ignored", nodes: []aiReviewNode{
			{State: "DISMISSED", AuthorLogin: "coderabbitai[bot]", CommentCount: 0},
		}, want: "-"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectAIReview(tc.nodes); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestFormatBranch(t *testing.T) {
	if got := formatBranch("feature/test"); got != "feature/test" {
		t.Fatalf("expected 'feature/test', got %q", got)
	}
	if got := formatBranch(""); got != "-" {
		t.Fatalf("expected '-', got %q", got)
	}
}

func TestRunVersionUpToDate(t *testing.T) {
	orig := fetchLatestReleaseFunc
	defer func() { fetchLatestReleaseFunc = orig }()
	fetchLatestReleaseFunc = func(owner, repo string) (string, error) {
		return "v1.2.3", nil
	}

	var buf bytes.Buffer
	err := runVersionTestable(&buf, "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "gh-prx v1.2.3 by HemSoft") {
		t.Fatalf("expected version line, got %q", out)
	}
	if !strings.Contains(out, "gh extension install HemSoft/gh-prx") {
		t.Fatalf("expected install command, got %q", out)
	}
	if !strings.Contains(out, "✓ Up to date") {
		t.Fatalf("expected up-to-date indicator, got %q", out)
	}
}

func TestRunVersionUpdateAvailable(t *testing.T) {
	orig := fetchLatestReleaseFunc
	defer func() { fetchLatestReleaseFunc = orig }()
	fetchLatestReleaseFunc = func(owner, repo string) (string, error) {
		return "v2.0.0", nil
	}

	var buf bytes.Buffer
	err := runVersionTestable(&buf, "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "↑ v2.0.0 available") {
		t.Fatalf("expected update indicator, got %q", out)
	}
	if !strings.Contains(out, "gh extension upgrade gh-prx") {
		t.Fatalf("expected upgrade command, got %q", out)
	}
}

func TestRunVersionDevBuild(t *testing.T) {
	orig := fetchLatestReleaseFunc
	defer func() { fetchLatestReleaseFunc = orig }()
	fetchLatestReleaseFunc = func(owner, repo string) (string, error) {
		return "v0.5.0", nil
	}

	var buf bytes.Buffer
	err := runVersionTestable(&buf, "dev")
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "gh-prx dev by HemSoft") {
		t.Fatalf("expected dev version, got %q", out)
	}
	if !strings.Contains(out, "⚙ Dev build · latest release: v0.5.0") {
		t.Fatalf("expected dev build indicator, got %q", out)
	}
}

func TestRunVersionAPIError(t *testing.T) {
	orig := fetchLatestReleaseFunc
	defer func() { fetchLatestReleaseFunc = orig }()
	fetchLatestReleaseFunc = func(owner, repo string) (string, error) {
		return "", fmt.Errorf("network error")
	}

	var buf bytes.Buffer
	err := runVersionTestable(&buf, "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "⚠ Could not check for updates") {
		t.Fatalf("expected error fallback, got %q", out)
	}
}

func TestPrintBanner(t *testing.T) {
	oldVersion := version
	oldDate := buildDate
	defer func() { version = oldVersion; buildDate = oldDate }()

	version = "v1.2.3"
	buildDate = "2026-05-10"
	var buf bytes.Buffer
	printBanner(&buf)
	if got := buf.String(); got != "gh-prx v1.2.3 (2026-05-10) by HemSoft\n" {
		t.Fatalf("unexpected banner: %q", got)
	}
}

func TestPrintBannerNoDate(t *testing.T) {
	oldVersion := version
	oldDate := buildDate
	defer func() { version = oldVersion; buildDate = oldDate }()

	version = "v1.2.3"
	buildDate = ""
	var buf bytes.Buffer
	printBanner(&buf)
	if got := buf.String(); got != "gh-prx v1.2.3 by HemSoft\n" {
		t.Fatalf("unexpected banner without date: %q", got)
	}
}

func TestFormatVersion(t *testing.T) {
	if got := formatVersion("v1.0.0", "2026-05-10"); got != "v1.0.0 (2026-05-10)" {
		t.Fatalf("expected date in parens, got %q", got)
	}
	if got := formatVersion("v1.0.0", ""); got != "v1.0.0" {
		t.Fatalf("expected no parens when date empty, got %q", got)
	}
}

func TestBannerOnRootUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_ = run(nil, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "gh-prx") {
		t.Fatalf("expected banner on stderr for root usage, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Available Commands") {
		t.Fatalf("expected usage on stdout, got %q", stdout.String())
	}
}

func TestBannerOnHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_ = run([]string{"--help"}, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "gh-prx") {
		t.Fatalf("expected banner on stderr for help, got %q", stderr.String())
	}
}

func TestNoBannerOnVersion(t *testing.T) {
	orig := fetchLatestReleaseFunc
	defer func() { fetchLatestReleaseFunc = orig }()
	fetchLatestReleaseFunc = func(owner, repo string) (string, error) {
		return "v1.0.0", nil
	}

	for _, arg := range []string{"version", "--version", "-v"} {
		var stdout, stderr bytes.Buffer
		_ = run([]string{arg}, &stdout, &stderr)
		if strings.Contains(stderr.String(), "gh-prx") {
			t.Fatalf("run(%q) should not print banner to stderr, got %q", arg, stderr.String())
		}
	}
}

func TestBannerOnUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_ = run([]string{"bogus"}, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "gh-prx") {
		t.Fatalf("expected banner on stderr for unknown command, got %q", stderr.String())
	}
}

func TestRunVersionRouting(t *testing.T) {
	orig := fetchLatestReleaseFunc
	defer func() { fetchLatestReleaseFunc = orig }()
	fetchLatestReleaseFunc = func(owner, repo string) (string, error) {
		return "v1.0.0", nil
	}

	for _, arg := range []string{"version", "--version", "-v"} {
		var buf bytes.Buffer
		err := run([]string{arg}, &buf, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("run(%q) returned error: %v", arg, err)
		}
		if !strings.Contains(buf.String(), "gh-prx") {
			t.Fatalf("run(%q) missing version output: %q", arg, buf.String())
		}
	}
}
