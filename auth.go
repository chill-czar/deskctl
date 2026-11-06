package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	sudoAllowlistPath = filepath.Join(deskctlDir, "sudo-allowlist.txt")
	sudoersDropInPath = "/etc/sudoers.d/deskctl"
	polkitDropInPath  = "/etc/polkit-1/rules.d/49-deskctl.rules"
)

type AuthPromptInfo struct {
	Detected   bool   `json:"detected"`
	PromptType string `json:"prompt_type,omitempty"` // polkit, keyring, sudo_terminal, atspi_password, lockscreen
	Program    string `json:"program,omitempty"`
	Title      string `json:"title,omitempty"`
	WindowID   string `json:"window_id,omitempty"`
	Details    string `json:"details,omitempty"`
}

// checkAuthPrompts scans X11 windows, terminal windows, and AT-SPI for active password/auth prompts.
// Enforces the core rule: Never bypass, disable, brute-force, read, store, or auto-type passwords.
func checkAuthPrompts() (*AuthPromptInfo, error) {
	// 1. Scan X11 window titles and classes
	wins, err := listWindows()
	if err == nil {
		for _, w := range wins {
			cLower := strings.ToLower(w.Class)
			tLower := strings.ToLower(w.Title)

			// Polkit dialogs
			if strings.Contains(cLower, "polkit") || strings.Contains(tLower, "authenticate") || strings.Contains(tLower, "authentication required") {
				return &AuthPromptInfo{
					Detected:   true,
					PromptType: "polkit",
					Program:    w.Class,
					Title:      w.Title,
					WindowID:   w.ID,
					Details:    "PolicyKit authentication agent is requesting administrative credentials",
				}, nil
			}

			// Keyring unlock dialogs
			if strings.Contains(cLower, "gcr-prompter") || strings.Contains(cLower, "keyring") || strings.Contains(tLower, "unlock keyring") {
				return &AuthPromptInfo{
					Detected:   true,
					PromptType: "keyring",
					Program:    w.Class,
					Title:      w.Title,
					WindowID:   w.ID,
					Details:    "Keyring prompter is requesting unlock password",
				}, nil
			}

			// Screen lock / screensaver
			if strings.Contains(cLower, "screensaver") || strings.Contains(cLower, "lock") || strings.Contains(tLower, "lock screen") {
				return &AuthPromptInfo{
					Detected:   true,
					PromptType: "lockscreen",
					Program:    w.Class,
					Title:      w.Title,
					WindowID:   w.ID,
					Details:    "Desktop session is locked",
				}, nil
			}
		}
	}

	// 2. Scan active terminal window for sudo prompts
	activeWin := getActiveWindowInfo()
	if activeWin != nil {
		cLower := strings.ToLower(activeWin.Class)
		if strings.Contains(cLower, "terminal") || strings.Contains(cLower, "term") || strings.Contains(cLower, "xterm") {
			if strings.Contains(strings.ToLower(activeWin.Title), "sudo") || strings.Contains(strings.ToLower(activeWin.Title), "su:") {
				return &AuthPromptInfo{
					Detected:   true,
					PromptType: "sudo_terminal",
					Program:    activeWin.Class,
					Title:      activeWin.Title,
					WindowID:   activeWin.ID,
					Details:    "Terminal prompt indicates sudo privilege escalation",
				}, nil
			}
		}
	}

	// 3. Scan AT-SPI for visible password entry fields
	helperScript := getHelperPath()
	out, err := exec.Command("python3", helperScript, "auth-check").Output()
	if err == nil {
		var atspiRes struct {
			OK      bool             `json:"ok"`
			Prompts []map[string]any `json:"prompts"`
		}
		if err := json.Unmarshal(out, &atspiRes); err == nil && atspiRes.OK && len(atspiRes.Prompts) > 0 {
			p := atspiRes.Prompts[0]
			return &AuthPromptInfo{
				Detected:   true,
				PromptType: "atspi_password",
				Program:    fmt.Sprintf("%v", p["app"]),
				Title:      fmt.Sprintf("%v", p["name"]),
				Details:    fmt.Sprintf("AT-SPI detected active password text field (role: %v)", p["role"]),
			}, nil
		}
	}

	return &AuthPromptInfo{
		Detected: false,
	}, nil
}

