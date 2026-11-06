package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

func doRun(commandStr string, timeoutSec int, dryRun bool) {
	logAction("run", []string{commandStr}, true, dryRun, "")

	if dryRun {
		outputJSON(map[string]interface{}{
			"ok":        true,
			"dry_run":   true,
			"command":   commandStr,
			"exit_code": 0,
			"stdout":    "(simulated dry run)",
			"stderr":    "",
		})
		return
	}

	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", commandStr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			outputJSON(map[string]interface{}{
				"ok":        false,
				"error":     fmt.Sprintf("Command timed out after %d seconds", timeoutSec),
				"exit_code": -1,
				"stdout":    stdout.String(),
				"stderr":    stderr.String(),
			})
			return
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	outputJSON(map[string]interface{}{
		"ok":        exitCode == 0,
		"command":   commandStr,
		"exit_code": exitCode,
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
	})
}

func doWait(ms int) {
	if ms < 0 {
		ms = 0
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	outputJSON(map[string]interface{}{
		"ok":        true,
		"waited_ms": ms,
	})
}
