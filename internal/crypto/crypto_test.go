package crypto

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test.key")

	encryptor := NewEncryptor(keyPath)

	// Test key generation
	err := encryptor.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Verify key file exists
	if !encryptor.KeyExists() {
		t.Fatal("Key file should exist after generation")
	}

	// Verify key file has correct permissions
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Failed to stat key file: %v", err)
	}

	expectedPerm := os.FileMode(0600)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("Expected key file permissions %v, got %v", expectedPerm, info.Mode().Perm())
	}

	// Test that generating again fails (file already exists)
	err = encryptor.GenerateKey()
	if err == nil {
		t.Error("Expected error when generating key that already exists")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test.key")
	inputPath := filepath.Join(tempDir, "input.txt")
	encryptedPath := filepath.Join(tempDir, "encrypted.age")
	decryptedPath := filepath.Join(tempDir, "decrypted.txt")

	// Generate key
	encryptor := NewEncryptor(keyPath)
	if err := encryptor.GenerateKey(); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create test file
	testContent := []byte("This is a secret message that needs to be encrypted!")
	if err := os.WriteFile(inputPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test encryption
	err := encryptor.Encrypt(inputPath, encryptedPath)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Verify encrypted file exists and is different from input
	encryptedData, err := os.ReadFile(encryptedPath)
	if err != nil {
		t.Fatalf("Failed to read encrypted file: %v", err)
	}

	if len(encryptedData) == 0 {
		t.Fatal("Encrypted file is empty")
	}

	// Encrypted content should be different from original
	if string(encryptedData) == string(testContent) {
		t.Error("Encrypted content should be different from original")
	}

	// Test decryption
	err = encryptor.Decrypt(encryptedPath, decryptedPath)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	// Verify decrypted content matches original
	decryptedData, err := os.ReadFile(decryptedPath)
	if err != nil {
		t.Fatalf("Failed to read decrypted file: %v", err)
	}

	if string(decryptedData) != string(testContent) {
		t.Errorf("Decrypted content doesn't match original.\nExpected: %s\nGot: %s",
			string(testContent), string(decryptedData))
	}
}

func TestEncryptDecryptLargeFile(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test.key")
	inputPath := filepath.Join(tempDir, "large.txt")
	encryptedPath := filepath.Join(tempDir, "large.age")
	decryptedPath := filepath.Join(tempDir, "large-decrypted.txt")

	// Generate key
	encryptor := NewEncryptor(keyPath)
	if err := encryptor.GenerateKey(); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create larger test file (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	if err := os.WriteFile(inputPath, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	// Encrypt
	if err := encryptor.Encrypt(inputPath, encryptedPath); err != nil {
		t.Fatalf("Failed to encrypt large file: %v", err)
	}

	// Decrypt
	if err := encryptor.Decrypt(encryptedPath, decryptedPath); err != nil {
		t.Fatalf("Failed to decrypt large file: %v", err)
	}

	// Verify
	decryptedData, err := os.ReadFile(decryptedPath)
	if err != nil {
		t.Fatalf("Failed to read decrypted file: %v", err)
	}

	if len(decryptedData) != len(largeContent) {
		t.Errorf("Decrypted file size mismatch. Expected %d, got %d",
			len(largeContent), len(decryptedData))
	}

	// Compare first and last 100 bytes
	for i := 0; i < 100; i++ {
		if decryptedData[i] != largeContent[i] {
			t.Errorf("Decrypted content mismatch at byte %d", i)
			break
		}
	}
}

func TestEncryptWithoutKey(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "nonexistent.key")
	inputPath := filepath.Join(tempDir, "input.txt")
	encryptedPath := filepath.Join(tempDir, "encrypted.age")

	// Create test file
	testContent := []byte("test content")
	if err := os.WriteFile(inputPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try to encrypt without generating key
	encryptor := NewEncryptor(keyPath)
	err := encryptor.Encrypt(inputPath, encryptedPath)
	if err == nil {
		t.Error("Expected error when encrypting without key")
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	tempDir := t.TempDir()
	key1Path := filepath.Join(tempDir, "key1.key")
	key2Path := filepath.Join(tempDir, "key2.key")
	inputPath := filepath.Join(tempDir, "input.txt")
	encryptedPath := filepath.Join(tempDir, "encrypted.age")
	decryptedPath := filepath.Join(tempDir, "decrypted.txt")

	// Generate first key and encrypt
	encryptor1 := NewEncryptor(key1Path)
	if err := encryptor1.GenerateKey(); err != nil {
		t.Fatalf("Failed to generate key1: %v", err)
	}

	testContent := []byte("secret data")
	if err := os.WriteFile(inputPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := encryptor1.Encrypt(inputPath, encryptedPath); err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Generate second key and try to decrypt
	encryptor2 := NewEncryptor(key2Path)
	if err := encryptor2.GenerateKey(); err != nil {
		t.Fatalf("Failed to generate key2: %v", err)
	}

	err := encryptor2.Decrypt(encryptedPath, decryptedPath)
	if err == nil {
		t.Error("Expected error when decrypting with wrong key")
	}
}

func TestKeyExists(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test.key")

	encryptor := NewEncryptor(keyPath)

	// Key should not exist initially
	if encryptor.KeyExists() {
		t.Error("Key should not exist before generation")
	}

	// Generate key
	if err := encryptor.GenerateKey(); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Key should exist now
	if !encryptor.KeyExists() {
		t.Error("Key should exist after generation")
	}
}

func TestEncryptNonexistentFile(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test.key")
	inputPath := filepath.Join(tempDir, "nonexistent.txt")
	encryptedPath := filepath.Join(tempDir, "encrypted.age")

	encryptor := NewEncryptor(keyPath)
	if err := encryptor.GenerateKey(); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	err := encryptor.Encrypt(inputPath, encryptedPath)
	if err == nil {
		t.Error("Expected error when encrypting nonexistent file")
	}
}

func TestDecryptNonexistentFile(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test.key")
	encryptedPath := filepath.Join(tempDir, "nonexistent.age")
	decryptedPath := filepath.Join(tempDir, "decrypted.txt")

	encryptor := NewEncryptor(keyPath)
	if err := encryptor.GenerateKey(); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	err := encryptor.Decrypt(encryptedPath, decryptedPath)
	if err == nil {
		t.Error("Expected error when decrypting nonexistent file")
	}
}
