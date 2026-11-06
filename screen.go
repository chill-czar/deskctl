package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/draw"
)

type ScreenInfo struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	W       int     `json:"w"`
	H       int     `json:"h"`
	Scale   float64 `json:"scale"`
	Primary bool    `json:"primary"`
	Active  bool    `json:"active"`
}

type VirtualDesktopInfo struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type ScreenshotState struct {
	Path           string              `json:"path"`
	TargetType     string              `json:"target_type"` // all, screen, window, region
	ScreenID       int                 `json:"screen_id"`
	ScreenName     string              `json:"screen_name"`
	ImageWidth     int                 `json:"image_width"`
	ImageHeight    int                 `json:"image_height"`
	CapturedWidth  int                 `json:"captured_width"`
	CapturedHeight int                 `json:"captured_height"`
	ScaleFactor    float64             `json:"scale_factor"`
	OffsetX        int                 `json:"offset_x"`
	OffsetY        int                 `json:"offset_y"`
	Format         string              `json:"format"`
	Screen         *ScreenInfo         `json:"screen,omitempty"`
	VirtualDesktop *VirtualDesktopInfo `json:"virtual_desktop,omitempty"`
	Timestamp      string              `json:"timestamp"`
}

func getDisplayScale() float64 {
	out, err := exec.Command("gsettings", "get", "org.cinnamon.desktop.interface", "text-scaling-factor").Output()
	if err == nil {
		s := strings.TrimSpace(string(out))
		f, err := strconv.ParseFloat(s, 64)
		if err == nil && f > 0 {
			return f
		}
	}
	return 1.0
}

func getMousePosition() (int, int, error) {
	out, err := exec.Command("xdotool", "getmouselocation").Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get mouse location: %w", err)
	}
	re := regexp.MustCompile(`x:(-?\d+)\s+y:(-?\d+)`)
	m := re.FindStringSubmatch(string(out))
	if len(m) == 3 {
		x, _ := strconv.Atoi(m[1])
		y, _ := strconv.Atoi(m[2])
		return x, y, nil
	}
	return 0, 0, fmt.Errorf("invalid xdotool getmouselocation output: %s", string(out))
}

// detectScreens queries xrandr at runtime for all connected monitors.
// Never assumes hardcoded screen counts, resolutions, or positive coordinates.
func detectScreens() ([]ScreenInfo, *VirtualDesktopInfo, error) {
	out, err := exec.Command("xrandr", "--listmonitors").CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to run xrandr --listmonitors: %s (%v)", strings.TrimSpace(string(out)), err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return nil, nil, fmt.Errorf("no monitors reported by xrandr")
	}

	sysScale := getDisplayScale()

	// Example line:
	// 0: +*eDP-1-1 1920/344x1080/193+1920+0  eDP-1-1
	// 1: +HDMI-1-1 1920/344x1080/193+0+0  HDMI-1-1
	// 1: VIRTUAL-TEST 1024/200x768/150+100+100
	re := regexp.MustCompile(`^\s*(\d+):\s+([\+\*]+)?(\S+)\s+(\d+)/\d+x(\d+)/\d+([+-]\d+)([+-]\d+)(?:\s+(\S+))?`)

	var screens []ScreenInfo
	minX, minY := math.MaxInt32, math.MaxInt32
	maxX, maxY := math.MinInt32, math.MinInt32

	for _, line := range lines[1:] {
		m := re.FindStringSubmatch(line)
		if len(m) >= 8 {
			idx, _ := strconv.Atoi(m[1])
			flags := m[2]
			shortName := m[3]
			w, _ := strconv.Atoi(m[4])
			h, _ := strconv.Atoi(m[5])
			x, _ := strconv.Atoi(m[6])
			y, _ := strconv.Atoi(m[7])
			name := m[8]
			if name == "" {
				name = shortName
			}
			isPrimary := strings.Contains(flags, "*")

			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x+w > maxX {
				maxX = x + w
			}
			if y+h > maxY {
				maxY = y + h
			}

			screens = append(screens, ScreenInfo{
				ID:      idx,
				Name:    name,
				X:       x,
				Y:       y,
				W:       w,
				H:       h,
				Scale:   sysScale,
				Primary: isPrimary,
				Active:  false,
			})
		}
	}

	if len(screens) == 0 {
		return nil, nil, fmt.Errorf("could not parse any monitors from xrandr output")
	}

	vDesk := &VirtualDesktopInfo{
		X: minX,
		Y: minY,
		W: maxX - minX,
		H: maxY - minY,
	}

	// Determine active screen based on cursor position or primary monitor.
	// We avoid querying window list here to prevent mutual recursion with listWindows().
	activeSet := false
	curX, curY, err := getMousePosition()
	if err == nil {
		for i := range screens {
			s := screens[i]
			if curX >= s.X && curX < s.X+s.W && curY >= s.Y && curY < s.Y+s.H {
				screens[i].Active = true
				activeSet = true
				break
			}
		}
	}

	if !activeSet && len(screens) > 0 {
		// Default to primary or first monitor
		for i := range screens {
			if screens[i].Primary {
				screens[i].Active = true
				activeSet = true
				break
			}
		}
		if !activeSet {
			screens[0].Active = true
		}
	}

	return screens, vDesk, nil
}

