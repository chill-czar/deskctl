# Forms and Data Entry Automation Recipe

Instructions for filling out dialog forms, web forms, and desktop application input fields reliably.

## 1. Input Navigation Hierarchy

When navigating complex forms, use this hierarchy:
1. **AT-SPI Direct Field Setting**: `deskctl set-text --name "Username" --value "admin"` (fastest, immune to cursor positioning).
2. **Tab Index Navigation**: Press `Tab` or `Shift+Tab` to advance between sequential fields.
3. **Visual / OCR Targeting**: Click inside text fields using OCR bounding boxes or visual coordinates.

---

## 2. Handling Different Form Controls

### Text Fields
```bash
# Method A: AT-SPI
deskctl set-text --name "First Name" --value "Jane"

# Method B: Focus and Type
deskctl click 450 320
# Clear existing text
deskctl key "ctrl+a"
deskctl key "BackSpace"
# Type new text with a small typing delay
deskctl type "Jane" --delay-ms 15
```

### Checkboxes
```bash
# Method A: AT-SPI toggle
deskctl click-element "I agree to terms" --role check-box

# Method B: Keyboard
deskctl key "Tab"
deskctl key "space"
```

### Radio Buttons
```bash
# Select option via Arrow keys
deskctl key "Tab"
deskctl key "Down"
```

### Dropdown Menus / Comboboxes
```bash
# 1. Focus the combobox
deskctl click-element "Country" --role combo-box
# Or open dropdown via keyboard
deskctl key "alt+Down"
deskctl wait --ms 200

# 2. Select option by typing first letters or pressing Down
deskctl type "Canada"
deskctl key "Return"
```

### Date and Time Pickers
Prefer typing ISO date strings (`YYYY-MM-DD`) directly when the text input is accessible, or use Tab navigation.

---

## 3. Human Confirmation Before Form Submission

> [!CRITICAL]
> **Always verify before submitting forms.**
> Under the Security and Scope Policy, an agent must **never** submit forms that:
> - Send communications (emails, chats, social posts, forum comments).
> - Make financial transactions or payment authorizations.
> - Modify user account credentials, recovery phone numbers, or passwords.
> - Apply system-wide security, network, or permission modifications.

When reaching a submit button for any sensitive action:
1. Pause before clicking.
2. Capture a screenshot of the filled form: `deskctl screenshot --screen active`.
3. Notify the user with details of the proposed submission and wait for their explicit approval.
