package crypto

import (
	"fmt"
	"io"
	"os"

	"filippo.io/age"
)

type Encryptor struct {
	keyPath string
}

func NewEncryptor(keyPath string) *Encryptor {
	return &Encryptor{
		keyPath: keyPath,
	}
}

func (e *Encryptor) GenerateKey() error {

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	keyFile, err := os.OpenFile(e.keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	if _, err := fmt.Fprintf(keyFile, "# created: %s\n", identity.Recipient()); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}
	if _, err := fmt.Fprintf(keyFile, "%s\n", identity); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

func (e *Encryptor) Encrypt(inputPath, outputPath string) error {

	recipient, err := e.loadRecipient()
	if err != nil {
		return err
	}

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	w, err := age.Encrypt(outputFile, recipient)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	if _, err := io.Copy(w, inputFile); err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to finalize encryption: %w", err)
	}

	return nil
}

func (e *Encryptor) Decrypt(inputPath, outputPath string) error {

	identity, err := e.loadIdentity()
	if err != nil {
		return err
	}

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	r, err := age.Decrypt(inputFile, identity)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	if _, err := io.Copy(outputFile, r); err != nil {
		return fmt.Errorf("failed to write decrypted content: %w", err)
	}

	return nil
}

func (e *Encryptor) KeyExists() bool {
	_, err := os.Stat(e.keyPath)
	return err == nil
}

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

func (e *Encryptor) loadRecipient() (age.Recipient, error) {
	identity, err := e.loadIdentity()
	if err != nil {
		return nil, err
	}

	x25519Identity, ok := identity.(*age.X25519Identity)
	if !ok {
		return nil, fmt.Errorf("key is not an X25519 identity")
	}

	return x25519Identity.Recipient(), nil
}