func doScreens() {
	screens, vDesk, err := detectScreens()
	if err != nil {
		outputError(err.Error())
	}
	outputJSON(map[string]interface{}{
		"ok":              true,
		"screens":         screens,
		"virtual_desktop": vDesk,
		"count":           len(screens),
	})
}

func loadLastScreenshotState() (*ScreenshotState, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}
	var st ScreenshotState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func saveScreenshotState(st *ScreenshotState) {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(stateFile, data, 0644)
}

// mapImageToScreenCoords translates image-space coordinates to global virtual-desktop coordinates.
// If screenArg is provided (e.g. "--screen 1"), it looks up that screen's geometry.
// Otherwise, it converts via the state of the last screenshot.
func mapImageToScreenCoords(imgX, imgY int, screenArg string) (int, int, float64, error) {
	screens, vDesk, err := detectScreens()
	if err != nil {
		return imgX, imgY, 1.0, err
	}

	if screenArg != "" {
		for _, s := range screens {
			if strconv.Itoa(s.ID) == screenArg || strings.EqualFold(s.Name, screenArg) {
				st, _ := loadLastScreenshotState()
				scale := 1.0
				if st != nil && st.ScreenID == s.ID && st.ScaleFactor > 0 {
					scale = st.ScaleFactor
				}
				realX := s.X + int(math.Round(float64(imgX)/scale))
				realY := s.Y + int(math.Round(float64(imgY)/scale))
				return realX, realY, scale, nil
			}
		}
	}

	st, err := loadLastScreenshotState()
	if err != nil || st == nil || st.ScaleFactor <= 0 {
		return imgX, imgY, 1.0, nil
	}

	realX := st.OffsetX + int(math.Round(float64(imgX)/st.ScaleFactor))
	realY := st.OffsetY + int(math.Round(float64(imgY)/st.ScaleFactor))
	_ = vDesk
	return realX, realY, st.ScaleFactor, nil
}

