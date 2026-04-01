// pre-flight runs all local quality checks before commit.
// This catches CI failures locally, reducing the feedback loop.
//
// Usage: go run ./cmd/pre-flight [--verbose] [--fix]
//
// Run from the tools/ directory:
//   cd tools && go run ./cmd/pre-flight
//
// Hook management:
//   cd tools && go run ./cmd/pre-flight --install-hook    # Install git pre-commit hook
//   cd tools && go run ./cmd/pre-flight --uninstall-hook  # Remove git pre-commit hook
//
// Checks:
//   - Go: build, lint (golangci-lint), tests
//   - TypeScript: typecheck, lint (eslint), tests (vitest)
//   - Manifest: coverage check (template context only)
//   - HAL: compliance verification
//   - Builds: backend bootstrap, frontend dist, CDK synth
//
// Context Detection:
//   - If .template-version exists → template context (runs manifest check)
//   - Otherwise → derived app context (skips manifest check)
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Colors for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorBlue   = "\033[0;34m"
	colorBold   = "\033[1m"
)

// Check represents a single quality check
type Check struct {
	Name        string
	Category    string
	Command     string
	Args        []string
	Dir         string // relative to repo root
	FixCommand  string // optional fix command
	FixArgs     []string
	TemplateOnly bool // only run in template context
}

// CheckResult holds the result of a single check
type CheckResult struct {
	Check    Check
	Passed   bool
	Duration time.Duration
	Output   string
	Error    error
}

var verbose = false
var fix = false

// Pre-commit hook script template
const preCommitHookScript = `#!/bin/sh
# Pre-commit hook installed by pre-flight tool
# Runs quality checks before allowing commit
# 
# To uninstall: cd tools && go run ./cmd/pre-flight --uninstall-hook
# Or manually: rm .git/hooks/pre-commit

set -e

# Find repo root
REPO_ROOT="$(git rev-parse --show-toplevel)"

echo "🔍 Running pre-flight checks..."
echo ""

# Run pre-flight from tools directory
cd "$REPO_ROOT/tools"
go run ./cmd/pre-flight

# If we get here, all checks passed
exit 0
`

// Marker to identify hooks installed by this tool
const hookMarker = "# Pre-commit hook installed by pre-flight tool"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s%sError: %v%s\n", colorBold, colorRed, err, colorReset)
		os.Exit(1)
	}
}

