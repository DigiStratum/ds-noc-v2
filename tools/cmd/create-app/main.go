// create-app spawns a new DS Ecosystem app from template.
//
// Usage: go run ./tools/cmd/create-app [OPTIONS] [output-directory]
//
// Creates a new app by:
// 1. Copying template to output directory
// 2. Prompting for configuration values (or using flags)
// 3. Replacing __PLACEHOLDER__ tokens
// 4. Initializing git repository
// 5. Optionally setting up GitHub remote
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Colors
const (
	red    = "\033[0;31m"
	green  = "\033[0;32m"
	yellow = "\033[1;33m"
	blue   = "\033[0;34m"
	reset  = "\033[0m"
)

// Config holds app configuration
type Config struct {
	AppName         string
	AppDisplayName  string
	RepoName        string // Computed: ds-app-{AppName}
	GitHubOrg       string
	OutputDir       string
	SetupGitHub     bool
	AutoYes         bool
	SkipGit         bool
	TemplateVersion string
	RepoRoot        string
}

const (
	defaultGitHubOrg = "DigiStratum"
	repoNamePrefix   = "ds-app-"
)

func info(msg string)    { fmt.Printf("%s[INFO]%s %s\n", blue, reset, msg) }
func warn(msg string)    { fmt.Printf("%s[WARN]%s %s\n", yellow, reset, msg) }
func success(msg string) { fmt.Printf("%s[SUCCESS]%s %s\n", green, reset, msg) }
func errorf(msg string)  { fmt.Fprintf(os.Stderr, "%s[ERROR]%s %s\n", red, reset, msg) }

func main() {
	if err := run(); err != nil {
		errorf(err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg := &Config{
		GitHubOrg: defaultGitHubOrg,
	}

	// Find repo root
	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("finding repo root: %w", err)
	}
	cfg.RepoRoot = repoRoot

	// Read template version
	cfg.TemplateVersion = readFileString(filepath.Join(repoRoot, ".template-version"), "unknown")

	// Parse arguments
	if err := parseArgs(cfg); err != nil {
		return err
	}

	// Collect configuration interactively if needed
	if err := collectConfig(cfg); err != nil {
		return err
	}

	// Show confirmation
	if err := confirmConfig(cfg); err != nil {
		return err
	}

	// Copy template
	if err := copyTemplate(cfg); err != nil {
		return fmt.Errorf("copying template: %w", err)
	}

	// Replace placeholders
	if err := replacePlaceholders(cfg); err != nil {
		return fmt.Errorf("replacing placeholders: %w", err)
	}

	// Validate all placeholders were substituted
	if err := validatePlaceholders(cfg); err != nil {
		return fmt.Errorf("placeholder validation failed: %w", err)
	}

	// Initialize git
	if err := initGit(cfg); err != nil {
		return fmt.Errorf("initializing git: %w", err)
	}

	// Setup GitHub
	if err := setupGitHub(cfg); err != nil {
		return fmt.Errorf("setting up GitHub: %w", err)
	}

	// Check AWS resources
	checkAWSResources(cfg)

	// Print next steps
	printNextSteps(cfg)

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
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find ds-app-template root")
		}
		dir = parent
	}
}

func parseArgs(cfg *Config) error {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--name":
			if i+1 < len(args) {
				cfg.AppName = args[i+1]
				i++
			}
		case "-N", "--display-name":
			if i+1 < len(args) {
				cfg.AppDisplayName = args[i+1]
				i++
			}
		case "-g", "--github":
			cfg.SetupGitHub = true
		case "-o", "--org":
			if i+1 < len(args) {
				cfg.GitHubOrg = args[i+1]
				i++
			}
		case "-y", "--yes":
			cfg.AutoYes = true
		case "--no-git":
			cfg.SkipGit = true
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		default:
			if !strings.HasPrefix(args[i], "-") {
				cfg.OutputDir = args[i]
			}
		}
	}
	return nil
}

