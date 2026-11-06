package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func doType(text string, delayMs int, dryRun, screenshotAfter bool) {
	if delayMs <= 0 {
		delayMs = 12
	}

	logAction("type", []string{fmt.Sprintf("len=%d", len(text)), fmt.Sprintf("sample=%q", truncateString(text, 20)), fmt.Sprintf("delay=%d", delayMs)}, false, dryRun, "")

	if !dryRun {
		cmd := exec.Command("xdotool", "type", "--delay", strconv.Itoa(delayMs), "--clearmodifiers", "--file", "-")
		cmd.Stdin = bytes.NewBufferString(text)
		out, err := cmd.CombinedOutput()
		if err != nil {
			recordActionFailure("type")
			outputError(fmt.Sprintf("xdotool type failed: %s (%v)", strings.TrimSpace(string(out)), err))
		}
		recordActionSuccess()
		performActionWait()
	}

	resp := map[string]interface{}{
		"ok":       true,
		"action":   "type",
		"length":   len(text),
		"delay_ms": delayMs,
		"dry_run":  dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}

func doKey(combo string, dryRun, screenshotAfter bool) {
	combo = strings.TrimSpace(combo)
	if combo == "" {
		outputError("Key combo cannot be empty")
	}

	destructive := isDestructiveKey(combo)
	logAction("key", []string{combo}, destructive, dryRun, "")

	if !dryRun {
		out, err := exec.Command("xdotool", "key", "--clearmodifiers", combo).CombinedOutput()
		if err != nil {
			recordActionFailure("key")
			outputError(fmt.Sprintf("xdotool key failed for '%s': %s (%v)", combo, strings.TrimSpace(string(out)), err))
		}
		recordActionSuccess()
		performActionWait()
	}

	resp := map[string]interface{}{
		"ok":          true,
		"action":      "key",
		"key":         combo,
		"destructive": destructive,
		"dry_run":     dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}

func doKeyHold(combo string, holdMs int, dryRun, screenshotAfter bool) {
	combo = strings.TrimSpace(combo)
	if combo == "" {
		outputError("Key combo cannot be empty")
	}
	if holdMs <= 0 {
		holdMs = 500
	}

	destructive := isDestructiveKey(combo)
	logAction("key-hold", []string{combo, fmt.Sprintf("hold_ms=%d", holdMs)}, destructive, dryRun, "")

	if !dryRun {
		_ = exec.Command("xdotool", "keydown", combo).Run()
		time.Sleep(time.Duration(holdMs) * time.Millisecond)
		_ = exec.Command("xdotool", "keyup", combo).Run()
		performActionWait()
	}

	resp := map[string]interface{}{
		"ok":          true,
		"action":      "key-hold",
		"key":         combo,
		"hold_ms":     holdMs,
		"destructive": destructive,
		"dry_run":     dryRun,
	}
	attachScreenshotAfter(resp, screenshotAfter)
	outputJSON(resp)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
