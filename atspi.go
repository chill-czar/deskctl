package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const atspiSocketPath = "/tmp/deskctl-atspi.sock"

func queryATSPIHelper(cmdArgs []string) ([]byte, error) {
	// 1. Attempt fast UNIX socket connection to persistent daemon
	conn, err := net.DialTimeout("unix", atspiSocketPath, 80*time.Millisecond)
	if err == nil {
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		req := map[string]interface{}{"args": cmdArgs}
		reqBytes, err := json.Marshal(req)
		if err == nil {
			reqBytes = append(reqBytes, '\n')
			if _, err := conn.Write(reqBytes); err == nil {
				reader := bufio.NewReader(conn)
				line, err := reader.ReadBytes('\n')
				if err == nil && len(line) > 0 {
					return bytes.TrimSpace(line), nil
				}
			}
		}
	}

	// 2. Fallback to direct python3 execution if daemon is unavailable
	helperScript := getHelperPath()
	args := append([]string{helperScript}, cmdArgs...)
	out, err := exec.Command("python3", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s (%v)", strings.TrimSpace(string(out)), err)
	}
	return bytes.TrimSpace(out), nil
}

type BBox struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type ATSPINode struct {
	Depth        int      `json:"depth"`
	Role         string   `json:"role"`
	Name         string   `json:"name"`
	BBox         *BBox    `json:"bbox"`
	Center       *Point   `json:"center"`
	ScreenCenter *Point   `json:"screen_center,omitempty"`
	States       []string `json:"states"`
}

type ATSPIHelperResponse struct {
	OK        bool        `json:"ok"`
	Error     string      `json:"error,omitempty"`
	Nodes     []ATSPINode `json:"nodes,omitempty"`
	Matches   []ATSPINode `json:"matches,omitempty"`
	Total     int         `json:"total,omitempty"`
	Truncated bool        `json:"truncated,omitempty"`
}

func getHelperPath() string {
	target := filepath.Join(helperDir, "atspi_helper.py")
	if _, err := os.Stat(target); os.IsNotExist(err) {
		unpackHelpers()
	}
	return target
}

func screenToImageCoords(screenX, screenY int) (int, int) {
	st, err := loadLastScreenshotState()
	if err != nil || st == nil || st.ScaleFactor <= 0 {
		return screenX, screenY
	}
	imgX := int(math.Round(float64(screenX-st.OffsetX) * st.ScaleFactor))
	imgY := int(math.Round(float64(screenY-st.OffsetY) * st.ScaleFactor))
	return imgX, imgY
}

func resolveWindowPID(windowID string) (int, string, error) {
	if windowID == "" {
		return 0, "", nil
	}
	wins, err := listWindows()
	if err != nil {
		return 0, "", err
	}
	norm := normalizeHex(windowID)
	for _, w := range wins {
		if w.ID == norm || strings.EqualFold(w.ID, norm) {
			return w.PID, w.Title, nil
		}
	}
	return 0, "", fmt.Errorf("window '%s' not found in window list", windowID)
}

