// check-manifest validates that all files in the repo are either:
// 1. Listed in .template-manifest (template-owned), OR
// 2. Listed in .template-overrides (app customizations of template files), OR
// 3. In a known app-owned location
//
// Usage: go run ./tools/cmd/check-manifest [--strict]
//
//	--strict: Exit with error if unknown files found
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Known app-owned locations (not in manifest, but expected)
// These are directories/patterns where apps put their custom code
var appOwnedPatterns = []string{
	// Backend app code - handlers and domain-specific packages
	`^backend/internal/handlers/`,
	`^backend/internal/models/`,
	`^backend/internal/storage/`,
	// Apps may create domain-specific packages directly under internal/
	// These are common patterns for app business logic
	`^backend/internal/apps/`,
	`^backend/internal/orgs/`,
	`^backend/internal/users/`,
	`^backend/internal/sso/`,
	`^backend/internal/authprovider/`,
	`^backend/internal/jwt/`,
	`^backend/internal/dynamo/`,
	`^backend/internal/postgres/`,
	`^backend/internal/services/`,
	`^backend/internal/domain/`,
	// Backend integrations and domain packages
	`^backend/internal/integrations/`,
	`^backend/internal/personas/`,
	`^backend/internal/pool/`,
	`^backend/internal/repositories/`,
	`^backend/internal/auth/`,
	`^backend/internal/dispatcher/`,
	`^backend/internal/workflow/`,

	// Frontend app code
	`^frontend/src/app/`,
	`^frontend/src/features/`,
	`^frontend/src/pages/`,
	`^frontend/src/types/`,
	`^frontend/src/hooks/`,
	// E2E tests (backend and frontend)
	`^tests/`,
	`^frontend/e2e/`,
	// Build/config files apps may add
	`^frontend/src/vite-env\.d\.ts$`,

	// Infrastructure app-specific stacks
	`^infra/lib/app/`,
	`^infra/lib/app-stack\.ts$`,
	`^infra/cdk\.context\.json$`,
	`^infra/package-lock\.json$`,

	// Root-level app files
	`^README\.md$`,
	`^REQUIREMENTS\.md$`,
	`^\.template-overrides$`,
	`^\.template-manifest$`,
	`^\.template-version$`,
	`^\.template-config$`,
	`^\.template-tokens$`,
	`^go\.work$`,

	// Environment and generated files
	`^\.env`,
	`^node_modules/`,
	`^\.git/`,
	`^dist/`,
	`^build/`,
	`^coverage/`,
	`^\.DS_Store$`,
	`^tools/`,

	// App-specific workflow files (PR validation, etc.)
	`^\.github/workflows/pr-validation\.yml$`,
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	strict := false
	for _, arg := range os.Args[1:] {
		if arg == "--strict" {
			strict = true
		}
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("finding repo root: %w", err)
	}

	manifestPath := filepath.Join(repoRoot, ".template-manifest")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf(".template-manifest not found")
	}

	// Load manifest patterns (template-owned files)
	templatePatterns, err := loadManifestPatterns(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	// Load overrides (app customizations of template files)
	overridesPath := filepath.Join(repoRoot, ".template-overrides")
	overrideFiles := make(map[string]bool)
	if _, err := os.Stat(overridesPath); err == nil {
		overrideFiles, err = loadOverrideFiles(overridesPath)
		if err != nil {
			return fmt.Errorf("loading overrides: %w", err)
		}
	}

	// Compile app-owned patterns
	appOwnedRE := make([]*regexp.Regexp, 0, len(appOwnedPatterns))
	for _, p := range appOwnedPatterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return fmt.Errorf("compiling app-owned pattern %q: %w", p, err)
		}
		appOwnedRE = append(appOwnedRE, re)
	}

	// Get tracked files
	trackedFiles, err := gitLsFiles(repoRoot)
	if err != nil {
		return fmt.Errorf("listing git files: %w", err)
	}

	// Check each file
	var unknownFiles []string
	for _, file := range trackedFiles {
		if isKnown(file, templatePatterns, appOwnedRE, overrideFiles) {
			continue
		}
		unknownFiles = append(unknownFiles, file)
	}

	// Report results
	if len(unknownFiles) == 0 {
		fmt.Println("✅ All files accounted for in manifest or app-owned locations")
		return nil
	}

	fmt.Printf("⚠️  Found %d file(s) not in manifest or app-owned locations:\n\n", len(unknownFiles))
	for _, f := range unknownFiles {
		fmt.Printf("  - %s\n", f)
	}
	fmt.Println()
	fmt.Println("Actions:")
	fmt.Println("  1. Add to .template-manifest if template-owned")
	fmt.Println("  2. Add to .template-overrides if app customization of template file")
	fmt.Println("  3. Move to app-owned location (backend/internal/handlers/, frontend/src/app/)")
	fmt.Println("  4. Add pattern to appOwnedPatterns in this tool if new app location")

	if strict {
		os.Exit(1)
	}
	return nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".template-manifest")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root")
		}
		dir = parent
	}
}

func loadManifestPatterns(path string) ([]*regexp.Regexp, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []*regexp.Regexp
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Escape dots and build pattern
		escaped := regexp.QuoteMeta(line)
		var pattern string
		if strings.HasSuffix(line, "/") {
			// Directory: match anything under it
			pattern = "^" + escaped
		} else {
			// File: exact match
			pattern = "^" + escaped + "$"
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling pattern %q: %w", line, err)
		}
		patterns = append(patterns, re)
	}

	return patterns, scanner.Err()
}

func loadOverrideFiles(path string) (map[string]bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	files := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		files[line] = true
	}

	return files, scanner.Err()
}

func gitLsFiles(dir string) ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func isKnown(file string, templatePatterns []*regexp.Regexp, appOwnedRE []*regexp.Regexp, overrideFiles map[string]bool) bool {
	// Check if file is in overrides (app customization)
	if overrideFiles[file] {
		return true
	}

	// Check template-owned
	for _, re := range templatePatterns {
		if re.MatchString(file) {
			return true
		}
	}

	// Check app-owned
	for _, re := range appOwnedRE {
		if re.MatchString(file) {
			return true
		}
	}

	return false
}
