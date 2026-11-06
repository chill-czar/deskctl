# Multi-Screen Geometry & Dynamic Display Modeling

`deskctl` never assumes a single monitor or hardcoded resolutions. It detects monitor topology dynamically on every command via `xrandr --listmonitors`.

## 1. Querying Display Topology

Run `deskctl screens` to inspect connected monitors:
```bash
deskctl screens
```
Output structure:
```json
{
  "count": 2,
  "ok": true,
  "screens": [
    {
      "id": 0,
      "name": "eDP-1-1",
      "x": 1920,
      "y": 0,
      "w": 1920,
      "h": 1080,
      "scale": 1.0,
      "primary": true,
      "active": true
    },
    {
      "id": 1,
      "name": "HDMI-1-1",
      "x": 0,
      "y": 0,
      "w": 1920,
      "h": 1080,
      "scale": 1.0,
      "primary": false,
      "active": false
    }
  ],
  "virtual_desktop": {
    "x": 0,
    "y": 0,
    "w": 3840,
    "h": 1080
  }
}
```

---

## 2. Multi-Screen Coordinate Space

- **Global Virtual Desktop**: Displays form a unified virtual bounding box. Displays can be placed side-by-side, stacked vertically, or offset with negative coordinates (e.g. `x = -1920` for a monitor to the left of primary).
- **Negative Coordinate Safety**: `deskctl` prefixes all low-level `xdotool` calls with `--` (`xdotool mousemove -- -100 200`) so negative positions are never parsed as CLI flags.
- **Image-to-Screen Mapping**:
  When you capture a downscaled screenshot:
  ```bash
  deskctl screenshot --screen all --max-width 1280
  ```
  `deskctl` records:
  - `ImageWidth`: 1280
  - `CapturedWidth`: 3840
  - `ScaleFactor`: 0.333333
  - `OffsetX`, `OffsetY`: 0, 0

  When you subsequently run `deskctl click 640 180`, `deskctl` automatically converts `(640, 180)` back to real virtual screen coordinates:
  $$\text{RealX} = \text{OffsetX} + \frac{\text{ImageX}}{\text{ScaleFactor}} = 0 + \frac{640}{0.333333} = 1920$$
  $$\text{RealY} = \text{OffsetY} + \frac{\text{ImageY}}{\text{ScaleFactor}} = 0 + \frac{180}{0.333333} = 540$$

---

## 3. Targeted Screenshots

You can capture specific targets without capturing the entire multi-monitor canvas:
- Specific monitor: `deskctl screenshot --screen 0` or `--screen HDMI-1-1`
- Primary monitor only: `deskctl screenshot --screen primary`
- Active monitor (where cursor/active window resides): `deskctl screenshot --screen active`
- Specific window: `deskctl screenshot --screen window:<window_id>`
- Sub-region: `deskctl screenshot --region 200,200,800,600`

---

## 4. Moving Windows Across Monitors

To move a window from one monitor to another:
```bash
# Move window to HDMI monitor (Screen #1)
deskctl move-to-screen --title "Google Chrome" --screen 1

# Move window to primary laptop display (Screen #0)
deskctl move-to-screen --id 0x04400060 --screen 0
```
`deskctl` calculates the center of the destination screen and adjusts the window position while preserving geometry.
