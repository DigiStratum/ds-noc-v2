package placeholders

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	if cfg.Values == nil {
		t.Error("NewConfig should initialize Values map")
	}
}

func TestMakeToken(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"APP_NAME", PlaceholderPrefix + "APP_NAME" + PlaceholderSuffix},
		{"CUSTOM", PlaceholderPrefix + "CUSTOM" + PlaceholderSuffix},
		{"A", PlaceholderPrefix + "A" + PlaceholderSuffix},
	}
	
	for _, tt := range tests {
		got := MakeToken(tt.key)
		if got != tt.want {
			t.Errorf("MakeToken(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestConfigGetSet(t *testing.T) {
	cfg := NewConfig()
	cfg.Set("APP_NAME", "test-app")
	
	if got := cfg.Get("APP_NAME"); got != "test-app" {
		t.Errorf("Get(APP_NAME) = %q, want %q", got, "test-app")
	}
	
	if got := cfg.Get("NONEXISTENT"); got != "" {
		t.Errorf("Get(NONEXISTENT) = %q, want empty string", got)
	}
}

func TestReplacements(t *testing.T) {
	cfg := NewConfig()
	cfg.Set("APP_NAME", "my-app")
	cfg.Set("CUSTOM_FIELD", "custom-value")
	
	replacements := cfg.Replacements()
	
	appNameToken := MakeToken("APP_NAME")
	if got := replacements[appNameToken]; got != "my-app" {
		t.Errorf("%s = %q, want %q", appNameToken, got, "my-app")
	}
	
	customToken := MakeToken("CUSTOM_FIELD")
	if got := replacements[customToken]; got != "custom-value" {
		t.Errorf("%s = %q, want %q", customToken, got, "custom-value")
	}
}

func TestReplacementsDerivedValues(t *testing.T) {
	cfg := NewConfig()
	cfg.Set("APP_NAME", "my-app")
	cfg.Set("APP_DOMAIN", "my.example.com")
	
	replacements := cfg.Replacements()
	
	// APP_ID should be derived (hyphens removed)
	appIDToken := MakeToken("APP_ID")
	if got := replacements[appIDToken]; got != "myapp" {
		t.Errorf("%s = %q, want %q", appIDToken, got, "myapp")
	}
	
	// APP_SUBDOMAIN should be derived (first part of domain)
	subdomainToken := MakeToken("APP_SUBDOMAIN")
	if got := replacements[subdomainToken]; got != "my" {
		t.Errorf("%s = %q, want %q", subdomainToken, got, "my")
	}
}

func TestReplacementsExplicitOverridesDerived(t *testing.T) {
	cfg := NewConfig()
	cfg.Set("APP_NAME", "my-app")
	cfg.Set("APP_ID", "customid")       // explicit override
	cfg.Set("APP_SUBDOMAIN", "custom")  // explicit override
	
	replacements := cfg.Replacements()
	
	// Explicit values should be preserved
	appIDToken := MakeToken("APP_ID")
	if got := replacements[appIDToken]; got != "customid" {
		t.Errorf("%s = %q, want %q (explicit value)", appIDToken, got, "customid")
	}
	subdomainToken := MakeToken("APP_SUBDOMAIN")
	if got := replacements[subdomainToken]; got != "custom" {
		t.Errorf("%s = %q, want %q (explicit value)", subdomainToken, got, "custom")
	}
}

func TestTokens(t *testing.T) {
	cfg := NewConfig()
	cfg.Set("APP_NAME", "test")
	cfg.Set("CUSTOM", "value")
	
	tokens := cfg.Tokens()
	
	tokenSet := make(map[string]bool)
	for _, tok := range tokens {
		tokenSet[tok] = true
	}
	
	appNameToken := MakeToken("APP_NAME")
	if !tokenSet[appNameToken] {
		t.Errorf("Tokens should include %s", appNameToken)
	}
	customToken := MakeToken("CUSTOM")
	if !tokenSet[customToken] {
		t.Errorf("Tokens should include %s", customToken)
	}
}

func TestHasPlaceholders(t *testing.T) {
	// Build test cases using the actual delimiters
	tests := []struct {
		content string
		want    bool
	}{
		{"Table: " + MakeToken("REPO_NAME") + "-dev", true},
		{"Name: " + MakeToken("APP_NAME"), true},
		{"Custom: " + MakeToken("ANY_UPPERCASE_TOKEN"), true},
		{"No placeholder here", false},
		{"lowercase: " + PlaceholderPrefix + "lowercase" + PlaceholderSuffix + " not a placeholder", false},
		{"Mixed: " + PlaceholderPrefix + "Mixed_Case" + PlaceholderSuffix + " not a placeholder", false},
		{"Empty: " + PlaceholderPrefix + " " + PlaceholderSuffix + " not a placeholder", false},
		{"Multiple: " + MakeToken("FOO") + " and " + MakeToken("BAR"), true},
	}
	
	for _, tt := range tests {
		got := HasPlaceholders(tt.content)
		if got != tt.want {
			t.Errorf("HasPlaceholders(%q) = %v, want %v", tt.content, got, tt.want)
		}
	}
}

func TestFindPlaceholders(t *testing.T) {
	content := "Name: " + MakeToken("APP_NAME") + ", ID: " + MakeToken("APP_ID") + ", Table: " + MakeToken("REPO_NAME")
	found := FindPlaceholders(content)
	
	if len(found) != 3 {
		t.Errorf("FindPlaceholders found %d placeholders, want 3", len(found))
	}
	
	expectedSet := map[string]bool{
		MakeToken("APP_NAME"):  true,
		MakeToken("APP_ID"):    true,
		MakeToken("REPO_NAME"): true,
	}
	
	for _, ph := range found {
		if !expectedSet[ph] {
			t.Errorf("Unexpected placeholder found: %s", ph)
		}
	}
}

func TestSubstituteContent(t *testing.T) {
	cfg := NewConfig()
	cfg.Set("APP_NAME", "my-app")
	cfg.Set("REPO_NAME", "ds-app-my-app")
	
	content := "App: " + MakeToken("APP_NAME") + ", Repo: " + MakeToken("REPO_NAME")
	result := SubstituteContent(content, cfg)
	
	expected := "App: my-app, Repo: ds-app-my-app"
	if result != expected {
		t.Errorf("SubstituteContent = %q, want %q", result, expected)
	}
}

func TestLoadFromTemplateConfig(t *testing.T) {
	// Create temp directory with .template-config
	tmpDir, err := os.MkdirTemp("", "placeholder-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	configContent := `# Comment line
APP_NAME=test-app
APP_DISPLAY_NAME=Test App
CUSTOM_FIELD=custom-value

# Another comment
AWS_REGION=us-west-2
`
	err = os.WriteFile(filepath.Join(tmpDir, ".template-config"), []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	
	cfg, err := LoadFromTemplateConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromTemplateConfig error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadFromTemplateConfig returned nil config")
	}
	
	if got := cfg.Get("APP_NAME"); got != "test-app" {
		t.Errorf("APP_NAME = %q, want %q", got, "test-app")
	}
	if got := cfg.Get("APP_DISPLAY_NAME"); got != "Test App" {
		t.Errorf("APP_DISPLAY_NAME = %q, want %q", got, "Test App")
	}
	if got := cfg.Get("CUSTOM_FIELD"); got != "custom-value" {
		t.Errorf("CUSTOM_FIELD = %q, want %q", got, "custom-value")
	}
	if got := cfg.Get("AWS_REGION"); got != "us-west-2" {
		t.Errorf("AWS_REGION = %q, want %q", got, "us-west-2")
	}
}

func TestLoadFromTemplateConfigNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "placeholder-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	cfg, err := LoadFromTemplateConfig(tmpDir)
	if err != nil {
		t.Errorf("LoadFromTemplateConfig should not error for missing file: %v", err)
	}
	if cfg != nil {
		t.Error("LoadFromTemplateConfig should return nil for missing file")
	}
}

func TestSubstituteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "placeholder-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create test file with placeholders
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main
const AppName = "` + MakeToken("APP_NAME") + `"
const RepoName = "` + MakeToken("REPO_NAME") + `"
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	
	cfg := NewConfig()
	cfg.Set("APP_NAME", "my-app")
	cfg.Set("REPO_NAME", "ds-app-my-app")
	
	modified, err := SubstituteFile(testFile, cfg)
	if err != nil {
		t.Fatalf("SubstituteFile error: %v", err)
	}
	if !modified {
		t.Error("SubstituteFile should return true when file is modified")
	}
	
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	
	if !strings.Contains(string(result), `"my-app"`) {
		t.Error("File should contain substituted APP_NAME")
	}
	if !strings.Contains(string(result), `"ds-app-my-app"`) {
		t.Error("File should contain substituted REPO_NAME")
	}
}

func TestSubstituteFileNoChange(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "placeholder-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create test file without placeholders
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main
const AppName = "already-substituted"
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	
	cfg := NewConfig()
	cfg.Set("APP_NAME", "my-app")
	
	modified, err := SubstituteFile(testFile, cfg)
	if err != nil {
		t.Fatalf("SubstituteFile error: %v", err)
	}
	if modified {
		t.Error("SubstituteFile should return false when file is not modified")
	}
}

func TestShouldProcess(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/path/to/file.go", true},
		{"/path/to/file.ts", true},
		{"/path/to/file.tsx", true},
		{"/path/to/file.json", true},
		{"/path/to/file.yaml", true},
		{"/path/to/file.yml", true},
		{"/path/to/file.md", true},
		{"/path/to/file.sh", true},
		{"/path/to/Makefile", true},
		{"/path/to/Dockerfile", true},
		{"/path/to/.env", true},
		{"/path/to/.env.local", true},
		{"/path/to/file.png", false},
		{"/path/to/file.jpg", false},
		{"/path/to/file.exe", false},
		{"/path/to/file.bin", false},
	}
	
	for _, tt := range tests {
		got := shouldProcess(tt.path)
		if got != tt.want {
			t.Errorf("shouldProcess(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestPlaceholderConstants(t *testing.T) {
	// Verify constants are set (basic sanity check)
	if PlaceholderPrefix == "" {
		t.Error("PlaceholderPrefix should not be empty")
	}
	if PlaceholderSuffix == "" {
		t.Error("PlaceholderSuffix should not be empty")
	}
	
	// Verify MakeToken uses the constants
	token := MakeToken("TEST")
	if !strings.HasPrefix(token, PlaceholderPrefix) {
		t.Errorf("MakeToken should use PlaceholderPrefix, got %q", token)
	}
	if !strings.HasSuffix(token, PlaceholderSuffix) {
		t.Errorf("MakeToken should use PlaceholderSuffix, got %q", token)
	}
}
