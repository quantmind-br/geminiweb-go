package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/history"
)

var (
	historyForceFlag     bool
	historyContentFlag   bool
	historyOutputFlag    string
	historyFormatFlag    string
	historyFavoritesFlag bool
)

// NewHistoryCmd creates a new history command
func NewHistoryCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Manage conversation history",
		Long:  "View and manage your local conversation history.\n\n" + history.ListAliases(),
	}

	cmd.AddCommand(NewHistoryListCmd(deps))
	cmd.AddCommand(NewHistoryShowCmd(deps))
	cmd.AddCommand(NewHistoryDeleteCmd(deps))
	cmd.AddCommand(NewHistoryClearCmd(deps))
	cmd.AddCommand(NewHistoryRenameCmd(deps))
	cmd.AddCommand(NewHistoryFavoriteCmd(deps))
	cmd.AddCommand(NewHistoryExportCmd(deps))
	cmd.AddCommand(NewHistorySearchCmd(deps))

	return cmd
}

func NewHistoryListCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all conversations",
		Long:  "List all conversations with indices and favorite indicators.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryList(cmd, args)
		},
	}
	cmd.Flags().BoolVar(&historyFavoritesFlag, "favorites", false, "List only favorite conversations")
	return cmd
}

func NewHistoryShowCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "show <ref>",
		Short: "Show a conversation",
		Long:  "Show the full content of a conversation.\n\n" + history.ListAliases(),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryShow(cmd, args)
		},
	}
}

func NewHistoryDeleteCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <ref>",
		Short: "Delete a conversation",
		Long:  "Delete a conversation with confirmation.\n\nUse --force to skip confirmation.\n\n" + history.ListAliases(),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryDelete(cmd, args)
		},
	}
	cmd.Flags().BoolVarP(&historyForceFlag, "force", "f", false, "Skip confirmation")
	return cmd
}

func NewHistoryClearCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Delete all conversations",
		Long:  "Delete all conversations. Use --force to skip confirmation.", 
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryClear(cmd, args)
		},
	}
	cmd.Flags().BoolVarP(&historyForceFlag, "force", "f", false, "Skip confirmation")
	return cmd
}

func NewHistoryRenameCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "rename <ref> <title>",
		Short: "Rename a conversation",
		Long:  "Rename a conversation to a new title.\n\n" + history.ListAliases(),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryRename(cmd, args)
		},
	}
}

func NewHistoryFavoriteCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "favorite <ref>",
		Short: "Toggle favorite status",
		Long:  "Toggle the favorite status of a conversation.\n\n" + history.ListAliases(),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryFavorite(cmd, args)
		},
	}
}

func NewHistoryExportCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <ref>",
		Short: "Export a conversation",
		Long:  "Export a conversation to a file.\n\n" + history.ListAliases(),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryExport(cmd, args)
		},
	}
	cmd.Flags().StringVarP(&historyOutputFlag, "output", "o", "", "Output file path")
	cmd.Flags().StringVarP(&historyFormatFlag, "format", "f", "", "Output format (markdown, json)")
	return cmd
}

func NewHistorySearchCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search conversations",
		Long:  "Search for conversations by title or content.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistorySearch(cmd, args)
		},
	}
	cmd.Flags().BoolVar(&historyContentFlag, "content", false, "Search in message content as well as titles")
	return cmd
}

// Backward compatibility globals
var historyCmd = NewHistoryCmd(nil)
var historyListCmd = NewHistoryListCmd(nil)
var historyShowCmd = NewHistoryShowCmd(nil)
var historyDeleteCmd = NewHistoryDeleteCmd(nil)
var historyClearCmd = NewHistoryClearCmd(nil)
var historyRenameCmd = NewHistoryRenameCmd(nil)
var historyFavoriteCmd = NewHistoryFavoriteCmd(nil)
var historyExportCmd = NewHistoryExportCmd(nil)
var historySearchCmd = NewHistorySearchCmd(nil)

func init() {
	// Root flags and commands are handled in NewRootCmd
}

func runHistoryList(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return err
	}

	convs, err := store.ListConversations()
	if err != nil {
		return err
	}

	if historyFavoritesFlag {
		var favorites []*history.Conversation
		for _, c := range convs {
			if c.IsFavorite {
				favorites = append(favorites, c)
			}
		}
		convs = favorites
	}

	if len(convs) == 0 {
		if historyFavoritesFlag {
			fmt.Println("No favorite conversations found.")
		} else {
			fmt.Println("No conversation history found.")
			fmt.Println("Start a new chat with 'geminiweb chat'")
		}
		return nil
	}

	fmt.Println("Conversation History:")
	for _, c := range convs {
		fav := " "
		if c.IsFavorite {
			fav = "★"
		}
		fmt.Printf("[%d] %s %s (%d msg, %s)\n", c.OrderIndex+1, fav, c.Title, len(c.Messages), history.FormatRelativeTime(c.UpdatedAt))
	}
	return nil
}

