package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CheckResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Details string `json:"details"`
}

func doDoctor() {
	var checks []CheckResult
	healthy := true

	// 1. Session Type & Display
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	display := os.Getenv("DISPLAY")
	if sessionType == "" {
		sessionType = "unknown"
	}
	if sessionType == "x11" || (sessionType != "wayland" && display != "") {
		checks = append(checks, CheckResult{
			Name:    "display_server",
			Passed:  true,
			Details: fmt.Sprintf("Session: %s, DISPLAY: %s", sessionType, display),
		})
	} else {
		healthy = false
		checks = append(checks, CheckResult{
			Name:    "display_server",
			Passed:  false,
			Details: fmt.Sprintf("Session: %s (Wayland detected; switch to X11 on login screen)", sessionType),
		})
	}

	// 2. Monitors Detection
	screens, _, err := detectScreens()
	if err == nil && len(screens) > 0 {
		var screenDesc []string
		for _, s := range screens {
			screenDesc = append(screenDesc, fmt.Sprintf("%s (%dx%d+%d+%d, primary: %v)", s.Name, s.W, s.H, s.X, s.Y, s.Primary))
		}
		checks = append(checks, CheckResult{
			Name:    "monitors",
			Passed:  true,
			Details: fmt.Sprintf("%d display(s) detected: %s", len(screens), strings.Join(screenDesc, ", ")),
		})
	} else {
		healthy = false
		checks = append(checks, CheckResult{
			Name:    "monitors",
			Passed:  false,
			Details: fmt.Sprintf("Failed to query monitors: %v", err),
		})
	}

	// 3. CLI Tools
	requiredTools := []string{"xdotool", "maim", "wmctrl", "xclip", "xrandr", "ffmpeg", "tesseract"}
	for _, tool := range requiredTools {
		path, err := exec.LookPath(tool)
		if err == nil {
			checks = append(checks, CheckResult{
				Name:    "tool_" + tool,
				Passed:  true,
				Details: fmt.Sprintf("Found at %s", path),
			})
		} else {
			healthy = false
			checks = append(checks, CheckResult{
				Name:    "tool_" + tool,
				Passed:  false,
				Details: fmt.Sprintf("Missing! Install via: sudo apt install -y %s", tool),
			})
		}
	}

	// 4. Python Modules
	pyModules := []string{"pyatspi", "PIL", "Xlib"}
	for _, mod := range pyModules {
		cmd := exec.Command("python3", "-c", fmt.Sprintf("import %s", mod))
		if err := cmd.Run(); err == nil {
			checks = append(checks, CheckResult{
				Name:    "python_" + mod,
				Passed:  true,
				Details: "Module import succeeded",
			})
		} else {
			healthy = false
			pkgName := "python3-" + strings.ToLower(mod)
			if mod == "PIL" {
				pkgName = "python3-pil"
			}
			checks = append(checks, CheckResult{
				Name:    "python_" + mod,
				Passed:  false,
				Details: fmt.Sprintf("Missing! Install via: sudo apt install -y %s", pkgName),
			})
		}
	}

	// 5. AT-SPI Accessibility Bus
	atspiCmd := exec.Command("python3", "-c", "import pyatspi; d = pyatspi.Registry.getDesktop(0); print(d.childCount)")
	if out, err := atspiCmd.Output(); err == nil {
		count := strings.TrimSpace(string(out))
		checks = append(checks, CheckResult{
			Name:    "atspi_accessibility",
			Passed:  true,
			Details: fmt.Sprintf("Accessible desktop found (%s top-level children)", count),
		})
	} else {
		checks = append(checks, CheckResult{
			Name:    "atspi_accessibility",
			Passed:  false,
			Details: "AT-SPI desktop inaccessible. Ensure accessibility is enabled in system settings.",
		})
	}

	// 6. Kill Switch Daemon
	if pid, running := isDaemonRunning(); running {
		checks = append(checks, CheckResult{
			Name:    "killswitch_daemon",
			Passed:  true,
			Details: fmt.Sprintf("Running (PID %d). Global hotkey Ctrl+Alt+Escape active.", pid),
		})
	} else {
		checks = append(checks, CheckResult{
			Name:    "killswitch_daemon",
			Passed:  false,
			Details: "Not running. Run 'deskctl run-killswitch-daemon' or use scripts/killswitch-daemon.py",
		})
	}

	// 7. Security & Sudo Allowlist
	allowlistPath := filepath.Join(deskctlDir, "sudo-allowlist.txt")
	sudoersDropin := "/etc/sudoers.d/deskctl"
	hasAllowlist := false
	if _, err := os.Stat(allowlistPath); err == nil {
		hasAllowlist = true
	}
	hasDropin := false
	if _, err := os.Stat(sudoersDropin); err == nil {
		hasDropin = true
	}
	checks = append(checks, CheckResult{
		Name:    "sudo_allowlist",
		Passed:  hasAllowlist,
		Details: fmt.Sprintf("Allowlist file: %v, /etc/sudoers.d dropin: %v", hasAllowlist, hasDropin),
	})

	// 8. Configuration File
	cfgPath := filepath.Join(deskctlDir, "config.toml")
	hasCfg := false
	if _, err := os.Stat(cfgPath); err == nil {
		hasCfg = true
	}
	checks = append(checks, CheckResult{
		Name:    "config_file",
		Passed:  hasCfg,
		Details: fmt.Sprintf("Configuration at %s (exists: %v)", cfgPath, hasCfg),
	})

	outputJSON(map[string]interface{}{
		"ok":      true,
		"healthy": healthy,
		"checks":  checks,
		"summary": fmt.Sprintf("%d checks performed. System healthy: %v", len(checks), healthy),
	})
}