func doAuthCheck() {
	info, err := checkAuthPrompts()
	if err != nil {
		outputError(fmt.Sprintf("Auth check failed: %v", err))
	}
	outputJSON(map[string]interface{}{
		"ok":     true,
		"prompt": info,
	})
}

func doAuthWait(timeoutSec int) {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}

	info, err := checkAuthPrompts()
	if err != nil {
		outputError(fmt.Sprintf("Failed to check auth prompts: %v", err))
	}

	if !info.Detected {
		outputJSON(map[string]interface{}{
			"ok":       true,
			"resolved": true,
			"message":  "No active authentication prompts detected",
		})
		return
	}

	// Notify user immediately
	msg := fmt.Sprintf("Authentication Required: %s is requesting credentials. Please enter your password directly on the screen.", info.Program)
	sendNotification("deskctl - Authentication Handoff", msg)

	start := time.Now()
	for time.Since(start) < time.Duration(timeoutSec)*time.Second {
		time.Sleep(500 * time.Millisecond)
		check, err := checkAuthPrompts()
		if err == nil && !check.Detected {
			// Prompt closed
			outputJSON(map[string]interface{}{
				"ok":          true,
				"resolved":    true,
				"elapsed_sec": int(time.Since(start).Seconds()),
				"message":     "Authentication prompt was resolved by user",
			})
			return
		}
	}

	outputJSON(map[string]interface{}{
		"ok":          false,
		"resolved":    false,
		"elapsed_sec": int(time.Since(start).Seconds()),
		"error":       fmt.Sprintf("Timed out waiting for user to resolve authentication prompt after %d seconds", timeoutSec),
	})
}

func initSudoAllowlist() error {
	if _, err := os.Stat(sudoAllowlistPath); err == nil {
		return nil
	}
	defaultContent := `# Deskctl Scoped Sudo Allowlist
# Only exact binaries and safe maintenance arguments are allowed.
# NOPASSWD: ALL is strictly prohibited.
/usr/bin/apt update
/usr/bin/apt-get update
/usr/bin/apt install *
/usr/bin/apt-get install *
/usr/bin/apt remove *
/usr/bin/apt-get remove *
/bin/systemctl status *
/usr/bin/systemctl status *
/bin/systemctl restart [a-zA-Z0-9_-]+
/usr/bin/systemctl restart [a-zA-Z0-9_-]+
`
	return os.WriteFile(sudoAllowlistPath, []byte(defaultContent), 0644)
}

func loadSudoAllowlist() ([]string, error) {
	if err := initSudoAllowlist(); err != nil {
		return nil, err
	}
	file, err := os.Open(sudoAllowlistPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cmds []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strict security validation: Never allow NOPASSWD: ALL or dangerous commands
		lineUpper := strings.ToUpper(line)
		if lineUpper == "ALL" || strings.Contains(lineUpper, "ALL=(ALL)") {
			return nil, fmt.Errorf("dangerous allowlist entry detected: 'ALL' is forbidden")
		}
		dangerous := []string{"/bin/sh", "/bin/bash", "/usr/bin/sudo", "/bin/su", "/usr/bin/passwd", "visudo"}
		for _, d := range dangerous {
			if strings.HasPrefix(line, d) {
				return nil, fmt.Errorf("dangerous command %q is strictly forbidden in allowlist", d)
			}
		}
		cmds = append(cmds, line)
	}
	return cmds, nil
}

