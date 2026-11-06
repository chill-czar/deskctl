# Media Applications Guide (`celluloid` / `pix`)

Guidelines for interacting with media players and viewers on Linux Mint:
- **Celluloid**: Default GTK frontend for MPV video playback.
- **Pix**: Default image organizer and viewer.

## Celluloid (Video Player)

### Launching Video Files
```bash
deskctl launch "celluloid /home/jimmy/Videos/demo.mp4"
deskctl wait-for-window --title "Celluloid" --timeout 5000
deskctl focus --title "Celluloid"
```

### Essential Celluloid Shortcuts

| Action | Shortcut | Deskctl Command |
| :--- | :--- | :--- |
| **Play / Pause** | `Space` | `deskctl key "space"` |
| **Toggle Fullscreen** | `F` or `Double-Click` | `deskctl key "f"` |
| **Seek Backward / Forward 5s** | `Left` / `Right` | `deskctl key "Left"` |
| **Seek Backward / Forward 60s** | `Down` / `Up` | `deskctl key "Down"` |
| **Volume Up / Down** | `9` / `0` | `deskctl key "0"` |
| **Mute / Unmute** | `M` | `deskctl key "m"` |
| **Next / Previous Track** | `>` / `<` | `deskctl key "greater"` |
| **Quit Player** | `Q` | `deskctl key "q"` |

---

## Pix (Image Viewer)

### Launching Images
```bash
deskctl launch "pix /home/jimmy/Pictures/sample.png"
deskctl wait-for-window --title "Pix" --timeout 5000
deskctl focus --title "Pix"
```

### Essential Pix Shortcuts

| Action | Shortcut | Deskctl Command |
| :--- | :--- | :--- |
| **Next / Previous Image** | `Page_Down` / `Page_Up` or `Space` / `Backspace` | `deskctl key "Next"` |
| **Zoom In / Out** | `+` / `-` | `deskctl key "plus"` |
| **Fit Window** | `F` | `deskctl key "f"` |
| **Actual Size (100%)** | `1` | `deskctl key "1"` |
| **Fullscreen** | `F11` | `deskctl key "F11"` |
| **Rotate Right / Left** | `Ctrl+R` / `Ctrl+Shift+R` | `deskctl key "ctrl+r"` |
| **Properties** | `Alt+Return` | `deskctl key "alt+Return"` |
| **Quit** | `Ctrl+Q` | `deskctl key "ctrl+q"` |

## Programmatic Alternative for Media Processing

When inspecting images or video streams for metadata, prefer `ffmpeg` or `ffprobe` via `deskctl run`:
```bash
deskctl run "ffprobe -v error -show_entries format=duration,size -of default=noprint_wrappers=1 /path/to/media.mp4"
```
