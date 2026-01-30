//go:build linux

package core

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
	"github.com/google/go-tpm/tpm2/transport/linuxtpm"
)

const (
	// Default TPM device path on Linux
	defaultTPMDevice = "/dev/tpmrm0"

	// Key size for the sealed key (32 bytes = 256 bits for AES-256)
	sealedKeySize = 32
)

var (
	// ErrTPMNotAvailable is returned when TPM device is not accessible
	ErrTPMNotAvailable = errors.New("TPM device not available")

	// ErrNoSealedKey is returned when no sealed key exists
	ErrNoSealedKey = errors.New("no sealed key found")

	// ErrSealFailed is returned when sealing operation fails
	ErrSealFailed = errors.New("failed to seal key to TPM")

	// ErrUnsealFailed is returned when unsealing operation fails
	ErrUnsealFailed = errors.New("failed to unseal key from TPM")
)

// TPMKeyManager handles TPM-based key operations
type TPMKeyManager struct {
	device string
}

// SealedData represents the data needed to unseal a key
type SealedData struct {
	// PublicArea is the public portion of the sealing key
	PublicArea []byte `json:"public_area"`
	// PrivateArea is the encrypted private portion
	PrivateArea []byte `json:"private_area"`
	// SealedBlob is the actual sealed data
	SealedBlob []byte `json:"sealed_blob"`
	// SealedBlobPublic is the public area of the sealed blob
	SealedBlobPublic []byte `json:"sealed_blob_public"`
}

// NewTPMKeyManager creates a new TPM key manager
func NewTPMKeyManager() (*TPMKeyManager, error) {
	device := os.Getenv("TPM_DEVICE")
	if device == "" {
		device = defaultTPMDevice
	}

	// Check if TPM device exists
	if _, err := os.Stat(device); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: device %s not found", ErrTPMNotAvailable, device)
	}

	return &TPMKeyManager{
		device: device,
	}, nil
}

// IsTPMAvailable checks if a TPM device is accessible
func IsTPMAvailable() bool {
	device := os.Getenv("TPM_DEVICE")
	if device == "" {
		device = defaultTPMDevice
	}

	// Check if device exists
	if _, err := os.Stat(device); os.IsNotExist(err) {
		return false
	}

	// Try to open the TPM to verify it's usable
	tpm, err := linuxtpm.Open(device)
	if err != nil {
		return false
	}

	defer func() { _ = tpm.Close() }()

	return true
}

// openTPM opens a connection to the TPM
func (t *TPMKeyManager) openTPM() (transport.TPMCloser, error) {
	tpm, err := linuxtpm.Open(t.device)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTPMNotAvailable, err)
	}

	return tpm, nil
}

// createPrimaryKey creates a primary key in the TPM for sealing operations
func (t *TPMKeyManager) createPrimaryKey(tpm transport.TPM) (*tpm2.CreatePrimaryResponse, error) {
	// Create a primary key under the storage hierarchy
	primaryTemplate := tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgRSA,
		NameAlg: tpm2.TPMAlgSHA256,
		ObjectAttributes: tpm2.TPMAObject{
			FixedTPM:            true,
			FixedParent:         true,
			SensitiveDataOrigin: true,
			UserWithAuth:        true,
			Decrypt:             true,
			Restricted:          true,
		},
		Parameters: tpm2.NewTPMUPublicParms(
			tpm2.TPMAlgRSA,
			&tpm2.TPMSRSAParms{
				Symmetric: tpm2.TPMTSymDefObject{
					Algorithm: tpm2.TPMAlgAES,
					KeyBits: tpm2.NewTPMUSymKeyBits(
						tpm2.TPMAlgAES,
						tpm2.TPMKeyBits(256),
					),
					Mode: tpm2.NewTPMUSymMode(
						tpm2.TPMAlgAES,
						tpm2.TPMAlgCFB,
					),
				},
				KeyBits: 2048,
			},
		),
	}

	createPrimary := tpm2.CreatePrimary{
		PrimaryHandle: tpm2.TPMRHOwner,
		InPublic:      tpm2.New2B(primaryTemplate),
	}

	return createPrimary.Execute(tpm)
}

