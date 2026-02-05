package cli

import (
    "context"
    "fmt"
    "time"

    "github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
    Use:   "start <container>",
    Short: "Start a stopped container",
    Long: `Start a stopped container on the remote server.

The container can be specified by ID or name.

Examples:
  remdoc start my-nginx
  remdoc start 186e01159dd1`,
    Args: cobra.ExactArgs(1),
    RunE: runStart,
}

func init() {
    rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
    containerID := args[0]

    client, err := getClient()
    if err != nil {
        return err
    }

    fmt.Printf("Starting container %s...\n", containerID)

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := client.StartContainer(ctx, containerID); err != nil {
        return fmt.Errorf("failed to start container: %w", err)
    }

    fmt.Println("✓ Container started successfully")
    return nil
}