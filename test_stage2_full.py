#!/usr/bin/env python3
"""
Stage 2 Full Action Toolkit Verification
Tests all Stage 2 commands on the live system in a contained, non-disruptive way.
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
        out = json.loads(res.stdout)
        return out
    except Exception as e:
        print(f"Error parsing JSON from {cmd}:")
        print("STDOUT:", res.stdout)
        print("STDERR:", res.stderr)
        return {"ok": False, "error": f"JSON parse error: {e}", "raw": res.stdout}

def test_stage2():
    results = {}
    print("==================================================")
    print("Starting Stage 2 Action Toolkit Verification Suite")
    print("==================================================")

    # Backup clipboard
    orig_clip = run_cmd(["clipboard", "get"]).get("text", "")

    try:
        # 0. Session Start
        print("\n--- [1/8] Sessions ---")
        s_start = run_cmd(["session", "start", "--task", "Stage 2 Verification", "--scope", "test-scope"])
        assert s_start.get("ok"), "session start failed"
        session_id = s_start.get("session_id")
        print(f"Session started: {session_id}")
        results["session_start"] = True

        # 1. Screen & Displays
        print("\n--- [2/8] Screens & Screenshots ---")
        screens = run_cmd(["screens"])
        assert screens.get("ok") and len(screens.get("screens", [])) > 0
        print(f"Screens query ok: found {len(screens.get('screens', []))} monitors")
        for s in screens["screens"]:
            print(f"  Monitor #{s['id']} [{s['name']}]: {s['w']}x{s['h']} at ({s['x']}, {s['y']}) primary={s['primary']}")
        results["screens"] = True

        shot = run_cmd(["screenshot", "--max-width", "1280"])
        assert shot.get("ok") and os.path.exists(shot["path"])
        print(f"Screenshot captured: {shot['path']} ({shot['image_width']}x{shot['image_height']}), scale={shot.get('scale_factor')}")
        results["screenshot"] = True

        # 2. Vision & OCR
        print("\n--- [3/8] Vision & OCR ---")
        zoom = run_cmd(["zoom", "--region", "100,100,200,200", "--factor", "2"])
        assert zoom.get("ok") and os.path.exists(zoom["path"])
        print(f"Zoom region ok: {zoom['path']}")
        results["zoom"] = True

        diff = run_cmd(["diff", "--screen", "0"])
        assert diff.get("ok")
        print(f"Diff ok: changed={diff.get('changed')}, diff_percent={diff.get('diff_percent')}")
        results["diff"] = True

        stable = run_cmd(["wait-for-stable", "--threshold", "6.0", "--timeout-ms", "3000"])
        assert stable.get("ok")
        print(f"Wait-for-stable ok: stable={stable.get('stable')}")
        results["wait-for-stable"] = True

        color = run_cmd(["color-at-pixel", "50", "50"])
        assert color.get("ok") and color.get("hex")
        print(f"Color-at-pixel ok: {color.get('hex')}")
        results["color-at-pixel"] = True

        ocr = run_cmd(["ocr", "--screen", "0"])
        assert ocr.get("ok")
        print(f"OCR ok: detected {len(ocr.get('words', []))} words")
        results["ocr"] = True

        # Screen recording test (brief 1s recording)
        rec_start = run_cmd(["record-screen", "start", "--fps", "10"])
        assert rec_start.get("ok")
        time.sleep(1.0)
        rec_status = run_cmd(["record-screen", "status"])
        assert rec_status.get("recording")
        rec_stop = run_cmd(["record-screen", "stop"])
        assert rec_stop.get("ok") and os.path.exists(rec_stop.get("file"))
        print(f"Screen recording ok: {rec_stop.get('file')} ({os.path.getsize(rec_stop.get('file'))} bytes)")
        results["record-screen"] = True

        # 3. Windows & Containment Test Window
        print("\n--- [4/8] Window Management ---")
        launch = run_cmd(["launch", "xed"])
        assert launch.get("ok")
        time.sleep(1.0)

        wait_w = run_cmd(["wait-for-window", "--title", "xed", "--timeout", "5"])
        assert wait_w.get("ok")
        win_id = wait_w["window"]["id"]
        print(f"Launched test window: ID={win_id}, title={wait_w['window']['title']}")
        results["launch_and_wait"] = True

        # Focus
        foc = run_cmd(["focus", "--id", win_id])
        assert foc.get("ok")
        print(f"Focus window ok: {win_id}")
        results["focus"] = True

        # Resize & Move window to known safe bounds
        res_w = run_cmd(["resize-window", "--id", win_id, "--w", "500", "--h", "400"])
        assert res_w.get("ok")
        mov_w = run_cmd(["move-window", "--id", win_id, "--x", "200", "--y", "200"])
        assert mov_w.get("ok")
        print("Window resized to 500x400 and moved to (200, 200)")
        results["resize_move_window"] = True

        # Workspaces list
        ws = run_cmd(["workspaces", "list"])
        assert ws.get("ok")
        print(f"Workspaces query ok: {ws.get('count')} workspaces")
        results["workspaces"] = True

        # Always-on-top toggle
        aot = run_cmd(["always-on-top", "--id", win_id])
        assert aot.get("ok")
        print("Window always-on-top toggled ok")
        results["always-on-top"] = True

        # 4. Safe Input Actions (Targeting the test window interior)
        print("\n--- [5/8] Input Operations ---")
        mpos = run_cmd(["mouse-position"])
        assert mpos.get("ok")
        print(f"Mouse position ok: ({mpos.get('x')}, {mpos.get('y')})")
        results["mouse-position"] = True

        # Move with dry-run
        move_dry = run_cmd(["move", "300", "300", "--dry-run"])
        assert move_dry.get("ok") and move_dry.get("dry_run")
        print("Move --dry-run ok")
        results["dry-run"] = True

        # Move inside test window (300, 300)
        move_live = run_cmd(["move", "300", "300"])
        assert move_live.get("ok")
        hover = run_cmd(["hover", "300", "300", "--dwell-ms", "50"])
        assert hover.get("ok")
        print("Move & hover live ok")
        results["move_hover"] = True

        # Click inside text area of xed with screenshot-after
        click = run_cmd(["click", "300", "300", "--screenshot-after"])
        assert click.get("ok") and click.get("screenshot_after")
        print(f"Click with --screenshot-after ok: {click.get('screenshot_after')}")
        results["click_screenshot_after"] = True

        # Scroll inside text area
        scroll = run_cmd(["scroll", "300", "300", "--direction", "down", "--amount", "2"])
        assert scroll.get("ok")
        print("Scroll ok")
        results["scroll"] = True

        # Hold and release
        hold = run_cmd(["hold-and-release", "300", "300", "--hold-ms", "50"])
        assert hold.get("ok")
        print("Hold-and-release ok")
        results["hold-and-release"] = True

        # Drag path inside editor text area
        drag_p = run_cmd(["drag-path", "300,300 320,320 340,300"])
        assert drag_p.get("ok")
        print("Drag-path ok")
        results["drag-path"] = True

        # Typing unicode and newlines into editor
        type_res = run_cmd(["type", "Hello Deskctl!\nUnicode: 🚀 ✓ 測試", "--delay", "10"])
        assert type_res.get("ok")
        print("Type unicode & newlines ok")
        results["type"] = True

        # Select all and delete via key combos
        key_res = run_cmd(["key", "ctrl+a"])
        assert key_res.get("ok")
        key_hold = run_cmd(["key-hold", "BackSpace", "--hold-ms", "50"])
        assert key_hold.get("ok")
        print("Key combo & key-hold ok")
        results["keys"] = True

        # Close test editor window
        close_res = run_cmd(["close-window", "--id", win_id])
        assert close_res.get("ok")
        print("Closed test window cleanly")
        results["close-window"] = True

        # 5. Accessibility (AT-SPI)
        print("\n--- [6/8] Accessibility (AT-SPI) ---")
        # Global UI tree
        tree = run_cmd(["ui-tree", "--max-nodes", "25"])
        assert tree.get("ok")
        print(f"AT-SPI global ui-tree ok: {tree.get('count')} nodes inspected")
        results["ui-tree"] = True

        # Targeted find & element click (using System Monitor or global search)
        find_elem = run_cmd(["find", "System"])
        assert find_elem.get("ok")
        print(f"AT-SPI find element ok: found {len(find_elem.get('matches', []))} matches")
        results["find_element"] = True

        # Documented fallback test on unsupported target
        fallback_test = run_cmd(["ui-tree", "--window", "0x99999999"])
        assert not fallback_test.get("ok") and fallback_test.get("fallback") == "screenshot_coordinates"
        print(f"AT-SPI fallback mechanism verified: fallback={fallback_test.get('fallback')}")
        results["atspi_fallback"] = True

        # 6. System & Utilities
        print("\n--- [7/8] System & Utilities ---")
        # Clipboard set, get, history
        clip_set = run_cmd(["clipboard", "set", "deskctl-token-test-xyz"])
        assert clip_set.get("ok")
        clip_get = run_cmd(["clipboard", "get"])
        assert clip_get.get("ok") and clip_get.get("text") == "deskctl-token-test-xyz"
        clip_hist = run_cmd(["clipboard", "history"])
        assert clip_hist.get("ok")
        print(f"Clipboard get/set/history ok: val='{clip_get.get('text')}'")
        results["clipboard"] = True

        # Shell execution
        shell_run = run_cmd(["run", "echo 'DESKCTL_SHELL_OK'", "--timeout", "5"])
        assert shell_run.get("ok") and "DESKCTL_SHELL_OK" in shell_run.get("stdout")
        print("Shell run ok")
        results["run"] = True

        # Notification
        notif = run_cmd(["notify", "Stage 2 Toolkit Verified Successfully", "--title", "deskctl"])
        assert notif.get("ok")
        print("Desktop notification ok")
        results["notify"] = True

        # File operations
        f_write = run_cmd(["file", "write", "/tmp/deskctl_test_file.txt", "Stage 2 File I/O Verified"])
        assert f_write.get("ok")
        f_read = run_cmd(["file", "read", "/tmp/deskctl_test_file.txt"])
        assert f_read.get("ok") and "Stage 2 File I/O" in f_read.get("content")
        f_list = run_cmd(["file", "list", "/tmp"])
        assert f_list.get("ok") and any(item["name"] == "deskctl_test_file.txt" for item in f_list.get("items", []))
        f_trash = run_cmd(["file", "trash", "/tmp/deskctl_test_file.txt"])
        assert f_trash.get("ok")
        print("File write/read/list/trash ok")
        results["file_ops"] = True

        # Processes, volume, brightness, network, battery
        procs = run_cmd(["processes", "list", "deskctl"])
        assert procs.get("ok")
        vol = run_cmd(["volume", "get"])
        assert vol.get("ok")
        bright = run_cmd(["brightness", "get"])
        assert bright.get("ok")
        net = run_cmd(["network"])
        assert net.get("ok")
        batt = run_cmd(["battery"])
        assert batt.get("ok")
        print(f"System metrics ok: vol={vol.get('volume')}%, bright={bright.get('brightness')}%, net={net.get('status')}, batt={batt.get('status')}")
        results["system_metrics"] = True

        # Schedule
        sched = run_cmd(["schedule", "--delay", "1", "--command", "echo deskctl_schedule_done > /tmp/sched_test.txt"])
        assert sched.get("ok")
        time.sleep(1.5)
        assert os.path.exists("/tmp/sched_test.txt")
        os.remove("/tmp/sched_test.txt")
        print("Schedule timer ok")
        results["schedule"] = True

        # 7. Reliability & Stuck Detector
        print("\n--- [8/8] Reliability & Stuck Detector ---")
        run_cmd(["reset"])
        fail1 = run_cmd(["focus", "--id", "0x99999999"])
        assert not fail1.get("ok")
        fail2 = run_cmd(["focus", "--id", "0x99999999"])
        assert not fail2.get("ok")
        fail3 = run_cmd(["focus", "--id", "0x99999999"])
        assert not fail3.get("ok")
        print(f"Stuck detector verified on 3rd failure: {fail3.get('error')}")
        assert "Stuck detector triggered" in fail3.get("error") or fail3.get("stuck") == True
        results["stuck_detector"] = True
        run_cmd(["reset"])

        # End session
        s_end = run_cmd(["session", "end"])
        assert s_end.get("ok")
        print(f"Session ended ok: {s_end.get('session_dir')}")
        results["session_end"] = True

    finally:
        # Restore clipboard
        if orig_clip:
            run_cmd(["clipboard", "set", orig_clip])

    print("\n==================================================")
    print("STAGE 2 VERIFICATION SUMMARY:")
    all_ok = True
    for k, v in results.items():
        status = "PASSED" if v else "FAILED"
        print(f"  [✓] {k:<25}: {status}")
        if not v:
            all_ok = False
    print("==================================================")
    return all_ok

if __name__ == "__main__":
    success = test_stage2()
    sys.exit(0 if success else 1)
