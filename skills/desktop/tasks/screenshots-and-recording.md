# Screenshots and Recording Recipe

Instructions for capturing images, isolating windows, recording video clips, and respecting privacy on Linux Mint.

## 1. Capturing Screenshots (`deskctl screenshot`)

`deskctl screenshot` dynamically queries display geometries and supports flexible target specifiers:

### Full Multi-Monitor Desktop
```bash
deskctl screenshot --screen all --format png
```

### Specific Screen by ID or Keyword
```bash
# Capture primary display
deskctl screenshot --screen primary

# Capture active screen (where the mouse cursor or active window is located)
deskctl screenshot --screen active

# Capture specific monitor index
deskctl screenshot --screen 0
deskctl screenshot --screen 1
```

### Specific Window by ID
Captures only the target window without capturing the surrounding desktop or background monitors:
```bash
deskctl screenshot --screen window:0x02400003
```

### Region Crop
```bash
deskctl screenshot --screen region:100,200,800,600
```

### Model Optimization with Downscaling
Downscale large captures to conserve context window while retaining coordinate accuracy:
```bash
deskctl screenshot --max-width 1280
```
The output JSON returns:
- `path`: Local image path.
- `scale`: Exact scaling factor (e.g. `0.6667`).
- `real_w` / `real_h`: Global desktop dimensions.

---

## 2. Privacy Mode (`--privacy`)

> [!IMPORTANT]
> When executing general tasks, background notifications, personal chat windows, or browser tabs on other monitors may contain private data. Always prefer privacy mode or window targeting.

```bash
# Automatically crops to the active window bounds and blanks surrounding monitors
deskctl screenshot --privacy
```

---

## 3. Zooming and Fine Inspection

When small UI elements, text, or icons need visual verification:
```bash
deskctl zoom --x 400 --y 300 --w 200 --h 150 --scale 2
```

---

## 4. Visual Stability and Screen Diffs

### Wait for UI Animation to Settle
```bash
deskctl wait-for-stable --threshold 4.0 --timeout 5000
```

### Compare Two Screenshots
Detect whether the screen actually updated after an action:
```bash
deskctl diff /path/to/before.png /path/to/after.png
```

---

## 5. Screen Video Recording (`record-screen`)

Capture video clips using FFmpeg for complex multi-step demonstrations or debugging:

```bash
# Start recording the active screen at 15 FPS
deskctl record-screen start --screen active --fps 15 --duration 30

# Perform the required desktop workflow
deskctl click 450 320
deskctl type "Sample input"

# Finalize and stop recording
deskctl record-screen stop
```
The output video MP4 file path will be returned in the JSON response.
