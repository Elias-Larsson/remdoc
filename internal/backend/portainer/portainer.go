package portainer

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"

    "github.com/Elias-Larsson/remdoc/internal/backend"
)

type Client struct {
    BaseURL    string
    JWT        string
    HTTPClient *http.Client
}

func NewClient(baseURL, jwt string) *Client {
    return &Client{
        BaseURL: strings.TrimRight(baseURL, "/"),
        JWT:     jwt,
        HTTPClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

// checkResponse validates the HTTP response and returns an error if unexpected
func checkResponse(resp *http.Response, expectedCodes ...int) error {
    for _, code := range expectedCodes {
        if resp.StatusCode == code {
            return nil
        }
    }
    body, _ := io.ReadAll(resp.Body)
    return fmt.Errorf("Portainer API error (status %d): %s", resp.StatusCode, string(body))
}

func (c *Client) Validate(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/status", nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.JWT)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to connect to Portainer: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusUnauthorized {
        return fmt.Errorf("invalid JWT token (unauthorized)")
    }

    if err := checkResponse(resp, http.StatusOK); err != nil {
        return err
    }

    return nil
}

func (c *Client) ListContainers(ctx context.Context) ([]backend.Container, error) {
    endpointID, err := c.getFirstEndpoint(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get endpoint: %w", err)
    }

    url := fmt.Sprintf("%s/api/endpoints/%d/docker/containers/json?all=true", c.BaseURL, endpointID)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.JWT)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch containers: %w", err)
    }
    defer resp.Body.Close()

    if err := checkResponse(resp, http.StatusOK); err != nil {
        return nil, err
    }

    var rawContainers []struct {
        ID     string   `json:"Id"`
        Names  []string `json:"Names"`
        Image  string   `json:"Image"`
        State  string   `json:"State"`
        Status string   `json:"Status"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&rawContainers); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    containers := make([]backend.Container, len(rawContainers))
    for i, raw := range rawContainers {
        name := "unknown"
        if len(raw.Names) > 0 {
            name = strings.TrimPrefix(raw.Names[0], "/")
        }

        containers[i] = backend.Container{
            ID:     raw.ID[:12],
            Name:   name,
            Image:  raw.Image,
            State:  raw.State,
            Status: raw.Status,
        }
    }

    return containers, nil
}

func (c *Client) getFirstEndpoint(ctx context.Context) (int, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/endpoints", nil)
    if err != nil {
        return 0, fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.JWT)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return 0, fmt.Errorf("failed to fetch endpoints: %w", err)
    }
    defer resp.Body.Close()

    if err := checkResponse(resp, http.StatusOK); err != nil {
        return 0, err
    }

    var endpoints []struct {
        ID int `json:"Id"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
        return 0, fmt.Errorf("failed to parse endpoints: %w", err)
    }

    if len(endpoints) == 0 {
        return 0, fmt.Errorf("no Docker endpoints configured in Portainer")
    }

    return endpoints[0].ID, nil
}

func (c *Client) DeployContainer(ctx context.Context, opts backend.DeployOptions) (*backend.Container, error) {
    endpointID, err := c.getFirstEndpoint(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get endpoint: %w", err)
    }

    containerID, err := c.createContainer(ctx, endpointID, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to create container: %w", err)
    }

    if err := c.startContainer(ctx, endpointID, containerID); err != nil {
        return nil, fmt.Errorf("failed to start container: %w", err)
    }

    return &backend.Container{
        ID:    containerID[:12],
        Name:  opts.Name,
        Image: opts.Image,
        State: "running",
    }, nil
}

