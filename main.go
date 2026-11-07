package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func printUsage() {
	usage := `deskctl - Linux Desktop Control Toolkit

Usage:
  deskctl <command> [arguments] [flags]

Input:
  click <x> <y> [--screen <ID>] [--button left|right|middle] [--double] [--triple]
  double-click <x> <y> [--screen <ID>]
  triple-click <x> <y> [--screen <ID>]
  right-click <x> <y> [--screen <ID>]
  middle-click <x> <y> [--screen <ID>]
  move <x> <y> [--screen <ID>]
  drag <x1> <y1> <x2> <y2> [--screen <ID>]
  drag-path "<x1,y1 x2,y2 x3,y3 ...>" [--screen <ID>]
  hold-and-release <x> <y> [--button left|right|middle] [--hold-ms <N>]
  scroll <x> <y> [--screen <ID>] [--direction up|down|left|right] [--amount <N>]
  hover <x> <y> [--screen <ID>] [--dwell-ms <N>]
  mouse-position
  type <text> [--delay <ms>]
  key <combo>
  key-hold <combo> [--hold-ms <N>]

Windows:
  windows
  focus [--id <ID>] [--title <SUBSTR>]
  close-window [--id <ID>] [--title <SUBSTR>]
  minimize [--id <ID>] [--title <SUBSTR>]
  maximize [--id <ID>] [--title <SUBSTR>] [--mode add|remove|toggle]
  move-window [--id <ID>] [--title <SUBSTR>] --x <N> --y <N>
  resize-window [--id <ID>] [--title <SUBSTR>] --w <N> --h <N>
  move-to-screen [--id <ID>] [--title <SUBSTR>] --screen <ID>
  always-on-top [--id <ID>] [--title <SUBSTR>]
  workspaces <list|switch|move-window> [args]
  launch <app_name_or_desktop_file>
  wait-for-window --title <SUBSTR> [--timeout <sec>]
  wait-for-window-close [--id <ID>] [--title <SUBSTR>] [--timeout <sec>]

Vision, Screen & OCR:
  screens
  screenshot [--screen ID|all|active|primary|window:ID|region:x,y,w,h] [--max-width <N>] [--format png|jpeg] [--output PATH]
  zoom --region <x,y,w,h> [--factor <N>] [--output PATH]
  diff [--file1 <A> --file2 <B>] [--screen <ID>]
  wait-for-change [--screen <ID>] [--threshold <P>] [--timeout-ms <N>]
  wait-for-stable [--screen <ID>] [--threshold <P>] [--timeout-ms <N>]
  color-at-pixel <x> <y> [--screen <ID>]
  find-image <template_path> [--screen <ID>] [--threshold <P>]
  record-screen <start|stop|status> [--screen <ID>] [--fps <N>] [--output PATH]
  ocr [--screen <ID>] [--region <x,y,w,h>]
  find-text <query> [--screen <ID>] [--region <x,y,w,h>] [--click]

Accessibility (AT-SPI):
  ui-tree [--window <ID>] [--filter <ROLE>] [--max-depth <N>] [--max-nodes <N>]
  find <query> [--window <ID>] [--role <ROLE>] [--limit <N>]
  click-element <name> [--role <ROLE>] [--window <ID>] [--button left|right|middle]
  get-text [--query <name>] [--role <ROLE>] [--window <ID>]
  set-text <text> [--query <name>] [--role <ROLE>] [--window <ID>]
  element-actions [--query <name>] [--role <ROLE>] [--window <ID>] [--action <NAME>]

System & Sessions:
  clipboard <get|set|history|get-image|set-image> [args]
  run <shell_command> [--timeout <seconds>]
  notify <message> [--title <title>]
  open <path_or_url>
  file <list|read|write|trash> [args]
  processes <list|kill> [args]
  volume <get|set|mute> [value]
  brightness <get|set> [value]
  network
  battery
  session <start|status|end> [flags]
  wait --ms <milliseconds>
  schedule --delay <sec> --command "<cmd>"
  auth-check
  auth-wait [--timeout <sec>]
  setup-sudo [--confirm] [--dry-run] [--revoke]
  setup-polkit [--confirm] [--dry-run] [--revoke]
  stop [reason]
  reset
  killswitch <start|stop|status>
  doctor
`
	fmt.Println(usage)
}

