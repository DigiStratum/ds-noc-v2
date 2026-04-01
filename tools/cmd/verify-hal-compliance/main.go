// verify-hal-compliance validates HAL/HATEOAS compliance for the backend.
//
// Validates:
// 1. All routes in main.go have corresponding links in discovery.go
// 2. Discovery links use proper HAL format
// 3. No hardcoded API paths in frontend (optional, with --frontend flag)
//
// Usage: go run ./tools/cmd/verify-hal-compliance [--frontend]
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Colors
const (
	red    = "\033[0;31m"
	green  = "\033[0;32m"
	yellow = "\033[1;33m"
	reset  = "\033[0m"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %v%s\n", red, err, reset)
		os.Exit(2)
	}
}

func run() error {
	checkFrontend := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--frontend":
			checkFrontend = true
		case "-h", "--help":
			fmt.Println("Usage: go run ./tools/cmd/verify-hal-compliance [--frontend]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --frontend  Also check frontend for hardcoded API paths")
			return nil
		}
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("finding repo root: %w", err)
	}

	mainFile := filepath.Join(repoRoot, "backend/cmd/api/main.go")
	discoveryFile := filepath.Join(repoRoot, "backend/internal/discovery/discovery.go")

	fmt.Println("=== HAL/HATEOAS Compliance Check ===")
	fmt.Println()

	// Check required files exist
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		return fmt.Errorf("main.go not found at %s", mainFile)
	}
	if _, err := os.Stat(discoveryFile); os.IsNotExist(err) {
		return fmt.Errorf("discovery.go not found at %s", discoveryFile)
	}

	fmt.Printf("Checking: %s\n", mainFile)
	fmt.Printf("Against:  %s\n", discoveryFile)
	fmt.Println()

	errors := 0
	warnings := 0

	// Extract routes from main.go
	routes, err := extractRoutes(mainFile)
	if err != nil {
		return fmt.Errorf("extracting routes: %w", err)
	}

	// Extract discovery paths
	discoveryPaths, err := extractDiscoveryPaths(discoveryFile)
	if err != nil {
		return fmt.Errorf("extracting discovery paths: %w", err)
	}

	fmt.Println("--- Routes in main.go ---")
	if len(routes) == 0 {
		fmt.Println("(none found)")
	} else {
		for _, r := range routes {
			fmt.Println(r)
		}
	}
	fmt.Println()

	fmt.Println("--- Links in discovery.go ---")
	if len(discoveryPaths) == 0 {
		fmt.Println("(none found)")
	} else {
		for _, p := range discoveryPaths {
			fmt.Println(p)
		}
	}
	fmt.Println()

	// Standard endpoints to skip
	standardEndpoints := map[string]bool{
		"/api/health":    true,
		"/api/discovery": true,
	}

	fmt.Println("--- Compliance Check ---")
	for _, route := range routes {
		// Extract just the path (remove method prefix if present)
		path := extractPath(route)

		// Skip standard endpoints
		if standardEndpoints[path] {
			continue
		}

		// Skip auth endpoints (internal)
		if strings.HasPrefix(path, "/api/auth/") {
			continue
		}

		// Check if path exists in discovery
		if containsPath(discoveryPaths, path) {
			fmt.Printf("%s✓%s %s\n", green, reset, path)
		} else {
			fmt.Printf("%s✗%s %s - NOT in discovery.go\n", red, reset, path)
			errors++
		}
	}
	fmt.Println()

	// Check discovery.go format
	fmt.Println("--- HAL Format Check ---")
	discoveryContent, _ := os.ReadFile(discoveryFile)
	dcStr := string(discoveryContent)

	// Check for _links pattern
	if strings.Contains(dcStr, `"_links"`) || strings.Contains(dcStr, "Links:") {
		fmt.Printf("%s✓%s Uses Links map\n", green, reset)
	} else {
		fmt.Printf("%s✗%s Missing Links map - not HAL compliant\n", red, reset)
		errors++
	}

	// Check for self link
	if strings.Contains(dcStr, `"self"`) {
		fmt.Printf("%s✓%s Has 'self' link\n", green, reset)
	} else {
		fmt.Printf("%s✗%s Missing 'self' link - required by HAL\n", red, reset)
		errors++
	}

	// Check for CURIEs
	if strings.Contains(dcStr, "curies") || strings.Contains(dcStr, "Curies") || strings.Contains(dcStr, "CURIE") {
		fmt.Printf("%s✓%s Has CURIEs defined\n", green, reset)
	} else {
		fmt.Printf("%s⚠%s No CURIEs found (recommended for documentation)\n", yellow, reset)
		warnings++
	}

	// Check Content-Type
	if strings.Contains(dcStr, "application/hal+json") {
		fmt.Printf("%s✓%s Sets Content-Type: application/hal+json\n", green, reset)
	} else {
		fmt.Printf("%s✗%s Missing Content-Type: application/hal+json\n", red, reset)
		errors++
	}
	fmt.Println()

	// Optional frontend check
	if checkFrontend {
		fmt.Println("--- Frontend Check ---")
		frontendDir := filepath.Join(repoRoot, "frontend/src")

		if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
			fmt.Printf("%s⚠%s Frontend directory not found at %s\n", yellow, reset, frontendDir)
		} else {
			hardcoded := findHardcodedPaths(frontendDir)
			if len(hardcoded) > 0 {
				fmt.Printf("%s✗%s Found hardcoded API paths in frontend:\n", red, reset)
				for _, h := range hardcoded {
					fmt.Println(h)
				}
				fmt.Println()
				fmt.Println("Use useHALNavigation hook instead:")
				fmt.Println("  const { getHref } = useHALNavigation();")
				fmt.Println("  const url = getHref('ds:items');")
				errors++
			} else {
				fmt.Printf("%s✓%s No hardcoded API paths found\n", green, reset)
			}
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("=== Summary ===")
	if errors == 0 {
		if warnings == 0 {
			fmt.Printf("%sAll checks passed!%s\n", green, reset)
		} else {
			fmt.Printf("%sAll checks passed%s (%s%d warnings%s)\n", green, reset, yellow, warnings, reset)
		}
		return nil
	}

	fmt.Printf("%s%d error(s)%s, %s%d warning(s)%s\n", red, errors, reset, yellow, warnings, reset)
	fmt.Println()
	fmt.Println("Fix the errors above before merging.")
	fmt.Println("Run: go run ./tools/cmd/add-endpoint ... to add new endpoints properly.")
	os.Exit(1)
	return nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "backend/cmd/api/main.go")); err == nil {
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

func extractRoutes(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`(GET|POST|PUT|PATCH|DELETE)?\s*/api/[a-zA-Z0-9/_{}:-]+`)
	matches := re.FindAllString(string(content), -1)

	seen := make(map[string]bool)
	var routes []string
	for _, m := range matches {
		m = strings.TrimSpace(m)
		if !seen[m] {
			seen[m] = true
			routes = append(routes, m)
		}
	}
	return routes, nil
}

func extractDiscoveryPaths(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`Href:\s*"(/api/[^"]+)"`)
	matches := re.FindAllStringSubmatch(string(content), -1)

	seen := make(map[string]bool)
	var paths []string
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			seen[m[1]] = true
			paths = append(paths, m[1])
		}
	}
	return paths, nil
}

func extractPath(route string) string {
	re := regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE)\s+`)
	return re.ReplaceAllString(route, "")
}

func containsPath(paths []string, path string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}

func findHardcodedPaths(dir string) []string {
	var results []string
	re := regexp.MustCompile(`"/api/`)

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx") {
			return nil
		}

		// Skip hal.ts
		if strings.HasSuffix(path, "hal.ts") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Skip imports and comments
			if strings.Contains(line, "import") ||
				strings.Contains(line, "// eslint-disable") ||
				strings.Contains(line, "discovery") {
				continue
			}

			if re.MatchString(line) {
				relPath, _ := filepath.Rel(dir, path)
				results = append(results, fmt.Sprintf("%s:%d: %s", relPath, lineNum, strings.TrimSpace(line)))
			}
		}
		return nil
	})

	return results
}
