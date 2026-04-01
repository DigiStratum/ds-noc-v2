package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPlaceholderConfigMergesGoModAndTemplateConfig(t *testing.T) {
	// Create temp directory with both .template-config and go.mod
	tmpDir, err := os.MkdirTemp("", "update-template-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend/go.mod with app name
	backendDir := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatal(err)
	}
	goModContent := `module github.com/DigiStratum/test-app/backend

go 1.23
`
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .template-config with ONLY extra fields (no APP_NAME)
	configContent := `# Custom fields only - APP_NAME should come from go.mod
CUSTOM_TABLE_PREFIX=myprefix
FEATURE_FLAG=enabled
AWS_REGION=eu-west-1
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".template-config"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadPlaceholderConfig(tmpDir)
	if err != nil {
		t.Fatalf("loadPlaceholderConfig error: %v", err)
	}

	// Should have APP_NAME from go.mod
	if got := cfg.Get("APP_NAME"); got != "test-app" {
		t.Errorf("APP_NAME = %q, want %q (from go.mod)", got, "test-app")
	}

	// Should have GITHUB_ORG from go.mod
	if got := cfg.Get("GITHUB_ORG"); got != "DigiStratum" {
		t.Errorf("GITHUB_ORG = %q, want %q (from go.mod)", got, "DigiStratum")
	}

	// Should have custom fields from .template-config
	if got := cfg.Get("CUSTOM_TABLE_PREFIX"); got != "myprefix" {
		t.Errorf("CUSTOM_TABLE_PREFIX = %q, want %q (from .template-config)", got, "myprefix")
	}
	if got := cfg.Get("FEATURE_FLAG"); got != "enabled" {
		t.Errorf("FEATURE_FLAG = %q, want %q (from .template-config)", got, "enabled")
	}

	// AWS_REGION from .template-config should override default
	if got := cfg.Get("AWS_REGION"); got != "eu-west-1" {
		t.Errorf("AWS_REGION = %q, want %q (from .template-config override)", got, "eu-west-1")
	}
}

func TestLoadPlaceholderConfigTemplateConfigOverridesGoMod(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "update-template-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend/go.mod
	backendDir := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatal(err)
	}
	goModContent := `module github.com/DigiStratum/test-app/backend

go 1.23
`
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .template-config that overrides APP_NAME
	configContent := `APP_NAME=custom-app-name
APP_DISPLAY_NAME=Custom App Display Name
GITHUB_ORG=CustomOrg
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".template-config"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadPlaceholderConfig(tmpDir)
	if err != nil {
		t.Fatalf("loadPlaceholderConfig error: %v", err)
	}

	// All values should come from .template-config (overriding go.mod)
	if got := cfg.Get("APP_NAME"); got != "custom-app-name" {
		t.Errorf("APP_NAME = %q, want %q (from .template-config)", got, "custom-app-name")
	}
	if got := cfg.Get("APP_DISPLAY_NAME"); got != "Custom App Display Name" {
		t.Errorf("APP_DISPLAY_NAME = %q, want %q (from .template-config)", got, "Custom App Display Name")
	}
	if got := cfg.Get("GITHUB_ORG"); got != "CustomOrg" {
		t.Errorf("GITHUB_ORG = %q, want %q (from .template-config)", got, "CustomOrg")
	}
}

func TestLoadPlaceholderConfigGoModOnlyFallback(t *testing.T) {
	// Create temp directory with ONLY go.mod (no .template-config)
	tmpDir, err := os.MkdirTemp("", "update-template-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend/go.mod
	backendDir := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatal(err)
	}
	goModContent := `module github.com/DigiStratum/another-app/backend

go 1.23
`
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadPlaceholderConfig(tmpDir)
	if err != nil {
		t.Fatalf("loadPlaceholderConfig error: %v", err)
	}

	// Should derive values from go.mod
	if got := cfg.Get("APP_NAME"); got != "another-app" {
		t.Errorf("APP_NAME = %q, want %q", got, "another-app")
	}
	if got := cfg.Get("GITHUB_ORG"); got != "DigiStratum" {
		t.Errorf("GITHUB_ORG = %q, want %q", got, "DigiStratum")
	}
	// Should have default AWS region
	if got := cfg.Get("AWS_REGION"); got != "us-west-2" {
		t.Errorf("AWS_REGION = %q, want %q (default)", got, "us-west-2")
	}
}
