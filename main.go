package main

import (
	_ "embed"
	"strings"

	"github.com/QuesmaOrg/git-prompt-story/cmd"
)

//go:embed VERSION
var version string

func main() {
	cmd.SetVersion(strings.TrimSpace(version))
	cmd.Execute()
}
