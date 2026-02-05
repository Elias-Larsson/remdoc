package cli

import (
    "context"
    "fmt"
    "time"

    "github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
    Use:   "stop <container>",
    Short: "Stop a running container",
    Long: `Stop a running container on the remote server.

The container can be specified by ID or name.

Examples:
  remdoc stop my-nginx
  remdoc stop 186e01159dd1`,
    Args: cobra.ExactArgs(1),
    RunE: runStop,
}

func init() {
    rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
    containerID := args[0]

    client, err := getClient()
    if err != nil {
        return err
    }

    fmt.Printf("Stopping container %s...\n", containerID)

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := client.StopContainer(ctx, containerID); err != nil {
        return fmt.Errorf("failed to stop container: %w", err)
    }

    fmt.Println("✓ Container stopped successfully")
    return nil
}