package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var pidFile = filepath.Join(deskctlDir, "killswitch.pid")

func doStop(reason string) {
	if reason == "" {
		reason = "Manually invoked via deskctl stop"
	}
	ts := time.Now().Format(time.RFC3339)
	content := fmt.Sprintf("TRIGGERED at %s: %s\n", ts, reason)
	_ = os.WriteFile(stopFile, []byte(content), 0644)
	logAction("stop", []string{reason}, true, false, "STOP file created")
	outputJSON(map[string]interface{}{
		"ok":        true,
		"message":   "STOP file created. All deskctl action commands are now blocked.",
		"stop_file": stopFile,
	})
}

func doReset() {
	if _, err := os.Stat(stopFile); err == nil {
		_ = os.Remove(stopFile)
		logAction("reset", []string{}, false, false, "STOP file removed")
		outputJSON(map[string]interface{}{
			"ok":      true,
			"message": "Kill switch cleared. deskctl commands resumed.",
		})
		return
	}
	outputJSON(map[string]interface{}{
		"ok":      true,
		"message": "Kill switch was not active. No STOP file existed.",
	})
}

func isDaemonRunning() (int, bool) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}
	// Signal 0 checks if process exists without actually sending a signal
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return pid, true
	}
	return 0, false
}

func startKillswitchDaemon() (int, error) {
	if pid, running := isDaemonRunning(); running {
		return pid, nil
	}

	daemonScript := filepath.Join(helperDir, "killswitch_daemon.py")
	if _, err := os.Stat(daemonScript); os.IsNotExist(err) {
		unpackHelpers()
	}

	cmd := exec.Command("python3", daemonScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}

	pid := cmd.Process.Pid
	_ = os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
	return pid, nil
}

func stopKillswitchDaemon() error {
	pid, running := isDaemonRunning()
	if !running {
		_ = os.Remove(pidFile)
		return nil
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	_ = process.Signal(syscall.SIGTERM)
	_ = os.Remove(pidFile)
	return nil
}

func doKillswitchCmd(subcmd string) {
	switch strings.ToLower(subcmd) {
	case "start":
		pid, err := startKillswitchDaemon()
		if err != nil {
			outputError(fmt.Sprintf("Failed to start killswitch daemon: %v", err))
		}
		outputJSON(map[string]interface{}{
			"ok":      true,
			"status":  "running",
			"pid":     pid,
			"hotkey":  "Ctrl+Alt+Escape",
			"message": "Kill switch daemon listening for Ctrl+Alt+Escape",
		})
	case "stop":
		_ = stopKillswitchDaemon()
		outputJSON(map[string]interface{}{
			"ok":      true,
			"status":  "stopped",
			"message": "Kill switch daemon stopped",
		})
	case "status":
		pid, running := isDaemonRunning()
		outputJSON(map[string]interface{}{
			"ok":      true,
			"running": running,
			"pid":     pid,
			"hotkey":  "Ctrl+Alt+Escape",
		})
	default:
		outputError(fmt.Sprintf("Unknown killswitch subcommand '%s'. Use start, stop, or status.", subcmd))
	}
}