// SealKey seals a key to the TPM
func (t *TPMKeyManager) SealKey(key []byte) (*SealedData, error) {
	tpm, err := t.openTPM()
	if err != nil {
		return nil, err
	}

	defer func() { _ = tpm.Close() }()

	// Create primary key for sealing
	primaryResp, err := t.createPrimaryKey(tpm)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create primary key: %v", ErrSealFailed, err)
	}

	defer func() {
		flushContext := tpm2.FlushContext{
			FlushHandle: primaryResp.ObjectHandle,
		}
		_, _ = flushContext.Execute(tpm)
	}()

	// Create a sealed object containing our key
	sealTemplate := tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgKeyedHash,
		NameAlg: tpm2.TPMAlgSHA256,
		ObjectAttributes: tpm2.TPMAObject{
			FixedTPM:     true,
			FixedParent:  true,
			UserWithAuth: true,
		},
	}

	// Create sensitive data structure with the key to seal
	inSensitive := tpm2.TPM2BSensitiveCreate{
		Sensitive: &tpm2.TPMSSensitiveCreate{
			Data: tpm2.NewTPMUSensitiveCreate(&tpm2.TPM2BSensitiveData{
				Buffer: key,
			}),
		},
	}

	create := tpm2.Create{
		ParentHandle: tpm2.AuthHandle{
			Handle: primaryResp.ObjectHandle,
			Name:   primaryResp.Name,
			Auth:   tpm2.PasswordAuth(nil),
		},
		InSensitive: inSensitive,
		InPublic:    tpm2.New2B(sealTemplate),
	}

	createResp, err := create.Execute(tpm)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create sealed object: %v", ErrSealFailed, err)
	}

	// Marshal the public area of the primary key
	pubBytes := tpm2.Marshal(primaryResp.OutPublic)

	// Get the private and public areas of the sealed object
	privBytes := tpm2.Marshal(createResp.OutPrivate)
	sealedPubBytes := tpm2.Marshal(createResp.OutPublic)

	return &SealedData{
		PublicArea:       pubBytes,
		PrivateArea:      privBytes,
		SealedBlob:       privBytes,
		SealedBlobPublic: sealedPubBytes,
	}, nil
}

// UnsealKey unseals a key from the TPM
func (t *TPMKeyManager) UnsealKey(data *SealedData) ([]byte, error) {
	tpm, err := t.openTPM()
	if err != nil {
		return nil, err
	}

	defer func() { _ = tpm.Close() }()

	// Recreate the primary key (deterministic, so same key as before)
	primaryResp, err := t.createPrimaryKey(tpm)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create primary key: %v", ErrUnsealFailed, err)
	}

	defer func() {
		flushContext := tpm2.FlushContext{
			FlushHandle: primaryResp.ObjectHandle,
		}
		_, _ = flushContext.Execute(tpm)
	}()

	// Unmarshal the sealed object using the new API
	outPrivate, err := tpm2.Unmarshal[tpm2.TPM2BPrivate](data.PrivateArea)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal private area: %v", ErrUnsealFailed, err)
	}

	outPublic, err := tpm2.Unmarshal[tpm2.TPM2BPublic](data.SealedBlobPublic)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal public area: %v", ErrUnsealFailed, err)
	}

	// Load the sealed object
	load := tpm2.Load{
		ParentHandle: tpm2.AuthHandle{
			Handle: primaryResp.ObjectHandle,
			Name:   primaryResp.Name,
			Auth:   tpm2.PasswordAuth(nil),
		},
		InPrivate: *outPrivate,
		InPublic:  *outPublic,
	}

	loadResp, err := load.Execute(tpm)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to load sealed object: %v", ErrUnsealFailed, err)
	}

	defer func() {
		flushContext := tpm2.FlushContext{
			FlushHandle: loadResp.ObjectHandle,
		}
		_, _ = flushContext.Execute(tpm)
	}()

	// Unseal the data
	unseal := tpm2.Unseal{
		ItemHandle: tpm2.AuthHandle{
			Handle: loadResp.ObjectHandle,
			Name:   loadResp.Name,
			Auth:   tpm2.PasswordAuth(nil),
		},
	}

	unsealResp, err := unseal.Execute(tpm)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to unseal data: %v", ErrUnsealFailed, err)
	}

	return unsealResp.OutData.Buffer, nil
}

// GenerateAndSealKey generates a random key and seals it to the TPM
func (t *TPMKeyManager) GenerateAndSealKey() (*SealedData, error) {
	// Generate a random 32-byte key
	key := make([]byte, sealedKeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	return t.SealKey(key)
}
