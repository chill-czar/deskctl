# Files and Folders Operations Recipe

Instructions for managing files, folders, and archives using `deskctl` and the Nemo file manager.

## 1. Safety and Scope Pre-Check

Before interacting with files, confirm:
1. The target path is within the session's authorized folders:
   ```bash
   deskctl session start --task "Organize downloads" --folders "/home/jimmy/Downloads,/home/jimmy/Documents"
   ```
2. The target path is **not** inside a protected zone (`~/.ssh`, `~/.gnupg`, `~/.aws`, `.env`, browser profiles, or keyrings). `deskctl` strictly rejects access to protected zones.
3. Permanent deletion without user confirmation is forbidden. Always use `trash` instead of `rm -rf`.

---

## 2. Programmatic File Management (`deskctl file`)

For structured and atomic operations, prefer `deskctl file` commands:

### List Files in a Directory
```bash
deskctl file list /home/jimmy/Downloads
```

### Read File Contents
```bash
deskctl file read /home/jimmy/Documents/notes.txt
```

### Write or Overwrite a File
```bash
deskctl file write /home/jimmy/Documents/output.txt "Generated report content\n"
```

### Move or Rename a File
```bash
deskctl file move /home/jimmy/Downloads/data.csv /home/jimmy/Documents/data.csv
```

### Safely Move to Trash
```bash
deskctl file trash /home/jimmy/Downloads/unneeded_temp_file.tmp
```

### Archive Folders
To compress files into a zip or tar archive:
```bash
deskctl run "tar -czf /home/jimmy/Documents/archive.tar.gz -C /home/jimmy/Downloads dataset"
```

---

## 3. GUI File Management in Nemo

When demonstrating operations to the user or when GUI interaction is requested:

### Create a Folder in Nemo
```bash
# 1. Open Nemo at the parent directory
deskctl launch "nemo /home/jimmy/Documents"
deskctl wait-for-window --title "Documents" --timeout 5000
deskctl focus --title "Documents"

# 2. Trigger new folder shortcut (Ctrl+Shift+N)
deskctl key "ctrl+shift+n"
deskctl wait --ms 300

# 3. Enter folder name and confirm
deskctl type "Projects_2026\n"
deskctl wait --ms 300

# 4. Verify folder creation
deskctl file list /home/jimmy/Documents
```

### Move Files Between Panes (F3 Split View)
```bash
deskctl focus --title "Documents"
# Open dual-pane view
deskctl key "F3"
deskctl wait --ms 300

# Switch active pane with Tab
deskctl key "Tab"
# Navigate right pane to destination (Ctrl+L)
deskctl key "ctrl+l"
deskctl type "/home/jimmy/Pictures\n"
deskctl wait --ms 300

# Switch back to source pane
deskctl key "Tab"
# Select item and press F2 or drag
deskctl key "ctrl+x"
deskctl key "Tab"
deskctl key "ctrl+v"
```

### Verification
Always verify by listing the folder or capturing `--screenshot-after`.
