package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var clipHistoryFile = filepath.Join(deskctlDir, "clipboard_history.json")

func doClipboardGet() {
	out, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output()
	if err != nil {
		outputJSON(map[string]interface{}{
			"ok":   true,
			"text": "",
		})
		return
	}
	outputJSON(map[string]interface{}{
		"ok":   true,
		"text": string(out),
	})
}

func doClipboardSet(text string, dryRun bool) {
	logAction("clipboard set", []string{fmt.Sprintf("len=%d", len(text))}, false, dryRun, "")

	if !dryRun {
		cmd := exec.Command("xclip", "-selection", "clipboard", "-i")
		cmd.Stdin = bytes.NewBufferString(text)
		cmd.Stdout = nil
		cmd.Stderr = nil
		_ = cmd.Run()

		// Append to history
		var history []string
		if data, err := os.ReadFile(clipHistoryFile); err == nil {
			_ = json.Unmarshal(data, &history)
		}
		history = append([]string{text}, history...)
		if len(history) > 20 {
			history = history[:20]
		}
		data, _ := json.MarshalIndent(history, "", "  ")
		_ = os.WriteFile(clipHistoryFile, data, 0644)
	}

	outputJSON(map[string]interface{}{
		"ok":      true,
		"action":  "set",
		"length":  len(text),
		"dry_run": dryRun,
	})
}

func doClipboardHistory() {
	var history []string
	if data, err := os.ReadFile(clipHistoryFile); err == nil {
		_ = json.Unmarshal(data, &history)
	}
	outputJSON(map[string]interface{}{
		"ok":      true,
		"history": history,
		"count":   len(history),
	})
}

func doClipboardGetImage(outputPath string) {
	if outputPath == "" {
		ts := time.Now().Format("20060102_150405")
		outputPath = filepath.Join(deskctlDir, "screenshots", fmt.Sprintf("clip_%s.png", ts))
	}
	_ = os.MkdirAll(filepath.Dir(outputPath), 0755)

	out, err := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o").CombinedOutput()
	if err != nil {
		outputError(fmt.Sprintf("Failed to read image from clipboard: %s (%v)", strings.TrimSpace(string(out)), err))
	}

	if err := os.WriteFile(outputPath, out, 0644); err != nil {
		outputError(fmt.Sprintf("Failed to write clipboard image to %s: %v", outputPath, err))
	}

	outputJSON(map[string]interface{}{
		"ok":    true,
		"path":  outputPath,
		"bytes": len(out),
	})
}

func doClipboardSetImage(imagePath string) {
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		outputError(fmt.Sprintf("Image file %q does not exist", imagePath))
	}

	cmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-i", imagePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outputError(fmt.Sprintf("Failed to copy image to clipboard: %s (%v)", strings.TrimSpace(string(out)), err))
	}

	outputJSON(map[string]interface{}{
		"ok":      true,
		"action":  "set-image",
		"path":    imagePath,
		"message": "Image loaded into clipboard",
	})
}

func sendNotification(title, message string) {
	if title == "" {
		title = "deskctl"
	}
	_ = exec.Command("notify-send", title, message).Run()
}

func doNotify(message, title string) {
	if title == "" {
		title = "deskctl"
	}
	logAction("notify", []string{title, message}, false, false, "")
	sendNotification(title, message)
	outputJSON(map[string]interface{}{
		"ok":      true,
		"action":  "notify",
		"title":   title,
		"message": message,
	})
}

func doOpen(target string) {
	logAction("open", []string{target}, false, false, "")
	cmd := exec.Command("xdg-open", target)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		outputError(fmt.Sprintf("xdg-open failed: %v", err))
	}
	outputJSON(map[string]interface{}{
		"ok":     true,
		"action": "open",
		"target": target,
	})
}

