#!/usr/bin/env python3
"""
Stage 3 Password and Privilege Handling Verification Suite
Tests:
1. Auth check & prompt detection
2. Auth wait and user handoff
3. Scoped sudo setup validation (visudo -c check, reject NOPASSWD: ALL)
4. Sudo setup confirmation requirements
5. Sudo setup revoke dry-run
6. Polkit rules generation
7. Real auth prompt interception (ensures agent NEVER types into password dialogs)
"""

import os
import sys
import json
import time
import subprocess

DESKCTL = "/home/jimmy/.local/bin/deskctl"

def run_cmd(args):
    cmd = [DESKCTL] + args
    res = subprocess.run(cmd, capture_output=True, text=True)
    try:
        return json.loads(res.stdout)
    except Exception as e:
        print(f"Error parsing JSON from {cmd}:")
        print("STDOUT:", res.stdout)
        print("STDERR:", res.stderr)
        return {"ok": False, "error": f"JSON parse error: {e}", "raw": res.stdout}

def test_stage3():
    results = {}
    print("==================================================")
    print("Starting Stage 3 Auth & Privilege Verification Suite")
    print("==================================================")

    # 1. Idle Auth Check
    print("\n--- [1/6] Idle Auth Check ---")
    chk = run_cmd(["auth-check"])
    assert chk.get("ok"), "auth-check failed"
    prompt = chk.get("prompt", {})
    assert not prompt.get("detected"), "Detected unexpected auth prompt on idle desktop"
    print("Idle auth check ok: detected=False")
    results["idle_auth_check"] = True

    # 2. Auth Wait when no prompt
    print("\n--- [2/6] Auth Wait (Idle) ---")
    wait_res = run_cmd(["auth-wait", "--timeout", "1"])
    assert wait_res.get("ok") and wait_res.get("resolved")
    print(f"Auth wait idle ok: message='{wait_res.get('message')}'")
    results["auth_wait_idle"] = True

    # 3. Scoped Sudo Setup Validation (visudo -c check)
    print("\n--- [3/6] Scoped Sudo Setup Validation ---")
    dry_sudo = run_cmd(["setup-sudo", "--dry-run"])
    assert dry_sudo.get("ok")
    assert "parsed OK" in dry_sudo.get("visudo_validation", "")
    assert "NOPASSWD: ALL" not in dry_sudo.get("proposed_content", "")
    assert "/usr/bin/apt update" in dry_sudo.get("proposed_content", "")
    print(f"visudo validation ok: {dry_sudo.get('visudo_validation')}")
    print("Explicit security check: NOPASSWD: ALL is NOT present.")
    results["sudo_visudo_validation"] = True

    # 4. Confirmation Requirement
    print("\n--- [4/6] Sudo Setup Confirmation Check ---")
    no_confirm = run_cmd(["setup-sudo"])
    assert no_confirm.get("ok")
    assert no_confirm.get("confirmation_required") == True
    print("setup-sudo safely paused requiring explicit --confirm flag before writing to /etc/sudoers.d/deskctl")
    results["sudo_confirmation_check"] = True

    # 5. Revoke Dry Run & Polkit Setup
    print("\n--- [5/6] Revoke Dry Run & Polkit Rule Setup ---")
    rev_dry = run_cmd(["setup-sudo", "--revoke", "--dry-run"])
    assert rev_dry.get("ok") and rev_dry.get("dry_run")
    print("setup-sudo --revoke --dry-run ok")

    polkit_dry = run_cmd(["setup-polkit", "--dry-run"])
    assert polkit_dry.get("ok")
    assert "org.debian.apt.update-cache" in polkit_dry.get("proposed_content", "")
    print("setup-polkit --dry-run ok: scoped to package operations")
    results["revoke_and_polkit"] = True

    # 6. Real Auth Prompt Detection & Input Interception
    print("\n--- [6/6] Real Auth Prompt Interception & Handoff ---")
    # Launch a simulated authentication dialog using zenity password dialog
    # Window class: zenity, title: "Authentication Required"
    zen_proc = subprocess.Popen(
        ["zenity", "--password", "--title=Authentication Required"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL
    )
    time.sleep(1.0)

    try:
        # Check that deskctl detects the active password dialog
        detected_chk = run_cmd(["auth-check"])
        assert detected_chk.get("ok")
        p_info = detected_chk.get("prompt", {})
        print(f"Auth check during active prompt: detected={p_info.get('detected')}, type={p_info.get('prompt_type')}, title='{p_info.get('title')}'")
        assert p_info.get("detected") == True

        # CRITICAL TEST: Verify that deskctl BLOCKS the agent from auto-typing while auth prompt is open!
        type_attempt = run_cmd(["type", "my_secret_password"])
        print(f"Type attempt result while auth prompt is open: ok={type_attempt.get('ok')}, auth_prompt={type_attempt.get('auth_prompt')}")
        assert type_attempt.get("ok") == False
        assert type_attempt.get("auth_prompt") == True
        print("PASS: Agent was strictly BLOCKED from auto-typing into authentication prompt!")
        results["input_blocked_on_auth"] = True

    finally:
        # Close the simulated dialog
        zen_proc.terminate()
        zen_proc.wait()
        time.sleep(0.5)

    # Verify that once the prompt is closed, auth-check clears
    chk_after = run_cmd(["auth-check"])
    assert not chk_after.get("prompt", {}).get("detected")
    print("Auth check after dialog closed: detected=False. Normal operations resume.")
    results["auth_cleared_after_close"] = True

    print("\n==================================================")
    print("STAGE 3 VERIFICATION SUMMARY:")
    all_ok = True
    for k, v in results.items():
        status = "PASSED" if v else "FAILED"
        print(f"  [✓] {k:<28}: {status}")
        if not v:
            all_ok = False
    print("==================================================")
    return all_ok

if __name__ == "__main__":
    success = test_stage3()
    sys.exit(0 if success else 1)
