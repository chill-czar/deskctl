package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	configFile     = filepath.Join(deskctlDir, "config.toml")
	violationsFile = filepath.Join(deskctlDir, "violations.log")

	rateLimitMutex sync.Mutex
	actionTimestamps []time.Time
	lastDestructiveTime time.Time
)

type Config struct {
	MaxActionsPerSession       int     `json:"max_actions_per_session"`
	MaxSessionDurationSec      int     `json:"max_session_duration_sec"`
	MinDelayBetweenDestructive int     `json:"min_delay_between_destructive_ms"`
	MaxActionsPerMinute        int     `json:"max_actions_per_minute"`
	PrivacyMode                bool    `json:"privacy_mode"`
	BlurBackgroundWindows      bool    `json:"blur_background_windows"`
	SuppressNotifications      bool    `json:"suppress_notifications"`
}

func defaultConfig() Config {
	return Config{
		MaxActionsPerSession:       100,
		MaxSessionDurationSec:      3600,
		MinDelayBetweenDestructive: 500,
		MaxActionsPerMinute:        60,
		PrivacyMode:                false,
		BlurBackgroundWindows:      true,
		SuppressNotifications:      true,
	}
}

func loadConfig() Config {
	cfg := defaultConfig()
	data, err := os.ReadFile(configFile)
	if err != nil {
		// Initialize config file with defaults
		_ = saveConfig(cfg)
		return cfg
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "max_actions_per_session":
			fmt.Sscanf(val, "%d", &cfg.MaxActionsPerSession)
		case "max_session_duration_sec":
			fmt.Sscanf(val, "%d", &cfg.MaxSessionDurationSec)
		case "min_delay_between_destructive_ms":
			fmt.Sscanf(val, "%d", &cfg.MinDelayBetweenDestructive)
		case "max_actions_per_minute":
			fmt.Sscanf(val, "%d", &cfg.MaxActionsPerMinute)
		case "privacy_mode":
			cfg.PrivacyMode = strings.ToLower(val) == "true"
		case "blur_background_windows":
			cfg.BlurBackgroundWindows = strings.ToLower(val) == "true"
		case "suppress_notifications":
			cfg.SuppressNotifications = strings.ToLower(val) == "true"
		}
	}
	return cfg
}

func saveConfig(cfg Config) error {
	var b strings.Builder
	b.WriteString("# ====================================================================\n")
	b.WriteString("# Deskctl Security, Budget and Privacy Configuration\n")
	b.WriteString("# ====================================================================\n\n")

	b.WriteString("[budget]\n")
	b.WriteString(fmt.Sprintf("max_actions_per_session = %d\n", cfg.MaxActionsPerSession))
	b.WriteString(fmt.Sprintf("max_session_duration_sec = %d\n", cfg.MaxSessionDurationSec))
	b.WriteString(fmt.Sprintf("min_delay_between_destructive_ms = %d\n\n", cfg.MinDelayBetweenDestructive))

	b.WriteString("[rate_limit]\n")
	b.WriteString(fmt.Sprintf("max_actions_per_minute = %d\n\n", cfg.MaxActionsPerMinute))

	b.WriteString("[privacy]\n")
	b.WriteString(fmt.Sprintf("privacy_mode = %t\n", cfg.PrivacyMode))
	b.WriteString(fmt.Sprintf("blur_background_windows = %t\n", cfg.BlurBackgroundWindows))
	b.WriteString(fmt.Sprintf("suppress_notifications = %t\n", cfg.SuppressNotifications))

	return os.WriteFile(configFile, []byte(b.String()), 0644)
}

func logViolation(violationType, target, details string) {
	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"type":      violationType,
		"target":    target,
		"details":   details,
	}
	if sess, err := getActiveSession(); err == nil && sess != nil {
		entry["session_id"] = sess.ID
		entry["session_task"] = sess.Task
	}

	data, _ := json.Marshal(entry)
	f, err := os.OpenFile(violationsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		_, _ = f.WriteString(string(data) + "\n")
		f.Close()
	}

	// Also log into active session log
	logSessionAction("security_violation", []string{violationType, target}, entry)
}

