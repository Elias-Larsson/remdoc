package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Elias-Larsson/remdoc/internal/backend/portainer"
	"github.com/Elias-Larsson/remdoc/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	username string
	password string
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with a Portainer instance",
	Long: `Authenticate with Portainer using your username and password.

Your credentials will be used to obtain a JWT token, which will be
saved in ~/.remdoc/config.json (permissions: 0600).

Usage:
  remdoc login --username admin
  remdoc login -u admin -p yourpassword`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().StringVarP(&username, "username", "u", "", "Portainer username (required)")
	loginCmd.Flags().StringVarP(&password, "password", "p", "", "Portainer password (will prompt securely if not provided)")
	loginCmd.MarkFlagRequired("username")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Portainer URL (e.g., https://portainer.example.com): ")
	url, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read URL: %w", err)
	}
	url = strings.TrimSpace(url)

	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	if password == "" {
		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() 
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password = string(passwordBytes)
	}

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	fmt.Print("Authenticating... ")
	jwt, err := getJWTFromPortainer(url, username, password)
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Println("✓")

	fmt.Print("Validating credentials... ")
	client := portainer.NewClient(url, jwt)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Validate(ctx); err != nil {
		fmt.Println("✗")
		return fmt.Errorf("validation failed: %w", err)
	}

	cfg := &config.Config{
		PortainerURL: url,
		JWT:          jwt,
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✓ Login successful. Config saved to ~/.remdoc/config.json")
	return nil
}

func getJWTFromPortainer(baseURL, username, password string) (string, error) {
	authURL := strings.TrimRight(baseURL, "/") + "/api/auth"

	payload := map[string]string{
		"username": username,
		"password": password,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode credentials: %w", err)
	}

	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Portainer: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnprocessableEntity || resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("invalid username or password")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Portainer API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		JWT string `json:"jwt"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.JWT == "" {
		return "", fmt.Errorf("no JWT returned from Portainer")
	}

	return result.JWT, nil
}