func parseBoolFlag(args []string, name string) (bool, []string) {
	var remaining []string
	found := false
	flag1 := "--" + name
	flag2 := "-" + name
	for _, a := range args {
		if a == flag1 || a == flag2 {
			found = true
		} else {
			remaining = append(remaining, a)
		}
	}
	return found, remaining
}

func parseStringFlag(args []string, name string, defaultValue string) (string, []string) {
	var remaining []string
	val := defaultValue
	flag1 := "--" + name
	flag2 := "-" + name
	prefix1 := flag1 + "="
	prefix2 := flag2 + "="

	skipNext := false
	for i := 0; i < len(args); i++ {
		if skipNext {
			skipNext = false
			continue
		}
		a := args[i]
		if a == flag1 || a == flag2 {
			if i+1 < len(args) {
				val = args[i+1]
				skipNext = true
			}
		} else if strings.HasPrefix(a, prefix1) {
			val = strings.TrimPrefix(a, prefix1)
		} else if strings.HasPrefix(a, prefix2) {
			val = strings.TrimPrefix(a, prefix2)
		} else {
			remaining = append(remaining, a)
		}
	}
	return val, remaining
}

func parseWindowIDFlag(args []string) (string, []string) {
	idOpt, remaining := parseStringFlag(args, "id", "")
	if idOpt == "" {
		idOpt, remaining = parseStringFlag(remaining, "window", "")
	}
	return idOpt, remaining
}

func parseIntFlag(args []string, name string, defaultValue int) (int, []string) {
	sVal, remaining := parseStringFlag(args, name, strconv.Itoa(defaultValue))
	iVal, err := strconv.Atoi(sVal)
	if err != nil {
		return defaultValue, remaining
	}
	return iVal, remaining
}

