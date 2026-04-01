// Package placeholders handles placeholder substitution for DS template files.
//
// Placeholders are tokens like ds-noc-v2 that get replaced with actual values
// during create-app and update-from-template operations.
//
// This package is fully dynamic: it reads ALL key=value pairs from .template-config
// and substitutes __KEY__ → value for each. No hardcoded fields.
//
// The placeholder syntax (prefix/suffix) is defined by constants and can be changed
// in one place if a different syntax is needed (e.g., {{KEY}} or ${KEY}).
package placeholders

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Placeholder delimiters - change these to modify placeholder syntax.
// For example, to use {{KEY}} syntax: Prefix="{{", Suffix="}}"
const (
	PlaceholderPrefix = "__"
	PlaceholderSuffix = "__"
)

// Config holds placeholder values derived from app configuration.
// Values is a dynamic map of KEY -> value (e.g., "APP_NAME" -> "my-app").
type Config struct {
	Values map[string]string
}

// NewConfig creates an empty Config with initialized map.
func NewConfig() *Config {
	return &Config{Values: make(map[string]string)}
}

// placeholderPattern matches placeholder tokens (e.g., __UPPERCASE_WITH_UNDERSCORES__).
// Built dynamically from PlaceholderPrefix and PlaceholderSuffix constants.
var placeholderPattern = regexp.MustCompile(
	regexp.QuoteMeta(PlaceholderPrefix) + `[A-Z][A-Z0-9_]*` + regexp.QuoteMeta(PlaceholderSuffix),
)

// MakeToken creates a placeholder token from a key name.
// e.g., MakeToken("APP_NAME") returns "ds-noc-v2" with default delimiters.
func MakeToken(key string) string {
	return PlaceholderPrefix + key + PlaceholderSuffix
}

// Extensions that should be processed for placeholder substitution
var Extensions = map[string]bool{
	".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".json": true,
	".md": true, ".yaml": true, ".yml": true, ".go": true, ".mod": true,
	".sum": true, ".sh": true, ".html": true, ".css": true, ".env": true,
}

// Replacements returns the map of token -> value for substitution.
// Generates __KEY__ tokens dynamically from all config values.
// Also computes derived values like APP_SUBDOMAIN and APP_ID if not explicitly set.
func (c *Config) Replacements() map[string]string {
	result := make(map[string]string)

	// Add derived values if not explicitly set
	if _, ok := c.Values["APP_SUBDOMAIN"]; !ok {
		if domain, ok := c.Values["APP_DOMAIN"]; ok && domain != "" {
			c.Values["APP_SUBDOMAIN"] = strings.Split(domain, ".")[0]
		}
	}
	if _, ok := c.Values["APP_ID"]; !ok {
		if appName, ok := c.Values["APP_NAME"]; ok && appName != "" {
			c.Values["APP_ID"] = strings.ReplaceAll(appName, "-", "")
		}
	}

	// Generate placeholder tokens dynamically for all config entries
	for key, value := range c.Values {
		result[MakeToken(key)] = value
	}

	return result
}

// Tokens returns all placeholder tokens from the current config.
// Computed dynamically from config values using PlaceholderPrefix/Suffix.
func (c *Config) Tokens() []string {
	tokens := make([]string, 0, len(c.Values))
	for key := range c.Values {
		tokens = append(tokens, MakeToken(key))
	}
	return tokens
}

// SubstituteFile reads a file, replaces placeholders, and writes it back.
// Returns true if the file was modified.
func SubstituteFile(path string, cfg *Config) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if !shouldProcess(path) {
		return false, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	original := string(content)
	modified := original

	for token, value := range cfg.Replacements() {
		modified = strings.ReplaceAll(modified, token, value)
	}

	if modified == original {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(modified), info.Mode()); err != nil {
		return false, err
	}

	return true, nil
}

// SubstituteDir walks a directory and substitutes placeholders in all eligible files.
// Returns the count of modified files.
func SubstituteDir(dir string, cfg *Config) (int, error) {
	count := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		modified, err := SubstituteFile(path, cfg)
		if err != nil {
			return nil // Skip files we can't process
		}
		if modified {
			count++
		}
		return nil
	})

	return count, err
}

// SubstituteContent replaces placeholders in content string
func SubstituteContent(content string, cfg *Config) string {
	for token, value := range cfg.Replacements() {
		content = strings.ReplaceAll(content, token, value)
	}
	return content
}

// HasPlaceholders checks if content contains any placeholder pattern (__UPPERCASE__).
// This is dynamic - it looks for the pattern, not a hardcoded list.
func HasPlaceholders(content string) bool {
	return placeholderPattern.MatchString(content)
}

// FindPlaceholders returns all placeholder tokens found in content.
func FindPlaceholders(content string) []string {
	return placeholderPattern.FindAllString(content, -1)
}

// LoadFromGoMod extracts app name and GitHub org from go.mod module path.
// e.g., "module github.com/DigiStratum/ds-kanban-v2/backend" -> ("ds-kanban-v2", "DigiStratum")
func LoadFromGoMod(appRoot string) (appName, githubOrg string, err error) {
	gomodPath := filepath.Join(appRoot, "backend", "go.mod")
	file, err := os.Open(gomodPath)
	if err != nil {
		return "", "", fmt.Errorf("opening go.mod: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			// Parse: module github.com/DigiStratum/ds-kanban-v2/backend
			parts := strings.Split(strings.TrimPrefix(line, "module "), "/")
			if len(parts) >= 3 && parts[0] == "github.com" {
				return parts[2], parts[1], nil
			}
		}
	}

	return "", "", fmt.Errorf("could not parse module path from go.mod")
}

// LoadFromTemplateConfig reads .template-config if it exists.
// Returns nil config if file doesn't exist (not an error).
// Reads ALL key=value pairs dynamically - no hardcoded field list.
func LoadFromTemplateConfig(appRoot string) (*Config, error) {
	configPath := filepath.Join(appRoot, ".template-config")
	file, err := os.Open(configPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := NewConfig()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Store ALL key=value pairs - no switch statement, fully dynamic
		cfg.Values[key] = value
	}

	return cfg, scanner.Err()
}

// shouldProcess returns true if the file should be processed for placeholder substitution
func shouldProcess(path string) bool {
	ext := filepath.Ext(path)
	base := filepath.Base(path)

	return Extensions[ext] ||
		base == "Makefile" ||
		base == "Dockerfile" ||
		strings.HasPrefix(base, ".env")
}

// Get returns the value for a key, or empty string if not found.
func (c *Config) Get(key string) string {
	if c.Values == nil {
		return ""
	}
	return c.Values[key]
}

// Set stores a key-value pair in the config.
func (c *Config) Set(key, value string) {
	if c.Values == nil {
		c.Values = make(map[string]string)
	}
	c.Values[key] = value
}
