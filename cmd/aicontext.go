package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// AIContextOptions configures the aicontext command behavior
type AIContextOptions struct {
	JSON     bool   // --json: output as structured JSON
	Compact  bool   // --compact: omit examples and long descriptions
	Category string // --category: filter to specific category
}

// AIContext represents the complete AI context document
type AIContext struct {
	Overview     AIOverview      `json:"overview"`
	Categories   []AICategory    `json:"categories"`
	Commands     []AICommandInfo `json:"commands"`
	Architecture AIArchitecture  `json:"architecture"`
}

// AIOverview describes the application
type AIOverview struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Principles  []string `json:"principles"`
	Features    []string `json:"features"`
}

// AICategory represents a command category
type AICategory struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Commands    []string `json:"commands"`
}

// AICommandInfo represents detailed command documentation
type AICommandInfo struct {
	Name        string       `json:"name"`
	Path        string       `json:"path"`
	Category    string       `json:"category"`
	Short       string       `json:"short"`
	Long        string       `json:"long,omitempty"`
	Usage       string       `json:"usage"`
	Flags       []AIFlagInfo `json:"flags,omitempty"`
	Examples    []string     `json:"examples,omitempty"`
	Subcommands []string     `json:"subcommands,omitempty"`
}

// AIFlagInfo represents a command flag
type AIFlagInfo struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default"`
	Description string `json:"description"`
}

// AIArchitecture describes the project structure
type AIArchitecture struct {
	Description string            `json:"description"`
	Structure   map[string]string `json:"structure"`
}

// aiCategoryMap maps command names to categories
var aiCategoryMap = map[string]string{
	// Repository Management
	"clone": "Repository Management", "add": "Repository Management",
	"remove": "Repository Management", "list": "Repository Management",
	"open": "Repository Management", "favorite": "Repository Management",
	"unfavorite": "Repository Management", "map": "Repository Management",

	// Git Operations
	"branches": "Git Operations", "diff": "Git Operations",
	"stats": "Git Operations", "status": "Git Operations",
	"reauthor": "Git Operations", "snapshot": "Git Operations",

	// GitHub Integration
	"gh": "GitHub Integration",

	// Organization
	"org": "Organization",

	// Project Management
	"pm": "Project Management",

	// Configuration
	"configure": "Configuration", "config": "Configuration",
	"profile": "Configuration",

	// Infrastructure
	"server": "Infrastructure", "service": "Infrastructure",
	"mirror": "Infrastructure",

	// Tooling
	"cmdtree": "Tooling", "aicontext": "Tooling",
	"version": "Tooling", "update": "Tooling",
	"nerds": "Tooling", "repo": "Tooling",
}

// aiCategoryDescriptions provides descriptions for each category
var aiCategoryDescriptions = map[string]string{
	"Repository Management": "Clone, organize, and manage Git repositories locally",
	"Git Operations":        "Git-related operations like branching, diffing, and statistics",
	"GitHub Integration":    "GitHub API integration for issues, PRs, actions, and releases",
	"Organization":          "Manage and mirror organization repositories",
	"Project Management":    "Integrate with project management tools (Jira, ZenHub)",
	"Configuration":         "Configure clonr settings and profiles",
	"Infrastructure":        "Server mode, services, and repository mirroring",
	"Tooling":               "Development and introspection tools",
}

// getAICategory returns the category for a command name
func getAICategory(name string) string {
	if cat, ok := aiCategoryMap[name]; ok {
		return cat
	}

	return "Other"
}

// aicontextCmd represents the aicontext command
var aicontextCmd = &cobra.Command{
	Use:   "aicontext",
	Short: "Generate AI-optimized context documentation",
	Long: `Generate comprehensive, AI-optimized documentation about clonr.

This command produces a detailed context document designed for AI consumption,
including command references, usage patterns, and architecture information.

Examples:
  clonr aicontext                    # Output markdown documentation
  clonr aicontext --json             # Output as structured JSON
  clonr aicontext --compact          # Omit examples and long descriptions
  clonr aicontext --category GitHub  # Filter to GitHub commands`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		compact, _ := cmd.Flags().GetBool("compact")
		category, _ := cmd.Flags().GetString("category")

		opts := AIContextOptions{
			JSON:     jsonOutput,
			Compact:  compact,
			Category: category,
		}

		return runAIContext(cmd.OutOrStdout(), rootCmd, opts)
	},
}

func init() {
	rootCmd.AddCommand(aicontextCmd)

	aicontextCmd.Flags().Bool("json", false, "output as structured JSON")
	aicontextCmd.Flags().Bool("compact", false, "omit examples and long descriptions")
	aicontextCmd.Flags().String("category", "", "filter to specific category")
}

