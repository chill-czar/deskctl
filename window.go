package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type WindowInfo struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Class      string `json:"class"`
	PID        int    `json:"pid"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	W          int    `json:"w"`
	H          int    `json:"h"`
	ScreenID   int    `json:"screen_id"`
	ScreenName string `json:"screen_name"`
	Active     bool   `json:"active"`
}

func getActiveWindowIDHex() string {
	out, err := exec.Command("xdotool", "getactivewindow").Output()
	if err != nil {
		return ""
	}
	decStr := strings.TrimSpace(string(out))
	val, err := strconv.ParseUint(decStr, 10, 64)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("0x%08x", val)
}

func getActiveWindowInfo() *WindowInfo {
	wins, err := listWindows()
	if err != nil {
		return nil
	}
	for _, w := range wins {
		if w.Active {
			return &w
		}
	}
	return nil
}

func normalizeHex(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.TrimPrefix(h, "0x")
	val, err := strconv.ParseUint(h, 16, 64)
	if err != nil {
		return h
	}
	return fmt.Sprintf("0x%08x", val)
}

func hexToDecString(h string) string {
	val, err := strconv.ParseUint(strings.TrimPrefix(strings.ToLower(h), "0x"), 16, 64)
	if err != nil {
		return h
	}
	return strconv.FormatUint(val, 10)
}

func listWindows() ([]WindowInfo, error) {
	out, err := exec.Command("wmctrl", "-l", "-p", "-G", "-x").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("wmctrl failed: %s (%v)", strings.TrimSpace(string(out)), err)
	}

	screens, _, _ := detectScreens()
	activeHex := getActiveWindowIDHex()

	var wins []WindowInfo
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		rawID := fields[0]
		normID := normalizeHex(rawID)
		pid, _ := strconv.Atoi(fields[2])
		x, _ := strconv.Atoi(fields[3])
		y, _ := strconv.Atoi(fields[4])
		w, _ := strconv.Atoi(fields[5])
		h, _ := strconv.Atoi(fields[6])
		wmClass := fields[7]

		title := ""
		if len(fields) > 9 {
			title = strings.Join(fields[9:], " ")
		}

		// Find which screen contains the center of this window
		screenID := -1
		screenName := "unknown"
		centerX := x + w/2
		centerY := y + h/2

		for _, s := range screens {
			if centerX >= s.X && centerX < s.X+s.W && centerY >= s.Y && centerY < s.Y+s.H {
				screenID = s.ID
				screenName = s.Name
				break
			}
		}

		wins = append(wins, WindowInfo{
			ID:         normID,
			Title:      title,
			Class:      wmClass,
			PID:        pid,
			X:          x,
			Y:          y,
			W:          w,
			H:          h,
			ScreenID:   screenID,
			ScreenName: screenName,
			Active:     (normID == activeHex),
		})
	}
	return wins, nil
}

func doWindows() {
	wins, err := listWindows()
	if err != nil {
		outputError(err.Error())
	}
	outputJSON(map[string]interface{}{
		"ok":      true,
		"windows": wins,
		"count":   len(wins),
	})
}

func findTargetWindow(idOpt, titleOpt string) (*WindowInfo, error) {
	wins, err := listWindows()
	if err != nil {
		return nil, err
	}

	if idOpt != "" {
		norm := normalizeHex(idOpt)
		for _, w := range wins {
			if w.ID == norm || strings.EqualFold(w.ID, norm) {
				return &w, nil
			}
		}
	} else if titleOpt != "" {
		q := strings.ToLower(titleOpt)
		// First pass: prefer real application windows over desktop background managers (e.g. nemo-desktop)
		for _, w := range wins {
			if !strings.Contains(strings.ToLower(w.Class), "-desktop") {
				if strings.Contains(strings.ToLower(w.Title), q) || strings.Contains(strings.ToLower(w.Class), q) {
					return &w, nil
				}
			}
		}
		// Second pass: match any window if no application window matched
		for _, w := range wins {
			if strings.Contains(strings.ToLower(w.Title), q) || strings.Contains(strings.ToLower(w.Class), q) {
				return &w, nil
			}
		}
	} else {
		// Use active window
		for _, w := range wins {
			if w.Active {
				return &w, nil
			}
		}
	}
	return nil, fmt.Errorf("window not found matching id=%q title=%q", idOpt, titleOpt)
}

func doFocus(idOpt, titleOpt string, dryRun, screenshotAfter bool) {
	win, err := findTargetWindow(idOpt, titleOpt)
	if err != nil {
		recordActionFailure("focus")
		outputError(err.Error())
	}
	recordActionSuccess()

	logAction("focus", []string{win.ID, win.Title}, false, dryRun, "")
	if !dryRun {
		_ = exec.Command("wmctrl", "-i", "-a", win.ID).Run()
		performActionWait()
	}

	resp := map[string]interface{}{
		"ok":      true,
		"action":  "focus",
		"window":  win,
		"dry_run": dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}

func doCloseWindow(idOpt, titleOpt string, dryRun bool) {
	win, err := findTargetWindow(idOpt, titleOpt)
	if err != nil {
		outputError(err.Error())
	}

	logAction("close-window", []string{win.ID, win.Title}, true, dryRun, "")
	if !dryRun {
		out, err := exec.Command("wmctrl", "-i", "-c", win.ID).CombinedOutput()
		if err != nil {
			outputError(fmt.Sprintf("wmctrl close failed: %s (%v)", strings.TrimSpace(string(out)), err))
		}
		performActionWait()
	}

	outputJSON(map[string]interface{}{
		"ok":      true,
		"action":  "close",
		"window":  win,
		"dry_run": dryRun,
	})
}

func doMinimizeWindow(idOpt, titleOpt string, dryRun bool) {
	win, err := findTargetWindow(idOpt, titleOpt)
	if err != nil {
		outputError(err.Error())
	}

	logAction("minimize", []string{win.ID, win.Title}, false, dryRun, "")
	if !dryRun {
		decID := hexToDecString(win.ID)
		_ = exec.Command("xdotool", "windowminimize", decID).Run()
		performActionWait()
	}

	outputJSON(map[string]interface{}{
		"ok":      true,
		"action":  "minimize",
		"window":  win,
		"dry_run": dryRun,
	})
}

func doMaximizeWindow(idOpt, titleOpt, mode string, dryRun bool) {
	win, err := findTargetWindow(idOpt, titleOpt)
	if err != nil {
		outputError(err.Error())
	}

	actionParam := "add"
	if mode == "remove" || mode == "unmaximize" {
		actionParam = "remove"
	} else if mode == "toggle" {
		actionParam = "toggle"
	}

	logAction("maximize", []string{win.ID, actionParam}, false, dryRun, "")
	if !dryRun {
		_ = exec.Command("wmctrl", "-i", "-r", win.ID, "-b", fmt.Sprintf("%s,maximized_vert,maximized_horz", actionParam)).Run()
		performActionWait()
	}

	outputJSON(map[string]interface{}{
		"ok":      true,
		"action":  "maximize",
		"mode":    actionParam,
		"window":  win,
		"dry_run": dryRun,
	})
}

func doMoveResizeWindow(idOpt, titleOpt string, x, y, w, h int, dryRun bool) {
	win, err := findTargetWindow(idOpt, titleOpt)
	if err != nil {
		outputError(err.Error())
	}

	// wmctrl -e <G>,<X>,<Y>,<W>,<H>
	// -1 retains existing value
	gravity := 0
	logAction("move-resize-window", []string{win.ID, fmt.Sprintf("%d,%d,%d,%d", x, y, w, h)}, false, dryRun, "")

	if !dryRun {
		_ = exec.Command("wmctrl", "-i", "-r", win.ID, "-b", "remove,maximized_vert,maximized_horz").Run()
		arg := fmt.Sprintf("%d,%d,%d,%d,%d", gravity, x, y, w, h)
		_ = exec.Command("wmctrl", "-i", "-r", win.ID, "-e", arg).Run()
		performActionWait()
	}

	outputJSON(map[string]interface{}{
		"ok":      true,
		"action":  "move-resize",
		"window":  win,
		"params":  map[string]int{"x": x, "y": y, "w": w, "h": h},
		"dry_run": dryRun,
	})
}

func doMoveToScreen(idOpt, titleOpt, screenArg string, dryRun bool) {
	win, err := findTargetWindow(idOpt, titleOpt)
	if err != nil {
		outputError(err.Error())
	}

	screens, _, err := detectScreens()
	if err != nil {
		outputError(err.Error())
	}

	var targetScreen *ScreenInfo
	for _, s := range screens {
		if strconv.Itoa(s.ID) == screenArg || strings.EqualFold(s.Name, screenArg) {
			targetScreen = &s
			break
		}
	}
	if targetScreen == nil {
		outputError(fmt.Sprintf("Target screen %q not found. Available screens: %s", screenArg, listScreenNames(screens)))
	}

	newX := targetScreen.X + 80
	newY := targetScreen.Y + 80

	logAction("move-to-screen", []string{win.ID, targetScreen.Name}, false, dryRun, "")
	if !dryRun {
		_ = exec.Command("wmctrl", "-i", "-r", win.ID, "-b", "remove,maximized_vert,maximized_horz").Run()
		_ = exec.Command("wmctrl", "-i", "-r", win.ID, "-e", fmt.Sprintf("0,%d,%d,-1,-1", newX, newY)).Run()
		performActionWait()
	}

	outputJSON(map[string]interface{}{
		"ok":            true,
		"action":        "move-to-screen",
		"window_id":     win.ID,
		"target_screen": targetScreen,
		"new_position":  map[string]int{"x": newX, "y": newY},
		"dry_run":       dryRun,
	})
}

func doAlwaysOnTop(idOpt, titleOpt string, dryRun bool) {
	win, err := findTargetWindow(idOpt, titleOpt)
	if err != nil {
		outputError(err.Error())
	}

	logAction("always-on-top", []string{win.ID}, false, dryRun, "")
	if !dryRun {
		_ = exec.Command("wmctrl", "-i", "-r", win.ID, "-b", "toggle,above").Run()
		performActionWait()
	}

	outputJSON(map[string]interface{}{
		"ok":      true,
		"action":  "always-on-top",
		"window":  win,
		"dry_run": dryRun,
	})
}

func doWorkspaces(subcmd, idOpt, wsArg string) {
	switch strings.ToLower(subcmd) {
	case "list", "":
		out, err := exec.Command("wmctrl", "-d").CombinedOutput()
		if err != nil {
			outputError(fmt.Sprintf("wmctrl -d failed: %v", err))
		}
		var list []map[string]interface{}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				id, _ := strconv.Atoi(fields[0])
				active := fields[1] == "*"
				name := fields[len(fields)-1]
				list = append(list, map[string]interface{}{
					"id":     id,
					"active": active,
					"name":   name,
				})
			}
		}
		outputJSON(map[string]interface{}{
			"ok":         true,
			"workspaces": list,
			"count":      len(list),
		})

	case "switch":
		wsNum, err := strconv.Atoi(wsArg)
		if err != nil {
			outputError("Workspace number must be an integer")
		}
		_ = exec.Command("wmctrl", "-s", strconv.Itoa(wsNum)).Run()
		outputJSON(map[string]interface{}{
			"ok":        true,
			"action":    "switch-workspace",
			"workspace": wsNum,
		})

	case "move-window":
		win, err := findTargetWindow(idOpt, "")
		if err != nil {
			outputError(err.Error())
		}
		wsNum, err := strconv.Atoi(wsArg)
		if err != nil {
			outputError("Workspace number must be an integer")
		}
		_ = exec.Command("wmctrl", "-i", "-r", win.ID, "-t", strconv.Itoa(wsNum)).Run()
		outputJSON(map[string]interface{}{
			"ok":        true,
			"action":    "move-window-to-workspace",
			"window_id": win.ID,
			"workspace": wsNum,
		})
	}
}

func doLaunch(targetApp string) {
	logAction("launch", []string{targetApp}, false, false, "")
	var cmd *exec.Cmd
	if strings.HasSuffix(targetApp, ".desktop") {
		appBase := strings.TrimSuffix(filepath.Base(targetApp), ".desktop")
		cmd = exec.Command("gtk-launch", appBase)
	} else {
		cmd = exec.Command("sh", "-c", "exec "+targetApp)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}

	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		outputError(fmt.Sprintf("Failed to launch %q: %v", targetApp, err))
	}

	if cmd.Process != nil {
		recordSessionPID(cmd.Process.Pid)
	}

	outputJSON(map[string]interface{}{
		"ok":      true,
		"action":  "launch",
		"target":  targetApp,
		"message": "Application launched asynchronously",
	})
}

func doWaitForWindow(titleSubstr string, timeoutSec int) {
	if timeoutSec <= 0 {
		timeoutSec = 10
	}
	start := time.Now()
	for time.Since(start) < time.Duration(timeoutSec)*time.Second {
		win, _ := findTargetWindow("", titleSubstr)
		if win != nil {
			outputJSON(map[string]interface{}{
				"ok":         true,
				"found":      true,
				"window":     win,
				"elapsed_ms": time.Since(start).Milliseconds(),
			})
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	outputJSON(map[string]interface{}{
		"ok":         false,
		"found":      false,
		"error":      fmt.Sprintf("Window matching %q did not appear within %d seconds", titleSubstr, timeoutSec),
		"elapsed_ms": time.Since(start).Milliseconds(),
	})
}

func doWaitForWindowClose(idOpt, titleOpt string, timeoutSec int) {
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	start := time.Now()
	for time.Since(start) < time.Duration(timeoutSec)*time.Second {
		win, _ := findTargetWindow(idOpt, titleOpt)
		if win == nil {
			outputJSON(map[string]interface{}{
				"ok":         true,
				"closed":     true,
				"elapsed_ms": time.Since(start).Milliseconds(),
			})
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	outputJSON(map[string]interface{}{
		"ok":         false,
		"closed":     false,
		"error":      fmt.Sprintf("Window matching id=%q title=%q did not close within %d seconds", idOpt, titleOpt, timeoutSec),
		"elapsed_ms": time.Since(start).Milliseconds(),
	})
}
