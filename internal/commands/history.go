package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/history"
)

var (
	historyForceFlag   bool
	historyContentFlag bool
	historyOutputFlag  string
	historyFormatFlag  string
	historyFavoritesFlag bool
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Manage conversation history",
	Long: `View and manage your local conversation history.

` + history.ListAliases() + `

Examples:
  geminiweb history list              # List all conversations
  geminiweb history list --favorites  # List only favorites
  geminiweb history show @last        # Show most recent conversation
  geminiweb history show 1            # Show first conversation in list
  geminiweb history delete @last      # Delete with confirmation
  geminiweb history delete 1 --force  # Delete without confirmation
  geminiweb history rename 1 "New Title"
  geminiweb history favorite @last    # Toggle favorite
  geminiweb history export @last -o chat.md
  geminiweb history search "API"      # Search in titles`,
}

var historyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all conversations",
	Long: `List all conversations with indices and favorite indicators.

The list shows:
  [index] ★ Title (relative time)

Use the index number to reference conversations in other commands.`,
	RunE: runHistoryList,
}

var historyShowCmd = &cobra.Command{
	Use:   "show <ref>",
	Short: "Show a conversation",
	Long: `Show the full content of a conversation.

` + history.ListAliases(),
	Args: cobra.ExactArgs(1),
	RunE: runHistoryShow,
}

var historyDeleteCmd = &cobra.Command{
	Use:   "delete <ref>",
	Short: "Delete a conversation",
	Long: `Delete a conversation with confirmation.

Use --force to skip confirmation.

` + history.ListAliases(),
	Args: cobra.ExactArgs(1),
	RunE: runHistoryDelete,
}

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all conversations",
	Long:  `Delete all conversations. Use --force to skip confirmation.`,
	RunE:  runHistoryClear,
}

var historyRenameCmd = &cobra.Command{
	Use:   "rename <ref> <title>",
	Short: "Rename a conversation",
	Long: `Rename a conversation to a new title.

` + history.ListAliases() + `

Example:
  geminiweb history rename @last "New Title"
  geminiweb history rename 1 "Project Discussion"`,
	Args: cobra.ExactArgs(2),
	RunE: runHistoryRename,
}

var historyFavoriteCmd = &cobra.Command{
	Use:   "favorite <ref>",
	Short: "Toggle favorite status",
	Long: `Toggle the favorite status of a conversation.

` + history.ListAliases() + `

Example:
  geminiweb history favorite @last
  geminiweb history favorite 1`,
	Args: cobra.ExactArgs(1),
	RunE: runHistoryFavorite,
}

var historyExportCmd = &cobra.Command{
	Use:   "export <ref>",
	Short: "Export a conversation",
	Long: `Export a conversation to a file.

Formats: markdown (default), json
If -o is not specified, prints to stdout.
Format is auto-detected from file extension (.md, .json).

` + history.ListAliases() + `

Example:
  geminiweb history export @last                  # Print markdown to stdout
  geminiweb history export @last -o chat.md       # Export as markdown
  geminiweb history export @last -o chat.json     # Export as JSON
  geminiweb history export 1 -f json              # Force JSON format`,
	Args: cobra.ExactArgs(1),
	RunE: runHistoryExport,
}

var historySearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search conversations",
	Long: `Search for conversations by title or content.

By default, searches only in titles. Use --content to also search in message content.

Example:
  geminiweb history search "API"
  geminiweb history search "error" --content`,
	Args: cobra.ExactArgs(1),
	RunE: runHistorySearch,
}

func init() {
	// Add flags
	historyDeleteCmd.Flags().BoolVarP(&historyForceFlag, "force", "f", false, "Skip confirmation")
	historyClearCmd.Flags().BoolVarP(&historyForceFlag, "force", "f", false, "Skip confirmation")
	historyExportCmd.Flags().StringVarP(&historyOutputFlag, "output", "o", "", "Output file path")
	historyExportCmd.Flags().StringVarP(&historyFormatFlag, "format", "f", "", "Export format (markdown, json)")
	historySearchCmd.Flags().BoolVarP(&historyContentFlag, "content", "c", false, "Search in message content")
	historyListCmd.Flags().BoolVar(&historyFavoritesFlag, "favorites", false, "Show only favorites")

	// Add commands
	historyCmd.AddCommand(historyListCmd)
	historyCmd.AddCommand(historyShowCmd)
	historyCmd.AddCommand(historyDeleteCmd)
	historyCmd.AddCommand(historyClearCmd)
	historyCmd.AddCommand(historyRenameCmd)
	historyCmd.AddCommand(historyFavoriteCmd)
	historyCmd.AddCommand(historyExportCmd)
	historyCmd.AddCommand(historySearchCmd)
}

