package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Elias-Larsson/remdoc/internal/backend"
	"github.com/spf13/cobra"
)

var (
	deployImage      string
	deployName       string
	deployPorts      []string
	deployEnv        []string
	deployRestart    string
	deployAutoRemove bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a new container to the remote server",
	Long: `Create and start a Docker container on the remote server via Portainer.

Examples:
  # Deploy nginx with port mapping
  remdoc deploy --image nginx:latest --name my-nginx --port 8080:80

  # Deploy with environment variables
  remdoc deploy --image postgres:14 --name my-db --port 5432:5432 \
    --env POSTGRES_PASSWORD=secret --env POSTGRES_DB=myapp

  # Deploy with restart policy
  remdoc deploy --image redis:alpine --name my-redis --port 6379:6379 \
    --restart unless-stopped`,
	RunE: runDeploy,
}

func init() {
	deployCmd.Flags().StringVar(&deployImage, "image", "", "Docker image to deploy (required)")
	deployCmd.Flags().StringVar(&deployName, "name", "", "Container name (optional, Docker will generate if not provided)")
	deployCmd.Flags().StringSliceVarP(&deployPorts, "port", "p", []string{}, "Port mappings (e.g., 8080:80, can be specified multiple times)")
	deployCmd.Flags().StringSliceVarP(&deployEnv, "env", "e", []string{}, "Environment variables (e.g., KEY=value, can be specified multiple times)")
	deployCmd.Flags().StringVar(&deployRestart, "restart", "unless-stopped", "Restart policy (no, always, unless-stopped, on-failure)")
	deployCmd.Flags().BoolVar(&deployAutoRemove, "rm", false, "Automatically remove the container when it stops")

	deployCmd.MarkFlagRequired("image")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	portMappings, err := parsePorts(deployPorts)
	if err != nil {
		return fmt.Errorf("invalid port mapping: %w", err)
	}

	envMap, err := parseEnv(deployEnv)
	if err != nil {
		return fmt.Errorf("invalid environment variable: %w", err)
	}

	opts := backend.DeployOptions{
		Name:       deployName,
		Image:      deployImage,
		Ports:      portMappings,
		Env:        envMap,
		Restart:    deployRestart,
		AutoRemove: deployAutoRemove,
	}

	fmt.Printf("Deploying container from image %s...\n", deployImage)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	container, err := client.DeployContainer(ctx, opts)
	if err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	fmt.Printf("✓ Container deployed successfully\n")
	fmt.Printf("  ID:    %s\n", container.ID)
	fmt.Printf("  Name:  %s\n", container.Name)
	fmt.Printf("  Image: %s\n", container.Image)
	fmt.Printf("  State: %s\n", container.State)

	return nil
}

func parsePorts(ports []string) ([]backend.PortMapping, error) {
	var mappings []backend.PortMapping

	for _, portStr := range ports {
		parts := strings.Split(portStr, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("port must be in format HOST:CONTAINER (got: %s)", portStr)
		}

		mappings = append(mappings, backend.PortMapping{
			HostPort:      parts[0],
			ContainerPort: parts[1],
			Protocol:      "tcp",
		})
	}

	return mappings, nil
}

func parseEnv(envVars []string) (map[string]string, error) {
	envMap := make(map[string]string)

	for _, envStr := range envVars {
		parts := strings.SplitN(envStr, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("env var must be in format KEY=value (got: %s)", envStr)
		}

		envMap[parts[0]] = parts[1]
	}

	return envMap, nil
}