func printUsage() {
	fmt.Println(`Usage: go run ./tools/cmd/create-app [OPTIONS] [output-directory]

Create a new DS Ecosystem app from template.

Arguments:
  output-directory    Where to create the app (default: ./ds-app-<name>)

Options:
  -n, --name NAME           App name (e.g., workforce, marketplace)
  -N, --display-name NAME   Display name (e.g., DS Marketplace)
  -g, --github              Set up GitHub remote after creation
  -o, --org ORG             GitHub organization (default: DigiStratum)
  -y, --yes                 Skip confirmation prompts (for automation)
  --no-git                  Skip git init (for refreshing existing repos)
  -h, --help                Show this help message

Example:
  go run ./tools/cmd/create-app -n workforce --github
  # Creates ds-app-workforce directory and DigiStratum/ds-app-workforce repo`)
}

func collectConfig(cfg *Config) error {
	fmt.Println()
	info(fmt.Sprintf("DS App Template v%s", cfg.TemplateVersion))
	fmt.Println("======================================")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// App name (short name, e.g., "workforce")
	if cfg.AppName == "" {
		cfg.AppName = prompt(reader, "App name (e.g., workforce, marketplace)", "", cfg.AutoYes)
	}

	// Strip any ds-app- or ds- prefix - we want the base name
	cfg.AppName = strings.TrimPrefix(cfg.AppName, "ds-app-")
	cfg.AppName = strings.TrimPrefix(cfg.AppName, "ds-")

	if !validateAppName(cfg.AppName) {
		return fmt.Errorf("app name must start with a letter, contain only lowercase letters, numbers, and hyphens")
	}

	// Compute repo name (always ds-app-{name})
	cfg.RepoName = repoNamePrefix + cfg.AppName

	// Display name
	if cfg.AppDisplayName == "" {
		suggested := toDisplayName(cfg.AppName)
		cfg.AppDisplayName = prompt(reader, "Display name", suggested, cfg.AutoYes)
	}

	// GitHub org
	cfg.GitHubOrg = prompt(reader, "GitHub organization", cfg.GitHubOrg, cfg.AutoYes)

	// Output directory (defaults to repo name)
	if cfg.OutputDir == "" {
		cfg.OutputDir = prompt(reader, "Output directory", "./"+cfg.RepoName, cfg.AutoYes)
	}

	// GitHub setup
	if !cfg.SetupGitHub && !cfg.AutoYes {
		response := prompt(reader, "Set up GitHub remote? [y/N]", "n", false)
		cfg.SetupGitHub = response == "y" || response == "Y"
	}

	return nil
}

func confirmConfig(cfg *Config) error {
	fmt.Println()
	fmt.Println("======================================")
	info("Configuration Summary")
	fmt.Println("======================================")
	fmt.Printf("  App Name:      %s\n", cfg.AppName)
	fmt.Printf("  Repo Name:     %s\n", cfg.RepoName)
	fmt.Printf("  Display Name:  %s\n", cfg.AppDisplayName)
	fmt.Printf("  GitHub Org:    %s\n", cfg.GitHubOrg)
	fmt.Printf("  Output:        %s\n", cfg.OutputDir)
	fmt.Printf("  GitHub Setup:  %v\n", cfg.SetupGitHub)
	fmt.Println("======================================")
	fmt.Println()

	if !cfg.AutoYes {
		reader := bufio.NewReader(os.Stdin)
		response := prompt(reader, "Proceed? [Y/n]", "y", false)
		if response == "n" || response == "N" {
			info("Aborted")
			os.Exit(0)
		}
	}

	return nil
}

func copyTemplate(cfg *Config) error {
	info(fmt.Sprintf("Creating app at %s...", cfg.OutputDir))

	// Check if output already exists
	if _, err := os.Stat(cfg.OutputDir); err == nil {
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, ".template-version")); err == nil {
			warn("Directory exists and appears to be a template-derived app")
			info("Re-running token replacement")
			return nil
		}
		return fmt.Errorf("directory already exists: %s", cfg.OutputDir)
	}

	// Create output directory
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return err
	}

	// Copy using rsync if available, otherwise use Go implementation
	rsync, err := exec.LookPath("rsync")
	if err == nil {
		cmd := exec.Command(rsync, "-a",
			"--exclude=.git",
			"--exclude=scripts/create-app.sh",
			"--exclude=node_modules",
			cfg.RepoRoot+"/",
			cfg.OutputDir+"/")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("rsync failed: %w", err)
		}
	} else {
		// Fallback to Go copy
		if err := copyDir(cfg.RepoRoot, cfg.OutputDir, []string{".git", "scripts/create-app.sh", "node_modules"}); err != nil {
			return err
		}
	}

	success("Template copied")
	return nil
}

