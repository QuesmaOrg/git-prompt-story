package ci

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed templates/*.css templates/*.tmpl
var templateFS embed.FS

// IndexData holds data for the index page template
type IndexData struct {
	PRNumber         int
	CSS              template.CSS
	Commits          []CommitViewData
	CommitsAnalyzed  int
	CommitsWithNotes int
	TotalPrompts     int
}

// CommitViewData holds data for displaying a commit
type CommitViewData struct {
	SHA         string
	ShortSHA    string
	Subject     string
	Sessions    []SessionSummary
	StartWork   time.Time
	EndWork     time.Time
	ToolNames   string
	PromptCount int
	CSS         template.CSS
}

// GenerateHTML creates HTML files for the summary in the output directory
func GenerateHTML(summary *Summary, outputDir string, prNumber int) error {
	// Load CSS
	cssBytes, err := templateFS.ReadFile("templates/styles.css")
	if err != nil {
		return fmt.Errorf("failed to load CSS: %w", err)
	}
	css := template.CSS(cssBytes)

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse templates with helper functions
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Local().Format("2006-01-02 15:04")
		},
		"formatTimeShort": func(t time.Time) string {
			return t.Local().Format("15:04")
		},
		"formatToolName": formatToolName,
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n-3] + "..."
		},
		"add": func(a, b int) int {
			return a + b
		},
	}

	// Load and parse index template
	indexTmplBytes, err := templateFS.ReadFile("templates/index.html.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load index template: %w", err)
	}
	indexTmpl, err := template.New("index").Funcs(funcMap).Parse(string(indexTmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse index template: %w", err)
	}

	// Load and parse commit template
	commitTmplBytes, err := templateFS.ReadFile("templates/commit.html.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load commit template: %w", err)
	}
	commitTmpl, err := template.New("commit").Funcs(funcMap).Parse(string(commitTmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse commit template: %w", err)
	}

	// Prepare commit view data
	var commits []CommitViewData
	for _, cs := range summary.Commits {
		cvd := CommitViewData{
			SHA:       cs.SHA,
			ShortSHA:  cs.ShortSHA,
			Subject:   cs.Subject,
			Sessions:  cs.Sessions,
			StartWork: cs.StartWork,
			EndWork:   cs.EndWork,
			CSS:       css,
		}

		// Calculate tool names and prompt count
		tools := make(map[string]bool)
		for _, sess := range cs.Sessions {
			tools[formatToolName(sess.Tool)] = true
			cvd.PromptCount += len(sess.Prompts)
		}
		var toolNames []string
		for t := range tools {
			toolNames = append(toolNames, t)
		}
		cvd.ToolNames = strings.Join(toolNames, ", ")

		commits = append(commits, cvd)
	}

	// Generate index.html
	indexData := IndexData{
		PRNumber:         prNumber,
		CSS:              css,
		Commits:          commits,
		CommitsAnalyzed:  summary.CommitsAnalyzed,
		CommitsWithNotes: summary.CommitsWithNotes,
		TotalPrompts:     summary.TotalPrompts,
	}

	indexPath := filepath.Join(outputDir, "index.html")
	indexFile, err := os.Create(indexPath)
	if err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}
	defer indexFile.Close()

	if err := indexTmpl.Execute(indexFile, indexData); err != nil {
		return fmt.Errorf("failed to render index.html: %w", err)
	}

	// Generate individual commit pages
	for _, cvd := range commits {
		commitPath := filepath.Join(outputDir, cvd.ShortSHA+".html")
		commitFile, err := os.Create(commitPath)
		if err != nil {
			return fmt.Errorf("failed to create %s.html: %w", cvd.ShortSHA, err)
		}

		if err := commitTmpl.Execute(commitFile, cvd); err != nil {
			commitFile.Close()
			return fmt.Errorf("failed to render %s.html: %w", cvd.ShortSHA, err)
		}
		commitFile.Close()
	}

	return nil
}
