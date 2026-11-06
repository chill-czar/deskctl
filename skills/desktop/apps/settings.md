# System Settings Guide (`cinnamon-settings`)

Guidelines for automating Cinnamon Desktop Environment configuration panels in Linux Mint.

## Opening Settings Modules Directly

Instead of navigating through nested GUI menus, launch specific settings modules directly using command line arguments:

```bash
# Display & Monitor Layout
deskctl launch "cinnamon-settings display"

# Sound & Audio Devices
deskctl launch "cinnamon-settings sound"

# Themes & Desktop Appearance
deskctl launch "cinnamon-settings themes"

# Power Management & Sleep Timers
deskctl launch "cinnamon-settings power"

# Keyboard & Keybindings
deskctl launch "cinnamon-settings keyboard"

# Mouse & Touchpad Settings
deskctl launch "cinnamon-settings mouse"

# Notifications Configuration
deskctl launch "cinnamon-settings notifications"

# Privacy & Activity Logging
deskctl launch "cinnamon-settings privacy"
```

## Available Modules Reference

| Module | Purpose | Direct Command |
| :--- | :--- | :--- |
| `display` | Screen resolution, arrangement, scaling, refresh rate | `cinnamon-settings display` |
| `sound` | Output volume, input microphones, alert sounds | `cinnamon-settings sound` |
| `themes` | Window borders, icons, controls, desktop theme | `cinnamon-settings themes` |
| `power` | Battery sleep thresholds, screen turn-off timers | `cinnamon-settings power` |
| `keyboard` | Keyboard layout, repeat delay, custom shortcuts | `cinnamon-settings keyboard` |
| `mouse` | Cursor speed, natural scrolling, double-click | `cinnamon-settings mouse` |
| `backgrounds` | Desktop wallpaper, slideshow intervals | `cinnamon-settings backgrounds` |
| `workspaces` | Virtual desktop count, workspace OSD behavior | `cinnamon-settings workspaces` |
| `screensaver` | Screen lock timer, lock on suspend | `cinnamon-settings screensaver` |

## Navigating Settings Windows

Cinnamon settings modules are GTK-based applications that expose standard AT-SPI trees:

```bash
# 1. Wait for module window
deskctl wait-for-window --title "Display" --timeout 5000
deskctl focus --title "Display"

# 2. Inspect UI controls via AT-SPI
deskctl ui-tree --window active

# 3. Target controls by role and label
deskctl find "Resolution" --window active
deskctl click-element "Apply" --role push-button
```

## Fast CLI Alternatives (`gsettings`)

For programmatic setting adjustments, prefer native `gsettings` commands via `deskctl run`, as they apply instantly without opening a GUI:

```bash
# Check current desktop theme
deskctl run "gsettings get org.cinnamon.desktop.interface gtk-theme"

# Change wallpaper
deskctl run "gsettings set org.cinnamon.desktop.background picture-uri 'file:///usr/share/backgrounds/linuxmint/default_background.jpg'"

# Adjust screen lock timeout (seconds)
deskctl run "gsettings set org.cinnamon.desktop.session idle-delay 900"
```

> [!WARNING]
> Changing network security, firewall, or user account privileges requires explicit human authorization.
