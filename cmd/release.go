package cmd

import (
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
	relMajor  bool
	relMinor  bool
	relPatch  bool
	relAuto   bool
	relTag    string
	relDryRun bool
	relPush   bool
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Create a tagged release with AI-generated release notes",
	Long: `Create a tagged release with AI-generated release notes.

Examples:
  commitai release --auto          # AI suggests version bump
  commitai release --major         # Bump major version (1.0.0 -> 2.0.0)
  commitai release --minor         # Bump minor version (1.0.0 -> 1.1.0)
  commitai release --patch         # Bump patch version (1.0.0 -> 1.0.1)
  commitai release --tag v1.2.3    # Use specific tag
  commitai release --auto --push   # Auto version + push tags`,
	RunE: runRelease,
}

func init() {
	releaseCmd.Flags().BoolVar(&relMajor, "major", false, "Bump major version")
	releaseCmd.Flags().BoolVar(&relMinor, "minor", false, "Bump minor version")
	releaseCmd.Flags().BoolVar(&relPatch, "patch", false, "Bump patch version")
	releaseCmd.Flags().BoolVarP(&relAuto, "auto", "a", false, "Let AI suggest version bump")
	releaseCmd.Flags().StringVar(&relTag, "tag", "", "Use specific tag (e.g. v1.2.3)")
	releaseCmd.Flags().BoolVarP(&relDryRun, "dry-run", "d", false, "Preview without creating tag")
	releaseCmd.Flags().BoolVarP(&relPush, "push", "p", false, "Push tag to origin after creation")
}

func runRelease(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		color.Yellow("⚠️  %s", err)
		return nil
	}

	client := ai.NewGeminiClient(cfg)

	// Get current tag
	currentTag, err := git.LatestTag()
	if err != nil {
		return err
	}

	color.Cyan("📦 Current version: %s", ifEmpty(currentTag, "none"))

	// Get commits since last tag
	commits, err := git.CommitsSinceTag(currentTag)
	if err != nil {
		return err
	}

	if len(commits) == 0 {
		color.Yellow("No commits since last tag. Nothing to release.")
		return nil
	}

	color.Cyan("📝 %d commit(s) since last tag", len(commits))

	// Determine new version
	var newVersion string
	if relTag != "" {
		newVersion = strings.TrimPrefix(relTag, "v")
	} else if relAuto {
		color.Cyan("\n🤖 Asking AI to suggest version bump...")
		newVersion, err = client.SuggestNextVersion(commits, currentTag)
		if err != nil {
			return fmt.Errorf("AI version suggestion failed: %w", err)
		}
	} else {
		newVersion = bumpVersion(currentTag, relMajor, relMinor, relPatch)
	}

	newTag := "v" + newVersion
	color.Cyan("🏷️  New version: %s", newTag)

	// Generate release notes
	color.Cyan("\n✨ Generating release notes with Gemini...")
	notes, err := client.GenerateReleaseNotes(commits, currentTag, newTag)
	if err != nil {
		return fmt.Errorf("failed to generate release notes: %w", err)
	}

	fmt.Println()
	color.Green("📋 Release Notes:")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(notes)
	fmt.Println(strings.Repeat("─", 60))

	if relDryRun {
		color.Yellow("\n🔍 Dry run — no tag was created.")
		return nil
	}

	// Confirm
	if !flagYes {
		fmt.Printf("\n⚡ Create tag %s? [Y/n]: ", newTag)
		var input string
		fmt.Scanln(&input)
		input = strings.ToLower(strings.TrimSpace(input))
		if input == "n" || input == "no" {
			color.Yellow("Release cancelled.")
			return nil
		}
	}

	// Create annotated tag
	if err := git.CreateTag(newTag, notes); err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}
	color.Green("\n✅ Tag %s created!", newTag)

	// Save release notes to file
	notesFile := fmt.Sprintf("RELEASE-%s.md", newTag)
	if err := os.WriteFile(notesFile, []byte(notes), 0644); err == nil {
		color.Cyan("📄 Release notes saved to %s", notesFile)
	}

	// Push if requested
	if relPush {
		color.Cyan("\n📤 Pushing tag to origin...")
		out, err := exec.Command("git", "push", "origin", newTag).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to push tag: %s\n%w", string(out), err)
		}
		color.Green("✅ Tag pushed to origin!")
	}

	return nil
}

func bumpVersion(currentTag string, major, minor, patch bool) string {
	tag := strings.TrimPrefix(currentTag, "v")
	if tag == "" {
		return "0.1.0"
	}

	parts := strings.Split(tag, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}

	var maj, min, pat int
	fmt.Sscanf(parts[0], "%d", &maj)
	fmt.Sscanf(parts[1], "%d", &min)
	fmt.Sscanf(parts[2], "%d", &pat)

	switch {
	case major:
		maj++
		min = 0
		pat = 0
	case minor:
		min++
		pat = 0
	default: // patch
		pat++
	}

	return fmt.Sprintf("%d.%d.%d", maj, min, pat)
}

func ifEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
