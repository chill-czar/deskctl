# Cross-App Data Exchange (Copy & Paste) Recipe

Instructions for exchanging text and data between desktop applications using the clipboard and keyboard shortcuts.

## 1. Direct Clipboard Management (`deskctl clipboard`)

Whenever possible, prefer setting and reading the clipboard directly with `deskctl` to eliminate UI timing and selection errors:

### Set Clipboard Text
```bash
deskctl clipboard set "https://github.com/example/repo.git"
```

### Inspect Current Clipboard Content
```bash
deskctl clipboard get
```

### View Clipboard History
```bash
deskctl clipboard history
```

---

## 2. Keyboard Shortcuts Across Application Types

Linux applications use different shortcut conventions for clipboard operations:

| Application Type | Copy Shortcut | Paste Shortcut | Notes |
| :--- | :--- | :--- | :--- |
| **Standard GUI** (`xed`, Chrome, Nemo) | `Ctrl+C` | `Ctrl+V` | Standard desktop shortcuts |
| **Terminal** (`gnome-terminal`) | `Ctrl+Shift+C` | `Ctrl+Shift+V` | `Ctrl+C` sends interrupt signal in terminal |
| **X11 Primary Selection** | Highlight text | Middle Mouse Click | Handled via X11 primary buffer |

---

## 3. End-to-End Workflow: Web Browser to Terminal

### Step 1: Select Text in Browser and Copy
```bash
# 1. Focus Chrome
deskctl focus --title "Google Chrome"

# 2. Select text or URL (Ctrl+L selects URL bar)
deskctl key "ctrl+l"
deskctl wait --ms 200

# 3. Copy to clipboard
deskctl key "ctrl+c"
deskctl wait --ms 200

# 4. Verify clipboard received the content
deskctl clipboard get
```

### Step 2: Paste Content into Terminal
```bash
# 1. Focus Terminal
deskctl focus --title "Terminal"

# 2. Type prefix command
deskctl type "git clone "

# 3. Paste via Ctrl+Shift+V
deskctl key "ctrl+shift+v"
deskctl wait --ms 200

# 4. Confirm command with Return
deskctl key "Return"
```

---

## 4. End-to-End Workflow: File Content to GUI Text Editor

```bash
# 1. Read file directly into clipboard
deskctl run "cat /home/jimmy/Documents/snippet.py | xclip -selection clipboard"

# 2. Focus Text Editor
deskctl focus --title "xed"

# 3. Paste at current cursor position
deskctl key "ctrl+v"
deskctl wait --ms 200

# 4. Save file
deskctl key "ctrl+s"
```

---

## 5. Security Precaution

> [!CAUTION]
> Never copy passwords, private keys, authentication tokens, or personal identity information to the system clipboard, as background desktop applications can monitor clipboard events.