func run() error {
	// Parse flags
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--verbose", "-v":
			verbose = true
		case "--fix":
			fix = true
		case "--install-hook":
			return installHook()
		case "--uninstall-hook":
			return uninstallHook()
		case "--help", "-h":
			printUsage()
			return nil
		default:
			return fmt.Errorf("unknown flag: %s", arg)
		}
	}

	// Find repo root
	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("finding repo root: %w", err)
	}

	// Detect context
	isTemplate := isTemplateContext(repoRoot)

	fmt.Printf("%s=== Pre-Flight Quality Gate ===%s\n", colorBold, colorReset)
	fmt.Printf("Root: %s\n", repoRoot)
	if isTemplate {
		fmt.Printf("Context: %sTemplate%s (running all checks including manifest)\n", colorBlue, colorReset)
	} else {
		fmt.Printf("Context: %sDerived App%s (skipping manifest check)\n", colorBlue, colorReset)
	}
	fmt.Println()

	// Define checks
	// Note: "go run" commands for other tools must run from tools/ dir
	checks := []Check{
		// Go checks
		{
			Name:     "Go Build",
			Category: "Backend",
			Command:  "go",
			Args:     []string{"build", "./..."},
			Dir:      "backend",
		},
		{
			Name:       "Go Lint",
			Category:   "Backend",
			Command:    "golangci-lint",
			Args:       []string{"run"},
			Dir:        "backend",
			FixCommand: "golangci-lint",
			FixArgs:    []string{"run", "--fix"},
		},
		{
			Name:     "Go Tests",
			Category: "Backend",
			Command:  "go",
			Args:     []string{"test", "./..."},
			Dir:      "backend",
		},

		// TypeScript Frontend checks
		{
			Name:     "TS Typecheck",
			Category: "Frontend",
			Command:  "pnpm",
			Args:     []string{"run", "typecheck"},
			Dir:      "frontend",
		},
		{
			Name:       "TS Lint",
			Category:   "Frontend",
			Command:    "pnpm",
			Args:       []string{"run", "lint"},
			Dir:        "frontend",
			FixCommand: "pnpm",
			FixArgs:    []string{"run", "lint", "--", "--fix"},
		},
		{
			Name:     "TS Tests",
			Category: "Frontend",
			Command:  "pnpm",
			Args:     []string{"vitest", "run", "--passWithNoTests"},
			Dir:      "frontend",
		},

		// Manifest check (template only) - runs from tools/
		{
			Name:         "Manifest Coverage",
			Category:     "Template",
			Command:      "go",
			Args:         []string{"run", "./cmd/check-manifest", "--strict"},
			Dir:          "tools",
			TemplateOnly: true,
		},

		// HAL compliance - runs from tools/
		{
			Name:     "HAL Compliance",
			Category: "API",
			Command:  "go",
			Args:     []string{"run", "./cmd/verify-hal-compliance"},
			Dir:      "tools",
		},

		// Build checks
		{
			Name:     "Backend Build",
			Category: "Builds",
			Command:  "go",
			Args:     []string{"run", "./cmd/build-backend"},
			Dir:      "tools",
		},
		{
			Name:     "Frontend Build",
			Category: "Builds",
			Command:  "pnpm",
			Args:     []string{"run", "build"},
			Dir:      "frontend",
		},
		{
			Name:     "CDK Synth",
			Category: "Builds",
			Command:  "npx",
			Args:     []string{"cdk", "synth", "--quiet"},
			Dir:      "infra",
		},
	}

	// Run checks
	results := make([]CheckResult, 0, len(checks))
	var currentCategory string

	for _, check := range checks {
		// Skip template-only checks in derived context
		if check.TemplateOnly && !isTemplate {
			continue
		}

		// Print category header
		if check.Category != currentCategory {
			if currentCategory != "" {
				fmt.Println()
			}
			fmt.Printf("%s--- %s ---%s\n", colorBold, check.Category, colorReset)
			currentCategory = check.Category
		}

		// Run check
		result := runCheck(check, repoRoot)
		results = append(results, result)

		// Print result
		if result.Passed {
			fmt.Printf("%s✓%s %s %s(%v)%s\n", colorGreen, colorReset, check.Name, colorBlue, result.Duration.Round(time.Millisecond), colorReset)
		} else {
			fmt.Printf("%s✗%s %s %s(%v)%s\n", colorRed, colorReset, check.Name, colorBlue, result.Duration.Round(time.Millisecond), colorReset)

			// Try fix if enabled and available
			if fix && check.FixCommand != "" {
				fmt.Printf("  %sAttempting fix...%s\n", colorYellow, colorReset)
				if runFix(check, repoRoot) {
					fmt.Printf("  %sFix applied, re-running...%s\n", colorGreen, colorReset)
					result = runCheck(check, repoRoot)
					results[len(results)-1] = result
					if result.Passed {
						fmt.Printf("  %s✓%s Fixed!\n", colorGreen, colorReset)
					}
				}
			}

			// Show output for failures
			if !result.Passed && result.Output != "" {
				indented := indentOutput(result.Output, "    ")
				fmt.Println(indented)
			}
		}

		if verbose && result.Passed && result.Output != "" {
			indented := indentOutput(result.Output, "    ")
			fmt.Println(indented)
		}
	}

	// Summary
	fmt.Println()
	fmt.Printf("%s=== Summary ===%s\n", colorBold, colorReset)

	passed := 0
	failed := 0
	var failedChecks []string

	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
			failedChecks = append(failedChecks, r.Check.Name)
		}
	}

	total := passed + failed
	fmt.Printf("Passed: %s%d/%d%s\n", colorGreen, passed, total, colorReset)

	if failed > 0 {
		fmt.Printf("Failed: %s%d%s\n", colorRed, failed, colorReset)
		fmt.Println()
		fmt.Println("Failed checks:")
		for _, name := range failedChecks {
			fmt.Printf("  %s✗%s %s\n", colorRed, colorReset, name)
		}
		fmt.Println()
		fmt.Println("Fix these issues before committing.")
		if !fix {
			fmt.Println("Tip: Run with --fix to auto-fix lint issues.")
		}
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("%s✓ All checks passed! Safe to commit.%s\n", colorGreen, colorReset)
	return nil
}

func printUsage() {
	fmt.Println("Usage: go run ./cmd/pre-flight [options]")
	fmt.Println()
	fmt.Println("Run from the tools/ directory:")
	fmt.Println("  cd tools && go run ./cmd/pre-flight")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --verbose, -v     Show output for passing checks")
	fmt.Println("  --fix             Auto-fix lint issues where possible")
	fmt.Println("  --install-hook    Install git pre-commit hook")
	fmt.Println("  --uninstall-hook  Remove git pre-commit hook")
	fmt.Println("  --help, -h        Show this help")
	fmt.Println()
	fmt.Println("Checks:")
	fmt.Println("  Backend:  go build, golangci-lint, go test")
	fmt.Println("  Frontend: tsc, eslint, vitest")
	fmt.Println("  Template: manifest coverage (template context only)")
	fmt.Println("  API:      HAL compliance")
	fmt.Println("  Builds:   backend binary, frontend dist, CDK synth")
	fmt.Println()
	fmt.Println("Context Detection:")
	fmt.Println("  Template context detected by presence of .template-version file.")
	fmt.Println("  Derived apps skip manifest check.")
	fmt.Println()
	fmt.Println("Git Hook:")
	fmt.Println("  Install a pre-commit hook to automatically run pre-flight before each commit.")
	fmt.Println("  The hook blocks commits if any checks fail.")
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Look for markers
		if _, err := os.Stat(filepath.Join(dir, "tools", "cmd", "pre-flight")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".template-manifest")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "backend", "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root")
		}
		dir = parent
	}
}

