package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type OCRWord struct {
	Text         string  `json:"text"`
	BBox         BBox    `json:"bbox"`
	Center       Point   `json:"center"`
	ScreenCenter Point   `json:"screen_center"`
	Confidence   float64 `json:"confidence"`
	LineNum      int     `json:"line_num"`
	BlockNum     int     `json:"block_num"`
}

func parseTesseractTSV(tsvData []byte, offsetX, offsetY int, scaleFactor float64) ([]OCRWord, error) {
	lines := strings.Split(string(tsvData), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("no OCR output from tesseract")
	}

	var words []OCRWord
	// header: level, page_num, block_num, par_num, line_num, word_num, left, top, width, height, conf, text
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) < 12 {
			continue
		}

		text := strings.TrimSpace(cols[11])
		if text == "" {
			continue
		}

		conf, _ := strconv.ParseFloat(cols[10], 64)
		if conf < 0 {
			continue
		}

		left, _ := strconv.Atoi(cols[6])
		top, _ := strconv.Atoi(cols[7])
		w, _ := strconv.Atoi(cols[8])
		h, _ := strconv.Atoi(cols[9])
		lineNum, _ := strconv.Atoi(cols[4])
		blockNum, _ := strconv.Atoi(cols[2])

		centerImgX := left + w/2
		centerImgY := top + h/2

		// Real screen coordinates
		screenCenterX := offsetX + int(float64(centerImgX)/scaleFactor)
		screenCenterY := offsetY + int(float64(centerImgY)/scaleFactor)

		words = append(words, OCRWord{
			Text: text,
			BBox: BBox{
				X: left,
				Y: top,
				W: w,
				H: h,
			},
			Center: Point{
				X: centerImgX,
				Y: centerImgY,
			},
			ScreenCenter: Point{
				X: screenCenterX,
				Y: screenCenterY,
			},
			Confidence: conf,
			LineNum:    lineNum,
			BlockNum:   blockNum,
		})
	}
	return words, nil
}

func runOCR(targetScreen, region string) ([]OCRWord, *ScreenshotState, error) {
	shotTarget := targetScreen
	if region != "" {
		shotTarget = "region:" + region
	}
	if shotTarget == "" {
		shotTarget = "active"
	}

	st, err := takeScreenshotMulti(shotTarget, 0, "png", "")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to take screenshot for OCR: %w", err)
	}

	cmd := exec.Command("tesseract", st.Path, "stdout", "tsv")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("tesseract failed: %s (%v)", strings.TrimSpace(stderr.String()), err)
	}

	words, err := parseTesseractTSV(stdout.Bytes(), st.OffsetX, st.OffsetY, st.ScaleFactor)
	if err != nil {
		return nil, nil, err
	}
	return words, st, nil
}

func doOCRCmd(targetScreen, region string) {
	words, st, err := runOCR(targetScreen, region)
	if err != nil {
		outputError(err.Error())
	}

	outputJSON(map[string]interface{}{
		"ok":              true,
		"words":           words,
		"count":           len(words),
		"screenshot_path": st.Path,
		"scale_factor":    st.ScaleFactor,
	})
}

func doFindTextCmd(query, targetScreen, region string, click bool, button string, double, triple, dryRun, screenshotAfter bool) {
	words, st, err := runOCR(targetScreen, region)
	if err != nil {
		outputError(err.Error())
	}

	qLower := strings.ToLower(strings.TrimSpace(query))
	var matches []OCRWord

	for _, w := range words {
		if strings.Contains(strings.ToLower(w.Text), qLower) {
			matches = append(matches, w)
		}
	}

	if len(matches) == 0 {
		outputError(fmt.Sprintf("Text %q not found via OCR. Found %d other words.", query, len(words)))
	}

	firstMatch := matches[0]

	if click {
		doClick(firstMatch.Center.X, firstMatch.Center.Y, button, double, triple, dryRun, screenshotAfter, "")
		return
	}

	outputJSON(map[string]interface{}{
		"ok":              true,
		"query":           query,
		"matches":         matches,
		"count":           len(matches),
		"screenshot_path": st.Path,
	})
}
