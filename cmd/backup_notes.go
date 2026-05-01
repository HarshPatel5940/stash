package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type backupNotesStore struct {
	Notes map[string]string `json:"notes"`
}

func notesFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".stash-notes.json"), nil
}

func normalizeBackupKey(name string) string {
	n := strings.TrimSpace(name)
	n = filepath.Base(n)
	n = strings.TrimSuffix(n, ".age")
	n = strings.TrimSuffix(n, ".tar.gz")
	return n
}

func loadBackupNotes() (*backupNotesStore, error) {
	path, err := notesFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &backupNotesStore{Notes: map[string]string{}}, nil
		}
		return nil, fmt.Errorf("failed to read notes store: %w", err)
	}

	var store backupNotesStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse notes store: %w", err)
	}
	if store.Notes == nil {
		store.Notes = map[string]string{}
	}
	return &store, nil
}

func saveBackupNote(backupName, note string) error {
	store, err := loadBackupNotes()
	if err != nil {
		return err
	}

	key := normalizeBackupKey(backupName)
	trimmed := strings.TrimSpace(note)
	if trimmed == "" {
		delete(store.Notes, key)
	} else {
		store.Notes[key] = trimmed
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal notes store: %w", err)
	}

	path, err := notesFilePath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to save notes store: %w", err)
	}
	return nil
}

func loadBackupNote(backupName string) (string, error) {
	store, err := loadBackupNotes()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(store.Notes[normalizeBackupKey(backupName)]), nil
}
