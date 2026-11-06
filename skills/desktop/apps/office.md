# Office Suite Guide (`libreoffice`)

Guidelines for automating office applications (LibreOffice Writer, Calc, Impress) on Linux Mint.

## Launching Applications

```bash
# Launch LibreOffice Writer (Word Processor)
deskctl launch "libreoffice --writer"
deskctl wait-for-window --title "LibreOffice Writer" --timeout 8000

# Launch LibreOffice Calc (Spreadsheets)
deskctl launch "libreoffice --calc"
deskctl wait-for-window --title "LibreOffice Calc" --timeout 8000

# Open an existing document
deskctl launch "libreoffice /home/jimmy/Documents/report.docx"
```

## Essential Office Shortcuts

| Action | Shortcut | Deskctl Command |
| :--- | :--- | :--- |
| **New Document** | `Ctrl+N` | `deskctl key "ctrl+n"` |
| **Open Document** | `Ctrl+O` | `deskctl key "ctrl+o"` |
| **Save Document** | `Ctrl+S` | `deskctl key "ctrl+s"` |
| **Save As...** | `Ctrl+Shift+S` | `deskctl key "ctrl+shift+s"` |
| **Print Preview** | `Ctrl+Shift+O` | `deskctl key "ctrl+shift+o"` |
| **Export Directly to PDF** | (Menu) `Alt+F`, `D` | `deskctl key "alt+f"`, `deskctl key "d"` |
| **Find & Replace** | `Ctrl+H` | `deskctl key "ctrl+h"` |
| **Spell Check** | `F7` | `deskctl key "F7"` |
| **Full Screen** | `Ctrl+Shift+J` | `deskctl key "ctrl+shift+j"` |

## Interacting with Documents

### 1. Navigating Calc Spreadsheets
In LibreOffice Calc:
```bash
deskctl focus --title "LibreOffice Calc"
# Jump to Name Box (Cell Address)
deskctl key "ctrl+shift+F5"
deskctl wait --ms 200
# Type cell coordinates (e.g. C12) and press Enter
deskctl type "C12\n"
# Enter value or formula
deskctl type "=SUM(A1:A10)\n"
```

### 2. Exporting to PDF via Headless CLI
When the task only requires converting documents or generating PDFs, prefer the headless command line over driving the GUI:
```bash
deskctl run "libreoffice --headless --convert-to pdf document.docx"
```
This is faster, fully deterministic, and avoids window focus or dialogue race conditions.
