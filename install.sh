#!/usr/bin/env bash
# Installation and setup script for deskctl toolkit and desktop agent skill
set -euo pipefail

echo "=========================================================="
echo "Installing deskctl Desktop Control System & Agent Skill"
echo "=========================================================="

PREFIX="${HOME}/.local/bin"
DESKCTL_DIR="${HOME}/.deskctl"
SKILLS_DIR="${HOME}/.gemini/config/skills/desktop"
REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p "${PREFIX}"
mkdir -p "${DESKCTL_DIR}/sessions"
mkdir -p "${DESKCTL_DIR}/helpers"
mkdir -p "${DESKCTL_DIR}/screenshots"

# 1. Check system dependencies
echo "1. Checking required Linux dependencies..."
DEPS=("xdotool" "maim" "wmctrl" "xclip" "xrandr" "ffmpeg" "tesseract" "python3")
MISSING=()
for dep in "${DEPS[@]}"; do
    if ! command -v "$dep" >/dev/null 2>&1; then
        MISSING+=("$dep")
    fi
done

if [ ${#MISSING[@]} -gt 0 ]; then
    echo "Warning: Missing recommended packages: ${MISSING[*]}"
    echo "Install them using: sudo apt install -y ${MISSING[*]}"
fi

# 2. Build or install deskctl binary
echo "2. Compiling and installing deskctl binary to ${PREFIX}/deskctl..."
if command -v go >/dev/null 2>&1; then
    (cd "${REPO_DIR}" && go build -o "${PREFIX}/deskctl" .)
    chmod +x "${PREFIX}/deskctl"
elif [ -f "${REPO_DIR}/deskctl" ]; then
    cp "${REPO_DIR}/deskctl" "${PREFIX}/deskctl"
    chmod +x "${PREFIX}/deskctl"
else
    echo "Error: Neither Go nor pre-compiled deskctl binary was found."
    exit 1
fi

# 3. Install Python helpers
echo "3. Installing Python helpers..."
if [ -d "${REPO_DIR}/helper" ]; then
    cp -r "${REPO_DIR}/helper/"* "${DESKCTL_DIR}/helpers/" 2>/dev/null || true
fi

# 4. Install Antigravity desktop skill
echo "4. Installing Antigravity desktop skill to ${SKILLS_DIR}..."
if [ -d "${REPO_DIR}/skills/desktop" ]; then
    mkdir -p "${SKILLS_DIR}"
    cp -r "${REPO_DIR}/skills/desktop/"* "${SKILLS_DIR}/"
fi

# 5. Run diagnostics
echo "5. Running deskctl doctor..."
"${PREFIX}/deskctl" doctor

echo ""
echo "=========================================================="
echo "Installation complete!"
echo "Binary: ${PREFIX}/deskctl"
echo "Skill:  ${SKILLS_DIR}"
echo "Run 'deskctl --help' or 'deskctl doctor' to get started."
echo "=========================================================="
