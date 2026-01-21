// Package recovery provides partial backup recovery functionality.
// When a backup fails midway, this package saves the progress state and
// allows users to resume or recover what was successfully backed up.
//
// Recovery states are persisted to disk as JSON files in a .recovery directory.
package recovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harshpatel5940/stash/internal/metadata"
)

// RecoveryState represents the state of a partial backup
type RecoveryState struct {
	BackupPath     string             `json:"backup_path"`
	Timestamp      time.Time          `json:"timestamp"`
	CompletedTasks []string           `json:"completed_tasks"`
	FailedTask     string             `json:"failed_task"`
	ErrorMessage   string             `json:"error_message"`
	Metadata       *metadata.Metadata `json:"metadata"`
	CanResume      bool               `json:"can_resume"`
}

// Manager handles backup recovery operations
type Manager struct {
	recoveryDir string
}

// NewManager creates a new recovery manager
func NewManager(backupDir string) *Manager {
	recoveryDir := filepath.Join(backupDir, ".recovery")
	os.MkdirAll(recoveryDir, 0755)

	return &Manager{
		recoveryDir: recoveryDir,
	}
}

// SaveState saves the current recovery state
func (m *Manager) SaveState(state *RecoveryState) error {
	stateFile := m.getStateFile(state.BackupPath)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal recovery state: %w", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to save recovery state: %w", err)
	}

	return nil
}

// LoadState loads a recovery state
func (m *Manager) LoadState(backupPath string) (*RecoveryState, error) {
	stateFile := m.getStateFile(backupPath)

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No recovery state exists
		}
		return nil, fmt.Errorf("failed to read recovery state: %w", err)
	}

	var state RecoveryState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal recovery state: %w", err)
	}

	return &state, nil
}

// DeleteState removes a recovery state
func (m *Manager) DeleteState(backupPath string) error {
	stateFile := m.getStateFile(backupPath)
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete recovery state: %w", err)
	}
	return nil
}

// ListRecoverableBackups lists all backups that can be recovered
func (m *Manager) ListRecoverableBackups() ([]RecoveryState, error) {
	files, err := os.ReadDir(m.recoveryDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read recovery directory: %w", err)
	}

	var states []RecoveryState
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".recovery.json") {
			continue
		}

		stateFile := filepath.Join(m.recoveryDir, file.Name())
		data, err := os.ReadFile(stateFile)
		if err != nil {
			continue
		}

		var state RecoveryState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		states = append(states, state)
	}

	return states, nil
}

// MarkTaskComplete marks a task as completed in the recovery state
func (m *Manager) MarkTaskComplete(backupPath, taskName string) error {
	state, err := m.LoadState(backupPath)
	if err != nil {
		return err
	}

	if state == nil {
		// Initialize new state
		state = &RecoveryState{
			BackupPath:     backupPath,
			Timestamp:      time.Now(),
			CompletedTasks: []string{},
			CanResume:      true,
		}
	}

	// Add task to completed list if not already there
	found := false
	for _, task := range state.CompletedTasks {
		if task == taskName {
			found = true
			break
		}
	}

	if !found {
		state.CompletedTasks = append(state.CompletedTasks, taskName)
	}

	return m.SaveState(state)
}

// MarkTaskFailed marks a task as failed
func (m *Manager) MarkTaskFailed(backupPath, taskName, errorMsg string) error {
	state, err := m.LoadState(backupPath)
	if err != nil {
		return err
	}

	if state == nil {
		state = &RecoveryState{
			BackupPath:     backupPath,
			Timestamp:      time.Now(),
			CompletedTasks: []string{},
		}
	}

	state.FailedTask = taskName
	state.ErrorMessage = errorMsg
	state.CanResume = isRecoverableTask(taskName)

	return m.SaveState(state)
}

// IsTaskComplete checks if a task has been completed
func (m *Manager) IsTaskComplete(backupPath, taskName string) (bool, error) {
	state, err := m.LoadState(backupPath)
	if err != nil {
		return false, err
	}

	if state == nil {
		return false, nil
	}

	for _, task := range state.CompletedTasks {
		if task == taskName {
			return true, nil
		}
	}

	return false, nil
}

// GetRemainingTasks returns tasks that haven't been completed
func (m *Manager) GetRemainingTasks(backupPath string, allTasks []string) ([]string, error) {
	state, err := m.LoadState(backupPath)
	if err != nil {
		return nil, err
	}

	if state == nil {
		return allTasks, nil
	}

	remaining := []string{}
	for _, task := range allTasks {
		completed := false
		for _, completedTask := range state.CompletedTasks {
			if task == completedTask {
				completed = true
				break
			}
		}
		if !completed {
			remaining = append(remaining, task)
		}
	}

	return remaining, nil
}

// SavePartialBackup saves a partial backup that can be resumed
func (m *Manager) SavePartialBackup(backupPath string, meta *metadata.Metadata, errorMsg string) (string, error) {
	// Create partial backup name
	timestamp := time.Now().Format("20060102-150405")
	dir := filepath.Dir(backupPath)
	base := filepath.Base(backupPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	partialName := fmt.Sprintf("%s-PARTIAL-%s%s", nameWithoutExt, timestamp, ext)
	partialPath := filepath.Join(dir, partialName)

	// Copy existing partial backup if it exists
	if _, err := os.Stat(backupPath); err == nil {
		os.Rename(backupPath, partialPath)
	}

	// Save recovery state
	recoveryState := &RecoveryState{
		BackupPath:   partialPath,
		Timestamp:    time.Now(),
		ErrorMessage: errorMsg,
		Metadata:     meta,
		CanResume:    true,
	}

	if err := m.SaveState(recoveryState); err != nil {
		return "", err
	}

	return partialPath, nil
}

// CleanupOldRecoveryStates removes recovery states older than a certain age
func (m *Manager) CleanupOldRecoveryStates(maxAge time.Duration) error {
	files, err := os.ReadDir(m.recoveryDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read recovery directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".recovery.json") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			stateFile := filepath.Join(m.recoveryDir, file.Name())
			os.Remove(stateFile)
		}
	}

	return nil
}

// getStateFile returns the path to the recovery state file
func (m *Manager) getStateFile(backupPath string) string {
	base := filepath.Base(backupPath)
	stateFile := strings.TrimSuffix(base, filepath.Ext(base)) + ".recovery.json"
	return filepath.Join(m.recoveryDir, stateFile)
}

// isRecoverableTask determines if a task failure is recoverable
func isRecoverableTask(taskName string) bool {
	// Most tasks are recoverable except critical ones like encryption
	unrecoverableTasks := []string{
		"encrypt",
		"finalize",
		"save_metadata",
	}

	for _, unrecoverable := range unrecoverableTasks {
		if strings.Contains(strings.ToLower(taskName), unrecoverable) {
			return false
		}
	}

	return true
}
