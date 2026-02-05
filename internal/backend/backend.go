package backend

import "context"

// Backend defines the interface for container management backends
// (Portainer, custom Docker agent, etc.)
type Backend interface {
	// Validate checks if the connection and credentials are valid
	Validate(ctx context.Context) error
	
	// ListContainers returns all containers on the remote server
	ListContainers(ctx context.Context) ([]Container, error)
	
	// DeployContainer creates and starts a new container
	DeployContainer(ctx context.Context, opts DeployOptions) (*Container, error)
	
	// RemoveContainer removes a container by ID or name
	RemoveContainer(ctx context.Context, containerID string, force bool) error
	
	// StopContainer stops a running container
	StopContainer(ctx context.Context, containerID string) error
	
	// StartContainer starts a stopped container
	StartContainer(ctx context.Context, containerID string) error

	// DeployComposeStack deploys a Docker Compose stack from content
	DeployComposeStack(ctx context.Context, name string, composeContent string) (int, error)
}

// Container represents a Docker container (simplified for now)
type Container struct {
	ID      string
	Name    string
	Image   string
	State   string
	Status  string
}

// DeployOptions contains all parameters needed to deploy a container
type DeployOptions struct {
	Name        string            // Container name
	Image       string            // Docker image (e.g., "nginx:latest")
	Ports       []PortMapping     // Port mappings (e.g., 8080:80)
	Env         map[string]string // Environment variables
	Restart     string            // Restart policy (e.g., "unless-stopped")
	AutoRemove  bool              // Remove container when stopped
}

// PortMapping represents a port binding
type PortMapping struct {
	HostPort      string // Port on the host (e.g., "8080")
	ContainerPort string // Port in the container (e.g., "80")
	Protocol      string // "tcp" or "udp" (default: tcp)
}