func parseFloatFlag(args []string, name string, defaultValue float64) (float64, []string) {
	sVal, remaining := parseStringFlag(args, name, fmt.Sprintf("%f", defaultValue))
	fVal, err := strconv.ParseFloat(sVal, 64)
	if err != nil {
		return defaultValue, remaining
	}
	return fVal, remaining
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	initEnv()

	cmd := os.Args[1]
	args := os.Args[2:]

	// Commands allowed even when STOP file is present
	switch cmd {
	case "reset":
		doReset()
		return
	case "panic":
		doPanic()
		return
	case "stop":
		reason := "Manual stop"
		if len(args) > 0 {
			reason = strings.Join(args, " ")
		}
		doStop(reason)
		return
	case "killswitch":
		subcmd := "status"
		if len(args) > 0 {
			subcmd = args[0]
		}
		doKillswitchCmd(subcmd)
		return
	case "--help", "-h", "help":
		printUsage()
		return
	}

	// For all other commands, verify killswitch, auth prompt safety, security & scope, and tool availability
	checkKillSwitch()
	checkAuthSafety(cmd)
	if err := checkSecurityAndScope(cmd, args); err != nil {
		outputJSON(map[string]interface{}{
			"ok":        false,
			"error":     err.Error(),
			"violation": true,
		})
		os.Exit(1)
	}
	if err := checkRequiredTools(); err != nil {
		outputError(err.Error())
	}

	// Global flags extraction
	dryRun, args := parseBoolFlag(args, "dry-run")
	screenshotAfter, args := parseBoolFlag(args, "screenshot-after")
	screenArg, args := parseStringFlag(args, "screen", "")

	switch cmd {
	// ==================== SCREEN & VISION ====================
	case "screens":
		noRefresh, _ := parseBoolFlag(args, "no-refresh")
		doScreens(!noRefresh)

	case "screenshot":
		maxWidth, args := parseIntFlag(args, "max-width", 0)
		format, args := parseStringFlag(args, "format", "png")
		outputPath, args := parseStringFlag(args, "output", "")
		region, args := parseStringFlag(args, "region", "")
		privacy, args := parseBoolFlag(args, "privacy")

		target := screenArg
		if target == "" && region != "" {
			target = "region:" + region
		}
		if target == "" && len(args) > 0 {
			target = args[0]
		}
		if target == "" {
			target = "all"
		}

		cfg := loadConfig()
		if (privacy || cfg.PrivacyMode) && (target == "all" || target == "") {
			target = "window:active"
		}

		st, err := takeScreenshotMulti(target, maxWidth, format, outputPath)
		if err != nil {
			outputError(fmt.Sprintf("Screenshot capture failed: %v", err))
		}
		outputJSON(map[string]interface{}{
			"ok":              true,
			"path":            st.Path,
			"target_type":     st.TargetType,
			"screen_id":       st.ScreenID,
			"screen_name":     st.ScreenName,
			"image_width":     st.ImageWidth,
			"image_height":    st.ImageHeight,
			"captured_width":  st.CapturedWidth,
			"captured_height": st.CapturedHeight,
			"scale_factor":    st.ScaleFactor,
			"offset_x":        st.OffsetX,
			"offset_y":        st.OffsetY,
			"format":          st.Format,
			"screen":          st.Screen,
			"virtual_desktop": st.VirtualDesktop,
		})

	case "zoom":
		region, args := parseStringFlag(args, "region", "")
		factor, args := parseIntFlag(args, "factor", 2)
		outputPath, _ := parseStringFlag(args, "output", "")
		if region == "" && len(args) > 0 {
			region = args[0]
		}
		doZoom(region, factor, screenArg, outputPath)

	case "diff":
		file1, args := parseStringFlag(args, "file1", "")
		file2, _ := parseStringFlag(args, "file2", "")
		doDiff(file1, file2, screenArg)

	case "wait-for-change":
		threshold, args := parseFloatFlag(args, "threshold", 0.2)
		timeoutMs, _ := parseIntFlag(args, "timeout-ms", 5000)
		doWaitForChange(screenArg, threshold, timeoutMs)

	case "wait-for-stable":
		threshold, args := parseFloatFlag(args, "threshold", 0.05)
		timeoutMs, _ := parseIntFlag(args, "timeout-ms", 5000)
		doWaitForStable(screenArg, threshold, timeoutMs)

	case "color-at-pixel":
		if len(args) < 2 {
			outputError("Usage: deskctl color-at-pixel <x> <y> [--screen <ID>]")
		}
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		doColorAtPixel(x, y, screenArg)

	case "find-image":
		if len(args) < 1 {
			outputError("Usage: deskctl find-image <template_path> [--screen <ID>] [--threshold <P>]")
		}
		threshold, _ := parseFloatFlag(args, "threshold", 0.8)
		doFindImage(args[0], screenArg, threshold)

	case "record-screen":
		action := "status"
		if len(args) > 0 {
			action = args[0]
		}
		fps, args := parseIntFlag(args, "fps", 15)
		outputPath, _ := parseStringFlag(args, "output", "")
		doRecordScreen(action, screenArg, outputPath, fps)

	case "ocr":
		region, _ := parseStringFlag(args, "region", "")
		doOCRCmd(screenArg, region)

	case "find-text":
		click, args := parseBoolFlag(args, "click")
		button, args := parseStringFlag(args, "button", "left")
		region, args := parseStringFlag(args, "region", "")
		if len(args) < 1 {
			outputError("Usage: deskctl find-text <query> [--screen <ID>] [--click]")
		}
		query := strings.Join(args, " ")
		doFindTextCmd(query, screenArg, region, click, button, false, false, dryRun, screenshotAfter)

	// ==================== MOUSE & INPUT ====================
	case "click":
		button, args := parseStringFlag(args, "button", "left")
		double, args := parseBoolFlag(args, "double")
		triple, args := parseBoolFlag(args, "triple")
		if len(args) < 2 {
			outputError("Usage: deskctl click <x> <y> [--screen <ID>] [--button left|right|middle]")
		}
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		doClick(x, y, button, double, triple, dryRun, screenshotAfter, screenArg)

	case "double-click":
		if len(args) < 2 {
			outputError("Usage: deskctl double-click <x> <y> [--screen <ID>]")
		}
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		doClick(x, y, "left", true, false, dryRun, screenshotAfter, screenArg)

	case "triple-click":
		if len(args) < 2 {
			outputError("Usage: deskctl triple-click <x> <y> [--screen <ID>]")
		}
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		doClick(x, y, "left", false, true, dryRun, screenshotAfter, screenArg)

	case "right-click":
		if len(args) < 2 {
			outputError("Usage: deskctl right-click <x> <y> [--screen <ID>]")
		}
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		doClick(x, y, "right", false, false, dryRun, screenshotAfter, screenArg)

	case "middle-click":
		if len(args) < 2 {
			outputError("Usage: deskctl middle-click <x> <y> [--screen <ID>]")
		}
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		doClick(x, y, "middle", false, false, dryRun, screenshotAfter, screenArg)

	case "move":
		if len(args) < 2 {
			outputError("Usage: deskctl move <x> <y> [--screen <ID>]")
		}
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		doMove(x, y, dryRun, screenshotAfter, screenArg)

	case "drag":
		if len(args) < 4 {
			outputError("Usage: deskctl drag <x1> <y1> <x2> <y2> [--screen <ID>]")
		}
		x1, _ := strconv.Atoi(args[0])
		y1, _ := strconv.Atoi(args[1])
		x2, _ := strconv.Atoi(args[2])
		y2, _ := strconv.Atoi(args[3])
		doDrag(x1, y1, x2, y2, dryRun, screenshotAfter, screenArg)

	case "drag-path":
		if len(args) < 1 {
			outputError("Usage: deskctl drag-path \"x1,y1 x2,y2 x3,y3\" [--screen <ID>]")
		}
		doDragPath(args[0], dryRun, screenshotAfter, screenArg)

	case "hold-and-release":
		button, args := parseStringFlag(args, "button", "left")
		holdMs, args := parseIntFlag(args, "hold-ms", 500)
		if len(args) < 2 {
			outputError("Usage: deskctl hold-and-release <x> <y> [--button left|right|middle] [--hold-ms <N>]")
		}
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		doHoldAndRelease(x, y, button, holdMs, dryRun, screenshotAfter, screenArg)

	case "hover":
		dwellMs, args := parseIntFlag(args, "dwell-ms", 500)
		if len(args) < 2 {
			outputError("Usage: deskctl hover <x> <y> [--dwell-ms <N>]")
		}
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		doHover(x, y, dwellMs, dryRun, screenshotAfter, screenArg)

	case "mouse-position":
		doMousePosition()

	case "scroll":
		direction, args := parseStringFlag(args, "direction", "down")
		amount, args := parseIntFlag(args, "amount", 3)
		if len(args) < 2 {
			outputError("Usage: deskctl scroll <x> <y> [--screen <ID>] [--direction up|down|left|right] [--amount N]")
		}
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		doScroll(x, y, direction, amount, dryRun, screenshotAfter, screenArg)

	case "type":
		delayMs, args := parseIntFlag(args, "delay", 12)
		if len(args) < 1 {
			outputError("Usage: deskctl type <text> [--delay <ms>]")
		}
		text := strings.Join(args, " ")
		doType(text, delayMs, dryRun, screenshotAfter)

	case "key":
		if len(args) < 1 {
			outputError("Usage: deskctl key <key-combo>")
		}
		combo := strings.Join(args, " ")
		doKey(combo, dryRun, screenshotAfter)

	case "key-hold":
		holdMs, args := parseIntFlag(args, "hold-ms", 500)
		if len(args) < 1 {
			outputError("Usage: deskctl key-hold <key-combo> [--hold-ms <N>]")
		}
		combo := strings.Join(args, " ")
		doKeyHold(combo, holdMs, dryRun, screenshotAfter)

	// ==================== WINDOWS ====================
	case "windows":
		doWindows()

	case "focus":
		idOpt, args := parseWindowIDFlag(args)
		titleOpt, _ := parseStringFlag(args, "title", "")
		doFocus(idOpt, titleOpt, dryRun, screenshotAfter)

	case "close-window":
		idOpt, args := parseWindowIDFlag(args)
		titleOpt, _ := parseStringFlag(args, "title", "")
		doCloseWindow(idOpt, titleOpt, dryRun)

	case "minimize":
		idOpt, args := parseWindowIDFlag(args)
		titleOpt, _ := parseStringFlag(args, "title", "")
		doMinimizeWindow(idOpt, titleOpt, dryRun)

	case "maximize":
		mode, args := parseStringFlag(args, "mode", "toggle")
		idOpt, args := parseWindowIDFlag(args)
		titleOpt, _ := parseStringFlag(args, "title", "")
		doMaximizeWindow(idOpt, titleOpt, mode, dryRun)

	case "move-window":
		idOpt, args := parseWindowIDFlag(args)
		titleOpt, args := parseStringFlag(args, "title", "")
		x, args := parseIntFlag(args, "x", -1)
		y, args := parseIntFlag(args, "y", -1)
		if x == -1 && len(args) >= 1 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				x = val
				args = args[1:]
			}
		}
		if y == -1 && len(args) >= 1 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				y = val
				args = args[1:]
			}
		}
		doMoveResizeWindow(idOpt, titleOpt, x, y, -1, -1, dryRun)

	case "resize-window":
		idOpt, args := parseWindowIDFlag(args)
		titleOpt, args := parseStringFlag(args, "title", "")
		w, args := parseIntFlag(args, "w", -1)
		h, args := parseIntFlag(args, "h", -1)
		if w == -1 && len(args) >= 1 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				w = val
				args = args[1:]
			}
		}
		if h == -1 && len(args) >= 1 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				h = val
				args = args[1:]
			}
		}
		doMoveResizeWindow(idOpt, titleOpt, -1, -1, w, h, dryRun)

	case "move-to-screen":
		idOpt, args := parseWindowIDFlag(args)
		titleOpt, args := parseStringFlag(args, "title", "")
		targetSc, _ := parseStringFlag(args, "screen", screenArg)
		if targetSc == "" {
			outputError("move-to-screen requires --screen <ID>")
		}
		doMoveToScreen(idOpt, titleOpt, targetSc, dryRun)

	case "always-on-top":
		idOpt, args := parseWindowIDFlag(args)
		titleOpt, _ := parseStringFlag(args, "title", "")
		doAlwaysOnTop(idOpt, titleOpt, dryRun)

	case "workspaces":
		subcmd := "list"
		if len(args) > 0 {
			subcmd = args[0]
			args = args[1:]
		}
		idOpt, args := parseWindowIDFlag(args)
		wsArg, _ := parseStringFlag(args, "workspace", "")
		if wsArg == "" && len(args) > 0 {
			wsArg = args[0]
		}
		doWorkspaces(subcmd, idOpt, wsArg)

	case "launch":
		if len(args) < 1 {
			outputError("Usage: deskctl launch <app_name_or_desktop_file>")
		}
		doLaunch(strings.Join(args, " "))

	case "wait-for-window":
		titleOpt, args := parseStringFlag(args, "title", "")
		timeoutSec, _ := parseIntFlag(args, "timeout", 10)
		if titleOpt == "" && len(args) > 0 {
			titleOpt = args[0]
		}
		doWaitForWindow(titleOpt, timeoutSec)

	case "wait-for-window-close":
		idOpt, args := parseWindowIDFlag(args)
		titleOpt, args := parseStringFlag(args, "title", "")
		timeoutSec, _ := parseIntFlag(args, "timeout", 60)
		doWaitForWindowClose(idOpt, titleOpt, timeoutSec)

	// ==================== ACCESSIBILITY ====================
	case "ui-tree":
		windowID, args := parseStringFlag(args, "window", "")
		filter, args := parseStringFlag(args, "filter", "")
		maxDepth, args := parseIntFlag(args, "max-depth", 5)
		maxNodes, args := parseIntFlag(args, "max-nodes", 150)
		interactiveOnly, _ := parseBoolFlag(args, "interactive-only")
		doUITree(windowID, filter, maxDepth, maxNodes, interactiveOnly)

	case "find":
		windowID, args := parseStringFlag(args, "window", "")
		roleFilter, args := parseStringFlag(args, "role", "")
		limit, args := parseIntFlag(args, "limit", 20)
		if len(args) < 1 {
			outputError("Usage: deskctl find <query> [--window <ID>] [--role <ROLE>]")
		}
		query := strings.Join(args, " ")
		doFindCmd(query, windowID, roleFilter, limit)

	case "click-element":
		roleFilter, args := parseStringFlag(args, "role", "")
		windowID, args := parseStringFlag(args, "window", "")
		button, args := parseStringFlag(args, "button", "left")
		double, args := parseBoolFlag(args, "double")
		triple, args := parseBoolFlag(args, "triple")
		if len(args) < 1 {
			outputError("Usage: deskctl click-element <name> [--role <ROLE>] [--window <ID>]")
		}
		name := strings.Join(args, " ")
		doClickElement(name, roleFilter, windowID, button, double, triple, dryRun, screenshotAfter)

	case "get-text":
		query, args := parseStringFlag(args, "query", "")
		roleFilter, args := parseStringFlag(args, "role", "")
		windowID, _ := parseStringFlag(args, "window", "")
		if query == "" && len(args) > 0 {
			query = args[0]
		}
		doGetText(query, windowID, roleFilter)

	case "set-text":
		query, args := parseStringFlag(args, "query", "")
		roleFilter, args := parseStringFlag(args, "role", "")
		windowID, args := parseStringFlag(args, "window", "")
		if len(args) < 1 {
			outputError("Usage: deskctl set-text <text> [--query <name>] [--role <ROLE>]")
		}
		text := strings.Join(args, " ")
		doSetText(query, text, windowID, roleFilter)

	case "element-actions":
		query, args := parseStringFlag(args, "query", "")
		roleFilter, args := parseStringFlag(args, "role", "")
		windowID, args := parseStringFlag(args, "window", "")
		actionName, _ := parseStringFlag(args, "action", "")
		if query == "" && len(args) > 0 {
			query = args[0]
		}
		doElementActions(query, windowID, roleFilter, actionName)

	// ==================== SYSTEM & SESSIONS ====================
	case "clipboard":
		if len(args) < 1 {
			outputError("Usage: deskctl clipboard <get|set|history|get-image|set-image> [args]")
		}
		sub := strings.ToLower(args[0])
		args = args[1:]
		switch sub {
		case "get":
			doClipboardGet()
		case "set":
			doClipboardSet(strings.Join(args, " "), dryRun)
		case "history":
			doClipboardHistory()
		case "get-image":
			outputPath, _ := parseStringFlag(args, "output", "")
			if outputPath == "" && len(args) > 0 {
				outputPath = args[0]
			}
			doClipboardGetImage(outputPath)
		case "set-image":
			if len(args) < 1 {
				outputError("Usage: deskctl clipboard set-image <image_path>")
			}
			doClipboardSetImage(args[0])
		default:
			outputError(fmt.Sprintf("Unknown clipboard action '%s'", sub))
		}

	case "session":
		if len(args) < 1 {
			outputError("Usage: deskctl session <start|status|end>")
		}
		sub := strings.ToLower(args[0])
		args = args[1:]
		switch sub {
		case "start":
			task, args := parseStringFlag(args, "task", "")
			apps, args := parseStringFlag(args, "apps", "")
			folders, args := parseStringFlag(args, "folders", "")
			domains, args := parseStringFlag(args, "domains", "")
			actions, _ := parseStringFlag(args, "actions", "")
			if task == "" && len(args) > 0 {
				task = strings.Join(args, " ")
			}
			doSessionStart(task, apps, folders, domains, actions)
		case "status":
			doSessionStatus()
		case "end":
			doSessionEnd()
		default:
			outputError(fmt.Sprintf("Unknown session action '%s'", sub))
		}

	case "run":
		timeoutSec, args := parseIntFlag(args, "timeout", 30)
		if len(args) < 1 {
			outputError("Usage: deskctl run <command> [--timeout N]")
		}
		cmdStr := strings.Join(args, " ")
		doRun(cmdStr, timeoutSec, dryRun)

	case "notify":
		title, args := parseStringFlag(args, "title", "deskctl")
		if len(args) < 1 {
			outputError("Usage: deskctl notify <message> [--title <title>]")
		}
		doNotify(strings.Join(args, " "), title)

	case "open":
		if len(args) < 1 {
			outputError("Usage: deskctl open <path_or_url>")
		}
		doOpen(args[0])

	case "file":
		if len(args) < 1 {
			outputError("Usage: deskctl file <list|read|write|trash> [args]")
		}
		sub := strings.ToLower(args[0])
		args = args[1:]
		switch sub {
		case "list":
			dirPath := "."
			if len(args) > 0 {
				dirPath = args[0]
			}
			doFileList(dirPath)
		case "read":
			maxBytes, args := parseIntFlag(args, "max-bytes", 64*1024)
			if len(args) < 1 {
				outputError("Usage: deskctl file read <path> [--max-bytes N]")
			}
			doFileRead(args[0], maxBytes)
		case "write":
			appendMode, args := parseBoolFlag(args, "append")
			if len(args) < 2 {
				outputError("Usage: deskctl file write <path> <content> [--append]")
			}
			doFileWrite(args[0], strings.Join(args[1:], " "), appendMode)
		case "trash":
			if len(args) < 1 {
				outputError("Usage: deskctl file trash <path>")
			}
			doFileTrash(args[0])
		default:
			outputError(fmt.Sprintf("Unknown file subcommand '%s'", sub))
		}

	case "processes":
		if len(args) < 1 {
			outputError("Usage: deskctl processes <list|kill> [args]")
		}
		sub := strings.ToLower(args[0])
		args = args[1:]
		switch sub {
		case "list":
			filter, _ := parseStringFlag(args, "filter", "")
			if filter == "" && len(args) > 0 {
				filter = args[0]
			}
			doProcessesList(filter)
		case "kill":
			force, args := parseBoolFlag(args, "force")
			if len(args) < 1 {
				outputError("Usage: deskctl processes kill <name|pid> [--force]")
			}
			doProcessesKill(args[0], force)
		default:
			outputError(fmt.Sprintf("Unknown processes action '%s'", sub))
		}

	case "volume":
		action := "get"
		val := ""
		if len(args) > 0 {
			action = args[0]
		}
		if len(args) > 1 {
			val = args[1]
		}
		doVolume(action, val)

	case "brightness":
		action := "get"
		val := ""
		if len(args) > 0 {
			action = args[0]
		}
		if len(args) > 1 {
			val = args[1]
		}
		doBrightness(action, val)

	case "network":
		doNetworkStatus()

	case "battery":
		doBatteryStatus()

	case "wait":
		ms, _ := parseIntFlag(args, "ms", 500)
		doWait(ms)

	case "schedule":
		delaySec, args := parseIntFlag(args, "delay", 5)
		commandStr, _ := parseStringFlag(args, "command", "")
		if commandStr == "" && len(args) > 0 {
			commandStr = strings.Join(args, " ")
		}
		doSchedule(delaySec, commandStr)

	// ==================== AUTH & PRIVILEGES (STAGE 3) ====================
	case "auth-check":
		doAuthCheck()

	case "auth-wait":
		timeoutSec, _ := parseIntFlag(args, "timeout", 120)
		doAuthWait(timeoutSec)

	case "setup-sudo":
		confirm, args := parseBoolFlag(args, "confirm")
		revoke, _ := parseBoolFlag(args, "revoke")
		doSetupSudo(confirm, dryRun, revoke)

	case "setup-polkit":
		confirm, args := parseBoolFlag(args, "confirm")
		revoke, _ := parseBoolFlag(args, "revoke")
		doSetupPolkit(confirm, dryRun, revoke)

	case "config":
		doConfigCmd(args)

	case "doctor":
		doDoctor()

	case "completion":
		shell := "bash"
		if len(args) > 0 {
			shell = args[0]
		}
		doCompletion(shell)

	default:
		outputError(fmt.Sprintf("Unknown command: %q. Run 'deskctl --help' for available commands.", cmd))
	}
}

