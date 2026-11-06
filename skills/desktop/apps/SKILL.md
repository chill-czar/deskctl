# Desktop Applications Guide (`apps/`)

This directory contains application-specific operation guides for standard desktop software on Linux Mint (Cinnamon desktop environment).

## General Application Lifecycle

When automating any GUI application, follow this reliable lifecycle:

### 1. Launching
Use `deskctl launch` or `deskctl run "xdg-open ..."`:
```bash
# Launch by application command or desktop entry
deskctl launch "nemo"
deskctl launch "google-chrome"
deskctl launch "xed"
```
Always wait for the window to appear before sending input:
```bash
deskctl wait-for-window --title "Nemo" --timeout 5000
```

### 2. Targeting and Focus
Never click or type blindly into an unverified window:
```bash
# Focus by title substring or hex/dec window ID
deskctl focus --title "Google Chrome"
```
Verify the active window matches before typing:
```bash
deskctl windows
```

### 3. Sizing and Placement
For consistent visual inspection, ensure the window is positioned predictably:
```bash
# Move to specific screen or maximize
deskctl move-to-screen --window <id> --screen 0
deskctl maximize --window <id>
```

### 4. Input Preference
Always prefer the fastest, most reliable input method:
1. **Application Shortcuts**: Instant and resolution-independent (`Ctrl+L`, `Ctrl+S`, `Ctrl+T`).
2. **AT-SPI Elements**: `deskctl find "<button_name>" --window <id>` and `deskctl click-element`.
3. **Visual / OCR**: Use `deskctl ocr` or `deskctl screenshot` if the app lacks AT-SPI (e.g. Electron apps, games).

### 5. Closing
Close applications gracefully when tasks are complete:
```bash
deskctl close-window --id <id>
```

---

## Application Guides Index

- [Terminal (`gnome-terminal`)](file:///home/jimmy/.gemini/config/skills/desktop/apps/terminal.md)
- [File Manager (`nemo`)](file:///home/jimmy/.gemini/config/skills/desktop/apps/file-manager.md)
- [Web Browser (`google-chrome`)](file:///home/jimmy/.gemini/config/skills/desktop/apps/browser.md)
- [Text Editor (`xed` / Code Editors)](file:///home/jimmy/.gemini/config/skills/desktop/apps/text-editor.md)
- [System Settings (`cinnamon-settings`)](file:///home/jimmy/.gemini/config/skills/desktop/apps/settings.md)
- [Office Applications (`libreoffice`)](file:///home/jimmy/.gemini/config/skills/desktop/apps/office.md)
- [Media Viewers (`celluloid` / `pix`)](file:///home/jimmy/.gemini/config/skills/desktop/apps/media.md)
