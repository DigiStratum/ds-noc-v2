// check-requirement-coverage scans E2E tests for requirement IDs and compares
// against REQUIREMENTS.md. Reports coverage and detects untested requirements.
//
// Usage: go run ./tools/cmd/check-requirement-coverage [--strict] [--json]
//
//	--strict: Exit with error if any requirements are untested
//	--json:   Output results as JSON
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Colors
const (
	red    = "\033[0;31m"
	green  = "\033[0;32m"
	yellow = "\033[0;33m"
	blue   = "\033[0;34m"
	reset  = "\033[0m"
)

type Report struct {
	TotalRequirements  int      `json:"total_requirements"`
	TestedRequirements int      `json:"tested_requirements"`
	SkippedRequirements int     `json:"skipped_requirements"`
	UntestedRequirements int    `json:"untested_requirements"`
	OrphanedTests      int      `json:"orphaned_tests"`
	CoveragePercentage float64  `json:"coverage_percentage"`
	Untested           []string `json:"untested"`
	Orphaned           []string `json:"orphaned"`
	Skipped            []string `json:"skipped"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	strictMode := false
	jsonOutput := false

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--strict":
			strictMode = true
		case "--json":
			jsonOutput = true
		}
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("finding repo root: %w", err)
	}

	requirementsFile := filepath.Join(repoRoot, "REQUIREMENTS.md")
	e2eDir := filepath.Join(repoRoot, "frontend/e2e")

	// Check required files exist
	if _, err := os.Stat(requirementsFile); os.IsNotExist(err) {
		return fmt.Errorf("REQUIREMENTS.md not found at %s", requirementsFile)
	}
	if _, err := os.Stat(e2eDir); os.IsNotExist(err) {
		return fmt.Errorf("E2E directory not found at %s", e2eDir)
	}

	// Extract requirements
	allRequirements, err := extractRequirements(requirementsFile)
	if err != nil {
		return fmt.Errorf("extracting requirements: %w", err)
	}

	// Extract tested requirements from E2E tests
	testedRequirements, err := extractTestedRequirements(e2eDir)
	if err != nil {
		return fmt.Errorf("extracting tested requirements: %w", err)
	}

	// Extract skipped requirements
	skippedRequirements, err := extractSkippedRequirements(e2eDir)
	if err != nil {
		return fmt.Errorf("extracting skipped requirements: %w", err)
	}

	// Calculate untested
	var untested []string
	for _, req := range allRequirements {
		if !contains(testedRequirements, req) {
			untested = append(untested, req)
		}
	}

	// Calculate orphaned (tests without matching requirement)
	var orphaned []string
	for _, req := range testedRequirements {
		if !contains(allRequirements, req) {
			orphaned = append(orphaned, req)
		}
	}

	// Build report
	totalCount := len(allRequirements)
	testedCount := len(testedRequirements)
	skippedCount := len(skippedRequirements)
	untestedCount := len(untested)
	orphanedCount := len(orphaned)

	var coverage float64
	if totalCount > 0 {
		coverage = float64(testedCount*100) / float64(totalCount)
	}

	report := Report{
		TotalRequirements:   totalCount,
		TestedRequirements:  testedCount,
		SkippedRequirements: skippedCount,
		UntestedRequirements: untestedCount,
		OrphanedTests:       orphanedCount,
		CoveragePercentage:  coverage,
		Untested:            untested,
		Orphaned:            orphaned,
		Skipped:             skippedRequirements,
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	// Human-readable output
	fmt.Println()
	fmt.Printf("%s═══════════════════════════════════════════════════════════════%s\n", blue, reset)
	fmt.Printf("%s           E2E Test to Requirement Traceability Report         %s\n", blue, reset)
	fmt.Printf("%s═══════════════════════════════════════════════════════════════%s\n", blue, reset)
	fmt.Println()
	fmt.Printf("Requirements file: %s\n", requirementsFile)
	fmt.Printf("E2E test directory: %s\n", e2eDir)
	fmt.Println()

	// Summary
	fmt.Printf("%s── Summary ──%s\n", blue, reset)
	fmt.Printf("  Total requirements:    %d\n", totalCount)
	fmt.Printf("  Tested requirements:   %s%d%s\n", green, testedCount, reset)
	fmt.Printf("  Skipped tests:         %s%d%s\n", yellow, skippedCount, reset)
	fmt.Printf("  Untested requirements: %s%d%s\n", red, untestedCount, reset)
	fmt.Printf("  Orphaned tests:        %s%d%s\n", yellow, orphanedCount, reset)
	fmt.Println()

	// Coverage bar
	if totalCount > 0 {
		filled := testedCount * 40 / totalCount
		empty := 40 - filled
		fmt.Print("  Coverage: [")
		for i := 0; i < filled; i++ {
			fmt.Print("█")
		}
		for i := 0; i < empty; i++ {
			fmt.Print("░")
		}
		fmt.Printf("] %.1f%%\n", coverage)
		fmt.Println()
	}

	// Untested requirements
	if len(untested) > 0 {
		fmt.Printf("%s── Untested Requirements ──%s\n", red, reset)
		for _, req := range untested {
			fmt.Printf("  %s✗%s %s\n", red, reset, req)
		}
		fmt.Println()
	}

	// Skipped tests
	if len(skippedRequirements) > 0 {
		fmt.Printf("%s── Skipped Tests ──%s\n", yellow, reset)
		for _, req := range skippedRequirements {
			fmt.Printf("  %s⊘%s %s (test.skip)\n", yellow, reset, req)
		}
		fmt.Println()
	}

	// Orphaned tests
	if len(orphaned) > 0 {
		fmt.Printf("%s── Orphaned Tests (no matching requirement) ──%s\n", yellow, reset)
		for _, req := range orphaned {
			fmt.Printf("  %s?%s %s\n", yellow, reset, req)
		}
		fmt.Println()
	}

	// Tested requirements by category
	if len(testedRequirements) > 0 {
		fmt.Printf("%s── Tested Requirements by Category ──%s\n", green, reset)

		categories := []string{"AUTH", "TENANT", "NAV", "THEME", "I18N", "PERF", "AVAIL", "SEC", "A11Y", "TEST", "MON", "APP"}
		for _, cat := range categories {
			var catReqs []string
			for _, req := range testedRequirements {
				if strings.Contains(req, "-"+cat+"-") {
					catReqs = append(catReqs, req)
				}
			}
			if len(catReqs) > 0 {
				fmt.Printf("  %s%s:%s\n", blue, cat, reset)
				for _, req := range catReqs {
					fmt.Printf("    %s✓%s %s\n", green, reset, req)
				}
			}
		}
		fmt.Println()
	}

	fmt.Printf("%s═══════════════════════════════════════════════════════════════%s\n", blue, reset)

	// Exit code based on strict mode
	if strictMode && untestedCount > 0 {
		fmt.Printf("%sError: %d requirements are not covered by E2E tests.%s\n", red, untestedCount, reset)
		fmt.Println("Run with --strict=false to allow deployment anyway.")
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
		if _, err := os.Stat(filepath.Join(dir, "REQUIREMENTS.md")); err == nil {
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

func extractRequirements(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`(FR|NFR)-[A-Z0-9]+-[0-9]+`)
	matches := re.FindAllString(string(content), -1)

	return uniqueSorted(matches), nil
}

func extractTestedRequirements(dir string) ([]string, error) {
	var matches []string
	re := regexp.MustCompile(`(FR|NFR)-[A-Z0-9]+-[0-9]+`)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".spec.ts") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Look for describe blocks
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.Contains(line, "describe(") && !strings.Contains(line, ".skip(") {
				found := re.FindAllString(line, -1)
				matches = append(matches, found...)
			}
		}
		return nil
	})

	return uniqueSorted(matches), err
}

func extractSkippedRequirements(dir string) ([]string, error) {
	var matches []string
	re := regexp.MustCompile(`(FR|NFR)-[A-Z0-9]+-[0-9]+`)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".spec.ts") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, ".skip(") {
				found := re.FindAllString(line, -1)
				matches = append(matches, found...)
			}
		}
		return nil
	})

	return uniqueSorted(matches), err
}

func uniqueSorted(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	sort.Strings(result)
	return result
}

func contains(items []string, item string) bool {
	for _, i := range items {
		if i == item {
			return true
		}
	}
	return false
}
