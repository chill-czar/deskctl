---
name: desktop-security
description: Hard security rules, non-negotiable protected zones, password policies, kill switches, and emergency panic procedures for deskctl.
---

# Deskctl Security Architecture & Policies

Security in `deskctl` is enforced directly in code at the CLI layer, never relying solely on system prompts or LLM compliance.

## 1. Zero-Bypass Password Policy

> [!IMPORTANT]
> **Strict Rule**: Never bypass, disable, brute-force, read, store, or auto-type any password or authentication credential.

- **Prohibited Sources**: Never read passwords from the screen, keyring, bash history (`~/.bash_history`), `.env` files, or configuration stores.
- **Automated Detection**: `deskctl` actively monitors X11 window classes, titles, and AT-SPI roles for authentication dialogs (`PolicyKit`, `gcr-prompter`, `cinnamon-polkit-agent`, `password text` fields).
- **Automated Input Interception**: If an authentication prompt appears, `deskctl` immediately halts input actions (`type`, `click`, `key`), triggers a desktop notification, and returns an `auth_prompt` error.
- **Human Handoff**: The agent must run `deskctl auth-wait` while the human user types the credentials on the physical keyboard.

---

## 2. Hardcoded Protected Zones

The following directories and patterns are hardcoded into `deskctl` and cannot be read, listed, modified, or accessed, regardless of session scope:

| Protected Zone | Scope / Files Protected |
| :--- | :--- |
| `~/.ssh` | Private keys (`id_rsa`, `id_ed25519`), `authorized_keys`, `known_hosts`, `config` |
| `~/.gnupg` | GPG secret keys, trustdb, private keyrings |
| `~/.aws` | Cloud credentials and configuration files |
| `.env*` | All environment and secret files (`.env`, `.env.local`, `.env.production`) |
| `~/.config/google-chrome` | Saved passwords, cookies, web history, profile databases |
| `~/.mozilla` | Browser profiles and credentials |
| `~/.local/share/keyrings` | System login and secret keyrings |
| `*.kdbx`, `*wallet*.dat` | KeePass databases, Bitcoin/Ethereum cryptocurrency wallets |
| `/etc/shadow`, `/etc/gshadow` | System password hashes |

Any command attempting to read, write, trash, or reference these paths in shell strings (`deskctl run`) is rejected with a `Protected zone violation` and logged to `~/.deskctl/violations.log`.

---

## 3. Kill Switch & Emergency Panic

### Global Hotkey (`Ctrl+Alt+Escape`)
- A background daemon (`killswitch_daemon.py`) intercepts `Ctrl+Alt+Escape` globally via XRecord.
- Instantly creates `~/.deskctl/STOP`.
- When `~/.deskctl/STOP` is present, all `deskctl` commands (except `reset`, `panic`, `stop`, `killswitch status`) are immediately aborted with exit code 1.

### Emergency Panic (`deskctl panic`)
If an action begins behaving erratically or an unauthorized state is detected, run:
```bash
deskctl panic
```
This performs a full containment sequence:
1. Creates `~/.deskctl/STOP`.
2. Terminates any running screen recordings (`ffmpeg`).
3. Kills all child processes spawned during the active session.
4. Closes all application windows created during the session.

To resume operations after user review:
```bash
deskctl reset
```

---

## 4. Privacy Mode

When executing sensitive tasks:
```bash
deskctl screenshot --privacy
```
Privacy mode automatically restricts screenshot capture to the active window rectangle (`target_type: window`), preventing background windows, open messengers, or desktop notification popups from being captured in vision inputs.
