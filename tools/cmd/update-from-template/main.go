// update-from-template pulls template updates into a derived app.
//
// Usage: go run ./tools/cmd/update-from-template [--dry-run] [--template-path /path/to/template]
//
// This tool:
// 1. Reads .template-manifest from the TEMPLATE repo (authoritative list of template files)
// 2. Reads .template-overrides from the APP repo (files the app has customized)
// 3. Syncs files from manifest MINUS overrides
// 4. Performs placeholder substitution using app's actual values
// 5. Updates .template-version
//
// Placeholder values are read from:
// 1. .template-config (if exists) - explicit config file
// 2. backend/go.mod - extracts app name and GitHub org from module path
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/DigiStratum/ds-app-template/tools/internal/placeholders"
)

// Colors
const (
	red    = "\033[0;31m"
	green  = "\033[0;32m"
	yellow = "\033[1;33m"
	blue   = "\033[0;34m"
	reset  = "\033[0m"
)

func info(msg string)    { fmt.Printf("%s[INFO]%s %s\n", blue, reset, msg) }
func warn(msg string)    { fmt.Printf("%s[WARN]%s %s\n", yellow, reset, msg) }
func success(msg string) { fmt.Printf("%s[OK]%s %s\n", green, reset, msg) }
func errorf(msg string)  { fmt.Fprintf(os.Stderr, "%s[ERROR]%s %s\n", red, reset, msg) }

// skipDirs are directories that should never be copied from template
var skipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"dist":         true,
	"coverage":     true,
	".turbo":       true,
	"cdk.out":      true,
}

func main() {
	if err := run(); err != nil {
		errorf(err.Error())
		os.Exit(1)
	}
}

func run() error {
	dryRun := false
	templatePath := ""

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			dryRun = true
		case "--template-path":
			if i+1 < len(args) {
				templatePath = args[i+1]
				i++
			}
		case "-h", "--help":
			fmt.Println("Usage: go run ./tools/cmd/update-from-template [--dry-run] [--template-path /path/to/template]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --dry-run           Show what would be updated without making changes")
			fmt.Println("  --template-path     Path to ds-app-template repo (default: auto-detect)")
			fmt.Println()
			fmt.Println("Files:")
			fmt.Println("  .template-manifest  (in template) - Files owned by template")
			fmt.Println("  .template-overrides (in app)      - Files app has customized (skip these)")
			fmt.Println("  .template-config    (in app)      - App placeholder values (optional)")
			fmt.Println()
			fmt.Println("Placeholder substitution:")
			fmt.Println("  Values read from .template-config or inferred from backend/go.mod")
			fmt.Println("  Tokens like ds-noc-v2 are replaced with actual values")
			return nil
		}
	}

	appRoot, err := findAppRoot()
	if err != nil {
		return fmt.Errorf("finding app root: %w", err)
	}

	// Find template repo
	if templatePath == "" {
		candidates := []string{
			filepath.Join(appRoot, "..", "ds-app-template"),
			filepath.Join(os.Getenv("HOME"), "repos/digistratum/ds-app-template"),
			filepath.Join(os.Getenv("HOME"), "ds-app-template"),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(filepath.Join(candidate, ".template-manifest")); err == nil {
				templatePath = candidate
				break
			}
		}
	}

	if templatePath == "" {
		return fmt.Errorf("cannot find ds-app-template repo - use --template-path to specify location")
	}

	templatePath, _ = filepath.Abs(templatePath)
	info(fmt.Sprintf("Template repo: %s", templatePath))
	info(fmt.Sprintf("App root: %s", appRoot))

	// Check for manifest in template
	manifestPath := filepath.Join(templatePath, ".template-manifest")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("no .template-manifest found in template repo")
	}

	// Get versions
	templateVersion := readFileString(filepath.Join(templatePath, ".template-version"), "unknown")
	currentVersion := readFileString(filepath.Join(appRoot, ".template-version"), "none")

	info(fmt.Sprintf("Template version: %s", templateVersion))
	info(fmt.Sprintf("Current app version: %s", currentVersion))

	// Load placeholder config
	phCfg, err := loadPlaceholderConfig(appRoot)
	if err != nil {
		return fmt.Errorf("loading placeholder config: %w", err)
	}
	info(fmt.Sprintf("App name: %s", phCfg.Get("APP_NAME")))
	info(fmt.Sprintf("GitHub org: %s", phCfg.Get("GITHUB_ORG")))

	// Load overrides from app repo
	overrides := loadOverrides(filepath.Join(appRoot, ".template-overrides"))
	if len(overrides) > 0 {
		info(fmt.Sprintf("App overrides: %d files excluded", len(overrides)))
	}

	fmt.Println()

	if dryRun {
		warn("DRY RUN MODE — no changes will be made")
		fmt.Println()
	}

	// Expand manifest entries to individual files
	manifestFiles, err := expandManifest(manifestPath, templatePath)
	if err != nil {
		return fmt.Errorf("expanding manifest: %w", err)
	}

	updatedCount := 0
	skippedCount := 0
	overrideCount := 0
	substitutedCount := 0

	for _, relPath := range manifestFiles {
		// Check if this file is overridden by app
		if overrides[relPath] {
			if dryRun {
				fmt.Printf("  [override] %s (skipped - app customized)\n", relPath)
			}
			overrideCount++
			continue
		}

		src := filepath.Join(templatePath, relPath)
		dst := filepath.Join(appRoot, relPath)

		// Check if source exists
		if _, err := os.Stat(src); os.IsNotExist(err) {
			warn(fmt.Sprintf("Template file does not exist: %s", relPath))
			continue
		}

		if dryRun {
			if _, err := os.Stat(dst); err == nil {
				srcContent, _ := os.ReadFile(src)
				substituted := placeholders.SubstituteContent(string(srcContent), phCfg)
				dstContent, _ := os.ReadFile(dst)

				if substituted != string(dstContent) {
					if placeholders.HasPlaceholders(string(srcContent)) {
						fmt.Printf("  [update+subst] %s\n", relPath)
						substitutedCount++
					} else {
						fmt.Printf("  [update] %s\n", relPath)
					}
					updatedCount++
				} else {
					skippedCount++
				}
			} else {
				fmt.Printf("  [new] %s\n", relPath)
				updatedCount++
			}
		} else {
			// Create parent dir if needed
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return fmt.Errorf("creating directory for %s: %w", relPath, err)
			}

			// Copy file
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("copying file %s: %w", relPath, err)
			}

			// Substitute placeholders
			modified, err := placeholders.SubstituteFile(dst, phCfg)
			if err != nil {
				warn(fmt.Sprintf("placeholder substitution failed for %s: %v", relPath, err))
			}
			if modified {
				substitutedCount++
			}

			updatedCount++
		}
	}

	// Update version file
	if !dryRun {
		if err := os.WriteFile(filepath.Join(appRoot, ".template-version"), []byte(templateVersion+"\n"), 0644); err != nil {
			return fmt.Errorf("updating .template-version: %w", err)
		}
	}

	fmt.Println()
	if dryRun {
		info(fmt.Sprintf("Dry run complete. %d files would be updated (%d with placeholder substitution), %d unchanged, %d overridden.",
			updatedCount, substitutedCount, skippedCount, overrideCount))
		info("Run without --dry-run to apply changes.")
	} else {
		success(fmt.Sprintf("Template update complete! %d files synced (%d with placeholder substitution), %d overridden.",
			updatedCount, substitutedCount, overrideCount))
		fmt.Println()
		info("Review changes with: git status")
		info(fmt.Sprintf("Commit with: git add -A && git commit -m 'chore: update from template v%s'", templateVersion))
	}

	return nil
}

