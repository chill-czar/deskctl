#!/usr/bin/env python3
"""
Stage 1: Multi-Screen Display Model Tests.
Tests dynamic runtime detection of monitors, different resolutions,
simulated virtual monitors (via xrandr), and simulated independent display
servers (via Xephyr) to ensure zero hardcoded resolutions or topologies.
"""

import os
import sys
import json
import subprocess
import time

DESKCTL = "/home/jimmy/.local/bin/deskctl"

def run_cmd(args, env=None):
    e = os.environ.copy()
    if env:
        e.update(env)
    res = subprocess.run([DESKCTL] + args, capture_output=True, text=True, env=e)
    try:
        return json.loads(res.stdout)
    except Exception as exc:
        print(f"Error parsing JSON from {args}: {res.stdout} (err: {res.stderr})")
        return {"ok": False, "error": str(exc), "raw": res.stdout}

def test_screens_list():
    print("\n--- [Test 1: Query Base Connected Screens] ---")
    res = run_cmd(["screens"])
    assert res.get("ok"), f"screens failed: {res}"
    print(f"Found {res['count']} physical screens:")
    for s in res["screens"]:
        print(f"  Screen #{s['id']} [{s['name']}]: {s['w']}x{s['h']} at ({s['x']}, {s['y']}), scale={s['scale']}, primary={s['primary']}, active={s['active']}")
    v = res["virtual_desktop"]
    print(f"Virtual Desktop Bounding Box: {v['w']}x{v['h']} at ({v['x']}, {v['y']})")
    assert res["count"] >= 1
    return res

def test_screenshot_targets():
    print("\n--- [Test 2: Screenshot Across All Target Modes] ---")
    # 1. Screen all
    shot_all = run_cmd(["screenshot", "--screen", "all", "--max-width", "1280"])
    assert shot_all.get("ok"), f"all screenshot failed: {shot_all}"
    print(f"  '--screen all' -> {shot_all['path']} ({shot_all['image_width']}x{shot_all['image_height']}), scale={shot_all['scale_factor']}")

    # 2. Screen primary
    shot_pri = run_cmd(["screenshot", "--screen", "primary", "--format", "jpeg"])
    assert shot_pri.get("ok"), f"primary screenshot failed: {shot_pri}"
    print(f"  '--screen primary' (JPEG) -> {shot_pri['path']} ({shot_pri['image_width']}x{shot_pri['image_height']})")

    # 3. Screen active
    shot_act = run_cmd(["screenshot", "--screen", "active"])
    assert shot_act.get("ok"), f"active screenshot failed: {shot_act}"
    print(f"  '--screen active' -> {shot_act['path']} ({shot_act['image_width']}x{shot_act['image_height']})")

    # 4. Screen window:active
    shot_win = run_cmd(["screenshot", "--screen", "window:active"])
    assert shot_win.get("ok"), f"window:active screenshot failed: {shot_win}"
    print(f"  '--screen window:active' -> {shot_win['path']} ({shot_win['captured_width']}x{shot_win['captured_height']})")

    # 5. Screen region
    shot_reg = run_cmd(["screenshot", "--screen", "region:150,150,300,200"])
    assert shot_reg.get("ok"), f"region screenshot failed: {shot_reg}"
    print(f"  '--screen region:...' -> {shot_reg['path']} ({shot_reg['captured_width']}x{shot_reg['captured_height']})")
    return True

