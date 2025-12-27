package crypto

import (
	"fmt"
	"io"
	"os"

	"filippo.io/age"
)

// Encryptor handles file encryption and decryption using age
type Encryptor struct {
	keyPath string
}

// NewEncryptor creates a new encryptor
func NewEncryptor(keyPath string) *Encryptor {
	return &Encryptor{
		keyPath: keyPath,
	}
}

// GenerateKey generates a new age key and saves it to the key path
func (e *Encryptor) GenerateKey() error {
	// Generate new identity
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Create key file
	keyFile, err := os.OpenFile(e.keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	// Write identity to file
	if _, err := fmt.Fprintf(keyFile, "# created: %s\n", identity.Recipient()); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}
	if _, err := fmt.Fprintf(keyFile, "%s\n", identity); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// Encrypt encrypts a file using the age key
func (e *Encryptor) Encrypt(inputPath, outputPath string) error {
	// Load recipient from key file
	recipient, err := e.loadRecipient()
	if err != nil {
		return err
	}

	// Open input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Create encryptor
	w, err := age.Encrypt(outputFile, recipient)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	// Encrypt and write
	if _, err := io.Copy(w, inputFile); err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	// Close encryptor
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to finalize encryption: %w", err)
	}

	return nil
}

// Decrypt decrypts a file using the age key
func (e *Encryptor) Decrypt(inputPath, outputPath string) error {
	// Load identity from key file
	identity, err := e.loadIdentity()
	if err != nil {
		return err
	}

	// Open input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	// Create decryptor
	r, err := age.Decrypt(inputFile, identity)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Decrypt and write
	if _, err := io.Copy(outputFile, r); err != nil {
		return fmt.Errorf("failed to write decrypted content: %w", err)
	}

	return nil
}

// KeyExists checks if the encryption key exists
func (e *Encryptor) KeyExists() bool {
	_, err := os.Stat(e.keyPath)
	return err == nil
}

// loadIdentity loads the age identity from the key file
func (e *Encryptor) loadIdentity() (age.Identity, error) {
	keyFile, err := os.Open(e.keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open key file: %w", err)
	}
	defer keyFile.Close()

	identities, err := age.ParseIdentities(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse identities: %w", err)
	}

	if len(identities) == 0 {
		return nil, fmt.Errorf("no identities found in key file")
	}

	return identities[0], nil
}

// loadRecipient loads the age recipient from the key file
func (e *Encryptor) loadRecipient() (age.Recipient, error) {
	identity, err := e.loadIdentity()
	if err != nil {
		return nil, err
	}

	// Convert identity to recipient
	x25519Identity, ok := identity.(*age.X25519Identity)
	if !ok {
		return nil, fmt.Errorf("key is not an X25519 identity")
	}

	return x25519Identity.Recipient(), nil
}
