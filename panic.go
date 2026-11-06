package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func doPanic() {
	// 1. Immediately create STOP file to freeze any deskctl commands
	reason := fmt.Sprintf("Emergency Panic Triggered at %s", time.Now().Format(time.RFC3339))
	_ = os.WriteFile(stopFile, []byte(reason), 0644)

	logAction("panic", []string{"emergency_stop"}, true, false, "")
	logViolation("panic", "system", "Emergency panic triggered by user or agent")

	terminatedPIDs := 0
	closedWindows := 0

	// 2. Terminate any active screen recording
	_ = exec.Command("pkill", "-2", "ffmpeg").Run()

	// 3. Inspect active session for spawned PIDs and windows
	sess, err := getActiveSession()
	if err == nil && sess != nil {
		sess.Status = "aborted_by_panic"
		sess.EndTime = time.Now().Format(time.RFC3339)

		// Terminate spawned processes
		for _, pid := range sess.SpawnedPIDs {
			if pid > 0 {
				if err := exec.Command("kill", "-15", strconv.Itoa(pid)).Run(); err == nil {
					terminatedPIDs++
				}
			}
		}

		// Close spawned windows
		for _, winID := range sess.SpawnedWindows {
			if winID != "" {
				if err := exec.Command("wmctrl", "-i", "-c", winID).Run(); err == nil {
					closedWindows++
				}
			}
		}

		// Update active session metadata
		data, _ := json.MarshalIndent(sess, "", "  ")
		_ = os.WriteFile(activeSessionFile, data, 0644)
	}

	outputJSON(map[string]interface{}{
		"ok":                   true,
		"panic":                true,
		"stop_file":            stopFile,
		"terminated_processes": terminatedPIDs,
		"closed_windows":       closedWindows,
		"message":              "Emergency panic completed: STOP file created, processes killed, session windows closed.",
	})
}
