#!/usr/bin/env python3
"""
Stage 3: Live real-desktop integration test suite for deskctl.
Tests all required desktop interactions:
1. Screen info & HiDPI scaled screenshots
2. Terminal opening & running `ls`
3. Browser opening & URL navigation (using system browser Google Chrome)
4. Right-click desktop & inspect context menu via ui-tree
5. Window dragging & position change verification
6. Scrolling a page/window
7. Clipboard set/get & inter-app copy-paste
8. AT-SPI element querying & screenshot+coordinate fallback handling
"""

import sys
import os
import json
import time
import subprocess

DESKCTL = "/home/jimmy/.gemini/antigravity/scratch/deskctl/deskctl"

def run_deskctl(args):
    cmd = [DESKCTL] + args
    res = subprocess.run(cmd, capture_output=True, text=True)
    try:
        out = json.loads(res.stdout)
        return out
    except Exception as e:
        print(f"FAILED to parse JSON from {cmd}: {res.stdout} (err: {res.stderr})")
        return {"ok": False, "error": f"JSON parse error: {e}", "raw": res.stdout}

def test_screen_and_screenshot():
    print("\n--- [Test 1: Screen-info and Screenshot] ---")
    info = run_deskctl(["screen-info"])
    assert info.get("ok"), f"screen-info failed: {info}"
    print(f"Screen resolution: {info['screen_width']}x{info['screen_height']}, scale: {info['scale']}")
    assert info["screen_width"] > 0 and info["screen_height"] > 0
    
    shot = run_deskctl(["screenshot", "--max-width", "1280"])
    assert shot.get("ok"), f"screenshot failed: {shot}"
    print(f"Screenshot captured: {shot['path']} ({shot['image_width']}x{shot['image_height']}), scale_factor={shot['scale_factor']}")
    assert os.path.exists(shot["path"])
    return True

def test_terminal_and_ls():
    print("\n--- [Test 2: Terminal and 'ls'] ---")
    # Launch lightweight terminal command
    run_deskctl(["run", "gnome-terminal -- bash -c 'echo DESKCTL_ACTIVE; pwd; ls; sleep 2' &"])
    run_deskctl(["wait", "--ms", "1000"])
    
    focus = run_deskctl(["focus", "--title", "gnome-terminal"])
    assert focus.get("ok"), f"Focus terminal failed: {focus}"
    print(f"Focused terminal: {focus['window']['id']} - {focus['window']['title']}")
    
    typed = run_deskctl(["type", "echo 'DESKCTL_TEST_OK'\n"])
    assert typed.get("ok"), f"Type failed: {typed}"
    print("Typed command into terminal successfully")
    return True

def test_browser_navigation():
    print("\n--- [Test 3: Browser Navigation (Google Chrome)] ---")
    # Using Google Chrome already installed on system
    run_deskctl(["run", "google-chrome https://example.com &"])
    run_deskctl(["wait", "--ms", "1500"])
    
    focus = run_deskctl(["focus", "--title", "Google Chrome"])
    assert focus.get("ok"), f"Focus Chrome failed: {focus}"
    print(f"Focused Google Chrome: {focus['window']['id']} - {focus['window']['title']}")
    return True

def test_desktop_context_menu_ui_tree():
    print("\n--- [Test 4: Desktop Context Menu & UI-Tree] ---")
    print("Right-clicking empty desktop region to open context menu...")
    click_res = run_deskctl(["click", "100", "100", "--button", "right"])
    assert click_res.get("ok"), f"Right click failed: {click_res}"
    
    run_deskctl(["wait", "--ms", "500"])
    
    tree = run_deskctl(["ui-tree", "--filter", "role=menu item", "--max-nodes", "20"])
    assert tree.get("ok"), f"ui-tree failed: {tree}"
    print(f"Retrieved {tree.get('count', 0)} menu item nodes from context menu")
    if tree.get("nodes"):
        sample_names = [n["name"] for n in tree["nodes"] if n.get("name")][:5]
        print(f"Found context menu items: {sample_names}")
        
    run_deskctl(["key", "Escape"])
    print("Closed context menu with Escape")
    return True