func doUITree(windowID, filter string, maxDepth, maxNodes int, interactiveOnly bool) {
	args := []string{"tree"}

	if windowID != "" {
		pid, title, err := resolveWindowPID(windowID)
		if err != nil {
			outputErrorWithFallback(fmt.Sprintf("%v. Run 'deskctl windows' to see valid window IDs.", err), "screenshot_coordinates", "Window not found or not accessible; fall back to screenshot and visual coordinates.")
		}
		if pid > 0 {
			args = append(args, "--pid", strconv.Itoa(pid))
		} else if title != "" {
			args = append(args, "--title", title)
		}
	}

	if filter != "" {
		args = append(args, "--filter", filter)
	}
	if maxDepth > 0 {
		args = append(args, "--max-depth", strconv.Itoa(maxDepth))
	}
	if maxNodes > 0 {
		args = append(args, "--max-nodes", strconv.Itoa(maxNodes))
	}
	if interactiveOnly {
		args = append(args, "--interactive-only")
	}

	out, err := queryATSPIHelper(args)
	if err != nil {
		outputError(fmt.Sprintf("Failed to query accessibility tree: %v", err))
	}

	var res ATSPIHelperResponse
	if err := json.Unmarshal(out, &res); err != nil {
		outputError(fmt.Sprintf("Invalid JSON from AT-SPI helper: %s", string(out)))
	}
	if !res.OK {
		outputErrorWithFallback(res.Error, "screenshot_coordinates", "This application does not expose an AT-SPI accessibility tree. Fall back to screenshot and visual coordinates.")
	}

	for i := range res.Nodes {
		if res.Nodes[i].Center != nil {
			sc := *res.Nodes[i].Center
			res.Nodes[i].ScreenCenter = &sc
			ix, iy := screenToImageCoords(sc.X, sc.Y)
			res.Nodes[i].Center = &Point{X: ix, Y: iy}
		}
	}

	outputJSON(map[string]interface{}{
		"ok":        true,
		"nodes":     res.Nodes,
		"count":     len(res.Nodes),
		"truncated": res.Truncated,
	})
}

func doFind(query, windowID, roleFilter string, limit int) []ATSPINode {
	args := []string{"find", query}

	if windowID != "" {
		pid, title, err := resolveWindowPID(windowID)
		if err != nil {
			outputErrorWithFallback(fmt.Sprintf("%v. Run 'deskctl windows' to see valid window IDs.", err), "screenshot_coordinates", "Window not found or not accessible; fall back to screenshot and visual coordinates.")
		}
		if pid > 0 {
			args = append(args, "--pid", strconv.Itoa(pid))
		} else if title != "" {
			args = append(args, "--title", title)
		}
	}

	if roleFilter != "" {
		args = append(args, "--role", roleFilter)
	}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}

	out, err := queryATSPIHelper(args)
	if err != nil {
		outputError(fmt.Sprintf("Failed to search accessible elements: %v", err))
	}

	var res ATSPIHelperResponse
	if err := json.Unmarshal(out, &res); err != nil {
		outputError(fmt.Sprintf("Invalid JSON from AT-SPI helper: %s", string(out)))
	}
	if !res.OK {
		outputErrorWithFallback(res.Error, "screenshot_coordinates", "This application does not expose an AT-SPI accessibility tree. Fall back to screenshot and visual coordinates.")
	}

	for i := range res.Matches {
		if res.Matches[i].Center != nil {
			sc := *res.Matches[i].Center
			res.Matches[i].ScreenCenter = &sc
			ix, iy := screenToImageCoords(sc.X, sc.Y)
			res.Matches[i].Center = &Point{X: ix, Y: iy}
		}
	}

	return res.Matches
}

func doFindCmd(query, windowID, roleFilter string, limit int) {
	matches := doFind(query, windowID, roleFilter, limit)
	outputJSON(map[string]interface{}{
		"ok":      true,
		"query":   query,
		"matches": matches,
		"count":   len(matches),
	})
}

