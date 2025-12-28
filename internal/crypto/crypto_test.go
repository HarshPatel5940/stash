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

	err := encryptor.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	if !encryptor.KeyExists() {
		t.Fatal("Key file should exist after generation")
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Failed to stat key file: %v", err)
	}

	expectedPerm := os.FileMode(0600)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("Expected key file permissions %v, got %v", expectedPerm, info.Mode().Perm())
	}

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

	encryptor := NewEncryptor(keyPath)
	if err := encryptor.GenerateKey(); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	testContent := []byte("This is a secret message that needs to be encrypted!")
	if err := os.WriteFile(inputPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := encryptor.Encrypt(inputPath, encryptedPath)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	encryptedData, err := os.ReadFile(encryptedPath)
	if err != nil {
		t.Fatalf("Failed to read encrypted file: %v", err)
	}

	if len(encryptedData) == 0 {
		t.Fatal("Encrypted file is empty")
	}

	if string(encryptedData) == string(testContent) {
		t.Error("Encrypted content should be different from original")
	}

	err = encryptor.Decrypt(encryptedPath, decryptedPath)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

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

	encryptor := NewEncryptor(keyPath)
	if err := encryptor.GenerateKey(); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	if err := os.WriteFile(inputPath, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	if err := encryptor.Encrypt(inputPath, encryptedPath); err != nil {
		t.Fatalf("Failed to encrypt large file: %v", err)
	}

	if err := encryptor.Decrypt(encryptedPath, decryptedPath); err != nil {
		t.Fatalf("Failed to decrypt large file: %v", err)
	}

	decryptedData, err := os.ReadFile(decryptedPath)
	if err != nil {
		t.Fatalf("Failed to read decrypted file: %v", err)
	}

	if len(decryptedData) != len(largeContent) {
		t.Errorf("Decrypted file size mismatch. Expected %d, got %d",
			len(largeContent), len(decryptedData))
	}

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

	testContent := []byte("test content")
	if err := os.WriteFile(inputPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

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

	if encryptor.KeyExists() {
		t.Error("Key should not exist before generation")
	}

	if err := encryptor.GenerateKey(); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

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
