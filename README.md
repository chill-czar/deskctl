# `deskctl` - Linux Desktop Control System

`deskctl` is a complete, agent-ready desktop control toolkit and guidance system built specifically for Linux Mint (Cinnamon, X11). It allows an AI agent to operate the computer the way a human would across any monitor topology and desktop application, backed by code-enforced security boundaries, a zero-bypass password policy, and an instant emergency kill switch.

---

## Architecture Overview

`deskctl` consists of two interconnected components:

1. **The CLI Engine (`deskctl`)**: A high-performance Go binary installed at `~/.local/bin/deskctl` delegating to native Linux desktop utilities (`xdotool`, `maim`, `wmctrl`, `xclip`, `xrandr`, `ffmpeg`, `tesseract`, and Python `pyatspi`). Every command produces a single, structured JSON output (`{"ok": true, ...}` or `{"ok": false, ...}`).
2. **The Antigravity Nested Skill (`desktop`)**: Located at `~/.gemini/config/skills/desktop/`, providing modular guidelines that teach agents how to observe, plan, act, and verify across applications, tasks, multi-monitor topologies, and authentication events.

---

## Core Capabilities

### 1. Dynamic Display Model (Zero Hardcoding)
- Discovers monitors dynamically at runtime on every command via `xrandr --listmonitors`.
- Supports arbitrary multi-screen arrangements, mixed resolutions, negative virtual coordinates, portrait orientations, and dynamic hotplugging.
- Captures full virtual desktop, specific screens (`--screen ID|all|active|primary`), specific windows (`--screen window:0x...`), or region crops (`--screen region:x,y,w,h`).
- Intelligently downscales images with `--max-width` while always returning exact scaling ratios and translating model clicks back to global screen coordinates.

### 2. Complete Action Toolkit
- **Input**: `click`, `double-click`, `triple-click`, `right-click`, `middle-click`, `move`, `hover`, `drag`, `drag-path`, `hold-and-release`, `scroll`, `type` (with configurable delay), `key`, `key-hold`, `mouse-position`.
- **Windows**: `windows`, `focus`, `close-window`, `minimize`, `maximize`, `move-window`, `resize-window`, `move-to-screen`, `always-on-top`, `workspaces`, `launch`, `wait-for-window`.
- **Vision & OCR**: `screenshot`, `zoom`, `diff`, `wait-for-change`, `wait-for-stable`, `color-at-pixel`, `find-image`, `find-text --click`, `ocr`, `record-screen` (FFmpeg MP4 capture).
- **Accessibility (AT-SPI)**: `ui-tree`, `find`, `click-element`, `set-text`, `get-text`, with automatic graceful fallback to visual coordinates when AT-SPI is unavailable.
- **System Operations**: `clipboard` (get/set/history), `run` (scoped shell), `notify` (desktop notifications), `open` (xdg-open), `file` (list/read/write/trash), `volume`, `brightness`, `network`, `battery`, `schedule`.
- **Sessions & Reliability**: `session start` with declared scopes, `session status`, `session end`, automatic settle times, `--dry-run`, `--screenshot-after`, and a stuck detector that halts after 3 identical consecutive failures.

### 3. Password & Privilege Zero-Bypass Policy
- Never reads, stores, guesses, or auto-types passwords.
- Real-time detection of authentication prompts: Polkit dialogs, GNOME Keyring unlocks, lock screens, terminal `[sudo]` prompts, and AT-SPI `password-text` inputs.
- Automatically pauses execution, sends a desktop notification to the user, and blocks agent input until the user enters their credentials.
- Scoped sudo configuration (`deskctl setup-sudo`) using `~/.deskctl/sudo-allowlist.txt` validated by `visudo -c`.

### 4. Code-Enforced Security & Protected Zones
- Declared session scopes enforce allowed `--apps`, `--folders`, `--domains`, and `--actions`.
- Hardcoded protected zones strictly reject access to:
  - `~/.ssh`, `~/.gnupg`, `~/.aws` and cloud keys
  - `.env*` configuration and secret files
  - Browser profile directories and saved credentials (`~/.config/google-chrome`, `~/.mozilla`)
  - Keyring databases (`~/.local/share/keyrings`)
  - Password manager databases (`*.kdbx`, `*wallet*.dat`)
  - System shadow files (`/etc/shadow`)
- Blocks dangerous shell command patterns (`cat ~/.ssh/...`, `curl ... | bash`, `rm -rf /`) and audits all violations to `~/.deskctl/violations.log`.
- Configurable session budgets and rate limits in `~/.deskctl/config.toml`.

### 5. Emergency Kill Switch
- Global hardware shortcut: **`Ctrl+Alt+Escape`** instantly halts actions and creates `~/.deskctl/STOP`.
- CLI emergency stop: `deskctl panic` or `deskctl stop` terminates running deskctl actions and resets session state.
- Resumed safely with `deskctl reset`.

---

## Installation & Setup

1. Run the automated installer:
   ```bash
   bash ~/.gemini/config/skills/desktop/scripts/install.sh
   ```
2. Verify all system requirements with `deskctl doctor`:
   ```bash
   deskctl doctor
   ```
3. Start the background kill switch daemon:
   ```bash
   python3 ~/.gemini/config/skills/desktop/scripts/killswitch-daemon.py &
   ```

---

## CLI Command Quick Reference

```bash
# Display & Screens
deskctl screens
deskctl screenshot --screen active --max-width 1280
deskctl screenshot --privacy

# Mouse & Keyboard
deskctl mouse-position
deskctl click 450 320
deskctl right-click 500 400
deskctl type "Hello World\n" --delay 10
deskctl key "ctrl+s"

# Windows
deskctl windows
deskctl focus --title "Google Chrome"
deskctl move-to-screen --window 0x02400003 --screen 1
deskctl close-window --title "Calculator"

# Accessibility & OCR
deskctl ui-tree --window active
deskctl find "Save" --window active
deskctl click-element "Save" --role push-button
deskctl find-text "Settings" --click

# System & Files
deskctl clipboard get
deskctl clipboard set "New copied content"
deskctl file list /home/jimmy/Documents
deskctl volume get
deskctl volume set 75
deskctl notify --title "Task Complete" --body "Your workflow finished successfully."

# Diagnostics & Safety
deskctl doctor
deskctl panic
deskctl reset
```

---

## Application AT-SPI Compatibility & Fallbacks

| Application | AT-SPI Support | Preferred Targeting Mode | Fallback |
| :--- | :--- | :--- | :--- |
| **Nemo** (File Manager) | Excellent | Keyboard (`Ctrl+L`, `Ctrl+Shift+N`) & AT-SPI | Visual Coordinates |
| **gnome-terminal** | Good | Keyboard shortcuts & `deskctl run` | OCR / Screenshots |
| **xed** (Text Editor) | Excellent | Keyboard shortcuts (`Ctrl+S`, `Ctrl+I`) & AT-SPI | OCR |
| **cinnamon-settings** | Excellent | Direct module launch (`cinnamon-settings <module>`) & AT-SPI | `gsettings` CLI |
| **Google Chrome** | Variable | Omnibox shortcuts (`Ctrl+L`), Tab indexing, DevTools | `deskctl find-text --click` / Visual Coordinates |
| **LibreOffice** | Good | Standard accelerators (`Alt+F`), headless CLI conversion | OCR / Menu items |
