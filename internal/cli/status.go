package cli

import (
    "context"
    "fmt"
    "os"
    "text/tabwriter"
    "time"

    "github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "List all containers on the remote server",
    Long:  `Display the status of all Docker containers managed via Portainer.`,
    RunE:  runStatus,
}

func init() {
    rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
    client, err := getClient()
    if err != nil {
        return err
    }

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    containers, err := client.ListContainers(ctx)
    if err != nil {
        return fmt.Errorf("failed to fetch containers: %w", err)
    }

    if len(containers) == 0 {
        fmt.Println("No containers found.")
        return nil
    }

    w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
    fmt.Fprintln(w, "CONTAINER ID\tNAME\tIMAGE\tSTATE\tSTATUS")

    for _, c := range containers {
        fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", c.ID, c.Name, c.Image, c.State, c.Status)
    }

    w.Flush()
    return nil
}