// runAIContext generates AI context documentation
func runAIContext(w io.Writer, root *cobra.Command, opts AIContextOptions) error {
	ctx := buildAIContext(root, opts)

	if opts.JSON {
		return writeAIContextJSON(w, ctx)
	}

	return writeAIContextMarkdown(w, ctx, opts)
}

// buildAIContext constructs the complete AI context
func buildAIContext(root *cobra.Command, opts AIContextOptions) AIContext {
	commands := collectAICommands(root, "", opts)
	categories := buildAICategories(commands)

	// Filter by category if specified
	if opts.Category != "" {
		var filtered []AICommandInfo

		for _, cmd := range commands {
			if strings.EqualFold(cmd.Category, opts.Category) {
				filtered = append(filtered, cmd)
			}
		}

		commands = filtered

		var filteredCats []AICategory

		for _, cat := range categories {
			if strings.EqualFold(cat.Name, opts.Category) {
				filteredCats = append(filteredCats, cat)
			}
		}

		categories = filteredCats
	}

	return AIContext{
		Overview:     buildAIOverview(),
		Categories:   categories,
		Commands:     commands,
		Architecture: buildAIArchitecture(),
	}
}

// buildAIOverview creates the application overview
func buildAIOverview() AIOverview {
	return AIOverview{
		Name:        "clonr",
		Description: "A Git repository manager for developers who work with multiple repositories. Provides an interactive interface for cloning, organizing, and working with multiple repositories efficiently.",
		Principles: []string{
			"Multi-repo first: Designed for managing many repositories",
			"GitHub integration: Deep integration with GitHub API",
			"Interactive: TUI interfaces for common operations",
			"Profile support: Multiple configurations for different contexts",
			"Cross-platform: Linux, macOS, Windows support",
		},
		Features: []string{
			"Clone and organize repositories with custom directory structure",
			"GitHub issues, PRs, actions, and releases management",
			"Organization repository mirroring",
			"Jira and ZenHub integration",
			"Git statistics and branch management",
			"Profile-based configuration",
			"Server mode with gRPC API",
			"Auto-update support",
		},
	}
}

// buildAIArchitecture creates the architecture documentation
func buildAIArchitecture() AIArchitecture {
	return AIArchitecture{
		Description: "Standard Go CLI architecture with Cobra commands and gRPC server support",
		Structure: map[string]string{
			"cmd/":      "Cobra CLI command definitions",
			"internal/": "Internal packages (cloner, config, git)",
			"pkg/":      "Public packages (database, profiles)",
			"proto/":    "Protocol buffer definitions for gRPC",
			"scripts/":  "Build and utility scripts",
			"docs/":     "Documentation",
			"main.go":   "Entry point calling cmd.Execute()",
		},
	}
}

// collectAICommands recursively collects all commands
func collectAICommands(cmd *cobra.Command, parentPath string, opts AIContextOptions) []AICommandInfo {
	var commands []AICommandInfo

	for _, c := range cmd.Commands() {
		// Skip help and completion commands
		if c.Name() == "help" || c.Name() == "completion" {
			continue
		}

		// Skip hidden commands
		if c.Hidden {
			continue
		}

		path := c.Name()
		if parentPath != "" {
			path = parentPath + " " + c.Name()
		}

		info := AICommandInfo{
			Name:     c.Name(),
			Path:     path,
			Category: getAICategory(c.Name()),
			Short:    c.Short,
			Usage:    c.UseLine(),
		}

		// Include long description unless compact mode
		if !opts.Compact && c.Long != "" {
			info.Long = c.Long
			info.Examples = extractAIExamples(c.Long)
		}

		// Collect flags
		info.Flags = collectAIFlags(c)

		// Collect subcommand names
		for _, sub := range c.Commands() {
			if sub.Name() != "help" && !sub.Hidden {
				info.Subcommands = append(info.Subcommands, sub.Name())
			}
		}

		commands = append(commands, info)

		// Recurse into subcommands
		subCommands := collectAICommands(c, path, opts)
		commands = append(commands, subCommands...)
	}

	return commands
}

// collectAIFlags collects flag information from a command
func collectAIFlags(cmd *cobra.Command) []AIFlagInfo {
	var flags []AIFlagInfo

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Skip help flag
		if f.Name == "help" {
			return
		}

		flags = append(flags, AIFlagInfo{
			Name:        f.Name,
			Shorthand:   f.Shorthand,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Description: f.Usage,
		})
	})

	return flags
}

