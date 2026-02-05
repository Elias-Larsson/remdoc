package cli

import (
    "fmt"
    "os"

    "github.com/Elias-Larsson/remdoc/internal/backend/portainer"
    "github.com/Elias-Larsson/remdoc/internal/config"
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "remdoc",
    Short: "Manage remote Docker containers via Portainer",
    Long: `remdoc is a CLI tool for deploying and managing Docker containers
on remote servers using the Portainer API.`,
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

// getClient loads config and returns a configured Portainer client
func getClient() (*portainer.Client, error) {
    cfg, err := config.Load()
    if err != nil {
        return nil, err
    }
    return portainer.NewClient(cfg.PortainerURL, cfg.JWT), nil
}