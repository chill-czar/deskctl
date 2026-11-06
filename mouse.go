package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func performActionWait() {
	time.Sleep(150 * time.Millisecond)
}

func maybeTakeScreenshotAfter(enabled bool) map[string]interface{} {
	if !enabled {
		return nil
	}
	st, err := takeScreenshotMulti("all", 0, "png", "")
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to take screenshot-after: %v", err),
		}
	}
	return map[string]interface{}{
		"path":         st.Path,
		"image_width":  st.ImageWidth,
		"image_height": st.ImageHeight,
		"scale_factor": st.ScaleFactor,
	}
}

func attachScreenshotAfter(resp map[string]interface{}, enabled bool) {
	if shot := maybeTakeScreenshotAfter(enabled); shot != nil {
		resp["screenshot"] = shot
		if p, ok := shot["path"]; ok {
			resp["screenshot_after"] = p
		}
	}
}

func doClick(imgX, imgY int, button string, double, triple, dryRun, screenshotAfter bool, screenArg string) {
	screenX, screenY, scale, err := mapImageToScreenCoords(imgX, imgY, screenArg)
	if err != nil {
		outputError(fmt.Sprintf("Coordinate mapping failed: %v", err))
	}

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

	logAction("click", []string{
		fmt.Sprintf("img=(%d,%d)", imgX, imgY),
		fmt.Sprintf("screen=(%d,%d)", screenX, screenY),
		fmt.Sprintf("btn=%s", button),
		fmt.Sprintf("type=%s", clickType),
		fmt.Sprintf("screen_arg=%s", screenArg),
	}, false, dryRun, "")

	if !dryRun {
		args := []string{"mousemove", "--", fmt.Sprintf("%d", screenX), fmt.Sprintf("%d", screenY)}
		if repeat > 1 {
			args = append(args, "click", "--repeat", fmt.Sprintf("%d", repeat), "--delay", "50", btnNum)
		} else {
			args = append(args, "click", btnNum)
		}

		out, err := exec.Command("xdotool", args...).CombinedOutput()
		if err != nil {
			recordActionFailure("click")
			outputError(fmt.Sprintf("xdotool click failed: %s (%v)", strings.TrimSpace(string(out)), err))
		}
		recordActionSuccess()
		performActionWait()
	}

	resp := map[string]interface{}{
		"ok":            true,
		"action":        "click",
		"button":        button,
		"click_type":    clickType,
		"image_coords":  map[string]int{"x": imgX, "y": imgY},
		"screen_coords": map[string]int{"x": screenX, "y": screenY},
		"scale_factor":  scale,
		"dry_run":       dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}

func doMove(imgX, imgY int, dryRun, screenshotAfter bool, screenArg string) {
	screenX, screenY, scale, err := mapImageToScreenCoords(imgX, imgY, screenArg)
	if err != nil {
		outputError(fmt.Sprintf("Coordinate mapping failed: %v", err))
	}

	logAction("move", []string{
		fmt.Sprintf("img=(%d,%d)", imgX, imgY),
		fmt.Sprintf("screen=(%d,%d)", screenX, screenY),
		fmt.Sprintf("screen_arg=%s", screenArg),
	}, false, dryRun, "")

	if !dryRun {
		out, err := exec.Command("xdotool", "mousemove", "--", fmt.Sprintf("%d", screenX), fmt.Sprintf("%d", screenY)).CombinedOutput()
		if err != nil {
			recordActionFailure("move")
			outputError(fmt.Sprintf("xdotool move failed: %s (%v)", strings.TrimSpace(string(out)), err))
		}
		recordActionSuccess()
		performActionWait()
	}

	resp := map[string]interface{}{
		"ok":            true,
		"action":        "move",
		"image_coords":  map[string]int{"x": imgX, "y": imgY},
		"screen_coords": map[string]int{"x": screenX, "y": screenY},
		"scale_factor":  scale,
		"dry_run":       dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}

func doDrag(imgX1, imgY1, imgX2, imgY2 int, dryRun, screenshotAfter bool, screenArg string) {
	screenX1, screenY1, _, err := mapImageToScreenCoords(imgX1, imgY1, screenArg)
	if err != nil {
		outputError(fmt.Sprintf("Coordinate mapping failed for start: %v", err))
	}
	screenX2, screenY2, scale, err := mapImageToScreenCoords(imgX2, imgY2, screenArg)
	if err != nil {
		outputError(fmt.Sprintf("Coordinate mapping failed for end: %v", err))
	}

	logAction("drag", []string{
		fmt.Sprintf("from_screen=(%d,%d)", screenX1, screenY1),
		fmt.Sprintf("to_screen=(%d,%d)", screenX2, screenY2),
		fmt.Sprintf("screen_arg=%s", screenArg),
	}, false, dryRun, "")

	if !dryRun {
		args := []string{
			"mousemove", "--", fmt.Sprintf("%d", screenX1), fmt.Sprintf("%d", screenY1),
			"mousedown", "1",
			"sleep", "0.05",
		}
		steps := 5
		for i := 1; i <= steps; i++ {
			interX := screenX1 + (screenX2-screenX1)*i/steps
			interY := screenY1 + (screenY2-screenY1)*i/steps
			args = append(args, "mousemove", "--", fmt.Sprintf("%d", interX), fmt.Sprintf("%d", interY), "sleep", "0.02")
		}
		args = append(args, "mouseup", "1")

		out, err := exec.Command("xdotool", args...).CombinedOutput()
		if err != nil {
			recordActionFailure("drag")
			outputError(fmt.Sprintf("xdotool drag failed: %s (%v)", strings.TrimSpace(string(out)), err))
		}
		recordActionSuccess()
		performActionWait()
	}

	resp := map[string]interface{}{
		"ok":     true,
		"action": "drag",
		"from": map[string]interface{}{
			"image":  map[string]int{"x": imgX1, "y": imgY1},
			"screen": map[string]int{"x": screenX1, "y": screenY1},
		},
		"to": map[string]interface{}{
			"image":  map[string]int{"x": imgX2, "y": imgY2},
			"screen": map[string]int{"x": screenX2, "y": screenY2},
		},
		"scale_factor": scale,
		"dry_run":      dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}

func doDragPath(pointsStr string, dryRun, screenshotAfter bool, screenArg string) {
	// Format: "x1,y1 x2,y2 x3,y3"
	coords := strings.Fields(pointsStr)
	if len(coords) < 2 {
		outputError("drag-path requires at least 2 coordinate pairs (e.g. \"100,200 200,300 300,400\")")
	}

	type Waypoint struct {
		ImgX, ImgY       int
		ScreenX, ScreenY int
	}
	var waypoints []Waypoint

	for _, pt := range coords {
		parts := strings.Split(pt, ",")
		if len(parts) != 2 {
			outputError(fmt.Sprintf("invalid waypoint %q in drag-path", pt))
		}
		ix, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		iy, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		sx, sy, _, err := mapImageToScreenCoords(ix, iy, screenArg)
		if err != nil {
			outputError(fmt.Sprintf("Coordinate mapping failed for waypoint %q: %v", pt, err))
		}
		waypoints = append(waypoints, Waypoint{ImgX: ix, ImgY: iy, ScreenX: sx, ScreenY: sy})
	}

	logAction("drag-path", []string{fmt.Sprintf("points=%d", len(waypoints))}, false, dryRun, "")

	if !dryRun {
		args := []string{
			"mousemove", "--", fmt.Sprintf("%d", waypoints[0].ScreenX), fmt.Sprintf("%d", waypoints[0].ScreenY),
			"mousedown", "1",
			"sleep", "0.05",
		}
		for i := 1; i < len(waypoints); i++ {
			args = append(args, "mousemove", "--", fmt.Sprintf("%d", waypoints[i].ScreenX), fmt.Sprintf("%d", waypoints[i].ScreenY), "sleep", "0.03")
		}
		args = append(args, "mouseup", "1")

		out, err := exec.Command("xdotool", args...).CombinedOutput()
		if err != nil {
			recordActionFailure("drag-path")
			outputError(fmt.Sprintf("xdotool drag-path failed: %s (%v)", strings.TrimSpace(string(out)), err))
		}
		recordActionSuccess()
		performActionWait()
	}

	resp := map[string]interface{}{
		"ok":        true,
		"action":    "drag-path",
		"waypoints": waypoints,
		"dry_run":   dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}

func doHoldAndRelease(imgX, imgY int, button string, holdMs int, dryRun, screenshotAfter bool, screenArg string) {
	screenX, screenY, scale, err := mapImageToScreenCoords(imgX, imgY, screenArg)
	if err != nil {
		outputError(fmt.Sprintf("Coordinate mapping failed: %v", err))
	}

	if holdMs <= 0 {
		holdMs = 500
	}
	btnNum := "1"
	if button == "right" {
		btnNum = "3"
	} else if button == "middle" {
		btnNum = "2"
	}

	logAction("hold-and-release", []string{fmt.Sprintf("btn=%s", button), fmt.Sprintf("hold_ms=%d", holdMs)}, false, dryRun, "")

	if !dryRun {
		args := []string{
			"mousemove", "--", fmt.Sprintf("%d", screenX), fmt.Sprintf("%d", screenY),
			"mousedown", btnNum,
			"sleep", fmt.Sprintf("%.3f", float64(holdMs)/1000.0),
			"mouseup", btnNum,
		}
		out, err := exec.Command("xdotool", args...).CombinedOutput()
		if err != nil {
			outputError(fmt.Sprintf("xdotool hold failed: %s (%v)", strings.TrimSpace(string(out)), err))
		}
		performActionWait()
	}

	resp := map[string]interface{}{
		"ok":            true,
		"action":        "hold-and-release",
		"button":        button,
		"hold_ms":       holdMs,
		"screen_coords": map[string]int{"x": screenX, "y": screenY},
		"scale_factor":  scale,
		"dry_run":       dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}

func doHover(imgX, imgY int, dwellMs int, dryRun, screenshotAfter bool, screenArg string) {
	screenX, screenY, scale, err := mapImageToScreenCoords(imgX, imgY, screenArg)
	if err != nil {
		outputError(fmt.Sprintf("Coordinate mapping failed: %v", err))
	}
	if dwellMs <= 0 {
		dwellMs = 500
	}

	logAction("hover", []string{fmt.Sprintf("dwell_ms=%d", dwellMs)}, false, dryRun, "")

	if !dryRun {
		_ = exec.Command("xdotool", "mousemove", "--", fmt.Sprintf("%d", screenX), fmt.Sprintf("%d", screenY)).Run()
		time.Sleep(time.Duration(dwellMs) * time.Millisecond)
	}

	resp := map[string]interface{}{
		"ok":            true,
		"action":        "hover",
		"dwell_ms":      dwellMs,
		"screen_coords": map[string]int{"x": screenX, "y": screenY},
		"scale_factor":  scale,
		"dry_run":       dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}

func doMousePosition() {
	x, y, err := getMousePosition()
	if err != nil {
		outputError(err.Error())
	}
	screens, _, _ := detectScreens()
	screenID := -1
	screenName := "unknown"
	for _, s := range screens {
		if x >= s.X && x < s.X+s.W && y >= s.Y && y < s.Y+s.H {
			screenID = s.ID
			screenName = s.Name
			break
		}
	}
	outputJSON(map[string]interface{}{
		"ok":          true,
		"x":           x,
		"y":           y,
		"screen_id":   screenID,
		"screen_name": screenName,
	})
}

func doScroll(imgX, imgY int, direction string, amount int, dryRun, screenshotAfter bool, screenArg string) {
	screenX, screenY, scale, err := mapImageToScreenCoords(imgX, imgY, screenArg)
	if err != nil {
		outputError(fmt.Sprintf("Coordinate mapping failed: %v", err))
	}

	if amount <= 0 {
		amount = 1
	}

	btn := "5"
	switch strings.ToLower(direction) {
	case "up":
		btn = "4"
	case "down":
		btn = "5"
	case "left":
		btn = "6"
	case "right":
		btn = "7"
	default:
		outputError(fmt.Sprintf("Invalid scroll direction '%s'. Allowed: up, down, left, right", direction))
	}

	logAction("scroll", []string{
		fmt.Sprintf("screen=(%d,%d)", screenX, screenY),
		fmt.Sprintf("dir=%s", direction),
		fmt.Sprintf("amount=%d", amount),
	}, false, dryRun, "")

	if !dryRun {
		args := []string{
			"mousemove", "--", fmt.Sprintf("%d", screenX), fmt.Sprintf("%d", screenY),
			"click", "--repeat", fmt.Sprintf("%d", amount), "--delay", "30", btn,
		}
		out, err := exec.Command("xdotool", args...).CombinedOutput()
		if err != nil {
			recordActionFailure("scroll")
			outputError(fmt.Sprintf("xdotool scroll failed: %s (%v)", strings.TrimSpace(string(out)), err))
		}
		recordActionSuccess()
		performActionWait()
	}

	resp := map[string]interface{}{
		"ok":            true,
		"action":        "scroll",
		"direction":     direction,
		"amount":        amount,
		"screen_coords": map[string]int{"x": screenX, "y": screenY},
		"scale_factor":  scale,
		"dry_run":       dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}
