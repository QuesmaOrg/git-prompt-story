package main

import (
	"github.com/QuesmaOrg/git-prompt-story/cmd"
)

// version is set via ldflags during build: -ldflags "-X main.version=x.y.z"
var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
