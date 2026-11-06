# Text Editor Guide (`xed` / Code Editors)

Guidelines for automating text editors on Linux Mint, focusing on `xed` (default editor) and standard GUI code editors.

## Launching and Opening Files

```bash
# Launch xed with target file
deskctl launch "xed /home/jimmy/notes.txt"
deskctl wait-for-window --title "notes.txt" --timeout 5000
deskctl focus --title "xed"
```

## Essential Editor Shortcuts (`xed`)

| Action | Shortcut | Deskctl Command |
| :--- | :--- | :--- |
| **New Document** | `Ctrl+N` | `deskctl key "ctrl+n"` |
| **Open File Dialog** | `Ctrl+O` | `deskctl key "ctrl+o"` |
| **Save File** | `Ctrl+S` | `deskctl key "ctrl+s"` |
| **Save As...** | `Ctrl+Shift+S` | `deskctl key "ctrl+shift+s"` |
| **Close Tab** | `Ctrl+W` | `deskctl key "ctrl+w"` |
| **Quit Editor** | `Ctrl+Q` | `deskctl key "ctrl+q"` |
| **Find Text** | `Ctrl+F` | `deskctl key "ctrl+f"` |
| **Find and Replace** | `Ctrl+H` | `deskctl key "ctrl+h"` |
| **Go to Line Number** | `Ctrl+I` | `deskctl key "ctrl+i"` |
| **Undo / Redo** | `Ctrl+Z` / `Ctrl+Shift+Z` | `deskctl key "ctrl+z"` |
| **Select All** | `Ctrl+A` | `deskctl key "ctrl+a"` |

## Common Workflows

### 1. Jump to Line and Insert Text
```bash
deskctl focus --title "xed"
# Jump to line 42
deskctl key "ctrl+i"
deskctl wait --ms 200
deskctl type "42\n"
deskctl wait --ms 200
# Insert text
deskctl key "End"
deskctl type "\n# Added by deskctl automated workflow\n"
deskctl key "ctrl+s"
```

### 2. Search and Replace
```bash
deskctl focus --title "xed"
deskctl key "ctrl+h"
deskctl wait --ms 300
# Type search target
deskctl type "old_variable_name\t"
# Type replacement
deskctl type "new_variable_name"
# Trigger replace all (Alt+A in xed replace dialog)
deskctl key "alt+a"
deskctl key "Escape"
deskctl key "ctrl+s"
```

## File Editing Safety

- For programmatic file edits, prefer `deskctl file write` or `deskctl run` with Python/sed when the editor GUI is not strictly required.
- Always check that protected paths (`.env`, `~/.ssh`, credentials) are never loaded into text editors.