// extractAIExamples parses examples from long description
func extractAIExamples(long string) []string {
	var examples []string

	lines := strings.Split(long, "\n")
	inExample := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect example blocks
		if strings.HasPrefix(trimmed, "clonr ") || strings.HasPrefix(trimmed, "$ clonr ") {
			examples = append(examples, strings.TrimPrefix(trimmed, "$ "))
			inExample = true
		} else if inExample && (trimmed == "" || !strings.HasPrefix(line, "  ")) {
			inExample = false
		}
	}

	return examples
}

// buildAICategories groups commands by category
func buildAICategories(commands []AICommandInfo) []AICategory {
	catMap := make(map[string][]string)

	for _, cmd := range commands {
		// Only include top-level commands in category listing
		if !strings.Contains(cmd.Path, " ") {
			catMap[cmd.Category] = append(catMap[cmd.Category], cmd.Name)
		}
	}

	categories := make([]AICategory, 0, len(catMap))

	for name, cmds := range catMap {
		sort.Strings(cmds)
		categories = append(categories, AICategory{
			Name:        name,
			Description: aiCategoryDescriptions[name],
			Commands:    cmds,
		})
	}

	// Sort categories by name
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})

	return categories
}

// writeAIContextJSON outputs the context as JSON
func writeAIContextJSON(w io.Writer, ctx AIContext) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(ctx)
}

// writeAIContextMarkdown outputs the context as Markdown
func writeAIContextMarkdown(w io.Writer, ctx AIContext, opts AIContextOptions) error {
	var sb strings.Builder

	// Title
	sb.WriteString("# clonr - AI Context Document\n\n")

	// Overview
	sb.WriteString("## Overview\n\n")
	sb.WriteString(ctx.Overview.Description + "\n\n")

	sb.WriteString("### Design Principles\n\n")

	for _, p := range ctx.Overview.Principles {
		sb.WriteString("- " + p + "\n")
	}

	sb.WriteString("\n")

	sb.WriteString("### Key Features\n\n")

	for _, f := range ctx.Overview.Features {
		sb.WriteString("- " + f + "\n")
	}

	sb.WriteString("\n")

	// Categories
	sb.WriteString("## Command Categories\n\n")

	for _, cat := range ctx.Categories {
		sb.WriteString(fmt.Sprintf("### %s\n\n", cat.Name))

		if cat.Description != "" {
			sb.WriteString(cat.Description + "\n\n")
		}

		sb.WriteString("Commands: `" + strings.Join(cat.Commands, "`, `") + "`\n\n")
	}

	// Command Reference
	sb.WriteString("## Complete Command Reference\n\n")

	for _, cmd := range ctx.Commands {
		sb.WriteString(fmt.Sprintf("### %s\n\n", cmd.Path))
		sb.WriteString(fmt.Sprintf("**Category:** %s\n\n", cmd.Category))
		sb.WriteString(fmt.Sprintf("**Usage:** `%s`\n\n", cmd.Usage))
		sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", cmd.Short))

		if !opts.Compact && cmd.Long != "" {
			sb.WriteString("**Details:**\n\n")
			sb.WriteString(cmd.Long + "\n\n")
		}

		if len(cmd.Flags) > 0 {
			sb.WriteString("**Flags:**\n\n")
			sb.WriteString("| Flag | Type | Default | Description |\n")
			sb.WriteString("|------|------|---------|-------------|\n")

			for _, f := range cmd.Flags {
				flag := "--" + f.Name
				if f.Shorthand != "" {
					flag = "-" + f.Shorthand + ", " + flag
				}

				def := f.Default
				if def == "" {
					def = "-"
				}

				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
					flag, f.Type, def, f.Description))
			}

			sb.WriteString("\n")
		}

		if !opts.Compact && len(cmd.Examples) > 0 {
			sb.WriteString("**Examples:**\n\n```bash\n")

			for _, ex := range cmd.Examples {
				sb.WriteString(ex + "\n")
			}

			sb.WriteString("```\n\n")
		}

		if len(cmd.Subcommands) > 0 {
			sb.WriteString(fmt.Sprintf("**Subcommands:** `%s`\n\n", strings.Join(cmd.Subcommands, "`, `")))
		}

		sb.WriteString("---\n\n")
	}

	// Architecture
	sb.WriteString("## Architecture\n\n")
	sb.WriteString(ctx.Architecture.Description + "\n\n")
	sb.WriteString("```\n")
	sb.WriteString("clonr/\n")

	// Sort keys for consistent output
	keys := make([]string, 0, len(ctx.Architecture.Structure))
	for k := range ctx.Architecture.Structure {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("  %s  # %s\n", k, ctx.Architecture.Structure[k]))
	}

	sb.WriteString("```\n")

	_, err := io.WriteString(w, sb.String())

	return err
}