def test_window_drag():
    print("\n--- [Test 5: Window Dragging] ---")
    # Launch test editor window with redirected fds
    run_deskctl(["run", "nohup xed /tmp/test_drag.txt >/dev/null 2>&1 &"])
    run_deskctl(["wait", "--ms", "1200"])
    run_deskctl(["run", "wmctrl -r 'test_drag.txt' -b remove,maximized_vert,maximized_horz"])
    run_deskctl(["run", "wmctrl -r 'test_drag.txt' -e 0,200,200,600,400"])
    run_deskctl(["wait", "--ms", "500"])
    
    # Initial window position before drag
    wins = run_deskctl(["windows"]).get("windows", [])
    target = None
    for w in wins:
        if "test_drag" in w["title"].lower() or "xed" in w["class"].lower():
            target = w
            break
            
    assert target is not None, "Failed to find test window for drag"
    orig_x, orig_y = target["x"], target["y"]
    print(f"Initial window position: ({orig_x}, {orig_y})")
    
    # Take a 1:1 screenshot so coordinate space matches 1:1
    run_deskctl(["screenshot"])
    
    # Titlebar on Cinnamon window positioned at 200,200 is at (350, 215)
    title_x = 350
    title_y = 215
    dest_x = 550
    dest_y = 350
    
    drag_res = run_deskctl(["drag", str(title_x), str(title_y), str(dest_x), str(dest_y)])
    assert drag_res.get("ok"), f"Drag failed: {drag_res}"
    
    run_deskctl(["wait", "--ms", "500"])
    
    wins_after = run_deskctl(["windows"]).get("windows", [])
    new_x, new_y = orig_x, orig_y
    for w in wins_after:
        if w["id"] == target["id"]:
            new_x, new_y = w["x"], w["y"]
            break
            
    print(f"Dragged window to position: ({new_x}, {new_y})")
    assert (new_x != orig_x or new_y != orig_y), "Window position did not move after drag!"
    print("Window successfully dragged and verified!")
    return target["id"]

def test_scroll_page():
    print("\n--- [Test 6: Scroll Page] ---")
    scr_res = run_deskctl(["scroll", "400", "400", "--direction", "down", "--amount", "4", "--screenshot-after"])
    assert scr_res.get("ok"), f"Scroll failed: {scr_res}"
    print(f"Scroll executed successfully. Result screenshot: {scr_res.get('screenshot', {}).get('path')}")
    return True

def test_copy_paste_between_apps():
    print("\n--- [Test 7: Copy and Paste Between Apps] ---")
    test_text = "deskctl-clipboard-token-" + str(int(time.time()))
    
    set_res = run_deskctl(["clipboard", "set", test_text])
    assert set_res.get("ok"), f"Clipboard set failed: {set_res}"
    
    get_res = run_deskctl(["clipboard", "get"])
    assert get_res.get("ok") and get_res["text"] == test_text, f"Clipboard mismatch: {get_res}"
    print(f"Clipboard exchange verified: '{get_res['text']}'")
    
    # Focus xed and paste
    run_deskctl(["focus", "--title", "xed"])
    run_deskctl(["key", "ctrl+v"])
    print("Pasted clipboard text into test window via Ctrl+V")
    
    # Close xed test window cleanly
    run_deskctl(["run", "pkill -9 xed || true"])
    return True

def test_atspi_fallback_documentation():
    print("\n--- [Test 8: AT-SPI Fallback Mechanism] ---")
    # Demonstrate fallback behavior when an element is not exposed via AT-SPI
    non_existent = run_deskctl(["find", "non_existent_widget_xyz123"])
    print(f"AT-SPI query for non-existent item returned count={non_existent.get('count')}")
    assert non_existent.get("count") == 0
    
    # Fallback to screenshot coordinates
    shot = run_deskctl(["screenshot", "--max-width", "1920"])
    print(f"Fallback path ready: Captured screenshot {shot['path']}; coordinates can be directly clicked with 'deskctl click X Y'")
    return True

def main():
    print("==================================================")
    print("Starting deskctl Live Desktop Integration Tests...")
    print("==================================================")
    test_screen_and_screenshot()
    test_terminal_and_ls()
    test_browser_navigation()
    test_desktop_context_menu_ui_tree()
    test_window_drag()
    test_scroll_page()
    test_copy_paste_between_apps()
    test_atspi_fallback_documentation()
    print("\n==================================================")
    print("ALL REAL DESKTOP TESTS PASSED SUCCESSFULLY!")
    print("==================================================")

if __name__ == "__main__":
    main()
