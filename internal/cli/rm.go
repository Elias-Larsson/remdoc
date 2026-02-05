package cli

import (
    "context"
    "fmt"
    "time"

    "github.com/spf13/cobra"
)

var rmForce bool

var rmCmd = &cobra.Command{
    Use:   "rm <container>",
    Short: "Remove a container",
    Long: `Remove a container from the remote server.

The container can be specified by ID or name.

Examples:
  remdoc rm test-nginx
  remdoc rm 186e01159dd1
  remdoc rm test-nginx --force  # Force remove even if running`,
    Args: cobra.ExactArgs(1),
    RunE: runRm,
}

func init() {
    rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force remove container (even if running)")
    rootCmd.AddCommand(rmCmd)
}

func runRm(cmd *cobra.Command, args []string) error {
    containerID := args[0]

    client, err := getClient()
    if err != nil {
        return err
    }

    fmt.Printf("Removing container %s...\n", containerID)

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if err := client.RemoveContainer(ctx, containerID, rmForce); err != nil {
        return fmt.Errorf("failed to remove container: %w", err)
    }

    fmt.Println("✓ Container removed successfully")
    return nil
}