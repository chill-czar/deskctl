#!/usr/bin/env python3
"""
Stage 6 End-to-End Real Desktop Test Suite for deskctl.
Verifies all 10 core desktop scenarios specified in Stage 6 on the live Linux Mint system.
"""

import os
import sys
import time
import json
import shutil
import subprocess
from pathlib import Path

DESKCTL = os.path.expanduser("~/.local/bin/deskctl")
if not os.path.exists(DESKCTL):
    DESKCTL = shutil.which("deskctl")
assert DESKCTL, "deskctl binary not found in PATH or ~/.local/bin"

def run_cmd(args, check_ok=True):
    cmd = [DESKCTL] + args
    res = subprocess.run(cmd, capture_output=True, text=True)
    try:
        data = json.loads(res.stdout)
    except Exception as e:
        print(f"FAILED TO PARSE JSON: stdout='{res.stdout}', stderr='{res.stderr}'")
        raise e
    if check_ok and not data.get("ok"):
        raise AssertionError(f"Command failed: {args} -> {data}")
    return data

passed_tests = []
failed_tests = []

def run_test(name, fn):
    print(f"\n==========================================")
    print(f"RUNNING: {name}")
    print(f"==========================================")
    try:
        fn()
        print(f"[PASS] {name}")
        passed_tests.append(name)
    except Exception as e:
        print(f"[FAIL] {name}: {e}")
        failed_tests.append((name, str(e)))

# ==============================================================================
# TEST 1: Open terminal and run ls
# ==============================================================================
def test_1_terminal_ls():
    before_wins = run_cmd(["windows"]).get("windows", [])
    before_ids = {w["id"] for w in before_wins}

    # Launch new terminal
    run_cmd(["launch", "gnome-terminal"])
    
    # Wait for terminal window by class
    wait_res = run_cmd(["wait-for-window", "--title", "Gnome-terminal", "--timeout", "8"])
    assert wait_res.get("found"), f"Terminal window not found: {wait_res}"
    
    # Identify the new terminal window
    time.sleep(0.5)
    after_wins = run_cmd(["windows"]).get("windows", [])
    new_win_id = None
    for w in after_wins:
        if "gnome-terminal" in w.get("class", "").lower() and w["id"] not in before_ids:
            new_win_id = w["id"]
            break
    if not new_win_id:
        new_win_id = wait_res["window"]["id"]
    
    # Focus
    run_cmd(["focus", "--id", new_win_id])
    time.sleep(0.5)
    
    # Type ls command
    run_cmd(["type", "ls -la /tmp\n", "--delay", "10"])
    time.sleep(0.6)
    
    # Capture screenshot for visual verification
    shot = run_cmd(["screenshot", "--screen", f"window:{new_win_id}"])
    assert shot.get("ok"), f"Screenshot failed: {shot}"
    assert os.path.exists(shot.get("path")), "Screenshot file does not exist"
    
    # Close window cleanly
    run_cmd(["close-window", "--id", new_win_id])
    time.sleep(0.5)

# ==============================================================================
# TEST 2: Create a folder in file manager
# ==============================================================================
def test_2_file_manager_folder():
    test_dir = Path.home() / "Documents"
    target_folder = test_dir / "deskctl_stage6_test_folder"
    if target_folder.exists():
        shutil.rmtree(target_folder)

    # Open nemo at Documents
    run_cmd(["launch", f"nemo {test_dir}"])
    wait_res = run_cmd(["wait-for-window", "--title", "nemo", "--timeout", "8"])
    assert wait_res.get("found"), f"Nemo window not found: {wait_res}"
    nemo_win_id = wait_res["window"]["id"]

    # Focus and create folder via keyboard shortcut (Ctrl+Shift+N)
    run_cmd(["focus", "--id", nemo_win_id])
    time.sleep(0.5)
    run_cmd(["key", "ctrl+shift+n"])
    time.sleep(0.5)
    run_cmd(["type", "deskctl_stage6_test_folder"])
    time.sleep(0.3)
    run_cmd(["key", "Return"])
    time.sleep(0.8)
    
    # Verify folder was created
    assert target_folder.exists(), f"Target folder {target_folder} was not created!"
    
    # Trash folder via deskctl file trash
    trash_res = run_cmd(["file", "trash", str(target_folder)])
    assert trash_res.get("ok"), f"Failed to trash folder: {trash_res}"
    
    # Close Nemo
    run_cmd(["close-window", "--id", nemo_win_id])
    time.sleep(0.5)