func runHistoryList(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	conversations, err := store.ListConversations()
	if err != nil {
		return fmt.Errorf("failed to list conversations: %w", err)
	}

	if len(conversations) == 0 {
		fmt.Println("No conversations found.")
		fmt.Println("\nStart a new conversation with: geminiweb chat")
		return nil
	}

	// Filter favorites if requested
	if historyFavoritesFlag {
		var filtered []*history.Conversation
		for _, conv := range conversations {
			if conv.IsFavorite {
				filtered = append(filtered, conv)
			}
		}
		conversations = filtered

		if len(conversations) == 0 {
			fmt.Println("No favorite conversations found.")
			fmt.Println("\nUse 'geminiweb history favorite <ref>' to mark favorites.")
			return nil
		}
	}

	// Print list with indices and favorites
	for i, conv := range conversations {
		// Format: [index] ★ Title (relative time)
		star := "  "
		if conv.IsFavorite {
			star = "★ "
		}

		title := conv.Title
		if len(title) > 50 {
			title = title[:50] + "..."
		}

		relTime := history.FormatRelativeTime(conv.UpdatedAt)
		msgCount := len(conv.Messages)
		msgLabel := "msgs"
		if msgCount == 1 {
			msgLabel = "msg"
		}

		fmt.Printf("[%d] %s%s (%d %s, %s)\n", i+1, star, title, msgCount, msgLabel, relTime)
	}

	return nil
}