func (c *Client) createContainer(ctx context.Context, endpointID int, opts backend.DeployOptions) (string, error) {
    url := fmt.Sprintf("%s/api/endpoints/%d/docker/containers/create", c.BaseURL, endpointID)

    if opts.Name != "" {
        url += "?name=" + opts.Name
    }

    portBindings := make(map[string][]map[string]string)
    exposedPorts := make(map[string]struct{})

    for _, pm := range opts.Ports {
        protocol := pm.Protocol
        if protocol == "" {
            protocol = "tcp"
        }

        containerPortKey := pm.ContainerPort + "/" + protocol
        exposedPorts[containerPortKey] = struct{}{}

        portBindings[containerPortKey] = []map[string]string{
            {"HostPort": pm.HostPort},
        }
    }

    var envVars []string
    for key, value := range opts.Env {
        envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
    }

    payload := map[string]interface{}{
        "Image":        opts.Image,
        "ExposedPorts": exposedPorts,
        "Env":          envVars,
        "HostConfig": map[string]interface{}{
            "PortBindings": portBindings,
            "RestartPolicy": map[string]interface{}{
                "Name": opts.Restart,
            },
            "AutoRemove": opts.AutoRemove,
        },
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return "", fmt.Errorf("failed to encode payload: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return "", fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.JWT)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if err := checkResponse(resp, http.StatusCreated, http.StatusOK); err != nil {
        return "", err
    }

    var result struct {
        ID       string   `json:"Id"`
        Warnings []string `json:"Warnings"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", fmt.Errorf("failed to parse response: %w", err)
    }

    return result.ID, nil
}

func (c *Client) startContainer(ctx context.Context, endpointID int, containerID string) error {
    url := fmt.Sprintf("%s/api/endpoints/%d/docker/containers/%s/start", c.BaseURL, endpointID, containerID)

    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.JWT)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if err := checkResponse(resp, http.StatusNoContent, http.StatusNotModified); err != nil {
        return err
    }

    return nil
}

func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
    endpointID, err := c.getFirstEndpoint(ctx)
    if err != nil {
        return fmt.Errorf("failed to get endpoint: %w", err)
    }

    url := fmt.Sprintf("%s/api/endpoints/%d/docker/containers/%s?force=%t",
        c.BaseURL, endpointID, containerID, force)

    req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.JWT)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if err := checkResponse(resp, http.StatusNoContent, http.StatusOK); err != nil {
        return err
    }

    return nil
}

func (c *Client) StopContainer(ctx context.Context, containerID string) error {
    endpointID, err := c.getFirstEndpoint(ctx)
    if err != nil {
        return fmt.Errorf("failed to get endpoint: %w", err)
    }

    url := fmt.Sprintf("%s/api/endpoints/%d/docker/containers/%s/stop", c.BaseURL, endpointID, containerID)

    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.JWT)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if err := checkResponse(resp, http.StatusNoContent, http.StatusNotModified); err != nil {
        return err
    }

    return nil
}

func (c *Client) StartContainer(ctx context.Context, containerID string) error {
    endpointID, err := c.getFirstEndpoint(ctx)
    if err != nil {
        return fmt.Errorf("failed to get endpoint: %w", err)
    }
    
    return c.startContainer(ctx, endpointID, containerID)
}

func (c *Client) DeployComposeStack(ctx context.Context, name string, composeContent string) (int, error) {
    if strings.TrimSpace(name) == "" {
        return 0, fmt.Errorf("stack name cannot be empty")
    }
    if strings.TrimSpace(composeContent) == "" {
        return 0, fmt.Errorf("compose content cannot be empty")
    }

    endpointID, err := c.getFirstEndpoint(ctx)
    if err != nil {
        return 0, fmt.Errorf("failed to get endpoint: %w", err)
    }

    url := fmt.Sprintf("%s/api/stacks?type=2&method=string&endpointId=%d", c.BaseURL, endpointID)

    payload := map[string]interface{}{
        "Name":             name,
        "StackFileContent": composeContent,
        "Env":              []interface{}{},
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return 0, fmt.Errorf("failed to encode payload: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return 0, fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.JWT)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return 0, fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if err := checkResponse(resp, http.StatusCreated, http.StatusOK); err != nil {
        return 0, err
    }

    var result struct {
        ID int `json:"Id"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, fmt.Errorf("failed to parse response: %w", err)
    }

    return result.ID, nil
}