package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed helper/*
var helperFS embed.FS

var (
	deskctlDir        = filepath.Join(os.Getenv("HOME"), ".deskctl")
	stopFile          = filepath.Join(deskctlDir, "STOP")
	logFile           = filepath.Join(deskctlDir, "actions.log")
	stateFile         = filepath.Join(deskctlDir, "last_screenshot.json")
	helperDir         = filepath.Join(deskctlDir, "helper")
	failureHistoryFile = filepath.Join(deskctlDir, "failure_tracker.json")
	screensCacheFile   = filepath.Join(deskctlDir, "screens_cache.json")
)

type FailureTracker struct {
	LastAction string `json:"last_action"`
	FailCount  int    `json:"fail_count"`
}

func initEnv() {
	_ = os.MkdirAll(deskctlDir, 0755)
	_ = os.MkdirAll(filepath.Join(deskctlDir, "screenshots"), 0755)
	_ = os.MkdirAll(filepath.Join(deskctlDir, "recordings"), 0755)
	_ = os.MkdirAll(filepath.Join(deskctlDir, "sessions"), 0755)
	_ = os.MkdirAll(helperDir, 0755)
	unpackHelpers()
}

func unpackHelpers() {
	entries, err := fs.ReadDir(helperFS, "helper")
	if err != nil {
		return
	}
	for _, entry := range entries {
		data, err := helperFS.ReadFile("helper/" + entry.Name())
		if err != nil {
			continue
		}
		targetPath := filepath.Join(helperDir, entry.Name())
		_ = os.WriteFile(targetPath, data, 0755)
	}
}

func checkKillSwitch() {
	if _, err := os.Stat(stopFile); err == nil {
		content, _ := os.ReadFile(stopFile)
		info := strings.TrimSpace(string(content))
		if info == "" {
			info = "Manual or hotkey trigger"
		}
		outputError(fmt.Sprintf("Kill switch active: STOP file detected at ~/.deskctl/STOP (%s). Aborting. Run 'deskctl reset' to clear.", info))
	}
}

func checkAuthSafety(cmd string) {
	inputCommands := map[string]bool{
		"type": true, "key": true, "key-hold": true,
		"click": true, "double-click": true, "triple-click": true, "right-click": true, "middle-click": true,
		"hold-and-release": true,
	}
	if !inputCommands[cmd] {
		return
	}

	info, err := checkAuthPrompts()
	if err == nil && info.Detected {
		msg := fmt.Sprintf("Authentication Required: %s is requesting password. AI input is paused.", info.Program)
		sendNotification("deskctl - Authentication Prompt Detected", msg)
		outputJSON(map[string]interface{}{
			"ok":          false,
			"blocked":     true,
			"auth_prompt": true,
			"prompt_type": info.PromptType,
			"program":     info.Program,
			"details":     info.Details,
			"error":       fmt.Sprintf("Active authentication prompt detected (%s). Input blocked for password safety. Please type your password directly on the screen or run 'deskctl auth-wait'.", info.PromptType),
		})
		os.Exit(0)
	}
}

func recordActionSuccess() {
	tracker := FailureTracker{LastAction: "", FailCount: 0}
	data, _ := json.Marshal(tracker)
	_ = os.WriteFile(failureHistoryFile, data, 0644)
}

func recordActionFailure(actionKey string) {
	var tracker FailureTracker
	if data, err := os.ReadFile(failureHistoryFile); err == nil {
		_ = json.Unmarshal(data, &tracker)
	}

	if tracker.LastAction == actionKey {
		tracker.FailCount++
	} else {
		tracker.LastAction = actionKey
		tracker.FailCount = 1
	}

	data, _ := json.Marshal(tracker)
	_ = os.WriteFile(failureHistoryFile, data, 0644)

	if tracker.FailCount >= 3 {
		outputError(fmt.Sprintf("Stuck detector triggered: action %q failed 3 times consecutively. Halting for user intervention to prevent loops.", actionKey))
	}
}

func checkTool(toolName, packageName string) error {
	_, err := exec.LookPath(toolName)
	if err != nil {
		return fmt.Errorf("required tool '%s' is not installed or not in PATH. Install with: sudo apt install %s", toolName, packageName)
	}
	return nil
}

func checkRequiredTools() error {
	tools := map[string]string{
		"xdotool": "xdotool",
		"maim":    "maim",
		"wmctrl":  "wmctrl",
		"xclip":   "xclip",
		"python3": "python3",
	}
	for tool, pkg := range tools {
		if err := checkTool(tool, pkg); err != nil {
			return err
		}
	}
	return nil
}

func isDestructiveKey(combo string) bool {
	c := strings.ToLower(combo)
	destructiveList := []string{
		"ctrl+w", "ctrl+q", "alt+f4", "ctrl+alt+del", "ctrl+d",
		"delete", "backspace", "ctrl+backspace", "ctrl+shift+w",
	}
	for _, d := range destructiveList {
		if strings.Contains(c, d) {
			return true
		}
	}
	return false
}

func logAction(cmdName string, args []string, destructive bool, dryRun bool, note string) {
	_ = os.MkdirAll(deskctlDir, 0755)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	ts := time.Now().Format(time.RFC3339)
	prefix := ""
	if dryRun {
		prefix += "[DRY-RUN] "
	}
	if destructive {
		prefix += "[DESTRUCTIVE] "
	}

	argStr := strings.Join(args, " ")
	line := fmt.Sprintf("[%s] %s%s %s", ts, prefix, cmdName, argStr)
	if note != "" {
		line += fmt.Sprintf(" | %s", note)
	}
	line += "\n"
	_, _ = f.WriteString(line)

	// Also pipe to active session jsonl if session is running
	logSessionAction(cmdName, args, map[string]interface{}{
		"destructive": destructive,
		"dry_run":     dryRun,
		"note":        note,
	})
}
