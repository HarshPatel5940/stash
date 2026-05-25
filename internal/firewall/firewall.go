// Package firewall snapshots and restores the macOS Application Firewall via
// /usr/libexec/ApplicationFirewall/socketfilterfw. Read-only queries do not
// need root; setters do, so restore prompts for sudo at apply time.
package firewall

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const sfwBin = "/usr/libexec/ApplicationFirewall/socketfilterfw"

type State struct {
	GlobalEnabled bool     `json:"global_enabled"`
	StealthMode   bool     `json:"stealth_mode"`
	BlockAll      bool     `json:"block_all"`
	AllowSigned   bool     `json:"allow_signed"`
	AllowSignedApp bool    `json:"allow_signed_app"`
	LoggingMode   bool     `json:"logging_mode"`
	Apps          []AppRule `json:"apps"`
}

type AppRule struct {
	Path    string `json:"path"`
	Allowed bool   `json:"allowed"`
}

type Manager struct {
	outputFile string
}

func NewManager(outputFile string) *Manager {
	return &Manager{outputFile: outputFile}
}

func Available() bool {
	_, err := os.Stat(sfwBin)
	return err == nil
}

func (m *Manager) BackupAll() error {
	if !Available() {
		return fmt.Errorf("socketfilterfw not present (non-macOS?)")
	}

	state := State{
		GlobalEnabled:  queryBool("--getglobalstate", "enabled"),
		StealthMode:    queryBool("--getstealthmode", "enabled"),
		BlockAll:       queryBool("--getblockall", "enabled"),
		AllowSigned:    queryBool("--getallowsigned", "enabled"),
		AllowSignedApp: queryBool("--getallowsignedapp", "enabled"),
		LoggingMode:    queryBool("--getloggingmode", "on"),
		Apps:           listApps(),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(parentDir(m.outputFile), 0755); err != nil {
		return err
	}
	return os.WriteFile(m.outputFile, data, 0644)
}

func (m *Manager) Restore(backupFile string) error {
	if !Available() {
		return fmt.Errorf("socketfilterfw not present (non-macOS?)")
	}

	data, err := os.ReadFile(backupFile)
	if err != nil {
		return err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	fmt.Println("  Restoring firewall state (sudo may prompt)...")
	runSudo(sfwBin, onOff("--setglobalstate", state.GlobalEnabled))
	runSudo(sfwBin, onOff("--setstealthmode", state.StealthMode))
	runSudo(sfwBin, onOff("--setblockall", state.BlockAll))
	runSudo(sfwBin, onOff("--setallowsigned", state.AllowSigned))
	runSudo(sfwBin, onOff("--setallowsignedapp", state.AllowSignedApp))
	runSudo(sfwBin, onOffLog("--setloggingmode", state.LoggingMode))

	for _, app := range state.Apps {
		if _, err := os.Stat(app.Path); err != nil {
			continue // App not installed yet; skip silently.
		}
		runSudo(sfwBin, "--add", app.Path)
		if !app.Allowed {
			runSudo(sfwBin, "--block", app.Path)
		} else {
			runSudo(sfwBin, "--unblockapp", app.Path)
		}
	}
	return nil
}

func queryBool(flag, truthy string) bool {
	out, err := exec.Command(sfwBin, flag).Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), truthy)
}

func listApps() []AppRule {
	out, err := exec.Command(sfwBin, "--listapps").Output()
	if err != nil {
		return nil
	}
	var apps []AppRule
	var current AppRule
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		// Lines look like:
		//   1 :  /Applications/Foo.app
		//        ( Allow incoming connections )
		if strings.HasPrefix(trimmed, "(") {
			current.Allowed = strings.Contains(strings.ToLower(trimmed), "allow")
			if current.Path != "" {
				apps = append(apps, current)
				current = AppRule{}
			}
			continue
		}
		if idx := strings.Index(trimmed, ":"); idx != -1 && strings.HasPrefix(trimmed, "" /* number prefix */) {
			path := strings.TrimSpace(trimmed[idx+1:])
			if strings.HasPrefix(path, "/") {
				current.Path = path
			}
		}
	}
	return apps
}

func onOff(flag string, enabled bool) []string {
	if enabled {
		return []string{flag, "on"}
	}
	return []string{flag, "off"}
}

func onOffLog(flag string, enabled bool) []string {
	if enabled {
		return []string{flag, "on"}
	}
	return []string{flag, "off"}
}

func runSudo(bin string, rest ...any) {
	var args []string
	args = append(args, bin)
	for _, r := range rest {
		switch v := r.(type) {
		case string:
			args = append(args, v)
		case []string:
			args = append(args, v...)
		}
	}
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func parentDir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}