// isProtectedZone checks hardcoded prohibited paths and patterns.
// Impossible to bypass or opt out, regardless of session scope.
func isProtectedZone(pathStr string) (bool, string) {
	if strings.TrimSpace(pathStr) == "" {
		return false, ""
	}

	home := os.Getenv("HOME")
	cleanPath := pathStr

	// Expand ~
	if strings.HasPrefix(cleanPath, "~/") || cleanPath == "~" {
		cleanPath = filepath.Join(home, strings.TrimPrefix(cleanPath, "~/"))
	} else if !filepath.IsAbs(cleanPath) {
		if abs, err := filepath.Abs(cleanPath); err == nil {
			cleanPath = abs
		}
	}
	cleanPath = filepath.Clean(cleanPath)

	// Resolve symlinks to prevent symlink traversal into protected directories
	if evalPath, err := filepath.EvalSymlinks(cleanPath); err == nil {
		cleanPath = filepath.Clean(evalPath)
	}

	protectedPrefixes := map[string]string{
		filepath.Join(home, ".ssh"):               "SSH credentials & keys (~/.ssh)",
		filepath.Join(home, ".gnupg"):             "GPG keys & secret keyring (~/.gnupg)",
		filepath.Join(home, ".aws"):               "AWS cloud credentials (~/.aws)",
		filepath.Join(home, ".local/share/keyrings"): "System keyrings (~/.local/share/keyrings)",
		filepath.Join(home, ".config/google-chrome"): "Browser profiles, history, and cookies (~/.config/google-chrome)",
		filepath.Join(home, ".mozilla"):           "Browser profiles and saved data (~/.mozilla)",
		filepath.Join(home, ".bitcoin"):           "Cryptocurrency wallet storage (~/.bitcoin)",
		filepath.Join(home, ".ethereum"):          "Cryptocurrency wallet storage (~/.ethereum)",
		"/etc/shadow":                             "System shadow password database",
		"/etc/gshadow":                            "System group shadow database",
	}

	for prefix, desc := range protectedPrefixes {
		if cleanPath == prefix || strings.HasPrefix(cleanPath, prefix+string(filepath.Separator)) {
			return true, desc
		}
	}

	base := filepath.Base(cleanPath)
	// Block .env files
	if base == ".env" || strings.HasPrefix(base, ".env.") || strings.HasSuffix(base, ".env") {
		return true, "Environment secrets file (" + base + ")"
	}

	// Block wallet dat and password database files
	lowerBase := strings.ToLower(base)
	if strings.HasSuffix(lowerBase, ".kdbx") || strings.HasSuffix(lowerBase, ".keypass") || strings.Contains(lowerBase, "wallet.dat") {
		return true, "Password manager / wallet vault (" + base + ")"
	}

	return false, ""
}

// checkShellCommandForProtectedZones inspects raw shell strings for protected zone access
func checkShellCommandForProtectedZones(cmdStr string) (bool, string) {
	patterns := []string{
		`\.ssh\b`, `\.gnupg\b`, `\.aws\b`, `\.env\b`, `keyrings\b`,
		`/etc/shadow\b`, `/etc/gshadow\b`, `wallet\.dat\b`, `\.kdbx\b`,
	}
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		if re.MatchString(cmdStr) {
			return true, fmt.Sprintf("Shell command references protected zone pattern '%s'", p)
		}
	}
	return false, ""
}