// takeScreenshotMulti captures a screenshot according to target specification:
// target: ID | all | active | primary | window:ID | region:x,y,w,h
func takeScreenshotMulti(target string, maxWidth int, format string, outputPath string) (*ScreenshotState, error) {
	screens, vDesk, err := detectScreens()
	if err != nil {
		return nil, err
	}

	target = strings.TrimSpace(target)
	if target == "" {
		target = "all"
	}

	var maimArgs []string
	var selectedScreen *ScreenInfo
	targetType := "screen"
	offsetX := 0
	offsetY := 0
	targetW := 0
	targetH := 0
	screenID := -1
	screenName := ""

	lowerTarget := strings.ToLower(target)

	if lowerTarget == "all" {
		targetType = "all"
		offsetX = vDesk.X
		offsetY = vDesk.Y
		targetW = vDesk.W
		targetH = vDesk.H
		maimArgs = append(maimArgs, "-g", fmt.Sprintf("%dx%d+%d+%d", vDesk.W, vDesk.H, vDesk.X, vDesk.Y))
	} else if lowerTarget == "primary" {
		for _, s := range screens {
			if s.Primary {
				selectedScreen = &s
				break
			}
		}
		if selectedScreen == nil && len(screens) > 0 {
			selectedScreen = &screens[0]
		}
	} else if lowerTarget == "active" {
		for _, s := range screens {
			if s.Active {
				selectedScreen = &s
				break
			}
		}
		if selectedScreen == nil && len(screens) > 0 {
			selectedScreen = &screens[0]
		}
	} else if strings.HasPrefix(lowerTarget, "window:") {
		targetType = "window"
		winArg := strings.TrimPrefix(target, "window:")
		winArg = strings.TrimPrefix(winArg, "WINDOW:")
		winArg = strings.TrimSpace(winArg)

		var targetWin *WindowInfo
		if strings.EqualFold(winArg, "active") || winArg == "" {
			targetWin = getActiveWindowInfo()
		} else {
			wins, _ := listWindows()
			norm := normalizeHex(winArg)
			for _, w := range wins {
				if w.ID == norm || strings.EqualFold(w.ID, norm) {
					targetWin = &w
					break
				}
			}
		}

		if targetWin == nil {
			return nil, fmt.Errorf("window '%s' not found for screenshot", winArg)
		}

		offsetX = targetWin.X
		offsetY = targetWin.Y
		targetW = targetWin.W
		targetH = targetWin.H

		// maim -i expects decimal window ID, not hex 0x...
		decID := targetWin.ID
		if val, err := strconv.ParseUint(strings.TrimPrefix(strings.ToLower(targetWin.ID), "0x"), 16, 64); err == nil {
			decID = strconv.FormatUint(val, 10)
		}
		maimArgs = append(maimArgs, "-i", decID)
	} else if strings.HasPrefix(lowerTarget, "region:") {
		targetType = "region"
		regStr := strings.TrimPrefix(target, "region:")
		regStr = strings.TrimPrefix(regStr, "REGION:")
		parts := strings.Split(regStr, ",")
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid region format, expected region:x,y,w,h (got %s)", regStr)
		}
		rx, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		ry, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		rw, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
		rh, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
		if rw <= 0 || rh <= 0 {
			return nil, fmt.Errorf("region width and height must be positive")
		}
		offsetX = rx
		offsetY = ry
		targetW = rw
		targetH = rh
		maimArgs = append(maimArgs, "-g", fmt.Sprintf("%dx%d+%d+%d", rw, rh, rx, ry))
	} else {
		// Target is a screen ID or name
		for _, s := range screens {
			if strconv.Itoa(s.ID) == target || strings.EqualFold(s.Name, target) {
				selectedScreen = &s
				break
			}
		}
		if selectedScreen == nil {
			return nil, fmt.Errorf("screen '%s' not found. Available screens: %s", target, listScreenNames(screens))
		}
	}

	if selectedScreen != nil {
		targetType = "screen"
		screenID = selectedScreen.ID
		screenName = selectedScreen.Name
		offsetX = selectedScreen.X
		offsetY = selectedScreen.Y
		targetW = selectedScreen.W
		targetH = selectedScreen.H
		maimArgs = append(maimArgs, "-g", fmt.Sprintf("%dx%d+%d+%d", selectedScreen.W, selectedScreen.H, selectedScreen.X, selectedScreen.Y))
	}

	tempShot := filepath.Join(deskctlDir, fmt.Sprintf("tmp_shot_%d.png", time.Now().UnixNano()))
	maimArgs = append(maimArgs, tempShot)

	out, err := exec.Command("maim", maimArgs...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("maim screenshot capture failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	defer os.Remove(tempShot)

	inFile, err := os.Open(tempShot)
	if err != nil {
		return nil, fmt.Errorf("failed to open raw screenshot: %w", err)
	}
	defer inFile.Close()

	img, err := png.Decode(inFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode captured screenshot: %w", err)
	}

	bounds := img.Bounds()
	curW := bounds.Dx()
	curH := bounds.Dy()

	if targetW == 0 {
		targetW = curW
	}
	if targetH == 0 {
		targetH = curH
	}

	finalImg := img
	finalW := curW
	finalH := curH

	if maxWidth > 0 && curW > maxWidth {
		finalW = maxWidth
		finalH = int(math.Round(float64(curH) * float64(maxWidth) / float64(curW)))
		dst := image.NewRGBA(image.Rect(0, 0, finalW, finalH))
		draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
		finalImg = dst
	}

	fmtLower := strings.ToLower(strings.TrimSpace(format))
	if fmtLower == "" {
		fmtLower = "png"
	}
	if fmtLower != "png" && fmtLower != "jpeg" && fmtLower != "jpg" {
		return nil, fmt.Errorf("unsupported format '%s', allowed: png, jpeg", format)
	}

	ext := "png"
	if fmtLower == "jpeg" || fmtLower == "jpg" {
		ext = "jpg"
	}

	if outputPath == "" {
		ts := time.Now().Format("20060102_150405_000")
		outputPath = filepath.Join(deskctlDir, "screenshots", fmt.Sprintf("shot_%s.%s", ts, ext))
	} else {
		_ = os.MkdirAll(filepath.Dir(outputPath), 0755)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file '%s': %w", outputPath, err)
	}
	defer outFile.Close()

	if ext == "jpg" {
		if err := jpeg.Encode(outFile, finalImg, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("failed to encode JPEG: %w", err)
		}
	} else {
		if err := png.Encode(outFile, finalImg); err != nil {
			return nil, fmt.Errorf("failed to encode PNG: %w", err)
		}
	}

	scaleFactor := float64(finalW) / float64(targetW)

	state := &ScreenshotState{
		Path:           outputPath,
		TargetType:     targetType,
		ScreenID:       screenID,
		ScreenName:     screenName,
		ImageWidth:     finalW,
		ImageHeight:    finalH,
		CapturedWidth:  targetW,
		CapturedHeight: targetH,
		ScaleFactor:    scaleFactor,
		OffsetX:        offsetX,
		OffsetY:        offsetY,
		Format:         fmtLower,
		Screen:         selectedScreen,
		VirtualDesktop: vDesk,
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	saveScreenshotState(state)
	return state, nil
}

func listScreenNames(screens []ScreenInfo) string {
	var names []string
	for _, s := range screens {
		names = append(names, fmt.Sprintf("%d (%s)", s.ID, s.Name))
	}
	return strings.Join(names, ", ")
}
