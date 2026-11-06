# Reliability, Settle Times & Verification Patterns

UI automation on X11 / Cinnamon requires defensive timing and continuous state verification to prevent brittle execution.

## 1. Settle Times & Animation Management

Never act immediately after launching an app, submitting a form, or clicking a button that triggers a transition:
- **Explicit Wait**: `deskctl wait --ms 800`
- **Dynamic Stability Check**:
  ```bash
  deskctl wait-for-stable --threshold 5.0 --timeout-ms 3000
  ```
  Compares successive screen frames and returns when visual change drops below the threshold percentage.

---

## 2. Verification via `--screenshot-after`

Every action command (`click`, `double-click`, `right-click`, `move`, `drag`, `scroll`, `type`, `key`, `focus`, `click-element`) supports the `--screenshot-after` flag:
```bash
deskctl click 450 320 --screenshot-after
```
Response includes the execution result plus the fresh screenshot path:
```json
{
  "action": "click",
  "ok": true,
  "screenshot": {
    "path": "/home/jimmy/.deskctl/screenshots/shot_20261107_011051_000.png",
    "image_width": 3840,
    "image_height": 1080,
    "scale_factor": 1.0
  },
  "screenshot_after": "/home/jimmy/.deskctl/screenshots/shot_20261107_011051_000.png"
}
```
Always inspect the post-action screenshot to confirm that:
1. The dropdown opened, or
2. The modal dismissed, or
3. The typed text appeared in the input field.

---

## 3. The 3-Failure Stuck Detector

`deskctl` includes an automated loop prevention circuit in `safety.go`:
- Tracks consecutive identical failures across commands.
- If the exact same action fails 3 times in a row, `deskctl` halts immediately:
  ```json
  {
    "ok": false,
    "error": "Stuck detector triggered: action \"focus\" failed 3 times consecutively. Halting for user intervention to prevent loops."
  }
  ```
- **Rule**: If an action fails twice, pivot your plan immediately (switch to AT-SPI, try a keyboard shortcut, or take a zoomed `--region` screenshot). Do not attempt a third identical try.
- To clear the stuck detector tracker: `deskctl reset`.

---

## 4. Visual Diffing & Waiting for Change

To verify that an asynchronous event occurred (e.g. download finished, page loaded):
```bash
# Check how much the screen changed compared to the last screenshot:
deskctl diff --screen 0

# Block execution until visual change exceeds threshold:
deskctl wait-for-change --threshold 2.0 --timeout-ms 5000
```

---

## 5. Window Dragging Physics

In Cinnamon/Muffin window managers:
- Instant mouse teleportation between `mousedown` and `mouseup` is ignored by the window manager.
- `deskctl drag` automatically interpolates movement across 5 intermediate steps with 20ms delays.
- Always target the window titlebar (top 30px decoration frame) rather than the client canvas.