// validateSessionScope checks whether the action is within the declared session boundaries
func validateSessionScope(cmdName string, target string) error {
	exemptCommands := map[string]bool{
		"session": true, "panic": true, "reset": true, "stop": true, "killswitch": true, "config": true,
	}
	if exemptCommands[cmdName] {
		return nil
	}

	sess, err := getActiveSession()
	if err != nil || sess == nil || sess.Status != "active" {
		// No active session or inactive session: scope checks pass
		return nil
	}

	// 1. Check allowed action types
	if len(sess.Scope.Actions) > 0 {
		allowed := false
		for _, a := range sess.Scope.Actions {
			if strings.EqualFold(a, cmdName) || strings.EqualFold(a, "all") {
				allowed = true
				break
			}
		}
		if !allowed {
			logViolation("action_scope", cmdName, fmt.Sprintf("Command %q not in allowed actions %v", cmdName, sess.Scope.Actions))
			return fmt.Errorf("Scope violation: action %q is outside declared session allowed actions %v", cmdName, sess.Scope.Actions)
		}
	}

	// 2. Check allowed folders for file operations
	fileCommands := map[string]bool{
		"file": true, "open": true,
	}
	if fileCommands[cmdName] && len(sess.Scope.Folders) > 0 && target != "" {
		cleanTarget := filepath.Clean(target)
		if strings.HasPrefix(cleanTarget, "~/") {
			cleanTarget = filepath.Join(os.Getenv("HOME"), strings.TrimPrefix(cleanTarget, "~/"))
		}
		inScope := false
		for _, f := range sess.Scope.Folders {
			cleanFolder := filepath.Clean(f)
			if strings.HasPrefix(cleanFolder, "~/") {
				cleanFolder = filepath.Join(os.Getenv("HOME"), strings.TrimPrefix(cleanFolder, "~/"))
			}
			if cleanTarget == cleanFolder || strings.HasPrefix(cleanTarget, cleanFolder+string(filepath.Separator)) {
				inScope = true
				break
			}
		}
		if !inScope {
			logViolation("folder_scope", target, fmt.Sprintf("Path %q is outside declared folders %v", target, sess.Scope.Folders))
			return fmt.Errorf("Scope violation: path %q is outside declared session folder boundaries %v", target, sess.Scope.Folders)
		}
	}

	// 3. Check allowed apps for window/launch operations
	appCommands := map[string]bool{
		"launch": true, "focus": true,
	}
	if appCommands[cmdName] && len(sess.Scope.Apps) > 0 && target != "" {
		tLower := strings.ToLower(target)
		inScope := false
		for _, app := range sess.Scope.Apps {
			if strings.Contains(tLower, strings.ToLower(app)) {
				inScope = true
				break
			}
		}
		if !inScope {
			logViolation("app_scope", target, fmt.Sprintf("App %q is outside declared allowed apps %v", target, sess.Scope.Apps))
			return fmt.Errorf("Scope violation: application %q is outside declared session app boundaries %v", target, sess.Scope.Apps)
		}
	}

	// 4. Check allowed domains for URLs
	if (cmdName == "open" || strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")) && len(sess.Scope.Domains) > 0 {
		u, err := url.Parse(target)
		if err == nil && u.Host != "" {
			inScope := false
			for _, d := range sess.Scope.Domains {
				if strings.EqualFold(u.Host, d) || strings.HasSuffix(strings.ToLower(u.Host), "."+strings.ToLower(d)) {
					inScope = true
					break
				}
			}
			if !inScope {
				logViolation("domain_scope", target, fmt.Sprintf("Domain %q is outside declared domains %v", u.Host, sess.Scope.Domains))
				return fmt.Errorf("Scope violation: domain %q is outside declared session allowed domains %v", u.Host, sess.Scope.Domains)
			}
		}
	}

	return nil
}

// checkBudgetAndRateLimit enforces max actions per session, max session duration, and rate limits
func checkBudgetAndRateLimit(cmdName string) error {
	cfg := loadConfig()

	// Check active session budget
	if sess, err := getActiveSession(); err == nil && sess != nil && sess.Status == "active" {
		if cfg.MaxActionsPerSession > 0 && sess.TotalActs >= cfg.MaxActionsPerSession {
			logViolation("budget_exceeded", cmdName, fmt.Sprintf("Reached max actions per session limit (%d)", cfg.MaxActionsPerSession))
			return fmt.Errorf("Session budget exceeded: reached maximum allowed actions (%d). Halting session for safety.", cfg.MaxActionsPerSession)
		}

		if cfg.MaxSessionDurationSec > 0 && sess.StartTime != "" {
			if t, err := time.Parse(time.RFC3339, sess.StartTime); err == nil {
				if int(time.Since(t).Seconds()) > cfg.MaxSessionDurationSec {
					logViolation("duration_exceeded", cmdName, fmt.Sprintf("Session duration exceeded limit (%d sec)", cfg.MaxSessionDurationSec))
					return fmt.Errorf("Session budget exceeded: duration exceeded %d seconds limit. Halting session for safety.", cfg.MaxSessionDurationSec)
				}
			}
		}
	}

	// Rate limiting per minute
	rateLimitMutex.Lock()
	defer rateLimitMutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-1 * time.Minute)
	var recent []time.Time
	for _, t := range actionTimestamps {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if cfg.MaxActionsPerMinute > 0 && len(recent) >= cfg.MaxActionsPerMinute {
		logViolation("rate_limit_exceeded", cmdName, fmt.Sprintf("Exceeded %d actions per minute", cfg.MaxActionsPerMinute))
		return fmt.Errorf("Rate limit exceeded: %d actions executed in the past minute (limit: %d/min). Please slow down.", len(recent), cfg.MaxActionsPerMinute)
	}
	actionTimestamps = append(recent, now)

	// Destructive action pacing
	if isDestructiveCommand(cmdName) {
		if cfg.MinDelayBetweenDestructive > 0 && !lastDestructiveTime.IsZero() {
			elapsed := time.Since(lastDestructiveTime)
			required := time.Duration(cfg.MinDelayBetweenDestructive) * time.Millisecond
			if elapsed < required {
				time.Sleep(required - elapsed)
			}
		}
		lastDestructiveTime = time.Now()
	}

	return nil
}

