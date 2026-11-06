# Terminal Guide (`gnome-terminal`)

Guidelines for interacting with the desktop terminal emulator (`gnome-terminal` / `x-terminal-emulator`).

## When to Use GUI Terminal vs `deskctl run`

- **Prefer `deskctl run "<cmd>"`**: For scripts, background commands, file inspection, and structured output. It is faster, captures exact stdout/stderr, and avoids GUI timing issues.
- **Use GUI Terminal**: When an interactive visual prompt or REPL is required, or when demonstrating a workflow to the human user on screen.

## Launching and Focusing

```bash
# Launch terminal
deskctl launch "gnome-terminal"

# Wait for window and focus
deskctl wait-for-window --title "Terminal" --timeout 5000
deskctl focus --title "Terminal"
```

## Key Shortcuts

| Action | Shortcut | Deskctl Command |
| :--- | :--- | :--- |
| **New Tab** | `Ctrl+Shift+T` | `deskctl key "ctrl+shift+t"` |
| **Close Tab** | `Ctrl+Shift+W` | `deskctl key "ctrl+shift+w"` |
| **Copy Selection** | `Ctrl+Shift+C` | `deskctl key "ctrl+shift+c"` |
| **Paste from Clipboard** | `Ctrl+Shift+V` | `deskctl key "ctrl+shift+v"` |
| **Cancel / Interrupt** | `Ctrl+C` | `deskctl key "ctrl+c"` |
| **Clear Terminal Screen** | `Ctrl+L` | `deskctl key "ctrl+l"` |
| **Exit / Close Session** | `Ctrl+D` | `deskctl key "ctrl+d"` |
| **Search Output** | `Ctrl+Shift+F` | `deskctl key "ctrl+shift+f"` |

## Executing Commands Safely in GUI Terminal

1. Focus the terminal window:
   ```bash
   deskctl focus --title "Terminal"
   ```
2. Type the command string followed by Enter:
   ```bash
   deskctl type "ls -la ~/Downloads\n" --delay-ms 10
   ```
3. Allow command to finish and inspect output:
   ```bash
   deskctl wait --ms 500
   deskctl screenshot --screen active
   ```

## Sudo / Password Detection

If a terminal command triggers a `[sudo] password for ...` prompt:
1. `deskctl`'s auth monitor detects the prompt text or AT-SPI terminal state.
2. **DO NOT** attempt to type the password.
3. Call `deskctl auth-wait` or `deskctl notify --title "Auth Required" --body "Please enter your sudo password in the terminal."`.
4. Wait for user completion before resuming.