func runHistoryShow(cmd *cobra.Command, args []string) error {
	ref := args[0]
	store, err := history.DefaultStore()
	if err != nil {
		return err
	}

	resolver := history.NewResolver(store)
	conv, err := resolver.ResolveWithInfo(ref)
	if err != nil {
		return err
	}

	fmt.Printf("ID:    %s\n", conv.ID)
	fmt.Printf("Title: %s\n", conv.Title)
	fmt.Printf("Model: %s\n", conv.Model)
	fmt.Printf("Time:  %s\n", conv.UpdatedAt.Format(time.RFC1123))
	fmt.Println(strings.Repeat("-", 40))

	for _, msg := range conv.Messages {
		role := "User"
		if msg.Role == "assistant" {
			role = "Gemini"
		}
		fmt.Printf("[%s]: %s\n\n", role, msg.Content)
	}

	return nil
}

func runHistoryDelete(cmd *cobra.Command, args []string) error {
	ref := args[0]
	store, err := history.DefaultStore()
	if err != nil {
		return err
	}

	resolver := history.NewResolver(store)
	conv, err := resolver.ResolveWithInfo(ref)
	if err != nil {
		return err
	}

	if !historyForceFlag {
		fmt.Printf("Are you sure you want to delete conversation '%s'? [y/N]: ", conv.Title)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := store.DeleteConversation(conv.ID); err != nil {
		return err
	}

	fmt.Printf("Deleted conversation '%s'.\n", conv.Title)
	return nil
}

func runHistoryClear(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return err
	}

	convs, _ := store.ListConversations()
	count := len(convs)

	if !historyForceFlag {
		fmt.Printf("Are you sure you want to delete ALL %d conversations? [y/N]: ", count)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := store.ClearAll(); err != nil {
		return err
	}

	fmt.Printf("Deleted %d conversations.\n", count)
	return nil
}

func runHistoryRename(cmd *cobra.Command, args []string) error {
	ref := args[0]
	newTitle := args[1]

	store, err := history.DefaultStore()
	if err != nil {
		return err
	}

	resolver := history.NewResolver(store)
	conv, err := resolver.ResolveWithInfo(ref)
	if err != nil {
		return err
	}

	if err := store.UpdateTitle(conv.ID, newTitle); err != nil {
		return err
	}

	fmt.Printf("Renamed conversation to '%s'.\n", newTitle)
	return nil
}

func runHistoryFavorite(cmd *cobra.Command, args []string) error {
	ref := args[0]

	store, err := history.DefaultStore()
	if err != nil {
		return err
	}

	resolver := history.NewResolver(store)
	conv, err := resolver.ResolveWithInfo(ref)
	if err != nil {
		return err
	}

	newStatus, err := store.ToggleFavorite(conv.ID)
	if err != nil {
		return err
	}

	statusStr := "removed from favorites"
	indicator := "☆"
	if newStatus {
		statusStr = "added to favorites"
		indicator = "★"
	}
	fmt.Printf("%s Conversation '%s' %s.\n", indicator, conv.Title, statusStr)
	return nil
}

func runHistoryExport(cmd *cobra.Command, args []string) error {
	ref := args[0]
	store, err := history.DefaultStore()
	if err != nil {
		return err
	}

	resolver := history.NewResolver(store)
	conv, err := resolver.ResolveWithInfo(ref)
	if err != nil {
		return err
	}

	format := historyFormatFlag
	if format == "" && historyOutputFlag != "" {
		ext := filepath.Ext(historyOutputFlag)
		if ext == ".json" {
			format = "json"
		} else {
			format = "markdown"
		}
	} else if format == "" {
		format = "markdown"
	}

	var content string
	var exportErr error

	if format == "json" {
		data, err := store.ExportToJSON(conv.ID)
		content = string(data)
		exportErr = err
	} else {
		content, exportErr = store.ExportToMarkdown(conv.ID)
	}

	if exportErr != nil {
		return exportErr
	}

	if historyOutputFlag != "" {
		if err := os.WriteFile(historyOutputFlag, []byte(content), 0644); err != nil {
			return err
		}
		fmt.Printf("Exported to %s\n", historyOutputFlag)
	} else {
		fmt.Println(content)
	}

	return nil
}

func runHistorySearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	store, err := history.DefaultStore()
	if err != nil {
		return err
	}

	results, err := store.SearchConversations(query, historyContentFlag)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Printf("No conversations matching '%s'.\n", query)
		return nil
	}

	fmt.Printf("Found %d matching conversations:\n", len(results))
	for _, res := range results {
		fmt.Printf("[%d] %s\n", res.Conversation.OrderIndex+1, res.Conversation.Title)
	}
	return nil
}
