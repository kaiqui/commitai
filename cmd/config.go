package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kaiqui/commitai/internal/config"
)

var (
	cfgAPIKey   string
	cfgLanguage string
	cfgStyle    string
	cfgModel    string
	cfgShow     bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure commitai settings",
	Long: `Configure commitai settings.

Examples:
  commitai config --key YOUR_GEMINI_API_KEY
  commitai config --lang pt-br
  commitai config --style conventional
  commitai config --model gemini-2.5-flash
  commitai config --show`,
	RunE: runConfig,
}

func init() {
	configCmd.Flags().StringVar(&cfgAPIKey, "key", "", "Gemini API key")
	configCmd.Flags().StringVar(&cfgLanguage, "lang", "", "Language (en, pt-br, es, fr, ...)")
	configCmd.Flags().StringVar(&cfgStyle, "style", "", "Commit style (conventional, simple)")
	configCmd.Flags().StringVar(&cfgModel, "model", "", "Gemini model (gemini-2.5-flash, gemini-1.5-pro, ...)")
	configCmd.Flags().BoolVar(&cfgShow, "show", false, "Show current configuration")
}

func runConfig(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if cfgShow || (!cmd.Flags().Changed("key") && !cmd.Flags().Changed("lang") &&
		!cmd.Flags().Changed("style") && !cmd.Flags().Changed("model")) {
		printConfig(cfg)
		return nil
	}

	if cfgAPIKey != "" {
		cfg.GeminiAPIKey = cfgAPIKey
		color.Green("✅ API key saved")
	}
	if cfgLanguage != "" {
		cfg.Language = cfgLanguage
		color.Green("✅ Language set to: %s", cfgLanguage)
	}
	if cfgStyle != "" {
		cfg.CommitStyle = cfgStyle
		color.Green("✅ Commit style set to: %s", cfgStyle)
	}
	if cfgModel != "" {
		cfg.Model = cfgModel
		color.Green("✅ Model set to: %s", cfgModel)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	color.Cyan("💾 Config saved to ~/.commitai.json")
	return nil
}

func printConfig(cfg *config.Config) {
	fmt.Println()
	color.Cyan("⚙️  commitai configuration:")
	fmt.Println()

	apiKeyDisplay := "(not set)"
	if cfg.GeminiAPIKey != "" {
		k := cfg.GeminiAPIKey
		if len(k) > 8 {
			apiKeyDisplay = k[:4] + strings.Repeat("*", len(k)-8) + k[len(k)-4:]
		} else {
			apiKeyDisplay = "****"
		}
	}

	fmt.Printf("  API Key:      %s\n", apiKeyDisplay)
	fmt.Printf("  Language:     %s\n", cfg.Language)
	fmt.Printf("  Style:        %s\n", cfg.CommitStyle)
	fmt.Printf("  Model:        %s\n", cfg.Model)
	fmt.Printf("  Max Tokens:   %d\n", cfg.MaxTokens)
	fmt.Println()
	fmt.Println("  Config file:  ~/.commitai.json")
	fmt.Println("  Env override: GEMINI_API_KEY")
	fmt.Println()
}