func doSetupSudo(confirm, dryRun, revoke bool) {
	logAction("setup-sudo", []string{fmt.Sprintf("confirm=%v", confirm), fmt.Sprintf("revoke=%v", revoke)}, true, dryRun, "")

	if revoke {
		if dryRun {
			outputJSON(map[string]interface{}{
				"ok":      true,
				"dry_run": true,
				"action":  "revoke",
				"target":  sudoersDropInPath,
				"message": "Would remove " + sudoersDropInPath + " via pkexec",
			})
			return
		}
		cmd := exec.Command("pkexec", "rm", "-f", sudoersDropInPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			outputError(fmt.Sprintf("Failed to revoke %s: %s (%v)", sudoersDropInPath, strings.TrimSpace(string(out)), err))
		}
		outputJSON(map[string]interface{}{
			"ok":      true,
			"action":  "revoke",
			"target":  sudoersDropInPath,
			"message": "Successfully revoked and removed " + sudoersDropInPath,
		})
		return
	}

	allowedCmds, err := loadSudoAllowlist()
	if err != nil {
		outputError(fmt.Sprintf("Failed to load allowlist: %v", err))
	}

	user := os.Getenv("USER")
	if user == "" {
		user = "jimmy"
	}

	// Generate scoped sudoers drop-in
	var b strings.Builder
	b.WriteString("# ====================================================================\n")
	b.WriteString("# Deskctl Scoped Privilege Configuration\n")
	b.WriteString(fmt.Sprintf("# Generated: %s for user: %s\n", time.Now().Format(time.RFC3339), user))
	b.WriteString("# Safety constraint: Full unrestricted root bypass is strictly prohibited.\n")
	b.WriteString("# Only explicit system maintenance commands from ~/.deskctl/sudo-allowlist.txt are granted.\n")
	b.WriteString("# ====================================================================\n\n")

	joinedCmds := strings.Join(allowedCmds, ", \\\n    ")
	b.WriteString(fmt.Sprintf("%s ALL=(ALL) NOPASSWD: \\\n    %s\n", user, joinedCmds))

	sudoersContent := b.String()

	// Validate syntax via visudo -c -f <tempfile>
	tempFile := filepath.Join(os.TempDir(), "deskctl_sudoers_preview")
	if err := os.WriteFile(tempFile, []byte(sudoersContent), 0644); err != nil {
		outputError(fmt.Sprintf("Failed to write temporary sudoers file: %v", err))
	}
	defer os.Remove(tempFile)

	valOut, valErr := exec.Command("visudo", "-c", "-f", tempFile).CombinedOutput()
	if valErr != nil {
		outputError(fmt.Sprintf("visudo validation failed for proposed configuration: %s (%v)", strings.TrimSpace(string(valOut)), valErr))
	}

	explanation := []string{
		"Each allowed line corresponds strictly to entries in " + sudoAllowlistPath,
		"Allows 'apt update', 'apt install', and 'apt remove' non-interactively for package installation",
		"Allows 'systemctl status' and 'systemctl restart' for managing system services",
		"NOPASSWD: ALL is explicitly forbidden and not included",
		"Validated with visudo -c prior to deployment",
	}

	if dryRun {
		outputJSON(map[string]interface{}{
			"ok":                   true,
			"dry_run":              true,
			"allowlist_path":       sudoAllowlistPath,
			"target_path":          sudoersDropInPath,
			"visudo_validation":    strings.TrimSpace(string(valOut)),
			"proposed_content":     sudoersContent,
			"line_by_line_explain": explanation,
		})
		return
	}

	if !confirm {
		outputJSON(map[string]interface{}{
			"ok":                    true,
			"confirmation_required": true,
			"message":               "Configuration validated with visudo. Review proposed content below. Re-run with '--confirm' to install via pkexec.",
			"allowlist_path":        sudoAllowlistPath,
			"target_path":           sudoersDropInPath,
			"visudo_validation":     strings.TrimSpace(string(valOut)),
			"proposed_content":      sudoersContent,
			"line_by_line_explain":  explanation,
		})
		return
	}

	// Install via pkexec
	installCmd := fmt.Sprintf("cp %s %s && chmod 0440 %s", tempFile, sudoersDropInPath, sudoersDropInPath)
	cmd := exec.Command("pkexec", "bash", "-c", installCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outputError(fmt.Sprintf("Failed to install %s: %s (%v)", sudoersDropInPath, strings.TrimSpace(string(out)), err))
	}

	outputJSON(map[string]interface{}{
		"ok":                true,
		"action":            "install",
		"target_path":       sudoersDropInPath,
		"visudo_validation": strings.TrimSpace(string(valOut)),
		"message":           "Successfully installed scoped sudoers configuration to " + sudoersDropInPath,
	})
}

