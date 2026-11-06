# Password & Privilege Handling Guide (`auth/`)

Guidelines for operating with strict zero-bypass password security on Linux Mint.

## Absolute Prohibitions

1. **NO Auto-Typing**: Never type a password into any password dialog, terminal prompt, keyring popup, or web field.
2. **NO Brute-Forcing or Guessing**: Never attempt dictionary attacks, common passwords, or guessing.
3. **NO Password Extraction**: Never attempt to extract passwords from `/etc/shadow`, shell history (`~/.bash_history`), memory, keyring databases (`~/.local/share/keyrings`), browser saved passwords, or `.env` files.
4. **NO Sudo Hijacking**: Never modify `/etc/sudoers` or `/etc/sudoers.d/` to grant unrestricted `NOPASSWD: ALL`.

---

## Authentication Prompt Detection

`deskctl` continuously checks for authentication prompts before every input action (`click`, `type`, `key`).

### Monitored Prompts:
1. **Polkit Dialogs**: `PolicyKit1`, `org.freedesktop.policykit-1`, `gnome-shell`, `mate-polkit`, `polkit-gnome-authentication-agent-1`.
2. **Keyring Prompts**: `gcr-prompter`, `gnome-keyring-prompt`.
3. **Terminal Sudo**: Prompts matching `[sudo] password for ...` in active terminal windows.
4. **AT-SPI Password Fields**: Text components with `password-text` role or `PASSWORD` state.
5. **Lock Screen / Screensaver**: `cinnamon-screensaver-dialog`, `lightdm`, `gdm`.

---

## Human Handoff Workflow

When `deskctl` detects an active authentication dialog, it returns:
```json
{
  "ok": false,
  "blocked": true,
  "auth_prompt": true,
  "error": "Authentication prompt detected (<dialog_title>). Human authorization required."
}
```

### Required Agent Response:
1. **Pause Action**: Do not retry typing or clicking inside the auth prompt.
2. **Notify Human User**:
   ```bash
   deskctl notify --title "Authentication Required" --body "Please type your password in the prompt to authorize <program_name>."
   ```
3. **Wait for Prompt Dismissal**:
   ```bash
   deskctl auth-wait --timeout 60
   ```
   This command blocks until the human user submits or cancels the prompt.
4. **Resume and Verify**: Once `deskctl auth-wait` returns `{"ok": true, "resolved": true}`, verify that the authorization succeeded before proceeding.

---

## Scoped Sudo Allowlist Configuration

Instead of prompting for passwords on routine administrative maintenance, `deskctl` supports a scoped sudoers drop-in restricted strictly to specific commands.

### Initial Setup (`deskctl setup-sudo`)
1. Create `~/.deskctl/sudo-allowlist.txt` containing only the approved commands:
   ```text
   /usr/bin/apt update
   /usr/bin/apt install -y *
   /usr/bin/apt remove -y *
   /bin/systemctl status *
   /bin/systemctl restart [a-zA-Z0-9_-]*
   ```
2. Preview proposed sudoers configuration:
   ```bash
   deskctl setup-sudo --dry-run
   ```
3. Show the proposed configuration to the human user and request explicit confirmation.
4. Once confirmed, apply the configuration:
   ```bash
   deskctl setup-sudo
   ```
   `deskctl` validates the configuration with `visudo -c` before writing to `/etc/sudoers.d/deskctl`.

### Polkit Package Management Rule (`deskctl setup-polkit`)
Enables GUI package managers (e.g. Update Manager) to install authorized system packages without repeatedly prompting:
```bash
deskctl setup-polkit --dry-run
deskctl setup-polkit
```

### Revoking Permissions
To remove all elevated permissions at any time:
```bash
deskctl setup-sudo --revoke
deskctl setup-polkit --revoke
```
