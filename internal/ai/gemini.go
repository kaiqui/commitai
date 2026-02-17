package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kaiqui/commitai/internal/config"
	"github.com/kaiqui/commitai/internal/git"
)

const geminiURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"

type GeminiClient struct {
	cfg    *config.Config
	client *http.Client
}

func NewGeminiClient(cfg *config.Config) *GeminiClient {
	return &GeminiClient{
		cfg:    cfg,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// --- Request/Response types ---

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// --- Public methods ---

// GenerateCommitMessages makes a SINGLE request to Gemini for all staged files.
// Returns a map of filepath -> commit message (or a single message if granular=false).
func (g *GeminiClient) GenerateCommitMessages(changes []git.FileChange, granular bool, recentCommits []string) (map[string]string, error) {
	prompt := g.buildCommitPrompt(changes, granular, recentCommits)

	raw, err := g.callGemini(prompt)
	if err != nil {
		return nil, err
	}

	return g.parseCommitResponse(raw, changes, granular), nil
}

// GenerateReleaseNotes generates release notes for a new version.
func (g *GeminiClient) GenerateReleaseNotes(commits []string, currentTag, newTag string) (string, error) {
	prompt := buildReleasePrompt(commits, currentTag, newTag)
	return g.callGemini(prompt)
}

// SuggestNextVersion suggests the next semver version based on commits.
func (g *GeminiClient) SuggestNextVersion(commits []string, currentTag string) (string, error) {
	prompt := buildVersionPrompt(commits, currentTag)
	raw, err := g.callGemini(prompt)
	if err != nil {
		return "", err
	}
	// Extract just the version string
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "v") || (len(l) > 0 && l[0] >= '0' && l[0] <= '9') {
			return strings.TrimPrefix(l, "v"), nil
		}
	}
	return strings.TrimSpace(raw), nil
}

// --- Internal ---

