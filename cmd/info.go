package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/harshpatel5940/stash/internal/backuputil"
	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/metadata"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var (
	infoDecryptKey string
	infoMessage    string
	infoYes        bool
)

var infoCmd = &cobra.Command{
	Use:   "info <backup-id|name>",
	Short: "Show backup metadata and note",
	Long: `Shows details for a backup by numeric ID (from stash list) or backup name.

Examples:
  stash info 1
  stash info backup-2026-04-06-171328.tar.gz.age
  stash info 1 -m "fresh baseline after macOS update"`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringVarP(&infoDecryptKey, "decrypt-key", "k", "", "Path to decryption key (default: ~/.stash.key)")
	infoCmd.Flags().StringVarP(&infoMessage, "message", "m", "", "Set/update backup note")
	infoCmd.Flags().BoolVarP(&infoYes, "yes", "y", false, "Confirm note update without prompt")
}

func runInfo(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	cfg.ExpandPaths()

	backup, err := resolveBackupInput(args[0], cfg.BackupDir)
	if err != nil {
		return err
	}

	if infoMessage != "" {
		trimmed := strings.TrimSpace(infoMessage)
		if !infoYes {
			confirmed, err := confirmNoteUpdate(backup.Name, trimmed)
			if err != nil {
				return err
			}
			if !confirmed {
				ui.PrintInfo("Note update cancelled")
				return nil
			}
		}

		if err := saveBackupNote(backup.Name, trimmed); err != nil {
			return err
		}
		ui.PrintSuccess("Updated note for %s", backup.Name)
	}

	keyPath := infoDecryptKey
	if keyPath == "" {
		keyPath = cfg.EncryptionKey
	}

	meta, err := backuputil.ExtractMetadata(backup.Path, keyPath)
	if err != nil {
		return fmt.Errorf("failed to read backup metadata: %w", err)
	}

	note := strings.TrimSpace(meta.Note)
	if storeNote, err := loadBackupNote(backup.Name); err == nil && storeNote != "" {
		note = storeNote
	}

	printBackupInfo(backup, meta, note)
	return nil
}

func confirmNoteUpdate(backupName, note string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Update note for %s to %q? [y/N]: ", backupName, note)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}
	normalized := strings.ToLower(strings.TrimSpace(response))
	return normalized == "y" || normalized == "yes", nil
}

func printBackupInfo(backup backupInfo, meta *metadata.Metadata, note string) {
	fmt.Printf("Name:      %s\n", backup.Name)
	if backup.Index > 0 {
		fmt.Printf("ID:        %d\n", backup.Index)
	}
	fmt.Printf("Path:      %s\n", backup.Path)
	fmt.Printf("Date:      %s\n", backupDate(meta.Timestamp, backup.ModTime).Format("2006-01-02 15:04:05"))
	fmt.Printf("Type:      %s\n", backupTypeLabel(meta.BackupType))
	fmt.Printf("Encrypted: %t\n", backup.Encrypted)
	fmt.Printf("Files:     %d\n", len(meta.Files))
	fmt.Printf("Size:      %s\n", metadata.FormatSize(backup.Size))
	if meta.BaseBackup != "" {
		fmt.Printf("Base:      %s\n", meta.BaseBackup)
	}
	if note != "" {
		fmt.Printf("Note:      %s\n", note)
	}
	if meta.Hostname != "" {
		fmt.Printf("Host:      %s\n", meta.Hostname)
	}
	if meta.Username != "" {
		fmt.Printf("User:      %s\n", meta.Username)
	}

	if len(meta.PackageCounts) == 0 {
		return
	}

	fmt.Println("Packages:")
	keys := make([]string, 0, len(meta.PackageCounts))
	for key := range meta.PackageCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("  - %s: %d\n", key, meta.PackageCounts[key])
	}
}

func backupTypeLabel(backupType string) string {
	if strings.TrimSpace(backupType) == "" {
		return "full"
	}
	return backupType
}