func replacePlaceholders(cfg *Config) error {
	info("Replacing placeholders...")

	// Derive values
	// APP_SUBDOMAIN matches APP_NAME (short name without ds-app- prefix)
	appID := strings.ReplaceAll(cfg.AppName, "-", "")

	replacements := map[string]string{
		"ds-noc-v2":         cfg.AppName,
		"DS Noc V2": cfg.AppDisplayName,
		"noc-v2":    cfg.AppName, // subdomain = app name
		"dsnocv2":           appID,
		"__REPO_NAME__":        cfg.RepoName,
		"DigiStratum":       cfg.GitHubOrg,
	}

	extensions := map[string]bool{
		".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".json": true,
		".md": true, ".yaml": true, ".yml": true, ".go": true, ".mod": true,
		".sum": true, ".sh": true, ".html": true, ".css": true, ".env": true,
	}

	count := 0
	err := filepath.Walk(cfg.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		base := filepath.Base(path)

		// Check if file should be processed
		shouldProcess := extensions[ext] ||
			base == "Makefile" ||
			base == "Dockerfile" ||
			strings.HasPrefix(base, ".env")

		if !shouldProcess {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		original := string(content)
		modified := original

		for token, value := range replacements {
			modified = strings.ReplaceAll(modified, token, value)
		}

		if modified != original {
			if err := os.WriteFile(path, []byte(modified), info.Mode()); err != nil {
				return err
			}
			count++
		}
		return nil
	})

	if err != nil {
		return err
	}

	success(fmt.Sprintf("Replaced placeholders in %d files", count))
	return nil
}

