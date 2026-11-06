package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/image/draw"
)

var recPidFile = filepath.Join(deskctlDir, "recording.pid")
var recMetaFile = filepath.Join(deskctlDir, "recording_meta.json")

func doZoom(regionStr string, factor int, targetScreen, outputPath string) {
	if factor <= 1 {
		factor = 2
	}
	if factor > 8 {
		factor = 8
	}

	parts := strings.Split(regionStr, ",")
	if len(parts) != 4 {
		outputError("zoom requires --region x,y,w,h")
	}
	rx, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	ry, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	rw, err3 := strconv.Atoi(strings.TrimSpace(parts[2]))
	rh, err4 := strconv.Atoi(strings.TrimSpace(parts[3]))
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || rw <= 0 || rh <= 0 {
		outputError(fmt.Sprintf("invalid zoom region %q", regionStr))
	}

	st, err := takeScreenshotMulti(fmt.Sprintf("region:%d,%d,%d,%d", rx, ry, rw, rh), 0, "png", "")
	if err != nil {
		outputError(fmt.Sprintf("failed to capture region for zoom: %v", err))
	}

	f, err := os.Open(st.Path)
	if err != nil {
		outputError(fmt.Sprintf("failed to open region capture: %v", err))
	}
	defer f.Close()

	srcImg, err := png.Decode(f)
	if err != nil {
		outputError(fmt.Sprintf("failed to decode region capture: %v", err))
	}

	bounds := srcImg.Bounds()
	zoomedW := bounds.Dx() * factor
	zoomedH := bounds.Dy() * factor

	dst := image.NewRGBA(image.Rect(0, 0, zoomedW, zoomedH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), srcImg, bounds, draw.Over, nil)

	if outputPath == "" {
		ts := time.Now().Format("20060102_150405_000")
		outputPath = filepath.Join(deskctlDir, "screenshots", fmt.Sprintf("zoom_%s.png", ts))
	}

	outF, err := os.Create(outputPath)
	if err != nil {
		outputError(fmt.Sprintf("failed to create zoom output file: %v", err))
	}
	defer outF.Close()

	if err := png.Encode(outF, dst); err != nil {
		outputError(fmt.Sprintf("failed to encode zoomed PNG: %v", err))
	}

	outputJSON(map[string]interface{}{
		"ok":            true,
		"path":          outputPath,
		"zoom_factor":   factor,
		"region":        map[string]int{"x": rx, "y": ry, "w": rw, "h": rh},
		"zoomed_width":  zoomedW,
		"zoomed_height": zoomedH,
	})
}

