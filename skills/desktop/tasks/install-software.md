# Software Installation & Package Management Recipe

Instructions for safely installing, updating, and removing software on Linux Mint.

> [!IMPORTANT]
> - Never run arbitrary `sudo` commands without checking authorization.
> - Never install Firefox under any circumstances (per user instruction).
> - Never auto-type passwords. If an auth prompt appears, hand off to the human user.

---

## 1. Checking If Software Is Already Installed

Before triggering package managers, check if the binary or flatpak already exists:
```bash
# Check PATH
deskctl run "which jq tree ripgrep"

# Check APT database
deskctl run "dpkg -l | grep -E 'jq|ripgrep'"

# Check Flatpaks
deskctl run "flatpak list"
```

---

## 2. Using the Scoped Sudo Allowlist

`deskctl` provides a scoped sudo mechanism configured at `/etc/sudoers.d/deskctl` based on `~/.deskctl/sudo-allowlist.txt`.

### Step A: Verify Sudo Configuration
Ensure the allowlist has been set up:
```bash
deskctl run "cat ~/.deskctl/sudo-allowlist.txt"
```
If not set up, run `deskctl setup-sudo --dry-run` to inspect proposed rules, request user confirmation, and run `deskctl setup-sudo`.

### Step B: Run Authorized Package Operations
Run standard package manager operations through `deskctl run`:
```bash
# Update package lists
deskctl run "sudo apt update"

# Install authorized packages
deskctl run "sudo apt install -y jq tree"

# Remove packages
deskctl run "sudo apt remove -y tree"
```

---

## 3. Human Handoff on Authentication Prompts

If a command or GUI installer triggers an authentication dialog:
1. `deskctl` automatically detects the prompt (polkit or terminal `[sudo]`).
2. Input commands are blocked to prevent leaking sensitive text.
3. Notify the human user:
   ```bash
   deskctl notify --title "Authorization Required" --body "Please enter your password to authorize package installation."
   deskctl auth-wait --timeout 60
   ```
4. Once the prompt dismisses, verify completion.

---

## 4. User-Space Flatpak Installation

When root privileges are unavailable or inappropriate, prefer user-scoped Flatpak:
```bash
# Search for application
deskctl run "flatpak search vlc"

# Install to user scope (no sudo required)
deskctl run "flatpak install --user -y flathub org.videolan.VLC"

# Run installed flatpak
deskctl launch "flatpak run org.videolan.VLC"
```

---

## 5. Verification

Always verify installation by checking the executable path and version:
```bash
deskctl run "jq --version"
```
If verification fails, inspect package manager output or logs before retrying.
