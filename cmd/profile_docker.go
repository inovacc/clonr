package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/inovacc/clonr/internal/client/grpc"
	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

var (
	dockerProfileListJSON    bool
	dockerProfileRemoveForce bool
	dockerProfileAddRegistry string
	dockerProfileAddUsername string
	dockerProfileAddToken    string
)

func init() {
	profileCmd.AddCommand(dockerProfileCmd)

	dockerProfileCmd.AddCommand(dockerProfileAddCmd)
	dockerProfileCmd.AddCommand(dockerProfileListCmd)
	dockerProfileCmd.AddCommand(dockerProfileRemoveCmd)
	dockerProfileCmd.AddCommand(dockerProfileLoginCmd)
	dockerProfileCmd.AddCommand(dockerProfileStatusCmd)

	dockerProfileListCmd.Flags().BoolVar(&dockerProfileListJSON, "json", false, "Output as JSON")
	dockerProfileRemoveCmd.Flags().BoolVarP(&dockerProfileRemoveForce, "force", "f", false, "Skip confirmation")
	dockerProfileAddCmd.Flags().StringVar(&dockerProfileAddRegistry, "registry", model.DefaultDockerRegistry(), "Container registry URL")
	dockerProfileAddCmd.Flags().StringVar(&dockerProfileAddUsername, "username", "", "Registry username")
	dockerProfileAddCmd.Flags().StringVar(&dockerProfileAddToken, "token", "", "Registry password/token")

	_ = dockerProfileAddCmd.MarkFlagRequired("username")
	_ = dockerProfileAddCmd.MarkFlagRequired("token")
}

var dockerProfileCmd = &cobra.Command{
	Use:   "docker",
	Short: "Manage container registry credentials",
	Long: `Manage container registry authentication profiles for Docker, ghcr.io, etc.

Each docker profile stores registry credentials securely encrypted.
Use these credentials to authenticate with container registries.

Available Commands:
  add          Create a new docker profile with registry credentials
  list         List all docker profiles
  remove       Delete a docker profile
  login        Login to a registry using a profile
  status       Show docker profile information

Supported Registries:
  docker.io    Docker Hub (default)
  ghcr.io      GitHub Container Registry
  gcr.io       Google Container Registry
  ecr.aws      Amazon ECR
  azurecr.io   Azure Container Registry

Examples:
  clonr profile docker add dockerhub --username myuser --token mytoken
  clonr profile docker add github --registry ghcr.io --username myuser --token ghp_xxx
  clonr profile docker list
  clonr profile docker login dockerhub
  clonr profile docker remove old-profile`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var dockerProfileAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new docker profile with registry credentials",
	Long: `Create a new container registry authentication profile.

The credentials will be stored securely encrypted using the keystore.

Examples:
  clonr profile docker add dockerhub --username myuser --token mytoken
  clonr profile docker add github --registry ghcr.io --username myuser --token ghp_xxx
  clonr profile docker add aws --registry ecr.aws --username AWS --token $(aws ecr get-login-password)`,
	Args: cobra.ExactArgs(1),
	RunE: runDockerProfileAdd,
}

