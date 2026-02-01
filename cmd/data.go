package cmd

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
)

var dataCmd = &cobra.Command{
	Use:   "data",
	Short: "Export and import clonr data",
	Long:  `Export and import clonr configuration data (profiles, workspaces, repositories, config).`,
}

var dataExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export all data encrypted with password",
	Long: `Export all clonr data (profiles, workspaces, repositories, config) to stdout.

The data is encrypted with a password using AES-256-GCM and encoded in base58
for easy copy/paste.

IMPORTANT: Profile tokens are included in the export. Make sure to use a strong
password and store the export securely.

Examples:
  clonr data export > backup.txt
  clonr data export --no-tokens > backup.txt`,
	RunE: runDataExport,
}

var dataImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import data from encrypted export",
	Long: `Import clonr data from a previously exported backup.

Reads base58-encoded encrypted data from stdin or a file and decrypts it
with the provided password.

Examples:
  clonr data import < backup.txt
  cat backup.txt | clonr data import
  clonr data import --file backup.txt`,
	RunE: runDataImport,
}

var (
	dataExportNoTokens bool
	dataImportFile     string
	dataImportMerge    bool
)

// ExportData represents the complete export structure
type ExportData struct {
	Version    int                `json:"version"`
	ExportedAt time.Time          `json:"exported_at"`
	Profiles   []model.Profile    `json:"profiles"`
	Workspaces []model.Workspace  `json:"workspaces"`
	Repos      []model.Repository `json:"repositories"`
	Config     *model.Config      `json:"config,omitempty"`
}

// Export format constants
const (
	exportVersion    = 1
	exportMagic      = "CLONR"
	pbkdf2Iterations = 100000
	saltSize         = 16
	nonceSize        = 12
)

func init() {
	rootCmd.AddCommand(dataCmd)

	dataCmd.AddCommand(dataExportCmd)
	dataCmd.AddCommand(dataImportCmd)

	dataExportCmd.Flags().BoolVar(&dataExportNoTokens, "no-tokens", false, "Exclude authentication tokens from export")

	dataImportCmd.Flags().StringVarP(&dataImportFile, "file", "f", "", "Read from file instead of stdin")
	dataImportCmd.Flags().BoolVar(&dataImportMerge, "merge", false, "Merge with existing data instead of replacing")
}

func runDataExport(_ *cobra.Command, _ []string) error {
	client, err := grpcclient.GetClient()
	if err != nil {
		return err
	}

	// Collect all data
	profiles, err := client.ListProfiles()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	repos, err := client.GetAllRepos()
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	config, err := client.GetConfig()
	if err != nil {
		// Config might not exist, that's ok
		config = nil
	}

	// Optionally strip tokens
	if dataExportNoTokens {
		for i := range profiles {
			profiles[i].EncryptedToken = nil
		}
	}

	exportData := ExportData{
		Version:    exportVersion,
		ExportedAt: time.Now(),
		Profiles:   profiles,
		Workspaces: workspaces,
		Repos:      repos,
		Config:     config,
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(exportData)
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}

	// Get password from user
	password, err := readPassword("Enter encryption password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Confirm password
	confirm, err := readPassword("Confirm password: ")
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}

	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	// Encrypt the data
	encrypted, err := encryptData(jsonData, password)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Encode to base58
	encoded := base58.Encode(encrypted)

	// Output with magic header for validation
	_, _ = fmt.Fprintf(os.Stdout, "%s:%s\n", exportMagic, encoded)

	_, _ = fmt.Fprintf(os.Stderr, "\nExported %d profiles, %d workspaces, %d repositories\n",
		len(profiles), len(workspaces), len(repos))

	if dataExportNoTokens {
		_, _ = fmt.Fprintln(os.Stderr, "Note: Tokens were excluded from export")
	}

	return nil
}