func runHistoryShow(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	// Resolve alias
	resolver := history.NewResolver(store)
	conv, err := resolver.ResolveWithInfo(args[0])
	if err != nil {
		return fmt.Errorf("✗ %w", err)
	}

	// Show favorite status
	star := ""
	if conv.IsFavorite {
		star = " ★"
	}

	fmt.Printf("ID: %s%s\n", conv.ID, star)
	fmt.Printf("Title: %s\n", conv.Title)
	fmt.Printf("Model: %s\n", conv.Model)
	fmt.Printf("Created: %s\n", conv.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s (%s)\n", conv.UpdatedAt.Format("2006-01-02 15:04:05"), history.FormatRelativeTime(conv.UpdatedAt))
	fmt.Printf("Messages: %d\n", len(conv.Messages))
	fmt.Println()

	for i, msg := range conv.Messages {
		role := "You"
		if msg.Role == "assistant" {
			role = "Gemini"
		}
		fmt.Printf("[%d] %s (%s):\n", i+1, role, msg.Timestamp.Format("15:04"))

		if msg.Thoughts != "" {
			fmt.Printf("  💭 %s\n", truncate(msg.Thoughts, 200))
		}

		content := msg.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		fmt.Printf("  %s\n\n", content)
	}

	return nil
}

func runHistoryDelete(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	// Resolve alias
	resolver := history.NewResolver(store)
	conv, err := resolver.ResolveWithInfo(args[0])
	if err != nil {
		return fmt.Errorf("✗ %w", err)
	}

	// Confirm unless --force
	if !historyForceFlag {
		fmt.Printf("Delete conversation?\n")
		fmt.Printf("  Title: %s\n", conv.Title)
		fmt.Printf("  Messages: %d\n", len(conv.Messages))
		fmt.Printf("  Updated: %s\n", history.FormatRelativeTime(conv.UpdatedAt))
		fmt.Print("\nConfirm [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := store.DeleteConversation(conv.ID); err != nil {
		return fmt.Errorf("✗ Failed to delete: %w", err)
	}

	fmt.Printf("✓ Deleted '%s'\n", conv.Title)
	return nil
}

func runHistoryClear(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	// Get count for confirmation
	conversations, err := store.ListConversations()
	if err != nil {
		return fmt.Errorf("failed to list conversations: %w", err)
	}

	if len(conversations) == 0 {
		fmt.Println("No conversations to delete.")
		return nil
	}

	// Confirm unless --force
	if !historyForceFlag {
		fmt.Printf("Delete ALL %d conversations? This cannot be undone.\n", len(conversations))
		fmt.Print("Type 'DELETE' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response != "DELETE" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := store.ClearAll(); err != nil {
		return fmt.Errorf("✗ Failed to clear history: %w", err)
	}

	fmt.Printf("✓ Deleted %d conversations.\n", len(conversations))
	return nil
}

func runHistoryRename(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	// Resolve alias
	resolver := history.NewResolver(store)
	id, err := resolver.Resolve(args[0])
	if err != nil {
		return fmt.Errorf("✗ %w", err)
	}

	newTitle := args[1]
	if err := store.UpdateTitle(id, newTitle); err != nil {
		return fmt.Errorf("✗ Failed to rename: %w", err)
	}

	fmt.Printf("✓ Renamed to '%s'\n", newTitle)
	return nil
}

func runHistoryFavorite(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	// Resolve alias
	resolver := history.NewResolver(store)
	conv, err := resolver.ResolveWithInfo(args[0])
	if err != nil {
		return fmt.Errorf("✗ %w", err)
	}

	isFavorite, err := store.ToggleFavorite(conv.ID)
	if err != nil {
		return fmt.Errorf("✗ Failed to toggle favorite: %w", err)
	}

	if isFavorite {
		fmt.Printf("★ Added '%s' to favorites\n", conv.Title)
	} else {
		fmt.Printf("☆ Removed '%s' from favorites\n", conv.Title)
	}

	return nil
}

func runHistoryExport(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	// Resolve alias
	resolver := history.NewResolver(store)
	id, err := resolver.Resolve(args[0])
	if err != nil {
		return fmt.Errorf("✗ %w", err)
	}

	// Determine format
	format := history.ExportFormatMarkdown
	if historyFormatFlag != "" {
		switch strings.ToLower(historyFormatFlag) {
		case "markdown", "md":
			format = history.ExportFormatMarkdown
		case "json":
			format = history.ExportFormatJSON
		default:
			return fmt.Errorf("✗ Unknown format: %s (use 'markdown' or 'json')", historyFormatFlag)
		}
	} else if historyOutputFlag != "" {
		// Auto-detect from extension
		ext := strings.ToLower(filepath.Ext(historyOutputFlag))
		switch ext {
		case ".json":
			format = history.ExportFormatJSON
		case ".md", ".markdown":
			format = history.ExportFormatMarkdown
		}
	}

	// Export
	var content []byte
	if format == history.ExportFormatJSON {
		content, err = store.ExportToJSON(id)
	} else {
		var md string
		md, err = store.ExportToMarkdown(id)
		content = []byte(md)
	}
	if err != nil {
		return fmt.Errorf("✗ Export failed: %w", err)
	}

	// Output
	if historyOutputFlag == "" {
		fmt.Print(string(content))
	} else {
		if err := os.WriteFile(historyOutputFlag, content, 0o644); err != nil {
			return fmt.Errorf("✗ Failed to write file: %w", err)
		}
		fmt.Printf("✓ Exported to %s\n", historyOutputFlag)
	}

	return nil
}

func runHistorySearch(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	query := args[0]
	results, err := store.SearchConversations(query, historyContentFlag)
	if err != nil {
		return fmt.Errorf("✗ Search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Printf("No conversations matching '%s'\n", query)
		if !historyContentFlag {
			fmt.Println("Tip: Use --content to also search in message content.")
		}
		return nil
	}

	fmt.Printf("Found %d conversation(s) matching '%s':\n\n", len(results), query)

	for _, r := range results {
		star := "  "
		if r.Conversation.IsFavorite {
			star = "★ "
		}

		title := r.Conversation.Title
		if len(title) > 40 {
			title = title[:40] + "..."
		}

		relTime := history.FormatRelativeTime(r.Conversation.UpdatedAt)

		fmt.Printf("%s%s (%s)\n", star, title, relTime)

		if r.MatchField == "content" {
			// Show snippet
			snippet := r.MatchSnippet
			if len(snippet) > 80 {
				snippet = snippet[:80] + "..."
			}
			fmt.Printf("   └─ \"%s\"\n", snippet)
		}
		fmt.Println()
	}

	return nil
}

// truncate truncates a string to maxLen and adds "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
