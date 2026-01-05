package main

import (
	"runtime/debug"

	"github.com/QuesmaOrg/git-prompt-story/cmd"
)

// These are set via ldflags during GoReleaser build:
// -ldflags "-X main.version=x.y.z -X main.commit=abc123 -X main.date=2025-01-05"
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	// If version is still "dev", try to get it from build info (go install @version)
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			// Module version (e.g., v0.12.0)
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
			}
			// Get vcs info from build settings
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					if commit == "" && len(setting.Value) >= 7 {
						commit = setting.Value[:7]
					}
				case "vcs.time":
					if date == "" {
						date = setting.Value
					}
				}
			}
		}
	}

	cmd.SetVersionInfo(version, commit, date)
	cmd.Execute()
}
