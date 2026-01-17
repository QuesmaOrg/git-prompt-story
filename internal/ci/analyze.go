package ci

// AnalysisResult represents the full PR analysis for GitHub Actions
type AnalysisResult struct {
	CommitsAnalyzed   int    `json:"commits_analyzed"`
	CommitsWithNotes  int    `json:"commits_with_notes"`
	ShouldPostComment bool   `json:"should_post_comment"`
	CommentType       string `json:"comment_type"` // "summary" or "none"
	MarkdownBody      string `json:"markdown_body,omitempty"`
	Error             string `json:"error,omitempty"`
}

// AnalyzePR performs full PR analysis and determines what action to take.
// It centralizes the decision logic that was previously split across bash and JavaScript.
//
// The key insight: only post a comment if we have actual notes. We no longer try to
// detect "Prompt-Story:" markers in commit messages because:
// 1. Marker detection is fragile (false positives from mentions in docs, etc.)
// 2. "Prompt-Story: none" markers should NOT trigger any action
// 3. The user experience is better when we only comment when we have actual content
func AnalyzePR(commitRange string, pagesURL string, version string) (*AnalysisResult, error) {
	summary, err := GenerateSummary(commitRange, false)
	if err != nil {
		return &AnalysisResult{Error: err.Error()}, err
	}

	result := &AnalysisResult{
		CommitsAnalyzed:  summary.CommitsAnalyzed,
		CommitsWithNotes: summary.CommitsWithNotes,
	}

	// Simple logic: only post if we have actual notes
	// This avoids false "Notes not found" warnings
	if summary.CommitsWithNotes > 0 {
		result.ShouldPostComment = true
		result.CommentType = "summary"
		result.MarkdownBody = RenderMarkdown(summary, pagesURL, version)
	} else {
		result.ShouldPostComment = false
		result.CommentType = "none"
	}

	return result, nil
}