func doFileList(dirPath string) {
	if dirPath == "" {
		dirPath = "."
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		outputError(fmt.Sprintf("Failed to read directory: %v", err))
	}

	var items []map[string]interface{}
	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		modTime := ""
		if info != nil {
			size = info.Size()
			modTime = info.ModTime().Format(time.RFC3339)
		}
		items = append(items, map[string]interface{}{
			"name":     e.Name(),
			"is_dir":   e.IsDir(),
			"size":     size,
			"modified": modTime,
		})
	}
	outputJSON(map[string]interface{}{
		"ok":    true,
		"path":  dirPath,
		"items": items,
		"count": len(items),
	})
}

func doFileRead(filePath string, maxBytes int) {
	f, err := os.Open(filePath)
	if err != nil {
		outputError(fmt.Sprintf("Failed to open file: %v", err))
	}
	defer f.Close()

	if maxBytes <= 0 {
		maxBytes = 64 * 1024
	}

	buf := make([]byte, maxBytes)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		outputError(fmt.Sprintf("Failed to read file: %v", err))
	}

	outputJSON(map[string]interface{}{
		"ok":        true,
		"path":      filePath,
		"content":   string(buf[:n]),
		"bytes":     n,
		"truncated": n == maxBytes,
	})
}

func doFileWrite(filePath, content string, appendMode bool) {
	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(filePath, flags, 0644)
	if err != nil {
		outputError(fmt.Sprintf("Failed to open file for write: %v", err))
	}
	defer f.Close()

	n, err := f.WriteString(content)
	if err != nil {
		outputError(fmt.Sprintf("Failed to write to file: %v", err))
	}
	outputJSON(map[string]interface{}{
		"ok":           true,
		"path":         filePath,
		"bytes_written": n,
	})
}

func doFileTrash(filePath string) {
	logAction("file trash", []string{filePath}, true, false, "")
	out, err := exec.Command("gio", "trash", filePath).CombinedOutput()
	if err != nil {
		outputError(fmt.Sprintf("gio trash failed: %s (%v)", strings.TrimSpace(string(out)), err))
	}
	outputJSON(map[string]interface{}{
		"ok":      true,
		"action":  "trash",
		"path":    filePath,
		"message": "Moved to trash via gio",
	})
}