// validatePlaceholders scans all text files for unsubstituted placeholders
// and fails with a clear error if any are found. This prevents creating
// repos with broken placeholder references.
func validatePlaceholders(cfg *Config) error {
	info("Validating placeholder substitution...")

	// Known placeholder patterns (only the ones we substitute)
	placeholders := []string{
		"ds-noc-v2",
		"DS Noc V2",
		"noc-v2",
		"dsnocv2",
		"__REPO_NAME__",
		"DigiStratum",
	}

	extensions := map[string]bool{
		".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".json": true,
		".md": true, ".yaml": true, ".yml": true, ".go": true, ".mod": true,
		".sum": true, ".sh": true, ".html": true, ".css": true, ".env": true,
	}

	// Track violations: map[placeholder][]filePath
	violations := make(map[string][]string)

	err := filepath.Walk(cfg.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Skip .git directory
		if strings.Contains(path, "/.git/") || strings.Contains(path, "\\.git\\") {
			return nil
		}

		ext := filepath.Ext(path)
		base := filepath.Base(path)

		// Check if file should be scanned
		shouldScan := extensions[ext] ||
			base == "Makefile" ||
			base == "Dockerfile" ||
			strings.HasPrefix(base, ".env")

		if !shouldScan {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		text := string(content)
		relPath, _ := filepath.Rel(cfg.OutputDir, path)

		for _, ph := range placeholders {
			if strings.Contains(text, ph) {
				violations[ph] = append(violations[ph], relPath)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(violations) > 0 {
		var sb strings.Builder
		sb.WriteString("unsubstituted placeholders found:\n")
		for ph, files := range violations {
			sb.WriteString(fmt.Sprintf("  %s in:\n", ph))
			for _, f := range files {
				sb.WriteString(fmt.Sprintf("    - %s\n", f))
			}
		}
		return fmt.Errorf(sb.String())
	}

	success("All placeholders substituted")
	return nil
}

func initGit(cfg *Config) error {
	if cfg.SkipGit {
		info("Skipping git init (--no-git specified)")
		return nil
	}

	info("Initializing git repository...")

	if _, err := os.Stat(filepath.Join(cfg.OutputDir, ".git")); err == nil {
		warn("Git repo already exists, skipping init")
		return nil
	}

	// git init
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = cfg.OutputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	// git add
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = cfg.OutputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	// git commit
	cmd = exec.Command("git", "commit", "-q", "-m",
		fmt.Sprintf("Initial commit from ds-app-template v%s", cfg.TemplateVersion))
	cmd.Dir = cfg.OutputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	// Create and checkout develop branch (develop-first workflow)
	cmd = exec.Command("git", "checkout", "-b", "develop")
	cmd.Dir = cfg.OutputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout -b develop: %w", err)
	}

	// Create main branch from develop
	cmd = exec.Command("git", "checkout", "-b", "main")
	cmd.Dir = cfg.OutputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout -b main: %w", err)
	}

	// Switch back to develop as the default working branch
	cmd = exec.Command("git", "checkout", "develop")
	cmd.Dir = cfg.OutputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout develop: %w", err)
	}

	success("Git repository initialized with develop-first branch workflow")
	return nil
}

func setupGitHub(cfg *Config) error {
	if !cfg.SetupGitHub {
		return nil
	}

	info("Setting up GitHub remote...")

	repoName := cfg.RepoName
	remoteURL := fmt.Sprintf("git@github.com:%s/%s.git", cfg.GitHubOrg, repoName)

	// Check if gh CLI is available
	gh, err := exec.LookPath("gh")
	if err != nil {
		warn("GitHub CLI (gh) not found. Skipping remote setup.")
		info("To set up manually:")
		fmt.Printf("  gh repo create %s/%s --private\n", cfg.GitHubOrg, repoName)
		fmt.Printf("  git remote add origin %s\n", remoteURL)
		fmt.Println("  git push -u origin main")
		return nil
	}

	// Check if repo exists
	cmd := exec.Command(gh, "repo", "view", fmt.Sprintf("%s/%s", cfg.GitHubOrg, repoName))
	cmd.Dir = cfg.OutputDir
	if err := cmd.Run(); err == nil {
		warn(fmt.Sprintf("Repository %s/%s already exists", cfg.GitHubOrg, repoName))
		// Add remote if not exists
		cmd = exec.Command("git", "remote", "get-url", "origin")
		cmd.Dir = cfg.OutputDir
		if cmd.Run() != nil {
			cmd = exec.Command("git", "remote", "add", "origin", remoteURL)
			cmd.Dir = cfg.OutputDir
			cmd.Run()
			success("Added remote origin")
		}
	} else {
		// Create repo
		if cfg.AutoYes {
			cmd = exec.Command(gh, "repo", "create", fmt.Sprintf("%s/%s", cfg.GitHubOrg, repoName),
				"--private", "--source=.", "--remote=origin")
			cmd.Dir = cfg.OutputDir
			if err := cmd.Run(); err != nil {
				warn("Failed to create GitHub repository")
			} else {
				success("Created GitHub repository")
			}
		} else {
			reader := bufio.NewReader(os.Stdin)
			response := prompt(reader, fmt.Sprintf("Create private repo %s/%s? [Y/n]", cfg.GitHubOrg, repoName), "y", false)
			if response != "n" && response != "N" {
				cmd = exec.Command(gh, "repo", "create", fmt.Sprintf("%s/%s", cfg.GitHubOrg, repoName),
					"--private", "--source=.", "--remote=origin")
				cmd.Dir = cfg.OutputDir
				if err := cmd.Run(); err != nil {
					warn("Failed to create GitHub repository")
				} else {
					success("Created GitHub repository")
				}
			}
		}
	}

	// Setup secrets
	setupGitHubSecrets(cfg, gh, repoName)

	// Push (develop-first workflow: push develop, then main, set develop as default)
	cmd = exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = cfg.OutputDir
	if cmd.Run() == nil {
		doPush := cfg.AutoYes
		if !cfg.AutoYes {
			reader := bufio.NewReader(os.Stdin)
			response := prompt(reader, "Push to origin? [Y/n]", "y", false)
			doPush = response != "n" && response != "N"
		}

		if doPush {
			// Push develop branch first
			cmd = exec.Command("git", "push", "-u", "origin", "develop")
			cmd.Dir = cfg.OutputDir
			if err := cmd.Run(); err != nil {
				warn("Failed to push develop branch")
			}

			// Push main branch
			cmd = exec.Command("git", "push", "-u", "origin", "main")
			cmd.Dir = cfg.OutputDir
			if err := cmd.Run(); err != nil {
				warn("Failed to push main branch")
			}

			// Set develop as default branch
			cmd = exec.Command(gh, "repo", "edit", fmt.Sprintf("%s/%s", cfg.GitHubOrg, repoName),
				"--default-branch", "develop")
			cmd.Dir = cfg.OutputDir
			if err := cmd.Run(); err != nil {
				warn("Failed to set develop as default branch")
			}

			success("Pushed to GitHub (develop-first workflow)")

			// Set up branch protection after push (main branch must exist)
			setupBranchProtection(cfg, gh, repoName)
		}
	}

	return nil
}

func setupBranchProtection(cfg *Config, gh, repoName string) {
	repoRef := fmt.Sprintf("%s/%s", cfg.GitHubOrg, repoName)

	info("Setting up branch protection for main...")

	// Build the API request for branch protection
	// Requirements:
	// - Require PR to merge (no direct push)
	// - Require CI status checks to pass (backend, frontend)
	// - Require 0 approvals (agent can self-merge)
	// - Allow admins to bypass (lucca-alma can force merge if needed)
	cmd := exec.Command(gh, "api",
		fmt.Sprintf("repos/%s/branches/main/protection", repoRef),
		"-X", "PUT",
		"-f", "required_status_checks[strict]=true",
		"-f", "required_status_checks[contexts][]=backend",
		"-f", "required_status_checks[contexts][]=frontend",
		"-F", "enforce_admins=false",
		"-F", "required_pull_request_reviews[required_approving_review_count]=0",
		"-F", "restrictions=null",
		"-F", "required_linear_history=false",
		"-F", "allow_force_pushes=false",
		"-F", "allow_deletions=false",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if this is a permissions error
		if strings.Contains(string(output), "403") || strings.Contains(string(output), "Resource not accessible") {
			warn("Cannot set branch protection - insufficient permissions (admin access required)")
			info("To set up manually, run:")
			fmt.Printf("  gh api repos/%s/branches/main/protection -X PUT \\\n", repoRef)
			fmt.Println("    -f required_status_checks[strict]=true \\")
			fmt.Println("    -f required_status_checks[contexts][]=backend \\")
			fmt.Println("    -f required_status_checks[contexts][]=frontend \\")
			fmt.Println("    -F enforce_admins=false \\")
			fmt.Println("    -F required_pull_request_reviews[required_approving_review_count]=0 \\")
			fmt.Println("    -F restrictions=null")
		} else if strings.Contains(string(output), "404") || strings.Contains(string(output), "Branch not found") {
			warn("Branch 'main' not found - branch protection will need to be set up after first push to main")
		} else {
			warn(fmt.Sprintf("Failed to set branch protection: %s", strings.TrimSpace(string(output))))
		}
		return
	}

	success("Branch protection configured for main")
	info("  - PRs required to merge")
	info("  - CI checks (backend, frontend) must pass")
	info("  - 0 approvals required (agent can self-merge)")
	info("  - Admin bypass enabled")
}

func setupGitHubSecrets(cfg *Config, gh, repoName string) {
	repoRef := fmt.Sprintf("%s/%s", cfg.GitHubOrg, repoName)

	// AWS secrets
	awsRoleARN := os.Getenv("AWS_ROLE_ARN")
	if awsRoleARN == "" {
		awsRoleARN = "arn:aws:iam::171949636152:role/DS-CDK-CF-Access"
	}

	info("Setting AWS_ROLE_ARN secret...")
	cmd := exec.Command(gh, "secret", "set", "AWS_ROLE_ARN", "--repo", repoRef, "--body", awsRoleARN)
	if err := cmd.Run(); err != nil {
		warn("Failed to set AWS_ROLE_ARN secret - configure manually")
	} else {
		success("AWS_ROLE_ARN secret configured")
	}

	// AWS_ACCOUNT_ID is set from environment if available
	awsAccountID := os.Getenv("AWS_ACCOUNT_ID")
	if awsAccountID == "" {
		awsAccountID = detectAWSAccount()
	}
	if awsAccountID != "" {
		info("Setting AWS_ACCOUNT_ID secret...")
		cmd = exec.Command(gh, "secret", "set", "AWS_ACCOUNT_ID", "--repo", repoRef, "--body", awsAccountID)
		if err := cmd.Run(); err != nil {
			warn("Failed to set AWS_ACCOUNT_ID secret - configure manually")
		} else {
			success("AWS_ACCOUNT_ID secret configured")
		}
	} else {
		warn("AWS_ACCOUNT_ID not detected - configure manually")
	}

	// NPM token
	npmToken := os.Getenv("NPM_TOKEN")
	if npmToken == "" {
		// Try loading from credentials file
		credFile := filepath.Join(os.Getenv("HOME"), ".openclaw/workspace/github-credentials.env")
		if data, err := os.ReadFile(credFile); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "NPM_TOKEN=") {
					npmToken = strings.TrimPrefix(line, "NPM_TOKEN=")
					break
				}
			}
		}
	}
	if npmToken != "" {
		info("Setting NPM_TOKEN secret...")
		cmd = exec.Command(gh, "secret", "set", "NPM_TOKEN", "--repo", repoRef, "--body", npmToken)
		if err := cmd.Run(); err != nil {
			warn("Failed to set NPM_TOKEN secret - configure manually")
		} else {
			success("NPM_TOKEN secret configured")
		}
	}

	// Create environments
	info("Creating GitHub environments...")
	for _, envName := range []string{"development", "staging", "production"} {
		cmd = exec.Command(gh, "api", fmt.Sprintf("repos/%s/environments/%s", repoRef, envName), "-X", "PUT")
		if err := cmd.Run(); err != nil {
			warn(fmt.Sprintf("Failed to create environment: %s", envName))
		} else {
			success(fmt.Sprintf("Created environment: %s", envName))
		}
	}
}

