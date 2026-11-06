# Session Scopes, Sudo Setup & PolicyKit Rules

## 1. Declaring Session Scopes

Every automated session should declare its operating boundaries upon initialization:
```bash
deskctl session start \
  --task "Update Node.js project dependencies" \
  --apps "gnome-terminal,google-chrome" \
  --folders "/home/jimmy/Desktop/DEV/my-project,/tmp" \
  --actions "screenshot,click,move,type,key,run,file" \
  --domains "github.com,npmjs.com"
```

### Scope Enforcement in Code
- **Apps**: If the agent attempts to launch or focus an application outside `--apps`, `deskctl` returns a `Scope violation: application outside declared session app boundaries`.
- **Folders**: If the agent attempts a `file` or `open` operation outside `--folders`, `deskctl` returns a `Scope violation: path outside declared session folder boundaries`.
- **Actions**: If the agent attempts a command not in `--actions`, execution is blocked.
- **Domains**: URLs passed to `open` or web navigators must match `--domains`.
- All scope violations are audited to `~/.deskctl/violations.log` and the session's `log.jsonl`.

---

## 2. Scoped Sudo Setup (`deskctl setup-sudo`)

`deskctl` never uses or allows `NOPASSWD: ALL`. For tasks requiring package installation or system service restarts, use the scoped sudo helper:

### Inspecting Proposed Configuration (Dry Run)
```bash
deskctl setup-sudo --dry-run
```
Output preview:
```json
{
  "allowlist_path": "/home/jimmy/.deskctl/sudo-allowlist.txt",
  "target_path": "/etc/sudoers.d/deskctl",
  "visudo_validation": "/tmp/deskctl_sudoers_preview: parsed OK",
  "proposed_content": "jimmy ALL=(ALL) NOPASSWD: \\\n    /usr/bin/apt update, \\\n    /usr/bin/apt install *, \\\n    /usr/bin/apt remove *, \\\n    /bin/systemctl status *, \\\n    /bin/systemctl restart [a-zA-Z0-9_-]+"
}
```

### Rules & Validation
1. Commands are read from `~/.deskctl/sudo-allowlist.txt`.
2. Any entry matching `ALL`, `/bin/sh`, `/bin/bash`, `sudo`, or `su` is rejected.
3. Syntax is verified with `visudo -c -f <file>` before installation.
4. Requires explicit confirmation:
   ```bash
   deskctl setup-sudo --confirm
   ```
   This triggers `pkexec` to install `/etc/sudoers.d/deskctl` with mode `0440`.

### Revoking Sudo Permissions
To completely clean up and remove the sudoers drop-in:
```bash
deskctl setup-sudo --revoke
```

---

## 3. Scoped PolicyKit Setup (`deskctl setup-polkit`)

For non-interactive graphical or D-Bus package operations without prompting:
```bash
# Preview
deskctl setup-polkit --dry-run

# Install via pkexec
deskctl setup-polkit --confirm

# Revoke
deskctl setup-polkit --revoke
```
Installs `/etc/polkit-1/rules.d/49-deskctl.rules` scoped strictly to:
- `org.debian.apt.update-cache`
- `org.debian.apt.install-or-remove-packages`
- `org.freedesktop.packagekit.system-sources-refresh`
- `org.freedesktop.packagekit.package-install`