func isTemplateContext(repoRoot string) bool {
	_, err := os.Stat(filepath.Join(repoRoot, ".template-version"))
	return err == nil
}

func runCheck(check Check, repoRoot string) CheckResult {
	start := time.Now()

	dir := repoRoot
	if check.Dir != "" && check.Dir != "." {
		dir = filepath.Join(repoRoot, check.Dir)
	}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return CheckResult{
			Check:    check,
			Passed:   false,
			Duration: time.Since(start),
			Output:   fmt.Sprintf("Directory does not exist: %s", check.Dir),
			Error:    err,
		}
	}

	// Check if command exists
	if _, err := exec.LookPath(check.Command); err != nil {
		// For go commands, this shouldn't fail, but for others...
		if check.Command != "go" && check.Command != "pnpm" && check.Command != "npx" {
			return CheckResult{
				Check:    check,
				Passed:   false,
				Duration: time.Since(start),
				Output:   fmt.Sprintf("Command not found: %s (install it or skip this check)", check.Command),
				Error:    err,
			}
		}
	}

	cmd := exec.Command(check.Command, check.Args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	// Truncate very long output
	if len(output) > 2000 {
		output = output[:2000] + "\n... (truncated)"
	}

	return CheckResult{
		Check:    check,
		Passed:   err == nil,
		Duration: duration,
		Output:   strings.TrimSpace(output),
		Error:    err,
	}
}

func runFix(check Check, repoRoot string) bool {
	if check.FixCommand == "" {
		return false
	}

	dir := repoRoot
	if check.Dir != "" && check.Dir != "." {
		dir = filepath.Join(repoRoot, check.Dir)
	}

	cmd := exec.Command(check.FixCommand, check.FixArgs...)
	cmd.Dir = dir

	err := cmd.Run()
	return err == nil
}

func indentOutput(output string, indent string) string {
	if output == "" {
		return ""
	}

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

// installHook installs the git pre-commit hook
func installHook() error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("finding repo root: %w", err)
	}

	gitDir := filepath.Join(repoRoot, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository (no .git directory found)")
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	hookPath := filepath.Join(hooksDir, "pre-commit")

	// Check if hook already exists
	if _, err := os.Stat(hookPath); err == nil {
		// Read existing hook to check if it's ours
		content, err := os.ReadFile(hookPath)
		if err != nil {
			return fmt.Errorf("reading existing hook: %w", err)
		}

		if strings.Contains(string(content), hookMarker) {
			fmt.Printf("%s✓%s Pre-commit hook already installed at %s\n", colorGreen, colorReset, hookPath)
			return nil
		}

		// Existing hook not installed by us
		return fmt.Errorf("pre-commit hook already exists and was not installed by this tool.\n"+
			"  Path: %s\n"+
			"  To replace it, first remove or rename the existing hook.", hookPath)
	}

	// Write the hook
	if err := os.WriteFile(hookPath, []byte(preCommitHookScript), 0755); err != nil {
		return fmt.Errorf("writing hook: %w", err)
	}

	fmt.Printf("%s✓%s Pre-commit hook installed at %s\n", colorGreen, colorReset, hookPath)
	fmt.Println()
	fmt.Println("The hook will run pre-flight checks before each commit.")
	fmt.Println("To uninstall: go run ./cmd/pre-flight --uninstall-hook")
	fmt.Println()
	fmt.Println("To bypass the hook for a single commit:")
	fmt.Println("  git commit --no-verify -m \"message\"")
	return nil
}

// uninstallHook removes the git pre-commit hook if it was installed by this tool
func uninstallHook() error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("finding repo root: %w", err)
	}

	hookPath := filepath.Join(repoRoot, ".git", "hooks", "pre-commit")

	// Check if hook exists
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		fmt.Printf("%s!%s No pre-commit hook found at %s\n", colorYellow, colorReset, hookPath)
		return nil
	}

	// Read hook to verify it's ours
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return fmt.Errorf("reading hook: %w", err)
	}

	if !strings.Contains(string(content), hookMarker) {
		return fmt.Errorf("pre-commit hook was not installed by this tool.\n"+
			"  Path: %s\n"+
			"  Remove it manually if you want to delete it.", hookPath)
	}

	// Remove the hook
	if err := os.Remove(hookPath); err != nil {
		return fmt.Errorf("removing hook: %w", err)
	}

	fmt.Printf("%s✓%s Pre-commit hook uninstalled from %s\n", colorGreen, colorReset, hookPath)
	return nil
}
