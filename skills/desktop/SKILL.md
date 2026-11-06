---
name: desktop
description: Operate the user's Linux desktop (Linux Mint, Cinnamon, X11) like a human across single or multi-screen setups. See the screen, click, drag, scroll, type, press shortcuts, manage windows, read UI elements via AT-SPI, run CLI commands, and handle authentication with strict security rules. Use whenever a task involves desktop applications, the file manager, system settings, or cross-app workflows.
---

# Desktop Control System (`deskctl`)

You operate the Linux desktop through the `deskctl` CLI. Every command returns a structured JSON object (`{"ok": true, ...}` or `{"ok": false, "error": "..."}`). Work in a strict observe-plan-act-verify loop and never act blind.

## Quick Decision Tree: How to Target Things

Always prefer the most reliable, least fragile targeting method:
1. **Keyboard Shortcuts & CLI**: Fastest and immune to UI shifts (e.g. `Ctrl+L` for address bar, `deskctl run "xdg-open <path>"`, `Ctrl+S` to save).
2. **Accessibility Targeting (AT-SPI)**: Use `deskctl find "<query>"`, `deskctl click-element "<name>" --role <role>`, `deskctl get-text`, and `deskctl set-text`. Immune to screen resolution, scaling, and window movement.
3. **OCR Text Targeting**: If AT-SPI is unavailable (Electron, canvas, legacy toolkits), use `deskctl find-text "<query>" --click`.
4. **Visual Pixel Coordinates**: Use `deskctl screenshot` or `deskctl zoom`, calculate the element center in image coordinates, and use `deskctl click <x> <y>`. Deskctl automatically maps image coordinates to global screen space across any display setup.

---

## The Observe-Plan-Act-Verify Loop

1. **Observe**:
   - Query window topology: `deskctl windows`
   - Capture the current screen state: `deskctl screenshot --max-width 1280` (or `deskctl screenshot --privacy` for window isolation).
   - Inspect accessibility tree if applicable: `deskctl ui-tree --window <id>` or `deskctl find "<name>"`.
2. **Plan One Step**:
   - State in one sentence what action you will perform and what UI response is expected.
3. **Act**:
   - Execute exactly one action (e.g. `deskctl click 300 200`, `deskctl type "text"`, `deskctl key "ctrl+t"`).
4. **Verify**:
   - Use `--screenshot-after` or take a new screenshot to verify that the UI transitioned as expected.
   - If an action fails or the screen does not change, do not repeat the exact same action more than twice; pivot to an alternative targeting method.
5. **Session Safety**:
   - Start each major task with `deskctl session start --task "..." --apps "..." --folders "..."`.
   - On completion, run `deskctl session end`.

---

## Core Operational Rules

1. **Focus First**: Always focus the target window (`deskctl focus --id <id>` or `--title <substr>`) before typing, clicking inside it, or sending key shortcuts.
2. **Dynamic Multi-Screen**: Never assume fixed resolutions or single monitors. Run `deskctl screens` to discover dynamic screen IDs, positions, and DPI scaling factors.
3. **Settle Times**: After opening a dialog, launching an app, or triggering a UI animation, allow the UI to settle (`deskctl wait --ms 500` or `deskctl wait-for-stable`).
4. **Protected Zones**: Never attempt to access, read, or automate `.ssh`, `.gnupg`, `.aws`, `.env`, browser credentials, or password managers. Deskctl enforces this in code.
5. **No Password Automation**: Never guess, store, or auto-type passwords into password dialogs. Follow the [Auth Handoff Guide](file:///home/jimmy/.gemini/config/skills/desktop/auth/SKILL.md) to notify the user.
6. **Kill Switch & Panic**: The user can press `Ctrl+Alt+Escape` at any time to freeze deskctl via `~/.deskctl/STOP`. You can also trigger `deskctl panic` for emergency containment.

---

## Detailed Topic Guides

- **Security & Policies**:
  - [Security Rules & Password Policy](file:///home/jimmy/.gemini/config/skills/desktop/security/SKILL.md)
  - [Permissions, Sudo & Polkit](file:///home/jimmy/.gemini/config/skills/desktop/security/permissions.md)
- **Core Mechanics**:
  - [Screen Topology & Targeting Hierarchy](file:///home/jimmy/.gemini/config/skills/desktop/core/SKILL.md)
  - [Multi-Monitor Coordinate Math](file:///home/jimmy/.gemini/config/skills/desktop/core/multi-screen.md)
  - [Reliability & Stuck Detection](file:///home/jimmy/.gemini/config/skills/desktop/core/reliability.md)
- **Application Guides**:
  - [Application Guides Index](file:///home/jimmy/.gemini/config/skills/desktop/apps/SKILL.md)
  - [Terminal (`gnome-terminal`)](file:///home/jimmy/.gemini/config/skills/desktop/apps/terminal.md)
  - [File Manager (`nemo`)](file:///home/jimmy/.gemini/config/skills/desktop/apps/file-manager.md)
  - [Web Browser (`google-chrome`)](file:///home/jimmy/.gemini/config/skills/desktop/apps/browser.md)
  - [Text Editor (`xed` / IDE)](file:///home/jimmy/.gemini/config/skills/desktop/apps/text-editor.md)
  - [System Settings (`cinnamon-settings`)](file:///home/jimmy/.gemini/config/skills/desktop/apps/settings.md)
  - [Office Applications (`libreoffice`)](file:///home/jimmy/.gemini/config/skills/desktop/apps/office.md)
  - [Media Players (`celluloid` / Pix)](file:///home/jimmy/.gemini/config/skills/desktop/apps/media.md)
- **Task Recipes**:
  - [Task Recipes Index](file:///home/jimmy/.gemini/config/skills/desktop/tasks/SKILL.md)
  - [Files and Folders Operations](file:///home/jimmy/.gemini/config/skills/desktop/tasks/files-and-folders.md)
  - [Software Installation & Package Management](file:///home/jimmy/.gemini/config/skills/desktop/tasks/install-software.md)
  - [Configuring System Settings](file:///home/jimmy/.gemini/config/skills/desktop/tasks/system-settings.md)
  - [Capturing Screenshots & Video Recording](file:///home/jimmy/.gemini/config/skills/desktop/tasks/screenshots-and-recording.md)
  - [Cross-App Data Exchange & Copy-Paste](file:///home/jimmy/.gemini/config/skills/desktop/tasks/copy-between-apps.md)
  - [Forms and Data Entry Automation](file:///home/jimmy/.gemini/config/skills/desktop/tasks/forms-and-data-entry.md)
- **Authentication**:
  - [Password Prompt Handoff Procedure](file:///home/jimmy/.gemini/config/skills/desktop/auth/SKILL.md)