func computeImageDiff(img1, img2 image.Image) (float64, *BBox) {
	b1 := img1.Bounds()
	b2 := img2.Bounds()

	minW := int(math.Min(float64(b1.Dx()), float64(b2.Dx())))
	minH := int(math.Min(float64(b1.Dy()), float64(b2.Dy())))
	totalPixels := minW * minH

	if totalPixels == 0 {
		return 0.0, nil
	}

	diffCount := 0
	minX, minY := math.MaxInt32, math.MaxInt32
	maxX, maxY := math.MinInt32, math.MinInt32

	for y := 0; y < minH; y++ {
		for x := 0; x < minW; x++ {
			r1, g1, b1, _ := img1.At(x, y).RGBA()
			r2, g2, b2, _ := img2.At(x, y).RGBA()

			diff := math.Abs(float64(r1)-float64(r2)) + math.Abs(float64(g1)-float64(g2)) + math.Abs(float64(b1)-float64(b2))
			if diff > 3000 { // slight noise tolerance
				diffCount++
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	pct := (float64(diffCount) / float64(totalPixels)) * 100.0
	var changeBox *BBox
	if diffCount > 0 {
		changeBox = &BBox{
			X: minX,
			Y: minY,
			W: maxX - minX + 1,
			H: maxY - minY + 1,
		}
	}
	return pct, changeBox
}

func doDiff(file1, file2 string, targetScreen string) {
	var img1, img2 image.Image
	var err error

	if file1 != "" && file2 != "" {
		img1, err = loadPNG(file1)
		if err != nil {
			outputError(fmt.Sprintf("Failed to load file1: %v", err))
		}
		img2, err = loadPNG(file2)
		if err != nil {
			outputError(fmt.Sprintf("Failed to load file2: %v", err))
		}
	} else {
		// Compare current screen to last screenshot
		st, err := loadLastScreenshotState()
		if err != nil || st == nil {
			outputError("No previous screenshot found. Provide --file1 and --file2 or take a screenshot first.")
		}
		img1, err = loadPNG(st.Path)
		if err != nil {
			outputError(fmt.Sprintf("Failed to load previous screenshot %s: %v", st.Path, err))
		}

		target := targetScreen
		if target == "" {
			target = st.TargetType
		}
		newSt, err := takeScreenshotMulti(target, st.ImageWidth, "png", "")
		if err != nil {
			outputError(fmt.Sprintf("Failed to capture fresh screenshot: %v", err))
		}
		img2, err = loadPNG(newSt.Path)
		if err != nil {
			outputError(fmt.Sprintf("Failed to load fresh screenshot: %v", err))
		}
	}

	pct, bbox := computeImageDiff(img1, img2)
	outputJSON(map[string]interface{}{
		"ok":             true,
		"change_percent": math.Round(pct*100) / 100,
		"changed":        pct > 0.05,
		"change_bbox":    bbox,
	})
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func doWaitForChange(targetScreen string, threshold float64, timeoutMs int) {
	if threshold <= 0 {
		threshold = 0.2
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}

	start := time.Now()
	st1, err := takeScreenshotMulti(targetScreen, 640, "png", "")
	if err != nil {
		outputError(fmt.Sprintf("Failed to take initial screenshot: %v", err))
	}
	img1, _ := loadPNG(st1.Path)

	for time.Since(start) < time.Duration(timeoutMs)*time.Millisecond {
		time.Sleep(150 * time.Millisecond)
		st2, err := takeScreenshotMulti(targetScreen, 640, "png", "")
		if err != nil {
			continue
		}
		img2, _ := loadPNG(st2.Path)
		pct, bbox := computeImageDiff(img1, img2)
		if pct >= threshold {
			outputJSON(map[string]interface{}{
				"ok":             true,
				"changed":        true,
				"change_percent": math.Round(pct*100) / 100,
				"change_bbox":    bbox,
				"elapsed_ms":     time.Since(start).Milliseconds(),
			})
			return
		}
	}

	outputJSON(map[string]interface{}{
		"ok":             true,
		"changed":        false,
		"change_percent": 0.0,
		"elapsed_ms":     time.Since(start).Milliseconds(),
		"timeout_ms":     timeoutMs,
	})
}

func doWaitForStable(targetScreen string, threshold float64, timeoutMs int) {
	if threshold <= 0 {
		threshold = 5.0
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}

	start := time.Now()
	for time.Since(start) < time.Duration(timeoutMs)*time.Millisecond {
		st1, err := takeScreenshotMulti(targetScreen, 640, "png", "")
		if err != nil {
			continue
		}
		img1, _ := loadPNG(st1.Path)

		time.Sleep(200 * time.Millisecond)

		st2, err := takeScreenshotMulti(targetScreen, 640, "png", "")
		if err != nil {
			continue
		}
		img2, _ := loadPNG(st2.Path)

		pct, _ := computeImageDiff(img1, img2)
		if pct <= threshold {
			outputJSON(map[string]interface{}{
				"ok":         true,
				"stable":     true,
				"elapsed_ms": time.Since(start).Milliseconds(),
			})
			return
		}
	}

	outputJSON(map[string]interface{}{
		"ok":         false,
		"stable":     false,
		"error":      fmt.Sprintf("Screen did not stabilize within %d ms", timeoutMs),
		"elapsed_ms": time.Since(start).Milliseconds(),
	})
}

func doColorAtPixel(imgX, imgY int, screenArg string) {
	screenX, screenY, _, err := mapImageToScreenCoords(imgX, imgY, screenArg)
	if err != nil {
		outputError(fmt.Sprintf("Coordinate mapping failed: %v", err))
	}

	tempShot := filepath.Join(deskctlDir, fmt.Sprintf("pixel_%d.png", time.Now().UnixNano()))
	out, err := exec.Command("maim", "-g", fmt.Sprintf("1x1+%d+%d", screenX, screenY), tempShot).CombinedOutput()
	if err != nil {
		outputError(fmt.Sprintf("Failed to capture pixel: %s (%v)", strings.TrimSpace(string(out)), err))
	}
	defer os.Remove(tempShot)

	img, err := loadPNG(tempShot)
	if err != nil {
		outputError(fmt.Sprintf("Failed to decode pixel image: %v", err))
	}

	rgba := color.RGBAModel.Convert(img.At(0, 0)).(color.RGBA)
	hexStr := fmt.Sprintf("#%02X%02X%02X", rgba.R, rgba.G, rgba.B)

	outputJSON(map[string]interface{}{
		"ok":            true,
		"screen_coords": map[string]int{"x": screenX, "y": screenY},
		"rgb":           []int{int(rgba.R), int(rgba.G), int(rgba.B)},
		"hex":           hexStr,
	})
}

func doFindImage(templatePath, targetScreen string, threshold float64) {
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		outputError(fmt.Sprintf("Template image %q does not exist", templatePath))
	}

	st, err := takeScreenshotMulti(targetScreen, 0, "png", "")
	if err != nil {
		outputError(fmt.Sprintf("Failed to capture screenshot: %v", err))
	}

	helperScript := filepath.Join(helperDir, "vision_helper.py")
	if _, err := os.Stat(helperScript); os.IsNotExist(err) {
		unpackHelpers()
	}

	args := []string{helperScript, "find-image", st.Path, templatePath}
	if threshold > 0 {
		args = append(args, "--threshold", fmt.Sprintf("%.2f", threshold))
	}

	out, err := exec.Command("python3", args...).CombinedOutput()
	if err != nil {
		outputError(fmt.Sprintf("Vision helper failed: %s (%v)", strings.TrimSpace(string(out)), err))
	}

	var res map[string]interface{}
	if err := json.Unmarshal(out, &res); err != nil {
		outputError(fmt.Sprintf("Invalid JSON from vision helper: %s", string(out)))
	}

	res["screenshot_path"] = st.Path
	outputJSON(res)
}

func doRecordScreen(action, screenArg, outputPath string, fps int) {
	switch strings.ToLower(action) {
	case "start":
		if data, err := os.ReadFile(recPidFile); err == nil {
			pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
			if pid > 0 {
				if proc, err := os.FindProcess(pid); err == nil && proc.Signal(syscall.Signal(0)) == nil {
					outputJSON(map[string]interface{}{
						"ok":        true,
						"recording": true,
						"pid":       pid,
						"message":   "Screen recording is already running",
					})
					return
				}
			}
		}

		screens, vDesk, err := detectScreens()
		if err != nil {
			outputError(fmt.Sprintf("Failed to detect displays: %v", err))
		}

		x, y, w, h := vDesk.X, vDesk.Y, vDesk.W, vDesk.H
		if screenArg != "" && screenArg != "all" {
			for _, s := range screens {
				if strconv.Itoa(s.ID) == screenArg || strings.EqualFold(s.Name, screenArg) {
					x, y, w, h = s.X, s.Y, s.W, s.H
					break
				}
			}
		}

		if fps <= 0 {
			fps = 15
		}
		if outputPath == "" {
			ts := time.Now().Format("20060102_150405")
			outputPath = filepath.Join(deskctlDir, "recordings", fmt.Sprintf("rec_%s.mp4", ts))
		}
		_ = os.MkdirAll(filepath.Dir(outputPath), 0755)

		// ffmpeg x11grab
		disp := os.Getenv("DISPLAY")
		if disp == "" {
			disp = ":0"
		}
		grabInput := fmt.Sprintf("%s.0+%d,%d", disp, x, y)
		cmd := exec.Command("ffmpeg", "-y", "-f", "x11grab", "-draw_mouse", "1",
			"-video_size", fmt.Sprintf("%dx%d", w, h),
			"-framerate", strconv.Itoa(fps),
			"-i", grabInput,
			"-c:v", "libx264", "-preset", "ultrafast", outputPath)

		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			outputError(fmt.Sprintf("Failed to start ffmpeg recording: %v", err))
		}

		pid := cmd.Process.Pid
		_ = os.WriteFile(recPidFile, []byte(strconv.Itoa(pid)), 0644)
		meta, _ := json.Marshal(map[string]interface{}{
			"pid":        pid,
			"path":       outputPath,
			"start_time": time.Now().Format(time.RFC3339),
			"fps":        fps,
		})
		_ = os.WriteFile(recMetaFile, meta, 0644)

		outputJSON(map[string]interface{}{
			"ok":        true,
			"recording": true,
			"pid":       pid,
			"output":    outputPath,
			"geometry":  fmt.Sprintf("%dx%d+%d+%d", w, h, x, y),
		})

	case "stop":
		data, err := os.ReadFile(recPidFile)
		if err != nil {
			outputJSON(map[string]interface{}{
				"ok":        true,
				"recording": false,
				"message":   "No active recording found",
			})
			return
		}
		pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		_ = os.Remove(recPidFile)

		if pid > 0 {
			if proc, err := os.FindProcess(pid); err == nil {
				_ = proc.Signal(syscall.SIGINT) // SIGINT tells ffmpeg to finalize container
				time.Sleep(500 * time.Millisecond)
			}
		}

		outPath := ""
		if metaData, err := os.ReadFile(recMetaFile); err == nil {
			var m map[string]interface{}
			_ = json.Unmarshal(metaData, &m)
			if p, ok := m["path"].(string); ok {
				outPath = p
			}
		}
		_ = os.Remove(recMetaFile)

		outputJSON(map[string]interface{}{
			"ok":        true,
			"recording": false,
			"path":      outPath,
			"file":      outPath,
			"message":   "Recording finalized and saved",
		})

	case "status":
		recording := false
		pid := 0
		if data, err := os.ReadFile(recPidFile); err == nil {
			p, _ := strconv.Atoi(strings.TrimSpace(string(data)))
			if p > 0 {
				if proc, err := os.FindProcess(p); err == nil && proc.Signal(syscall.Signal(0)) == nil {
					recording = true
					pid = p
				}
			}
		}
		outputJSON(map[string]interface{}{
			"ok":        true,
			"recording": recording,
			"pid":       pid,
		})
	default:
		outputError("Unknown record-screen action. Allowed: start, stop, status")
	}
}