def test_xrandr_virtual_monitor():
    print("\n--- [Test 3: Runtime Hotplug with xrandr Virtual Monitor] ---")
    # Add a virtual monitor with custom resolution (1024x768 at +500+200)
    print("Creating xrandr virtual monitor SIM-VIRT (1024x768 at +500+200)...")
    subprocess.run(["xrandr", "--setmonitor", "SIM-VIRT", "1024/200x768/150+500+200", "none"], check=True)
    
    try:
        res = run_cmd(["screens"])
        assert res.get("ok"), f"screens query failed: {res}"
        names = [s["name"] for s in res["screens"]]
        print(f"Detected screens after virtual monitor added: {names}")
        assert "SIM-VIRT" in names, "deskctl failed to detect virtual monitor at runtime!"
        
        sim = [s for s in res["screens"] if s["name"] == "SIM-VIRT"][0]
        print(f"Virtual Monitor verified: {sim['w']}x{sim['h']} at ({sim['x']}, {sim['y']})")
        assert sim["w"] == 1024 and sim["h"] == 768
        
        # Test taking screenshot specifically targeting SIM-VIRT
        shot_sim = run_cmd(["screenshot", "--screen", "SIM-VIRT", "--max-width", "512"])
        assert shot_sim.get("ok"), f"screenshot of virtual monitor failed: {shot_sim}"
        print(f"Screenshot targeting virtual monitor: {shot_sim['path']} ({shot_sim['image_width']}x{shot_sim['image_height']}), scale={shot_sim['scale_factor']}")
        assert shot_sim["image_width"] == 512
        assert shot_sim["scale_factor"] == 0.5
        
        # Test coordinate conversion inside virtual monitor
        click_res = run_cmd(["click", "256", "192", "--screen", "SIM-VIRT", "--dry-run"])
        assert click_res.get("ok"), f"click on virtual monitor failed: {click_res}"
        # With scale 0.5, (256, 192) in image space maps to (512, 384) relative to monitor offset (500, 200) -> global (1012, 584)
        sc = click_res["screen_coords"]
        print(f"Mapped click (256, 192) on SIM-VIRT -> Global coordinates ({sc['x']}, {sc['y']})")
        assert sc["x"] == 1012 and sc["y"] == 584
        print("Virtual monitor coordinate conversion verified perfectly!")
    finally:
        print("Cleaning up xrandr virtual monitor...")
        subprocess.run(["xrandr", "--delmonitor", "SIM-VIRT"], check=True)
        
    res_cleaned = run_cmd(["screens"])
    names_cleaned = [s["name"] for s in res_cleaned["screens"]]
    assert "SIM-VIRT" not in names_cleaned
    print("Virtual monitor removed and verified clean.")
    return True

def test_xephyr_simulated_display():
    print("\n--- [Test 4: Xephyr Simulated Foreign Display (1440x900)] ---")
    display_num = ":10"
    print(f"Launching Xephyr X11 server on {display_num} with resolution 1440x900...")
    proc = subprocess.Popen(["Xephyr", display_num, "-screen", "1440x900", "-ac"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    time.sleep(1.0)
    
    try:
        env = {"DISPLAY": display_num}
        res = run_cmd(["screens"], env=env)
        assert res.get("ok"), f"Xephyr screens query failed: {res}"
        print(f"Xephyr environment detected {res['count']} screens:")
        for s in res["screens"]:
            print(f"  Screen #{s['id']} [{s['name']}]: {s['w']}x{s['h']} at ({s['x']}, {s['y']})")
        assert res["count"] == 1
        assert res["screens"][0]["w"] == 1440 and res["screens"][0]["h"] == 900
        
        shot = run_cmd(["screenshot", "--screen", "all", "--max-width", "720"], env=env)
        assert shot.get("ok"), f"Xephyr screenshot failed: {shot}"
        print(f"Xephyr screenshot: {shot['path']} ({shot['image_width']}x{shot['image_height']}), scale={shot['scale_factor']}")
        assert shot["image_width"] == 720 and shot["scale_factor"] == 0.5
        print("Xephyr foreign resolution dynamically supported without hardcoding!")
    finally:
        proc.terminate()
        proc.wait()
        print("Xephyr server terminated.")
    return True

def main():
    print("==================================================")
    print("STAGE 1: Display Model & Multi-Screen Test Suite")
    print("==================================================")
    test_screens_list()
    test_screenshot_targets()
    test_xrandr_virtual_monitor()
    test_xephyr_simulated_display()
    print("\n==================================================")
    print("STAGE 1 DISPLAY MODEL TESTS PASSED COMPLETELY!")
    print("==================================================")

if __name__ == "__main__":
    main()