func (g *GeminiClient) callGemini(prompt string) (string, error) {
	req := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     0.3,
			MaxOutputTokens: g.cfg.MaxTokens,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(geminiURL, g.cfg.Model, g.cfg.GeminiAPIKey)
	resp, err := g.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("request to Gemini failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(data, &gemResp); err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %w\nBody: %s", err, string(data))
	}

	if gemResp.Error != nil {
		return "", fmt.Errorf("Gemini API error: %s", gemResp.Error.Message)
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	return gemResp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *GeminiClient) buildCommitPrompt(changes []git.FileChange, granular bool, recentCommits []string) string {
	var sb strings.Builder

	style := g.cfg.CommitStyle
	lang := g.cfg.Language

	sb.WriteString("You are an expert developer writing git commit messages.\n\n")

	if style == "conventional" {
		sb.WriteString("Use Conventional Commits format: <type>(<scope>): <description>\n")
		sb.WriteString("Types: feat, fix, docs, style, refactor, test, chore, perf, ci, build\n\n")
	}

	if lang == "pt" || lang == "pt-br" {
		sb.WriteString("Write commit messages in Portuguese (pt-BR).\n\n")
	} else {
		sb.WriteString("Write commit messages in English.\n\n")
	}

	if len(recentCommits) > 0 {
		sb.WriteString("Recent commits for context:\n")
		for _, c := range recentCommits {
			sb.WriteString("  " + c + "\n")
		}
		sb.WriteString("\n")
	}

	if granular {
		sb.WriteString(fmt.Sprintf("I have %d staged file(s). Generate ONE commit message per file.\n", len(changes)))
		sb.WriteString("Rules:\n")
		sb.WriteString("- Each message must be concise (max 72 chars for subject line)\n")
		sb.WriteString("- Add a blank line then a short body if needed\n")
		sb.WriteString("- Output format must be EXACTLY:\n\n")
		sb.WriteString("FILE: <filepath>\nMESSAGE:\n<commit message>\n---\n\n")
		sb.WriteString("Now here are the diffs:\n\n")

		for _, c := range changes {
			sb.WriteString(fmt.Sprintf("FILE: %s (status: %s)\n", c.Path, c.Status))
			if c.Diff != "" {
				// Limit diff size per file to avoid token overflow
				diff := c.Diff
				if len(diff) > 3000 {
					diff = diff[:3000] + "\n... (truncated)"
				}
				sb.WriteString("DIFF:\n```\n")
				sb.WriteString(diff)
				sb.WriteString("\n```\n")
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("Generate ONE single commit message that summarizes ALL the following staged changes.\n")
		sb.WriteString("Rules:\n")
		sb.WriteString("- Subject line: max 72 chars\n")
		sb.WriteString("- Add a blank line then bullet points listing key changes if there are multiple files\n")
		sb.WriteString("- Output ONLY the commit message, nothing else.\n\n")
		sb.WriteString("Staged changes:\n\n")

		for _, c := range changes {
			sb.WriteString(fmt.Sprintf("FILE: %s (status: %s)\n", c.Path, c.Status))
			if c.Diff != "" {
				diff := c.Diff
				if len(diff) > 2000 {
					diff = diff[:2000] + "\n... (truncated)"
				}
				sb.WriteString("```\n")
				sb.WriteString(diff)
				sb.WriteString("\n```\n")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (g *GeminiClient) parseCommitResponse(raw string, changes []git.FileChange, granular bool) map[string]string {
	result := make(map[string]string)

	if !granular {
		result["__all__"] = strings.TrimSpace(raw)
		return result
	}

	// Parse FILE: / MESSAGE: / --- blocks
	blocks := strings.Split(raw, "---")
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.SplitN(block, "\n", -1)
		var filePath, message string
		inMessage := false

		for _, line := range lines {
			if strings.HasPrefix(line, "FILE:") {
				filePath = strings.TrimSpace(strings.TrimPrefix(line, "FILE:"))
				inMessage = false
			} else if strings.HasPrefix(line, "MESSAGE:") {
				inMessage = true
				rest := strings.TrimSpace(strings.TrimPrefix(line, "MESSAGE:"))
				if rest != "" {
					message = rest
				}
			} else if inMessage {
				if message == "" {
					message = line
				} else {
					message += "\n" + line
				}
			}
		}

		if filePath != "" && message != "" {
			result[filePath] = strings.TrimSpace(message)
		}
	}

	// Fallback: if parsing failed, assign same message to all files
	if len(result) == 0 && len(changes) > 0 {
		for _, c := range changes {
			result[c.Path] = strings.TrimSpace(raw)
		}
	}

	return result
}

func buildReleasePrompt(commits []string, currentTag, newTag string) string {
	var sb strings.Builder
	sb.WriteString("You are a developer writing GitHub release notes.\n\n")
	sb.WriteString(fmt.Sprintf("Generate release notes for version %s", newTag))
	if currentTag != "" {
		sb.WriteString(fmt.Sprintf(" (previous: %s)", currentTag))
	}
	sb.WriteString(".\n\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- Use markdown\n")
	sb.WriteString("- Group into sections: ## 🚀 Features, ## 🐛 Bug Fixes, ## 🔧 Improvements, ## 📚 Docs (omit empty sections)\n")
	sb.WriteString("- Be concise and user-friendly\n")
	sb.WriteString("- Start with a one-sentence summary\n")
	sb.WriteString("- Output ONLY the release notes markdown\n\n")
	sb.WriteString("Commits since last release:\n")
	for _, c := range commits {
		sb.WriteString("- " + c + "\n")
	}
	return sb.String()
}

func buildVersionPrompt(commits []string, currentTag string) string {
	var sb strings.Builder
	sb.WriteString("You are a versioning expert using Semantic Versioning (semver).\n\n")

	if currentTag == "" {
		sb.WriteString("Current version: none (first release)\n")
	} else {
		sb.WriteString(fmt.Sprintf("Current version: %s\n", currentTag))
	}

	sb.WriteString("\nBased on these commits, suggest the next version number.\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- MAJOR: breaking changes (feat! or BREAKING CHANGE)\n")
	sb.WriteString("- MINOR: new features (feat:)\n")
	sb.WriteString("- PATCH: fixes and other changes\n")
	sb.WriteString("- If no current version, suggest 0.1.0\n")
	sb.WriteString("- Output ONLY the version number (e.g. 1.2.3), no 'v' prefix, no explanation\n\n")
	sb.WriteString("Commits:\n")
	for _, c := range commits {
		sb.WriteString("- " + c + "\n")
	}
	return sb.String()
}