# ==============================================================================
# TEST 3: Right-click and read context menu via ui-tree
# ==============================================================================
def test_3_right_click_context_menu():
    # Launch xed
    run_cmd(["launch", "xed"])
    wait_res = run_cmd(["wait-for-window", "--title", "xed", "--timeout", "8"])
    assert wait_res.get("found"), f"xed window not found: {wait_res}"
    win_id = wait_res["window"]["id"]
    
    run_cmd(["focus", "--id", win_id])
    time.sleep(0.5)
    
    # Right-click inside xed window
    run_cmd(["right-click", "400", "300"])
    time.sleep(0.5)
    
    # Read UI tree
    tree_res = run_cmd(["ui-tree", "--window", win_id], check_ok=False)
    assert tree_res.get("ok") or tree_res.get("fallback"), f"ui-tree failed without fallback: {tree_res}"
    
    # Dismiss menu and close
    run_cmd(["key", "Escape"])
    time.sleep(0.2)
    run_cmd(["close-window", "--id", win_id])
    time.sleep(0.5)

# ==============================================================================
# TEST 4: Drag a window to a second screen and back
# ==============================================================================
def test_4_move_window_screens():
    screens_res = run_cmd(["screens"])
    screens = screens_res.get("screens", [])
    assert len(screens) >= 2, f"Dual screen required for this test, found: {len(screens)}"
    
    # Launch xed as test window
    run_cmd(["launch", "xed"])
    wait_res = run_cmd(["wait-for-window", "--title", "xed", "--timeout", "8"])
    assert wait_res.get("found"), f"xed window not found: {wait_res}"
    win_id = wait_res["window"]["id"]
    
    # Move to screen 0
    move0 = run_cmd(["move-to-screen", "--window", win_id, "--screen", "0"])
    assert move0.get("ok"), f"Move to screen 0 failed: {move0}"
    time.sleep(0.5)
    
    # Move to screen 1
    move1 = run_cmd(["move-to-screen", "--window", win_id, "--screen", "1"])
    assert move1.get("ok"), f"Move to screen 1 failed: {move1}"
    time.sleep(0.5)
    
    # Move back to screen 0
    move_back = run_cmd(["move-to-screen", "--window", win_id, "--screen", "0"])
    assert move_back.get("ok"), f"Move back failed: {move_back}"
    
    run_cmd(["close-window", "--id", win_id])
    time.sleep(0.5)

# ==============================================================================
# TEST 5: Scroll a long page
# ==============================================================================
def test_5_scroll_page():
    # Test deskctl scroll command in all directions
    scroll_down = run_cmd(["scroll", "500", "500", "--direction", "down", "--amount", "3"])
    assert scroll_down.get("ok"), f"Scroll down failed: {scroll_down}"
    time.sleep(0.2)
    
    scroll_up = run_cmd(["scroll", "500", "500", "--direction", "up", "--amount", "3"])
    assert scroll_up.get("ok"), f"Scroll up failed: {scroll_up}"

# ==============================================================================
# TEST 6: Copy and paste between two apps
# ==============================================================================
def test_6_copy_paste():
    test_token = f"TOKEN_{int(time.time())}_DESKCTL"
    
    # Set token into clipboard
    set_res = run_cmd(["clipboard", "set", test_token])
    assert set_res.get("ok"), f"Clipboard set failed: {set_res}"
    
    # Verify clipboard content
    get_res = run_cmd(["clipboard", "get"])
    assert get_res.get("text") == test_token, f"Clipboard mismatch: {get_res}"
    
    # Create empty test file
    test_file = Path("/tmp/deskctl_paste_test.txt")
    test_file.touch()
        
    run_cmd(["launch", f"xed {test_file}"])
    wait_res = run_cmd(["wait-for-window", "--title", "xed", "--timeout", "8"])
    assert wait_res.get("found"), f"xed window not found: {wait_res}"
    win_id = wait_res["window"]["id"]
    
    run_cmd(["focus", "--id", win_id])
    time.sleep(0.5)
    run_cmd(["key", "ctrl+v"])
    time.sleep(0.5)
    run_cmd(["key", "ctrl+s"])
    time.sleep(0.5)
    
    # Close xed
    run_cmd(["close-window", "--id", win_id])
    time.sleep(0.5)
    
    # Verify file on disk received the pasted clipboard text
    assert test_file.exists(), "Pasted file was not saved"
    content = test_file.read_text()
    assert test_token in content, f"Pasted token not found in file: {content}"
    test_file.unlink()