func doProcessesList(filter string) {
	out, err := exec.Command("ps", "-eo", "pid,ppid,%cpu,%mem,comm,args").CombinedOutput()
	if err != nil {
		outputError(fmt.Sprintf("ps failed: %v", err))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var procs []map[string]interface{}
	fLower := strings.ToLower(filter)

	for i, l := range lines {
		if i == 0 {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) >= 5 {
			pid, _ := strconv.Atoi(fields[0])
			comm := fields[4]
			args := strings.Join(fields[5:], " ")
			if fLower == "" || strings.Contains(strings.ToLower(comm), fLower) || strings.Contains(strings.ToLower(args), fLower) {
				procs = append(procs, map[string]interface{}{
					"pid":  pid,
					"name": comm,
					"cmd":  args,
					"cpu":  fields[2],
					"mem":  fields[3],
				})
			}
		}
	}

	outputJSON(map[string]interface{}{
		"ok":        true,
		"processes": procs,
		"count":     len(procs),
	})
}

func doProcessesKill(target string, force bool) {
	logAction("processes kill", []string{target, fmt.Sprintf("force=%v", force)}, true, false, "")
	var cmd *exec.Cmd
	sig := "-15"
	if force {
		sig = "-9"
	}

	if pid, err := strconv.Atoi(target); err == nil && pid > 0 {
		cmd = exec.Command("kill", sig, strconv.Itoa(pid))
	} else {
		cmd = exec.Command("pkill", sig, "-f", target)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		outputError(fmt.Sprintf("kill failed: %s (%v)", strings.TrimSpace(string(out)), err))
	}
	outputJSON(map[string]interface{}{
		"ok":      true,
		"action":  "kill",
		"target":  target,
		"message": "Signal sent to process(es)",
	})
}

func doVolume(action, val string) {
	switch strings.ToLower(action) {
	case "get", "":
		out, _ := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@").Output()
		re := regexp.MustCompile(`(\d+)%`)
		m := re.FindStringSubmatch(string(out))
		vol := 0
		if len(m) == 2 {
			vol, _ = strconv.Atoi(m[1])
		}
		outputJSON(map[string]interface{}{
			"ok":     true,
			"volume": vol,
		})
	case "set":
		pct, err := strconv.Atoi(val)
		if err != nil || pct < 0 || pct > 150 {
			outputError("Volume percent must be between 0 and 150")
		}
		_ = exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", fmt.Sprintf("%d%%", pct)).Run()
		outputJSON(map[string]interface{}{
			"ok":     true,
			"action": "set-volume",
			"volume": pct,
		})
	case "mute":
		_ = exec.Command("pactl", "set-sink-mute", "@DEFAULT_SINK@", "toggle").Run()
		outputJSON(map[string]interface{}{
			"ok":     true,
			"action": "toggle-mute",
		})
	}
}

func doBrightness(action, val string) {
	switch strings.ToLower(action) {
	case "get", "":
		// Try reading backlight interface
		data, err := os.ReadFile("/sys/class/backlight/intel_backlight/actual_brightness")
		maxData, err2 := os.ReadFile("/sys/class/backlight/intel_backlight/max_brightness")
		if err == nil && err2 == nil {
			cur, _ := strconv.Atoi(strings.TrimSpace(string(data)))
			maxB, _ := strconv.Atoi(strings.TrimSpace(string(maxData)))
			pct := int(float64(cur) * 100.0 / float64(maxB))
			outputJSON(map[string]interface{}{
				"ok":         true,
				"brightness": pct,
			})
			return
		}
		outputJSON(map[string]interface{}{
			"ok":         true,
			"brightness": 100,
		})
	case "set":
		pct, err := strconv.Atoi(val)
		if err != nil || pct < 1 || pct > 100 {
			outputError("Brightness must be between 1 and 100")
		}
		// Set via xrandr software brightness
		screens, _, _ := detectScreens()
		for _, s := range screens {
			factor := float64(pct) / 100.0
			_ = exec.Command("xrandr", "--output", s.Name, "--brightness", fmt.Sprintf("%.2f", factor)).Run()
		}
		outputJSON(map[string]interface{}{
			"ok":         true,
			"action":     "set-brightness",
			"brightness": pct,
		})
	}
}

func doNetworkStatus() {
	out, err := exec.Command("ip", "-brief", "address").CombinedOutput()
	if err != nil {
		outputError(fmt.Sprintf("Failed to query network interfaces: %v", err))
	}
	var ifaces []map[string]string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 {
			addr := ""
			if len(f) >= 3 {
				addr = f[2]
			}
			ifaces = append(ifaces, map[string]string{
				"interface": f[0],
				"status":    f[1],
				"ip":        addr,
			})
		}
	}
	outputJSON(map[string]interface{}{
		"ok":         true,
		"interfaces": ifaces,
	})
}

func doBatteryStatus() {
	files, err := filepath.Glob("/sys/class/power_supply/BAT*")
	if err != nil || len(files) == 0 {
		outputJSON(map[string]interface{}{
			"ok":      true,
			"battery": nil,
			"message": "No battery detected (Desktop machine)",
		})
		return
	}
	batDir := files[0]
	capData, _ := os.ReadFile(filepath.Join(batDir, "capacity"))
	statData, _ := os.ReadFile(filepath.Join(batDir, "status"))
	pct, _ := strconv.Atoi(strings.TrimSpace(string(capData)))
	outputJSON(map[string]interface{}{
		"ok": true,
		"battery": map[string]interface{}{
			"percentage": pct,
			"status":     strings.TrimSpace(string(statData)),
		},
	})
}

func doSchedule(delaySec int, commandStr string) {
	if delaySec <= 0 || strings.TrimSpace(commandStr) == "" {
		outputError("schedule requires --delay <seconds> and a command")
	}

	script := fmt.Sprintf("sleep %d && deskctl run %q", delaySec, commandStr)
	cmd := exec.Command("bash", "-c", script)
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Start()

	outputJSON(map[string]interface{}{
		"ok":            true,
		"action":        "schedule",
		"delay_seconds": delaySec,
		"command":       commandStr,
		"message":       fmt.Sprintf("Command scheduled in background to run in %d seconds", delaySec),
	})
}