func checkAWSResources(cfg *Config) {
	info("Checking AWS resource availability...")

	aws, err := exec.LookPath("aws")
	if err != nil {
		warn("AWS CLI not found. Skipping AWS resource check.")
		return
	}

	// Check if credentials work
	cmd := exec.Command(aws, "sts", "get-caller-identity", "--query", "Account", "--output", "text")
	out, err := cmd.Output()
	if err != nil {
		warn("Cannot access AWS. Skipping resource check.")
		return
	}

	currentAccount := strings.TrimSpace(string(out))

	// Get region from environment or default
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "us-west-2"
	}

	// Check CDK bootstrap
	cmd = exec.Command(aws, "cloudformation", "describe-stacks", "--stack-name", "CDKToolkit",
		"--query", "Stacks[0].StackStatus", "--output", "text")
	out, _ = cmd.Output()
	status := strings.TrimSpace(string(out))
	if status == "CREATE_COMPLETE" || status == "UPDATE_COMPLETE" {
		fmt.Printf("  %s✓%s CDK bootstrap ready\n", green, reset)
	} else {
		fmt.Printf("  %s✗%s CDK bootstrap missing - run: cdk bootstrap aws://%s/%s\n",
			red, reset, currentAccount, awsRegion)
	}
}

func printNextSteps(cfg *Config) {
	fmt.Println()
	success("App created successfully!")
	fmt.Println()
	info("Next steps:")
	fmt.Printf("  1. cd %s\n", cfg.OutputDir)
	fmt.Println("  2. pnpm install")
	fmt.Println("  3. cd backend && go mod tidy")
	fmt.Println("  4. Update AGENTS.md with app-specific context")
	fmt.Println("  5. Review and update REQUIREMENTS.md")
	fmt.Println()
}

// Helper functions

func prompt(reader *bufio.Reader, question, defaultVal string, autoYes bool) string {
	if autoYes && defaultVal != "" {
		return defaultVal
	}

	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", question, defaultVal)
	} else {
		fmt.Printf("%s: ", question)
	}

	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultVal
	}
	return text
}

func validateAppName(name string) bool {
	re := regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	return re.MatchString(name)
}

func toDisplayName(name string) string {
	// Name is already the short form (e.g., "workforce")
	// Replace hyphens with spaces and title case
	words := strings.Split(name, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return "DS " + strings.Join(words, " ")
}

func detectAWSAccount() string {
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--query", "Account", "--output", "text")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func readFileString(path, defaultVal string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultVal
	}
	return strings.TrimSpace(string(data))
}

func copyDir(src, dst string, excludes []string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Check excludes
		for _, excl := range excludes {
			if strings.HasPrefix(relPath, excl) || relPath == excl {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
