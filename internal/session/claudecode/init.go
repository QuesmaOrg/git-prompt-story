package claudecode

import (
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

func init() {
	session.RegisterDiscoverer(NewDiscoverer())
	session.RegisterParser(NewParser())
}