func doCompletion(shell string) {
	if strings.ToLower(shell) != "bash" {
		outputError(fmt.Sprintf("Unsupported shell '%s'. Supported shells: bash", shell))
	}

	script := `# deskctl bash completion
_deskctl_completion() {
    local cur prev words cword
    _init_completion -n : || return

    local commands="reset panic stop killswitch help screens screenshot zoom diff wait-for-change wait-for-stable color-at-pixel find-image record-screen ocr find-text click double-click triple-click right-click middle-click move drag drag-path hold-and-release hover mouse-position scroll type key key-hold windows focus close-window minimize maximize move-window resize-window move-to-screen always-on-top workspaces launch wait-for-window wait-for-window-close ui-tree find click-element get-text set-text element-actions clipboard session run notify open file processes volume brightness network battery wait schedule auth-check auth-wait setup-sudo setup-polkit config doctor completion"

    if [ "$cword" -eq 1 ]; then
        COMPREPLY=( $(compgen -W "${commands}" -- "$cur") )
        return 0
    fi

    local subcmd="${words[1]}"
    case "$subcmd" in
        screenshot)
            COMPREPLY=( $(compgen -W "--target --max-width --format --output --screen" -- "$cur") )
            ;;
        click|move|hover|double-click|triple-click|right-click|middle-click)
            COMPREPLY=( $(compgen -W "--screen --dry-run --screenshot-after" -- "$cur") )
            ;;
        windows|focus|close-window|minimize|maximize|move-window|resize-window|move-to-screen)
            COMPREPLY=( $(compgen -W "--id --window --title --dry-run" -- "$cur") )
            ;;
        ui-tree)
            COMPREPLY=( $(compgen -W "--window --id --filter --max-depth --max-nodes --interactive-only" -- "$cur") )
            ;;
        find|click-element|get-text|set-text|element-actions)
            COMPREPLY=( $(compgen -W "--window --id --role --limit" -- "$cur") )
            ;;
        diff|wait-for-change)
            COMPREPLY=( $(compgen -W "--file1 --file2 --screen --threshold --timeout" -- "$cur") )
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash" -- "$cur") )
            ;;
        *)
            COMPREPLY=( $(compgen -W "--help --dry-run" -- "$cur") )
            ;;
    esac
}
complete -F _deskctl_completion deskctl
`
	fmt.Print(script)
}
