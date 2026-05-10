package main

import (
	"bytes"
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
		{Number: 42, Title: "My PR", Author: "user", State: "open", Review: "approved", Approvals: 2, Checks: "pass", Comments: "3/5", Branch: "feat", Updated: "2h"},
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
		{Number: 7, Title: "Add colors", Author: "dev", State: "open", Review: "review", Approvals: 0, Checks: "pending", Comments: "-", Branch: "color", Updated: "5m"},
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
		{Number: 1, Title: "Short", Author: "a", State: "open", Review: "-", Approvals: 0, Checks: "-", Comments: "-", Branch: "x", Updated: "1h"},
		{Number: 999, Title: "Longer title here", Author: "longuser", State: "merged", Review: "approved", Approvals: 3, Checks: "pass", Comments: "5/5", Branch: "feature/long", Updated: "30d"},
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

func TestFormatBranch(t *testing.T) {
	if got := formatBranch("feature/test"); got != "feature/test" {
		t.Fatalf("expected 'feature/test', got %q", got)
	}
	if got := formatBranch(""); got != "-" {
		t.Fatalf("expected '-', got %q", got)
	}
}
