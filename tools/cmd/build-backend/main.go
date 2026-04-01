// build-backend builds the backend with build info injected via ldflags.
// Usage: go run ./tools/cmd/build-backend [output-path]
//
// Environment variables:
//
//	VERSION - Override version (default: from git tag or "dev")
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Find repo root (where go.mod is)
	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("finding repo root: %w", err)
	}

	output := "backend/bootstrap"
	if len(os.Args) > 1 {
		output = os.Args[1]
	}

	// Git metadata
	commitSHA := gitOutput(repoRoot, "rev-parse", "--short", "HEAD")
	if commitSHA == "" {
		commitSHA = "unknown"
	}

	commitTime := gitOutput(repoRoot, "log", "-1", "--format=%cI")
	branch := gitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if branch == "" {
		branch = "unknown"
	}

	// Version from env, tag, or default
	version := os.Getenv("VERSION")
	if version == "" {
		version = gitOutput(repoRoot, "describe", "--tags", "--always")
		if version == "" {
			version = "dev"
		}
	}

	// Build timestamp
	buildTime := time.Now().UTC().Format(time.RFC3339)

	// Package path for ldflags - read from go.mod to find module name
	moduleName, err := getModuleName(filepath.Join(repoRoot, "backend", "go.mod"))
	if err != nil {
		return fmt.Errorf("reading backend module name: %w", err)
	}
	pkgPath := moduleName + "/internal/buildinfo"

	fmt.Println("Building backend...")
	fmt.Printf("  Commit: %s (%s)\n", commitSHA, branch)
	fmt.Printf("  Version: %s\n", version)
	fmt.Printf("  Build time: %s\n", buildTime)

	// Build with ldflags
	ldflags := fmt.Sprintf("-s -w -X '%s.CommitSHA=%s' -X '%s.CommitTime=%s' -X '%s.BuildTime=%s' -X '%s.Version=%s' -X '%s.Branch=%s'",
		pkgPath, commitSHA,
		pkgPath, commitTime,
		pkgPath, buildTime,
		pkgPath, version,
		pkgPath, branch,
	)

	cmd := exec.Command("go", "build",
		"-ldflags", ldflags,
		"-o", filepath.Join(repoRoot, output),
		"./cmd/api",
	)
	cmd.Dir = filepath.Join(repoRoot, "backend")
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=arm64",
		"CGO_ENABLED=0",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	fmt.Printf("Built: %s\n", output)
	return nil
}

func findRepoRoot() (string, error) {
	// Start from current directory and walk up looking for go.mod or .git
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "backend", "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			return "", fmt.Errorf("could not find repo root (no .git or backend/go.mod found)")
		}
		dir = parent
	}
}

func gitOutput(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func getModuleName(goModPath string) (string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}
	return "", fmt.Errorf("module directive not found in %s", goModPath)
}
