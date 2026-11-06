# System Settings Recipe

Instructions for inspecting and modifying Linux Mint system settings, audio, displays, and desktop environment behavior.

## 1. System Metrics and Hardware Status

Inspect current hardware metrics using native `deskctl` status commands:

```bash
# Query master audio volume and mute state
deskctl volume get

# Query display backlight brightness
deskctl brightness get

# Check active network interfaces and IP addresses
deskctl network

# Check battery percentage and charging state
deskctl battery
```

---

## 2. Audio Control

Adjust audio levels programmatically or via media shortcuts:

```bash
# Set volume percentage (0-100)
deskctl volume set 65

# Increase / decrease volume
deskctl run "pactl set-sink-volume @DEFAULT_SINK@ +5%"
deskctl run "pactl set-sink-volume @DEFAULT_SINK@ -5%"

# Toggle audio mute
deskctl run "pactl set-sink-mute @DEFAULT_SINK@ toggle"
```

---

## 3. Display Configuration & Window Placement

### Inspect Dynamic Screen Layout
```bash
deskctl screens
```

### Move Windows Across Monitors
Move windows between screens without fragile pixel dragging:
```bash
# Find window ID
deskctl windows

# Move window to screen 1 (e.g. secondary monitor)
deskctl move-to-screen --window <id> --screen 1

# Maximize on target screen
deskctl maximize --window <id>
```

### Open Display Settings GUI
```bash
deskctl launch "cinnamon-settings display"
deskctl wait-for-window --title "Display" --timeout 5000
```

---

## 4. Desktop Themes and Appearance

### Inspect Active Theme
```bash
deskctl run "gsettings get org.cinnamon.desktop.interface gtk-theme"
deskctl run "gsettings get org.cinnamon.desktop.interface icon-theme"
```

### Switch Desktop Theme
```bash
# Switch GTK theme
deskctl run "gsettings set org.cinnamon.desktop.interface gtk-theme 'Mint-Y-Dark'"

# Switch icon theme
deskctl run "gsettings set org.cinnamon.desktop.interface icon-theme 'Mint-Y-Dark'"
```

---

## 5. Security Precaution

> [!WARNING]
> Do not disable screen lock, relax firewall policies, or modify network proxies without explicit instruction and approval from the user.
