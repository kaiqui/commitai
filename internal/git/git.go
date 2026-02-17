package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// FileChange represents a staged file and its diff
type FileChange struct {
	Path   string
	Status string // A=added, M=modified, D=deleted, R=renamed
	Diff   string
}

// StagedChanges returns all staged changes grouped by file
func StagedChanges() ([]FileChange, error) {
	// Get list of staged files with status
	out, err := run("git", "diff", "--cached", "--name-status")
	if err != nil {
		return nil, fmt.Errorf("failed to get staged files: %w", err)
	}

	if strings.TrimSpace(out) == "" {
		return nil, fmt.Errorf("no staged changes found. Use 'git add' to stage files first")
	}

	var changes []FileChange
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		path := parts[len(parts)-1] // Handle renames: R old -> new

		changes = append(changes, FileChange{
			Path:   path,
			Status: status,
		})
	}

	// Get unified diff for all staged changes
	fullDiff, err := run("git", "diff", "--cached", "--unified=3")
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	// Split diff by file
	fileDiffs := splitDiffByFile(fullDiff)
	for i := range changes {
		if diff, ok := fileDiffs[changes[i].Path]; ok {
			changes[i].Diff = diff
		}
	}

	return changes, nil
}

// AllStagedDiff returns a single combined diff string (for single-request mode)
func AllStagedDiff() (string, error) {
	out, err := run("git", "diff", "--cached", "--unified=3", "--stat")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("no staged changes found. Use 'git add' to stage files first")
	}

	diff, err := run("git", "diff", "--cached", "--unified=3")
	if err != nil {
		return "", err
	}

	return out + "\n---\n" + diff, nil
}

// Commit creates a commit with the given message
func Commit(message string) error {
	out, err := run("git", "commit", "-m", message)
	if err != nil {
		return fmt.Errorf("commit failed: %s\n%w", out, err)
	}
	return nil
}

// IsGitRepo checks if current directory is inside a git repo
func IsGitRepo() bool {
	_, err := run("git", "rev-parse", "--git-dir")
	return err == nil
}

// RecentCommits returns recent commit messages for context
func RecentCommits(n int) ([]string, error) {
	out, err := run("git", "log", fmt.Sprintf("--oneline"), fmt.Sprintf("-n%d", n))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var msgs []string
	for _, l := range lines {
		if l != "" {
			msgs = append(msgs, l)
		}
	}
	return msgs, nil
}

// CommitsSinceTag returns commits since the last tag
func CommitsSinceTag(tag string) ([]string, error) {
	var out string
	var err error
	if tag == "" {
		out, err = run("git", "log", "--oneline")
	} else {
		out, err = run("git", "log", "--oneline", tag+"..HEAD")
	}
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var msgs []string
	for _, l := range lines {
		if l != "" {
			msgs = append(msgs, l)
		}
	}
	return msgs, nil
}

// LatestTag returns the most recent git tag
func LatestTag() (string, error) {
	out, err := run("git", "describe", "--tags", "--abbrev=0")
	if err != nil {
		return "", nil // No tags yet
	}
	return strings.TrimSpace(out), nil
}

// CreateTag creates an annotated git tag
func CreateTag(tag, message string) error {
	_, err := run("git", "tag", "-a", tag, "-m", message)
	return err
}

func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func splitDiffByFile(diff string) map[string]string {
	result := make(map[string]string)
	var currentFile string
	var currentLines []string

	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			if currentFile != "" && len(currentLines) > 0 {
				result[currentFile] = strings.Join(currentLines, "\n")
			}
			// Extract filename: diff --git a/file b/file
			parts := strings.Split(line, " b/")
			if len(parts) >= 2 {
				currentFile = parts[len(parts)-1]
			}
			currentLines = []string{line}
		} else {
			currentLines = append(currentLines, line)
		}
	}

	if currentFile != "" && len(currentLines) > 0 {
		result[currentFile] = strings.Join(currentLines, "\n")
	}

	return result
}