// loadPlaceholderConfig loads placeholder values from .template-config and go.mod.
// Priority: .template-config values override go.mod-derived values.
// This ensures all fields from .template-config are preserved, even when APP_NAME
// is inferred from go.mod.
func loadPlaceholderConfig(appRoot string) (*placeholders.Config, error) {
	// Load .template-config if it exists
	fileCfg, err := placeholders.LoadFromTemplateConfig(appRoot)
	if err != nil {
		return nil, err
	}

	// Start with go.mod-derived values as base
	appName, githubOrg, goModErr := placeholders.LoadFromGoMod(appRoot)

	// Build config with go.mod values as defaults
	cfg := placeholders.NewConfig()

	// If go.mod parsing succeeded, use those as base values
	if goModErr == nil {
		cfg.Set("APP_NAME", appName)
		cfg.Set("APP_DISPLAY_NAME", formatDisplayName(appName))
		cfg.Set("GITHUB_ORG", githubOrg)
	}

	// Set default AWS region
	cfg.Set("AWS_REGION", "us-west-2")

	// Overlay .template-config values (these take priority)
	if fileCfg != nil {
		for key, value := range fileCfg.Values {
			cfg.Set(key, value)
		}
	}

	// Ensure we have at least APP_NAME from one source
	if cfg.Get("APP_NAME") == "" {
		if goModErr != nil {
			return nil, fmt.Errorf("could not determine app config: %w\n\nCreate .template-config with:\n  APP_NAME=your-app-name\n  GITHUB_ORG=YourOrg", goModErr)
		}
		return nil, fmt.Errorf("APP_NAME not found in .template-config or go.mod")
	}

	return cfg, nil
}

// formatDisplayName converts "ds-kanban-v2" to "DS Kanban V2"
func formatDisplayName(name string) string {
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if part == "ds" {
			parts[i] = "DS"
		} else {
			parts[i] = strings.Title(part)
		}
	}
	return strings.Join(parts, " ")
}

// expandManifest reads the manifest and expands directory entries to individual files
func expandManifest(manifestPath, templatePath string) ([]string, error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var files []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fullPath := filepath.Join(templatePath, line)
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			// File doesn't exist in template, skip
			continue
		}
		if err != nil {
			return nil, err
		}

		if info.IsDir() || strings.HasSuffix(line, "/") {
			// Expand directory to individual files
			dirFiles, err := expandDirectory(fullPath, templatePath)
			if err != nil {
				return nil, err
			}
			files = append(files, dirFiles...)
		} else {
			files = append(files, line)
		}
	}

	return files, scanner.Err()
}

// expandDirectory walks a directory and returns all file paths relative to templatePath
func expandDirectory(dirPath, templatePath string) ([]string, error) {
	var files []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded directories
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path from template root
		relPath, err := filepath.Rel(templatePath, path)
		if err != nil {
			return err
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}

// loadOverrides reads .template-overrides file and returns a set of paths to skip
func loadOverrides(path string) map[string]bool {
	overrides := make(map[string]bool)

	file, err := os.Open(path)
	if err != nil {
		return overrides // File doesn't exist = no overrides
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		overrides[line] = true
	}

	return overrides
}

func findAppRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Look for .template-version (sign of a derived app)
		if _, err := os.Stat(filepath.Join(dir, ".template-version")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find app root")
		}
		dir = parent
	}
}

func readFileString(path, defaultVal string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultVal
	}
	return strings.TrimSpace(string(data))
}

func filesEqual(a, b string) bool {
	dataA, err := os.ReadFile(a)
	if err != nil {
		return false
	}
	dataB, err := os.ReadFile(b)
	if err != nil {
		return false
	}
	return bytes.Equal(dataA, dataB)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
