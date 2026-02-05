package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	composeFile string
	composeName string
)

var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Deploy a Docker Compose file as a stack",
	Long: `Deploy a local docker-compose file to the remote server via Portainer.

Examples:
  remdoc compose --file ./docker-compose.yml --name my-stack
  remdoc compose -f ./compose.yaml -n my-stack`,
	RunE: runCompose,
}

func init() {
	composeCmd.Flags().StringVarP(&composeFile, "file", "f", "", "Path to docker-compose file (required)")
	composeCmd.Flags().StringVarP(&composeName, "name", "n", "", "Stack name (optional; defaults to file name)")
	composeCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(composeCmd)
}

func runCompose(cmd *cobra.Command, args []string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	content, err := os.ReadFile(composeFile)
	if err != nil {
		return fmt.Errorf("failed to read compose file: %w", err)
	}

	name := strings.TrimSpace(composeName)
	if name == "" {
		base := filepath.Base(composeFile)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	fmt.Printf("Deploying compose stack %s from %s...\n", name, composeFile)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stackID, err := client.DeployComposeStack(ctx, name, string(content))
	if err != nil {
		return fmt.Errorf("compose deployment failed: %w", err)
	}

	fmt.Printf("✓ Stack deployed successfully (ID: %d)\n", stackID)
	return nil
}
