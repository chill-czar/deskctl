# File Manager Guide (`nemo`)

Guidelines for automating Nemo, the default desktop file manager in Linux Mint Cinnamon.

## Launching and Navigation

### Direct Path Launch
To open Nemo directly at a specific directory:
```bash
deskctl launch "nemo /home/jimmy/Documents"
deskctl wait-for-window --title "Documents" --timeout 5000
deskctl focus --title "Documents"
```

### Navigating to a New Path via Path Bar
Nemo supports an interactive text path bar:
```bash
# Toggle text entry path bar
deskctl key "ctrl+l"
# Type target folder path and press Enter
deskctl type "/home/jimmy/Pictures\n"
```

## Essential Nemo Keyboard Shortcuts

| Action | Shortcut | Deskctl Command |
| :--- | :--- | :--- |
| **Location Bar (Type Path)** | `Ctrl+L` | `deskctl key "ctrl+l"` |
| **New Folder** | `Ctrl+Shift+N` | `deskctl key "ctrl+shift+n"` |
| **New Tab** | `Ctrl+T` | `deskctl key "ctrl+t"` |
| **Close Tab** | `Ctrl+W` | `deskctl key "ctrl+w"` |
| **Dual Pane (Split View)** | `F3` | `deskctl key "F3"` |
| **Toggle Hidden Files** | `Ctrl+H` | `deskctl key "ctrl+h"` |
| **Search Files** | `Ctrl+F` | `deskctl key "ctrl+f"` |
| **Up One Directory** | `Alt+Up` | `deskctl key "alt+Up"` |
| **Back / Forward** | `Alt+Left` / `Alt+Right` | `deskctl key "alt+Left"` |
| **Select All** | `Ctrl+A` | `deskctl key "ctrl+a"` |
| **Rename Selected** | `F2` | `deskctl key "F2"` |
| **Context Menu** | `Shift+F10` | `deskctl key "shift+F10"` |
| **File Properties** | `Alt+Enter` | `deskctl key "alt+Return"` |
| **Move to Trash** | `Delete` | `deskctl key "Delete"` |

## Common Workflows

### 1. Creating a New Folder
```bash
deskctl focus --title "nemo"
deskctl key "ctrl+shift+n"
deskctl wait --ms 300
deskctl type "Project_Archive\n"
```

### 2. Renaming a File
```bash
# Navigate to the file using arrow keys or find
deskctl key "Down"
deskctl key "F2"
deskctl wait --ms 200
deskctl type "report_2026_final.pdf\n"
```

### 3. Context Menu Actions
```bash
# Right-click selection or press Shift+F10
deskctl key "shift+F10"
deskctl wait --ms 300
# Inspect menu options with OCR or AT-SPI
deskctl ui-tree --window active
```
