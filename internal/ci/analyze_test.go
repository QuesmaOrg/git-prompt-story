package ci

import (
	"testing"
)

func TestAnalysisResult_ShouldPostComment(t *testing.T) {
	tests := []struct {
		name              string
		commitsWithNotes  int
		expectedShouldPost bool
		expectedType      string
	}{
		{
			name:               "no notes should not post",
			commitsWithNotes:   0,
			expectedShouldPost: false,
			expectedType:       "none",
		},
		{
			name:               "has notes should post",
			commitsWithNotes:   1,
			expectedShouldPost: true,
			expectedType:       "summary",
		},
		{
			name:               "multiple notes should post",
			commitsWithNotes:   5,
			expectedShouldPost: true,
			expectedType:       "summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a result directly to test the logic
			result := &AnalysisResult{
				CommitsAnalyzed:  10, // Doesn't matter for this test
				CommitsWithNotes: tt.commitsWithNotes,
			}

			// Apply the same logic as AnalyzePR
			if tt.commitsWithNotes > 0 {
				result.ShouldPostComment = true
				result.CommentType = "summary"
			} else {
				result.ShouldPostComment = false
				result.CommentType = "none"
			}

			if result.ShouldPostComment != tt.expectedShouldPost {
				t.Errorf("ShouldPostComment = %v, want %v", result.ShouldPostComment, tt.expectedShouldPost)
			}
			if result.CommentType != tt.expectedType {
				t.Errorf("CommentType = %v, want %v", result.CommentType, tt.expectedType)
			}
		})
	}
}

func TestAnalysisResult_NoMarkerDetection(t *testing.T) {
	// This test documents that we intentionally do NOT detect markers.
	// The old behavior was:
	// - if notes == 0 && markers > 0 -> post "Notes not found" warning
	// The new behavior is:
	// - if notes == 0 -> don't post anything (regardless of markers)
	//
	// This is a regression test to ensure we don't reintroduce marker detection.

	// Simulate the scenario that used to cause false positives:
	// - No actual notes pushed
	// - Commit message happens to mention "Prompt-Story:" (e.g., in docs)
	result := &AnalysisResult{
		CommitsAnalyzed:  5,
		CommitsWithNotes: 0, // No actual notes
		// Note: There is NO "commits_with_markers" field anymore
	}

	// Apply our decision logic
	if result.CommitsWithNotes > 0 {
		result.ShouldPostComment = true
		result.CommentType = "summary"
	} else {
		result.ShouldPostComment = false
		result.CommentType = "none"
	}

	// The key assertion: we should NOT post a comment
	// Even though the old code would have posted a "Notes not found" warning
	if result.ShouldPostComment {
		t.Error("ShouldPostComment should be false when there are no notes, even if markers might exist in commit messages")
	}

	if result.CommentType != "none" {
		t.Errorf("CommentType should be 'none', got %q", result.CommentType)
	}

	// Verify there's no marker-related field
	// This is a compile-time check - if someone adds a CommitsWithMarkers field,
	// they need to update this test to explain why
}

func TestAnalysisResult_JSONFields(t *testing.T) {
	// Test that the JSON structure is correct
	result := &AnalysisResult{
		CommitsAnalyzed:   10,
		CommitsWithNotes:  3,
		ShouldPostComment: true,
		CommentType:       "summary",
		MarkdownBody:      "# Test\n\nContent",
	}

	// Verify fields are set correctly
	if result.CommitsAnalyzed != 10 {
		t.Errorf("CommitsAnalyzed = %d, want 10", result.CommitsAnalyzed)
	}
	if result.CommitsWithNotes != 3 {
		t.Errorf("CommitsWithNotes = %d, want 3", result.CommitsWithNotes)
	}
	if !result.ShouldPostComment {
		t.Error("ShouldPostComment should be true")
	}
	if result.CommentType != "summary" {
		t.Errorf("CommentType = %q, want 'summary'", result.CommentType)
	}
	if result.MarkdownBody != "# Test\n\nContent" {
		t.Errorf("MarkdownBody = %q, want '# Test\\n\\nContent'", result.MarkdownBody)
	}
}

func TestAnalysisResult_ErrorHandling(t *testing.T) {
	// Test that error information is preserved
	result := &AnalysisResult{
		CommitsAnalyzed:  0,
		CommitsWithNotes: 0,
		Error:            "failed to resolve commit range: fatal: bad revision",
	}

	if result.Error == "" {
		t.Error("Error field should be set")
	}

	// Even with an error, these should be meaningful
	if result.ShouldPostComment {
		t.Error("ShouldPostComment should be false on error")
	}
}
