package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SessionScope struct {
	Apps    []string `json:"apps"`
	Folders []string `json:"folders"`
	Domains []string `json:"domains"`
	Actions []string `json:"actions"`
}

type SessionMeta struct {
	ID             string       `json:"id"`
	Task           string       `json:"task"`
	Scope          SessionScope `json:"scope"`
	StartTime      string       `json:"start_time"`
	EndTime        string       `json:"end_time,omitempty"`
	Status         string       `json:"status"`
	TotalActs      int          `json:"total_actions"`
	ScratchDir     string       `json:"scratch_dir"`
	LogFile        string       `json:"log_file"`
	SpawnedPIDs    []int        `json:"spawned_pids,omitempty"`
	SpawnedWindows []string     `json:"spawned_windows,omitempty"`
}

func recordSessionPID(pid int) {
	if pid <= 0 {
		return
	}
	sess, err := getActiveSession()
	if err == nil && sess != nil && sess.Status == "active" {
		sess.SpawnedPIDs = append(sess.SpawnedPIDs, pid)
		data, _ := json.MarshalIndent(sess, "", "  ")
		_ = os.WriteFile(activeSessionFile, data, 0644)
	}
}

func recordSessionWindow(winID string) {
	if winID == "" {
		return
	}
	sess, err := getActiveSession()
	if err == nil && sess != nil && sess.Status == "active" {
		sess.SpawnedWindows = append(sess.SpawnedWindows, winID)
		data, _ := json.MarshalIndent(sess, "", "  ")
		_ = os.WriteFile(activeSessionFile, data, 0644)
	}
}

var activeSessionFile = filepath.Join(deskctlDir, "active_session.json")

func getActiveSession() (*SessionMeta, error) {
	data, err := os.ReadFile(activeSessionFile)
	if err != nil {
		return nil, err
	}
	var sess SessionMeta
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func logSessionAction(cmdName string, args []string, res map[string]interface{}) {
	sess, err := getActiveSession()
	if err != nil || sess == nil || sess.LogFile == "" {
		return
	}

	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"command":   cmdName,
		"args":      args,
		"result":    res,
	}

	bytes, err := json.Marshal(entry)
	if err != nil {
		return
	}

	f, err := os.OpenFile(sess.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		_, _ = f.WriteString(string(bytes) + "\n")
		f.Close()
	}

	// Update action count in active session
	sess.TotalActs++
	metaBytes, _ := json.MarshalIndent(sess, "", "  ")
	_ = os.WriteFile(activeSessionFile, metaBytes, 0644)
	metaPath := filepath.Join(deskctlDir, "sessions", sess.ID, "meta.json")
	_ = os.WriteFile(metaPath, metaBytes, 0644)
}

func doSessionStart(task string, appsStr, foldersStr, domainsStr, actionsStr string) {
	if strings.TrimSpace(task) == "" {
		outputError("session start requires --task \"<description>\"")
	}

	parseList := func(s string) []string {
		if strings.TrimSpace(s) == "" {
			return []string{}
		}
		parts := strings.Split(s, ",")
		var res []string
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if t != "" {
				res = append(res, t)
			}
		}
		return res
	}

	sessID := fmt.Sprintf("sess_%s", time.Now().Format("20060102_150405"))
	sessDir := filepath.Join(deskctlDir, "sessions", sessID)
	scratchDir := filepath.Join(sessDir, "scratch")
	logFile := filepath.Join(sessDir, "log.jsonl")

	_ = os.MkdirAll(scratchDir, 0755)

	meta := SessionMeta{
		ID:   sessID,
		Task: task,
		Scope: SessionScope{
			Apps:    parseList(appsStr),
			Folders: parseList(foldersStr),
			Domains: parseList(domainsStr),
			Actions: parseList(actionsStr),
		},
		StartTime:  time.Now().Format(time.RFC3339),
		Status:     "active",
		TotalActs:  0,
		ScratchDir: scratchDir,
		LogFile:    logFile,
	}

	data, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(filepath.Join(sessDir, "meta.json"), data, 0644)
	_ = os.WriteFile(activeSessionFile, data, 0644)

	outputJSON(map[string]interface{}{
		"ok":         true,
		"action":     "session-start",
		"session_id": sessID,
		"session":    meta,
		"message":    fmt.Sprintf("Session %s created with declared scope", sessID),
	})
}

func doSessionStatus() {
	sess, err := getActiveSession()
	if err != nil || sess == nil {
		outputJSON(map[string]interface{}{
			"ok":      true,
			"active":  false,
			"message": "No active session. Run 'deskctl session start --task ...'",
		})
		return
	}

	outputJSON(map[string]interface{}{
		"ok":      true,
		"active":  true,
		"session": sess,
	})
}

func doSessionEnd() {
	sess, err := getActiveSession()
	if err != nil || sess == nil {
		outputJSON(map[string]interface{}{
			"ok":      true,
			"message": "No active session to end",
		})
		return
	}

	sess.EndTime = time.Now().Format(time.RFC3339)
	sess.Status = "completed"

	metaBytes, _ := json.MarshalIndent(sess, "", "  ")
	metaPath := filepath.Join(deskctlDir, "sessions", sess.ID, "meta.json")
	_ = os.WriteFile(metaPath, metaBytes, 0644)
	_ = os.Remove(activeSessionFile)

	outputJSON(map[string]interface{}{
		"ok":             true,
		"action":         "session-end",
		"session_id":     sess.ID,
		"total_actions":  sess.TotalActs,
		"log_path":       sess.LogFile,
		"scratch_dir":    sess.ScratchDir,
		"message":        "Session ended and finalized",
	})
}
