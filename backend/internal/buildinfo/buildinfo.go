// Package buildinfo provides build metadata injected at compile time.
// See scripts/build.sh for how these values are set via -ldflags.
package buildinfo

import (
	"encoding/json"
	"net/http"
	"time"
)

// Build metadata - injected via ldflags at build time
var (
	// CommitSHA is the git commit hash
	CommitSHA = "dev"
	// CommitTime is the commit timestamp (RFC3339)
	CommitTime = ""
	// BuildTime is when the binary was built (RFC3339)
	BuildTime = ""
	// Version is the semantic version or tag
	Version = "dev"
	// Branch is the git branch name
	Branch = ""
)

// Info contains build metadata.
type Info struct {
	CommitSHA  string `json:"commit_sha"`
	CommitTime string `json:"commit_time,omitempty"`
	BuildTime  string `json:"build_time"`
	Version    string `json:"version"`
	Branch     string `json:"branch,omitempty"`
}

// Get returns the current build info.
func Get() Info {
	buildTime := BuildTime
	if buildTime == "" {
		buildTime = time.Now().UTC().Format(time.RFC3339)
	}
	
	return Info{
		CommitSHA:  CommitSHA,
		CommitTime: CommitTime,
		BuildTime:  buildTime,
		Version:    Version,
		Branch:     Branch,
	}
}

// Handler returns an HTTP handler for the /api/build endpoint.
// This endpoint is unauthenticated - build info is not sensitive.
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	
	info := Get()
	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