# ==============================================================================
# TEST 7: Trigger a password prompt and verify pause & handoff
# ==============================================================================
def test_7_auth_prompt_handoff():
    # Spawn a password prompt window via zenity in background
    p = subprocess.Popen(["zenity", "--password", "--title=Authentication Required"])
    time.sleep(1.0)
    
    try:
        # Check auth detection
        check = run_cmd(["auth-check"])
        assert check.get("prompt", {}).get("detected"), f"auth-check failed to detect prompt: {check}"
        
        # Verify an agent input command is blocked with auth_prompt
        click_blocked = run_cmd(["click", "100", "100"], check_ok=False)
        assert click_blocked.get("auth_prompt"), f"Click was not blocked during auth: {click_blocked}"
        assert click_blocked.get("blocked"), f"Click response missing blocked=true: {click_blocked}"
    finally:
        p.terminate()
        p.wait()
        time.sleep(0.5)
        
    # Verify prompt is now resolved
    check_after = run_cmd(["auth-check"])
    assert not check_after.get("prompt", {}).get("detected"), f"auth prompt should be cleared: {check_after}"

# ==============================================================================
# TEST 8: Out-of-scope action blocked
# ==============================================================================
def test_8_scope_enforcement():
    # Start restricted session
    start_res = run_cmd(["session", "start", "--task", "Scoped Test", "--apps", "xed", "--actions", "click,type,key"])
    assert start_res.get("ok"), f"Session start failed: {start_res}"
    
    try:
        # Attempt action outside declared actions (e.g. volume)
        vol_blocked = run_cmd(["volume", "set", "50"], check_ok=False)
        assert not vol_blocked.get("ok"), "Out-of-scope action was not blocked!"
        assert vol_blocked.get("violation"), f"Expected violation=true: {vol_blocked}"
    finally:
        run_cmd(["session", "end"])

# ==============================================================================
# TEST 9: Try to read ~/.ssh and verify blocked
# ==============================================================================
def test_9_protected_zone_ssh():
    # File read check
    ssh_file = os.path.expanduser("~/.ssh/id_rsa")
    res1 = run_cmd(["file", "read", ssh_file], check_ok=False)
    assert not res1.get("ok"), f"Protected zone read should fail: {res1}"
    assert res1.get("violation"), f"Missing violation=true on ~/.ssh: {res1}"
    
    # Run command check
    res2 = run_cmd(["run", f"cat {ssh_file}"], check_ok=False)
    assert not res2.get("ok"), f"Protected zone run should fail: {res2}"
    assert res2.get("violation"), f"Missing violation=true on cat ~/.ssh: {res2}"

# ==============================================================================
# TEST 10: Kill switch mid-task abort
# ==============================================================================
def test_10_killswitch_abort():
    # Trigger stop
    stop_res = run_cmd(["stop", "Test abort scenario"])
    assert stop_res.get("ok"), f"Stop command failed: {stop_res}"
    
    try:
        # Attempt an action while STOP file is present
        action_blocked = run_cmd(["click", "200", "200"], check_ok=False)
        assert not action_blocked.get("ok"), "Action succeeded despite kill switch!"
        assert "STOP" in action_blocked.get("error", "") or action_blocked.get("killswitch"), \
            f"Expected STOP error: {action_blocked}"
    finally:
        # Reset kill switch
        reset_res = run_cmd(["reset"])
        assert reset_res.get("ok"), f"Reset failed: {reset_res}"
        
    # Verify action resumes
    resume_action = run_cmd(["mouse-position"])
    assert resume_action.get("ok"), f"Actions did not resume after reset: {resume_action}"

# ==============================================================================
# MAIN RUNNER
# ==============================================================================
def main():
    print("==================================================================")
    print("STAGE 6: REAL DESKTOP END-TO-END TEST SUITE")
    print("==================================================================")
    
    run_test("1. Open terminal and run ls", test_1_terminal_ls)
    run_test("2. Create a folder in file manager", test_2_file_manager_folder)
    run_test("3. Right-click and read context menu via ui-tree", test_3_right_click_context_menu)
    run_test("4. Drag/move window to a second screen and back", test_4_move_window_screens)
    run_test("5. Scroll a long page", test_5_scroll_page)
    run_test("6. Copy and paste between two apps", test_6_copy_paste)
    run_test("7. Trigger a password prompt and verify pause & handoff", test_7_auth_prompt_handoff)
    run_test("8. Out-of-scope action blocked", test_8_scope_enforcement)
    run_test("9. Try to read ~/.ssh and verify blocked", test_9_protected_zone_ssh)
    run_test("10. Kill switch mid-task abort", test_10_killswitch_abort)
    
    print("\n==================================================================")
    print(f"RESULTS: {len(passed_tests)} Passed, {len(failed_tests)} Failed")
    print("==================================================================")
    
    if failed_tests:
        for name, err in failed_tests:
            print(f"  - {name}: {err}")
        sys.exit(1)
    else:
        print("ALL STAGE 6 REAL DESKTOP TESTS PASSED SUCCESSFULLY!")
        sys.exit(0)

if __name__ == "__main__":
    main()