func runDockerProfileAdd(_ *cobra.Command, args []string) error {
	name := args[0]

	client, err := grpc.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Check if profile already exists
	exists, err := client.DockerProfileExists(name)
	if err != nil {
		return fmt.Errorf("failed to check profile existence: %w", err)
	}

	if exists {
		return fmt.Errorf("docker profile '%s' already exists", name)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Creating docker profile: %s\n", name)
	_, _ = fmt.Fprintf(os.Stdout, "Registry: %s\n", dockerProfileAddRegistry)
	_, _ = fmt.Fprintf(os.Stdout, "Username: %s\n", dockerProfileAddUsername)

	// Encrypt the token
	encryptedToken, err := tpm.EncryptToken(dockerProfileAddToken, name, dockerProfileAddRegistry)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Determine storage type
	tokenStorage := model.TokenStorageEncrypted
	if tpm.IsDataOpen(encryptedToken) {
		tokenStorage = model.TokenStorageOpen
	}

	// Create profile
	profile := &model.DockerProfile{
		Name:           name,
		Registry:       dockerProfileAddRegistry,
		Username:       dockerProfileAddUsername,
		EncryptedToken: encryptedToken,
		TokenStorage:   tokenStorage,
		CreatedAt:      time.Now(),
	}

	if err := client.SaveDockerProfile(profile); err != nil {
		return fmt.Errorf("failed to save docker profile: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "\nDocker profile created successfully!")
	_, _ = fmt.Fprintf(os.Stdout, "Storage: %s\n", tokenStorage)
	_, _ = fmt.Fprintf(os.Stdout, "\nTo login: clonr profile docker login %s\n", name)

	return nil
}

var dockerProfileListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all docker profiles",
	Aliases: []string{"ls"},
	RunE:    runDockerProfileList,
}

// DockerProfileListItem represents a docker profile in JSON output
type DockerProfileListItem struct {
	Name       string    `json:"name"`
	Registry   string    `json:"registry"`
	Username   string    `json:"username"`
	Storage    string    `json:"storage"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
}

func runDockerProfileList(_ *cobra.Command, _ []string) error {
	client, err := grpc.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	profiles, err := client.ListDockerProfiles()
	if err != nil {
		return fmt.Errorf("failed to list docker profiles: %w", err)
	}

	if len(profiles) == 0 {
		if dockerProfileListJSON {
			_, _ = fmt.Fprintln(os.Stdout, "[]")
			return nil
		}

		printEmptyResult("docker profiles", "clonr profile docker add <name>")

		return nil
	}

	if dockerProfileListJSON {
		items := make([]DockerProfileListItem, 0, len(profiles))
		for _, p := range profiles {
			items = append(items, DockerProfileListItem{
				Name:       p.Name,
				Registry:   p.Registry,
				Username:   p.Username,
				Storage:    formatTokenStorage(p.TokenStorage),
				CreatedAt:  p.CreatedAt,
				LastUsedAt: p.LastUsedAt,
			})
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(items)
	}

	// Text output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tREGISTRY\tUSERNAME\tSTORAGE")
	_, _ = fmt.Fprintln(w, "----\t--------\t--------\t-------")

	for _, p := range profiles {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			p.Name,
			p.Registry,
			p.Username,
			formatTokenStorage(p.TokenStorage),
		)
	}

	return w.Flush()
}

var dockerProfileRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Short:   "Delete a docker profile",
	Aliases: []string{"rm", "delete"},
	Args:    cobra.ExactArgs(1),
	RunE:    runDockerProfileRemove,
}

func runDockerProfileRemove(_ *cobra.Command, args []string) error {
	name := args[0]

	client, err := grpc.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Check if profile exists
	profile, err := client.GetDockerProfile(name)
	if err != nil {
		return fmt.Errorf("failed to get docker profile: %w", err)
	}

	if profile == nil {
		return fmt.Errorf("docker profile '%s' not found", name)
	}

	if !dockerProfileRemoveForce {
		_, _ = fmt.Fprintf(os.Stdout, "Delete docker profile '%s' for %s? [y/N]: ", name, profile.Registry)
		if !promptConfirm("") {
			_, _ = fmt.Fprintln(os.Stdout, "Cancelled.")
			return nil
		}
	}

	if err := client.DeleteDockerProfile(name); err != nil {
		return fmt.Errorf("failed to delete docker profile: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Docker profile '%s' deleted.\n", name)

	return nil
}

var dockerProfileLoginCmd = &cobra.Command{
	Use:   "login <name>",
	Short: "Login to a registry using a profile",
	Long: `Login to a container registry using stored credentials.

This executes 'docker login' with the stored credentials.

Examples:
  clonr profile docker login dockerhub
  clonr profile docker login github`,
	Args: cobra.ExactArgs(1),
	RunE: runDockerProfileLogin,
}

func runDockerProfileLogin(_ *cobra.Command, args []string) error {
	name := args[0]

	client, err := grpc.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := client.GetDockerProfile(name)
	if err != nil {
		return fmt.Errorf("failed to get docker profile: %w", err)
	}

	if profile == nil {
		return fmt.Errorf("docker profile '%s' not found", name)
	}

	// Decrypt token
	token, err := tpm.DecryptToken(profile.EncryptedToken, name, profile.Registry)
	if err != nil {
		return fmt.Errorf("failed to decrypt token: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Logging in to %s as %s...\n", profile.Registry, profile.Username)

	// Execute docker login
	cmd := exec.Command("docker", "login", profile.Registry, "-u", profile.Username, "--password-stdin")
	cmd.Stdin = strings.NewReader(token)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker login failed: %w", err)
	}

	// Update last used timestamp
	profile.LastUsedAt = time.Now()
	_ = client.SaveDockerProfile(profile)

	return nil
}

var dockerProfileStatusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show docker profile information",
	Args:  cobra.ExactArgs(1),
	RunE:  runDockerProfileStatus,
}

func runDockerProfileStatus(_ *cobra.Command, args []string) error {
	name := args[0]

	client, err := grpc.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := client.GetDockerProfile(name)
	if err != nil {
		return fmt.Errorf("failed to get docker profile: %w", err)
	}

	if profile == nil {
		return fmt.Errorf("docker profile '%s' not found", name)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Profile:  %s\n", profile.Name)
	_, _ = fmt.Fprintf(os.Stdout, "Registry: %s\n", profile.Registry)
	_, _ = fmt.Fprintf(os.Stdout, "Username: %s\n", profile.Username)
	_, _ = fmt.Fprintf(os.Stdout, "Storage:  %s\n", formatTokenStorage(profile.TokenStorage))
	_, _ = fmt.Fprintf(os.Stdout, "Created:  %s\n", profile.CreatedAt.Format(time.RFC3339))

	if !profile.LastUsedAt.IsZero() {
		_, _ = fmt.Fprintf(os.Stdout, "Last used: %s\n", profile.LastUsedAt.Format(time.RFC3339))
	}

	return nil
}