func runDataImport(_ *cobra.Command, _ []string) error {
	// Read input
	var (
		input string
		err   error
	)

	if dataImportFile != "" {
		data, err := os.ReadFile(dataImportFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		input = strings.TrimSpace(string(data))
	} else {
		// Read from stdin
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input = strings.TrimSpace(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
	}

	if input == "" {
		return fmt.Errorf("no input data provided")
	}

	// Validate magic header
	if !strings.HasPrefix(input, exportMagic+":") {
		return fmt.Errorf("invalid export format: missing CLONR header")
	}

	encoded := strings.TrimPrefix(input, exportMagic+":")

	// Decode from base58
	encrypted := base58.Decode(encoded)
	if len(encrypted) == 0 {
		return fmt.Errorf("failed to decode data: invalid base58")
	}

	// Get password from user
	password, err := readPassword("Enter decryption password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	// Decrypt the data
	jsonData, err := decryptData(encrypted, password)
	if err != nil {
		return fmt.Errorf("failed to decrypt data (wrong password?): %w", err)
	}

	// Parse JSON
	var exportData ExportData
	if err := json.Unmarshal(jsonData, &exportData); err != nil {
		return fmt.Errorf("failed to parse data: %w", err)
	}

	// Validate version
	if exportData.Version > exportVersion {
		return fmt.Errorf("export version %d is newer than supported version %d", exportData.Version, exportVersion)
	}

	// Get client
	client, err := grpcclient.GetClient()
	if err != nil {
		return err
	}

	// Import data
	var stats struct {
		profiles   int
		workspaces int
		repos      int
	}

	// Import workspaces first (profiles depend on them)
	for _, ws := range exportData.Workspaces {
		exists, err := client.WorkspaceExists(ws.Name)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to check workspace %s: %v\n", ws.Name, err)
			continue
		}

		if exists && !dataImportMerge {
			_, _ = fmt.Fprintf(os.Stderr, "Skipping existing workspace: %s\n", ws.Name)
			continue
		}

		if err := client.SaveWorkspace(&ws); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to import workspace %s: %v\n", ws.Name, err)
			continue
		}

		stats.workspaces++
	}

	// Import profiles
	for _, p := range exportData.Profiles {
		exists, err := client.ProfileExists(p.Name)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to check profile %s: %v\n", p.Name, err)
			continue
		}

		if exists && !dataImportMerge {
			_, _ = fmt.Fprintf(os.Stderr, "Skipping existing profile: %s\n", p.Name)
			continue
		}

		if err := client.SaveProfile(&p); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to import profile %s: %v\n", p.Name, err)
			continue
		}

		stats.profiles++
	}

	// Import repositories
	for _, r := range exportData.Repos {
		exists, err := client.RepoExistsByURL(parseURL(r.URL))
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to check repo %s: %v\n", r.URL, err)
			continue
		}

		if exists && !dataImportMerge {
			_, _ = fmt.Fprintf(os.Stderr, "Skipping existing repo: %s\n", r.URL)
			continue
		}

		if err := client.SaveRepoWithWorkspace(parseURL(r.URL), r.Path, r.Workspace); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to import repo %s: %v\n", r.URL, err)
			continue
		}

		stats.repos++
	}

	// Import config if present and not merging
	if exportData.Config != nil && !dataImportMerge {
		if err := client.SaveConfig(exportData.Config); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to import config: %v\n", err)
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Imported %d profiles, %d workspaces, %d repositories\n",
		stats.profiles, stats.workspaces, stats.repos)

	return nil
}

// encryptData encrypts data using AES-256-GCM with PBKDF2 key derivation
func encryptData(data []byte, password string) ([]byte, error) {
	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, 32, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Combine: salt + nonce + ciphertext
	result := make([]byte, 0, saltSize+nonceSize+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// decryptData decrypts data using AES-256-GCM with PBKDF2 key derivation
func decryptData(data []byte, password string) ([]byte, error) {
	if len(data) < saltSize+nonceSize+16 { // 16 is minimum ciphertext with GCM tag
		return nil, fmt.Errorf("data too short")
	}

	// Extract salt, nonce, and ciphertext
	salt := data[:saltSize]
	nonce := data[saltSize : saltSize+nonceSize]
	ciphertext := data[saltSize+nonceSize:]

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, 32, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// readPassword reads a password from the terminal without echoing
func readPassword(prompt string) (string, error) {
	_, _ = fmt.Fprint(os.Stderr, prompt)

	// Check if stdin is a terminal
	fd := int(syscall.Stdin)
	if term.IsTerminal(fd) {
		password, err := term.ReadPassword(fd)
		_, _ = fmt.Fprintln(os.Stderr) // New line after password input

		if err != nil {
			return "", err
		}

		return string(password), nil
	}

	// Fallback for non-terminal (piped input)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text(), nil
	}

	return "", fmt.Errorf("failed to read password")
}

// parseURL is a helper to parse URL strings
func parseURL(urlStr string) *url.URL {
	u, _ := url.Parse(urlStr)
	return u
}
