package packager

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// BrewfileItem represents a single item in a Brewfile
type BrewfileItem struct {
	Type    string // "tap", "brew", "cask", "mas"
	Name    string // package name
	RawLine string // original line from Brewfile
}

// ParseBrewfile parses a Brewfile and returns individual items
func ParseBrewfile(brewfilePath string) ([]BrewfileItem, error) {
	file, err := os.Open(brewfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Brewfile: %w", err)
	}
	defer file.Close()

	var items []BrewfileItem
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Parse the line
		item := parseBrewfileLine(trimmed)
		if item != nil {
			items = append(items, *item)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read Brewfile: %w", err)
	}

	return items, nil
}

// parseBrewfileLine parses a single Brewfile line
func parseBrewfileLine(line string) *BrewfileItem {
	// Handle different Brewfile formats
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	itemType := parts[0]
	
	// Extract name (remove quotes if present)
	name := parts[1]
	name = strings.Trim(name, `"'`)
	
	// Handle comma at end
	name = strings.TrimSuffix(name, ",")

	// Only process known types
	if itemType == "tap" || itemType == "brew" || itemType == "cask" || itemType == "mas" {
		return &BrewfileItem{
			Type:    itemType,
			Name:    name,
			RawLine: line,
		}
	}

	return nil
}

// CreateFilteredBrewfile creates a new Brewfile with only selected items
func CreateFilteredBrewfile(items []BrewfileItem, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create Brewfile: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, item := range items {
		if _, err := writer.WriteString(item.RawLine + "\n"); err != nil {
			return fmt.Errorf("failed to write to Brewfile: %w", err)
		}
	}

	return nil
}

// FormatBrewfileItem creates a display label for a Brewfile item
func FormatBrewfileItem(item BrewfileItem) string {
	var icon string
	switch item.Type {
	case "tap":
		icon = "🚰"
	case "brew":
		icon = "🍺"
	case "cask":
		icon = "📦"
	case "mas":
		icon = "🏪"
	default:
		icon = "  "
	}

	return fmt.Sprintf("%s %s", icon, item.Name)
}