func doSetupPolkit(confirm, dryRun, revoke bool) {
	logAction("setup-polkit", []string{fmt.Sprintf("confirm=%v", confirm), fmt.Sprintf("revoke=%v", revoke)}, true, dryRun, "")

	if revoke {
		if dryRun {
			outputJSON(map[string]interface{}{
				"ok":      true,
				"dry_run": true,
				"action":  "revoke",
				"target":  polkitDropInPath,
				"message": "Would remove " + polkitDropInPath + " via pkexec",
			})
			return
		}
		cmd := exec.Command("pkexec", "rm", "-f", polkitDropInPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			outputError(fmt.Sprintf("Failed to revoke %s: %s (%v)", polkitDropInPath, strings.TrimSpace(string(out)), err))
		}
		outputJSON(map[string]interface{}{
			"ok":      true,
			"action":  "revoke",
			"target":  polkitDropInPath,
			"message": "Successfully revoked and removed " + polkitDropInPath,
		})
		return
	}

	user := os.Getenv("USER")
	if user == "" {
		user = "jimmy"
	}

	ruleContent := fmt.Sprintf(`// Deskctl Scoped PolicyKit Configuration
// Grants specific non-interactive permissions for package operations without full bypass.
polkit.addRule(function(action, subject) {
    if (subject.user == "%s") {
        if (action.id == "org.debian.apt.update-cache" ||
            action.id == "org.debian.apt.install-or-remove-packages" ||
            action.id == "org.freedesktop.packagekit.system-sources-refresh" ||
            action.id == "org.freedesktop.packagekit.package-install") {
            return polkit.Result.YES;
        }
    }
});
`, user)

	explanation := []string{
		"Allows package cache refresh and package installations non-interactively for user " + user,
		"Restricted strictly to APT and PackageKit actions; no other policykit actions are authorized",
		"Does not grant broad administrator rights",
	}

	if dryRun {
		outputJSON(map[string]interface{}{
			"ok":                   true,
			"dry_run":              true,
			"target_path":          polkitDropInPath,
			"proposed_content":     ruleContent,
			"line_by_line_explain": explanation,
		})
		return
	}

	if !confirm {
		outputJSON(map[string]interface{}{
			"ok":                    true,
			"confirmation_required": true,
			"message":               "Polkit rule prepared. Review proposed content below. Re-run with '--confirm' to install via pkexec.",
			"target_path":           polkitDropInPath,
			"proposed_content":      ruleContent,
			"line_by_line_explain":  explanation,
		})
		return
	}

	tempFile := filepath.Join(os.TempDir(), "deskctl_polkit_preview")
	if err := os.WriteFile(tempFile, []byte(ruleContent), 0644); err != nil {
		outputError(fmt.Sprintf("Failed to write temporary polkit file: %v", err))
	}
	defer os.Remove(tempFile)

	installCmd := fmt.Sprintf("cp %s %s && chmod 0644 %s", tempFile, polkitDropInPath, polkitDropInPath)
	cmd := exec.Command("pkexec", "bash", "-c", installCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outputError(fmt.Sprintf("Failed to install %s: %s (%v)", polkitDropInPath, strings.TrimSpace(string(out)), err))
	}

	outputJSON(map[string]interface{}{
		"ok":          true,
		"action":      "install",
		"target_path": polkitDropInPath,
		"message":     "Successfully installed scoped PolicyKit rules to " + polkitDropInPath,
	})
}
