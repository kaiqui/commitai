package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kaiqui/commitai/internal/ai"
	"github.com/kaiqui/commitai/internal/config"
	"github.com/kaiqui/commitai/internal/git"
)

var (
	flagGranular bool
	flagAll      bool
	flagAutoMode bool
	flagDryRun   bool
	flagYes      bool
	flagLanguage string
	flagStyle    string
)

var rootCmd = &cobra.Command{
	Use:   "commitai",
	Short: "🤖 AI-powered git commit messages using Google Gemini",
	Long: `commitai generates intelligent git commit messages using Google Gemini AI.

It analyzes your staged changes and suggests meaningful commit messages.

Examples:
  commitai              # Auto-detect: single message or granular based on file count
  commitai --all        # One message for all staged changes
  commitai --granular   # Separate message per file
  commitai --dry-run    # Preview messages without committing
  commitai config       # Configure API key and preferences
  commitai release      # Create a tagged release with AI-generated notes`,
	RunE: runCommit,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().BoolVarP(&flagGranular, "granular", "g", false, "Generate separate commit per staged file")
	rootCmd.Flags().BoolVarP(&flagAll, "all", "a", false, "Generate one commit message for all staged changes")
	rootCmd.Flags().BoolVar(&flagAutoMode, "auto", true, "Auto-detect commit mode based on staged files (default)")
	rootCmd.Flags().BoolVarP(&flagDryRun, "dry-run", "d", false, "Preview commit messages without committing")
	rootCmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation prompts")
	rootCmd.Flags().StringVarP(&flagLanguage, "lang", "l", "", "Language for messages (en, pt-br)")
	rootCmd.Flags().StringVar(&flagStyle, "style", "", "Commit style (conventional, simple)")

	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(versionCmd)
}

func runCommit(cmd *cobra.Command, args []string) error {
	// Validate git repo
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		color.Yellow("⚠️  %s", err)
		return nil
	}

	// Override config with flags
	if flagLanguage != "" {
		cfg.Language = flagLanguage
	}
	if flagStyle != "" {
		cfg.CommitStyle = flagStyle
	}

	// Get staged changes
	color.Cyan("🔍 Analyzing staged changes...")
	changes, err := git.StagedChanges()
	if err != nil {
		return err
	}

	if len(changes) == 0 {
		color.Yellow("No staged changes found. Use 'git add' to stage files.")
		return nil
	}

	// Determine mode
	granular := determineMode(changes)

	// Print what we found
	color.Cyan("\n📂 Staged files (%d):", len(changes))
	for _, c := range changes {
		statusIcon := statusToIcon(c.Status)
		fmt.Printf("  %s %s\n", statusIcon, c.Path)
	}

	// Get recent commits for context
	recentCommits, _ := git.RecentCommits(5)

	// Generate messages (ONE request to Gemini for all files)
	color.Cyan("\n✨ Generating commit message(s) with Gemini...")
	client := ai.NewGeminiClient(cfg)
	messages, err := client.GenerateCommitMessages(changes, granular, recentCommits)
	if err != nil {
		return fmt.Errorf("AI generation failed: %w", err)
	}

	// Display and confirm
	if granular {
		return handleGranularCommits(changes, messages, flagDryRun, flagYes)
	}
	return handleSingleCommit(messages["__all__"], flagDryRun, flagYes)
}

func determineMode(changes []git.FileChange) bool {
	if flagGranular {
		return true
	}
	if flagAll {
		return false
	}
	// Auto: granular if multiple files with different concerns
	if len(changes) <= 1 {
		return false
	}
	// Heuristic: granular if files are in different directories or different types
	dirs := make(map[string]bool)
	for _, c := range changes {
		parts := strings.Split(c.Path, "/")
		if len(parts) > 1 {
			dirs[parts[0]] = true
		}
	}
	return len(dirs) > 1 || len(changes) >= 3
}

func handleSingleCommit(message string, dryRun, skipConfirm bool) error {
	fmt.Println()
	color.Green("💬 Suggested commit message:")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(message)
	fmt.Println(strings.Repeat("─", 60))

	if dryRun {
		color.Yellow("\n🔍 Dry run — no commit was made.")
		return nil
	}

	msg, confirmed := confirmOrEdit(message, skipConfirm)
	if !confirmed {
		color.Yellow("Commit cancelled.")
		return nil
	}

	if err := git.Commit(msg); err != nil {
		return err
	}
	color.Green("\n✅ Committed successfully!")
	return nil
}

func handleGranularCommits(changes []git.FileChange, messages map[string]string, dryRun, skipConfirm bool) error {
	fmt.Println()
	color.Green("💬 Suggested commit messages (per file):")

	type plan struct {
		file    string
		message string
	}
	var plans []plan

	for _, c := range changes {
		msg, ok := messages[c.Path]
		if !ok {
			// Fallback: use generic message
			msg = fmt.Sprintf("chore: update %s", c.Path)
		}
		plans = append(plans, plan{c.Path, msg})
	}

	for i, p := range plans {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(plans), p.file)
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println(p.message)
		fmt.Println(strings.Repeat("─", 60))
	}

	if dryRun {
		color.Yellow("\n🔍 Dry run — no commits were made.")
		return nil
	}

	if !skipConfirm {
		fmt.Print("\n⚡ Commit all with these messages? [Y/n/e(dit)]: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "n" || input == "no" {
			color.Yellow("Commit cancelled.")
			return nil
		}
	}

	// Unstage all, then stage+commit one file at a time
	exec.Command("git", "restore", "--staged", ".").Run()

	for i, p := range plans {
		// Re-stage just this file
		if out, err2 := exec.Command("git", "add", p.file).CombinedOutput(); err2 != nil {
			return fmt.Errorf("failed to stage %s: %s\n%w", p.file, string(out), err2)
		}
		if err2 := git.Commit(p.message); err2 != nil {
			return fmt.Errorf("failed to commit %s: %w", p.file, err2)
		}
		color.Green("  ✅ [%d/%d] %s", i+1, len(plans), p.file)
	}

	color.Green("\n🎉 All %d files committed!", len(plans))
	return nil
}

func confirmOrEdit(message string, skip bool) (string, bool) {
	if skip {
		return message, true
	}

	fmt.Print("\n⚡ Use this message? [Y/n/e(dit)]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "n", "no":
		return "", false
	case "e", "edit":
		fmt.Print("Enter your message: ")
		newMsg, _ := reader.ReadString('\n')
		return strings.TrimSpace(newMsg), true
	default:
		return message, true
	}
}

func statusToIcon(s string) string {
	switch {
	case strings.HasPrefix(s, "A"):
		return color.GreenString("✚")
	case strings.HasPrefix(s, "M"):
		return color.YellowString("●")
	case strings.HasPrefix(s, "D"):
		return color.RedString("✖")
	case strings.HasPrefix(s, "R"):
		return color.CyanString("→")
	default:
		return "?"
	}
}