func doConfigCmd(args []string) {
	cfg := loadConfig()
	if len(args) == 0 || args[0] == "show" || args[0] == "get" {
		outputJSON(map[string]interface{}{
			"ok":          true,
			"config_path": configFile,
			"config":      cfg,
		})
		return
	}
	if args[0] == "set" && len(args) >= 3 {
		key := args[1]
		val := args[2]
		switch key {
		case "max_actions_per_session":
			fmt.Sscanf(val, "%d", &cfg.MaxActionsPerSession)
		case "max_session_duration_sec":
			fmt.Sscanf(val, "%d", &cfg.MaxSessionDurationSec)
		case "min_delay_between_destructive_ms":
			fmt.Sscanf(val, "%d", &cfg.MinDelayBetweenDestructive)
		case "max_actions_per_minute":
			fmt.Sscanf(val, "%d", &cfg.MaxActionsPerMinute)
		case "privacy_mode":
			cfg.PrivacyMode = strings.ToLower(val) == "true"
		default:
			outputError(fmt.Sprintf("Unknown config key %q", key))
		}
		_ = saveConfig(cfg)
		outputJSON(map[string]interface{}{
			"ok":     true,
			"action": "set-config",
			"key":    key,
			"value":  val,
			"config": cfg,
		})
		return
	}
	outputError("Usage: deskctl config <show|set <key> <val>>")
}

func isDestructiveCommand(cmd string) bool {
	destructive := map[string]bool{
		"close-window": true, "file trash": true, "processes kill": true,
		"setup-sudo": true, "setup-polkit": true, "panic": true,
	}
	return destructive[cmd]
}

// checkSecurityAndScope validates protected zones, declared session scopes, budgets, and rate limits
func checkSecurityAndScope(cmdName string, args []string) error {
	// 1. Budget and rate limits check
	if err := checkBudgetAndRateLimit(cmdName); err != nil {
		return err
	}

	// 2. Validate session action scope
	if err := validateSessionScope(cmdName, ""); err != nil {
		return err
	}

	// 2. Check for protected zones on file and shell commands
	switch cmdName {
	case "file":
		if len(args) >= 2 {
			filePath := args[1]
			if protected, desc := isProtectedZone(filePath); protected {
				logViolation("protected_zone", filePath, desc)
				return fmt.Errorf("Protected zone violation: access to %q is strictly prohibited (%s)", filePath, desc)
			}
			if err := validateSessionScope("file", filePath); err != nil {
				return err
			}
		}
	case "open":
		if len(args) > 0 {
			target := args[0]
			if protected, desc := isProtectedZone(target); protected {
				logViolation("protected_zone", target, desc)
				return fmt.Errorf("Protected zone violation: opening %q is strictly prohibited (%s)", target, desc)
			}
			if err := validateSessionScope("open", target); err != nil {
				return err
			}
		}
	case "run":
		if len(args) > 0 {
			cmdStr := strings.Join(args, " ")
			if protected, desc := checkShellCommandForProtectedZones(cmdStr); protected {
				logViolation("protected_zone_shell", cmdStr, desc)
				return fmt.Errorf("Protected zone violation in shell command: %s", desc)
			}
		}
	case "launch", "focus":
		if len(args) > 0 {
			if err := validateSessionScope(cmdName, args[0]); err != nil {
				return err
			}
		}
	}

	return nil
}