func doClickElement(name, roleFilter, windowID, button string, double, triple, dryRun, screenshotAfter bool) {
	matches := doFind(name, windowID, roleFilter, 10)
	if len(matches) == 0 {
		recordActionFailure("click-element")
		outputError(fmt.Sprintf("Element matching %q (role=%q) not found. Try 'deskctl ui-tree' or fallback to coordinates with 'deskctl click X Y'.", name, roleFilter))
	}

	target := matches[0]
	if target.ScreenCenter == nil {
		outputError(fmt.Sprintf("Element %q found but has no clickable screen position/bounding box.", name))
	}

	logAction("click-element", []string{
		fmt.Sprintf("name=%q", target.Name),
		fmt.Sprintf("role=%s", target.Role),
		fmt.Sprintf("screen=(%d,%d)", target.ScreenCenter.X, target.ScreenCenter.Y),
	}, false, dryRun, "")

	btnNum := "1"
	switch strings.ToLower(button) {
	case "left", "1", "":
		btnNum = "1"
	case "middle", "2":
		btnNum = "2"
	case "right", "3":
		btnNum = "3"
	default:
		outputError(fmt.Sprintf("Invalid button '%s'. Allowed: left, right, middle", button))
	}

	repeat := 1
	clickType := "single"
	if triple {
		repeat = 3
		clickType = "triple"
	} else if double {
		repeat = 2
		clickType = "double"
	}

	if !dryRun {
		args := []string{"mousemove", "--", strconv.Itoa(target.ScreenCenter.X), strconv.Itoa(target.ScreenCenter.Y)}
		if repeat > 1 {
			args = append(args, "click", "--repeat", strconv.Itoa(repeat), "--delay", "50", btnNum)
		} else {
			args = append(args, "click", btnNum)
		}
		out, err := exec.Command("xdotool", args...).CombinedOutput()
		if err != nil {
			recordActionFailure("click-element")
			outputError(fmt.Sprintf("Click element execution failed: %s (%v)", strings.TrimSpace(string(out)), err))
		}
		recordActionSuccess()
		performActionWait()
	}

	resp := map[string]interface{}{
		"ok":            true,
		"action":        "click-element",
		"matched_name":  target.Name,
		"matched_role":  target.Role,
		"screen_coords": target.ScreenCenter,
		"image_coords":  target.Center,
		"button":        button,
		"click_type":    clickType,
		"dry_run":       dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}

func doSetText(query, text, windowID, roleFilter string) {
	args := []string{"set-text", text}
	if query != "" {
		args = append(args, "--query", query)
	}
	if roleFilter != "" {
		args = append(args, "--role", roleFilter)
	}
	if windowID != "" {
		pid, title, _ := resolveWindowPID(windowID)
		if pid > 0 {
			args = append(args, "--pid", strconv.Itoa(pid))
		} else if title != "" {
			args = append(args, "--title", title)
		}
	}

	out, err := queryATSPIHelper(args)
	if err != nil {
		outputError(fmt.Sprintf("set-text failed: %v", err))
	}

	var res map[string]interface{}
	_ = json.Unmarshal(out, &res)
	outputJSON(res)
}

func doGetText(query, windowID, roleFilter string) {
	args := []string{"get-text"}
	if query != "" {
		args = append(args, "--query", query)
	}
	if roleFilter != "" {
		args = append(args, "--role", roleFilter)
	}
	if windowID != "" {
		pid, title, _ := resolveWindowPID(windowID)
		if pid > 0 {
			args = append(args, "--pid", strconv.Itoa(pid))
		} else if title != "" {
			args = append(args, "--title", title)
		}
	}

	out, err := queryATSPIHelper(args)
	if err != nil {
		outputError(fmt.Sprintf("get-text failed: %v", err))
	}

	var res map[string]interface{}
	_ = json.Unmarshal(out, &res)
	outputJSON(res)
}

func doElementActions(query, windowID, roleFilter, actionName string) {
	args := []string{"actions"}
	if query != "" {
		args = append(args, "--query", query)
	}
	if roleFilter != "" {
		args = append(args, "--role", roleFilter)
	}
	if actionName != "" {
		args = append(args, "--do-action", actionName)
	}
	if windowID != "" {
		pid, title, _ := resolveWindowPID(windowID)
		if pid > 0 {
			args = append(args, "--pid", strconv.Itoa(pid))
		} else if title != "" {
			args = append(args, "--title", title)
		}
	}

	out, err := queryATSPIHelper(args)
	if err != nil {
		outputError(fmt.Sprintf("element-actions failed: %v", err))
	}

	var res map[string]interface{}
	_ = json.Unmarshal(out, &res)
	outputJSON(res)
}
