package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/diogo/geminiweb/internal/history"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Manage conversation history",
	Long:  `View and manage your local conversation history.`,
}

var historyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all conversations",
	RunE:  runHistoryList,
}

var historyShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a conversation",
	Args:  cobra.ExactArgs(1),
	RunE:  runHistoryShow,
}

var historyDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a conversation",
	Args:  cobra.ExactArgs(1),
	RunE:  runHistoryDelete,
}

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all conversations",
	RunE:  runHistoryClear,
}

func init() {
	historyCmd.AddCommand(historyListCmd)
	historyCmd.AddCommand(historyShowCmd)
	historyCmd.AddCommand(historyDeleteCmd)
	historyCmd.AddCommand(historyClearCmd)
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
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tMODEL\tMESSAGES\tUPDATED")
	_, _ = fmt.Fprintln(w, "--\t-----\t-----\t--------\t-------")

	for _, conv := range conversations {
		updated := conv.UpdatedAt.Format("2006-01-02 15:04")
		title := conv.Title
		if len(title) > 40 {
			title = title[:40] + "..."
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			conv.ID, title, conv.Model, len(conv.Messages), updated)
	}

	return w.Flush()
}

func runHistoryShow(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	conv, err := store.GetConversation(args[0])
	if err != nil {
		return fmt.Errorf("conversation not found: %w", err)
	}

	fmt.Printf("ID: %s\n", conv.ID)
	fmt.Printf("Title: %s\n", conv.Title)
	fmt.Printf("Model: %s\n", conv.Model)
	fmt.Printf("Created: %s\n", conv.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", conv.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Messages: %d\n", len(conv.Messages))
	fmt.Println()

	for i, msg := range conv.Messages {
		role := "You"
		if msg.Role == "assistant" {
			role = "Gemini"
		}
		fmt.Printf("[%d] %s (%s):\n", i+1, role, msg.Timestamp.Format("15:04"))

		if msg.Thoughts != "" {
			fmt.Printf("  💭 %s\n", msg.Thoughts)
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

	if err := store.DeleteConversation(args[0]); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	fmt.Printf("Deleted conversation: %s\n", args[0])
	return nil
}

func runHistoryClear(cmd *cobra.Command, args []string) error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("failed to open history: %w", err)
	}

	if err := store.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}

	fmt.Println("All conversations deleted.")
	return nil
}
