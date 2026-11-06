#!/usr/bin/env bash
# Installation and setup script for deskctl
set -euo pipefail

echo "=== Installing deskctl Desktop Control Toolkit ==="

PREFIX="${HOME}/.local/bin"
DESKCTL_DIR="${HOME}/.deskctl"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../antigravity/scratch/deskctl" && pwd)"

mkdir -p "${PREFIX}"
mkdir -p "${DESKCTL_DIR}/sessions"
mkdir -p "${DESKCTL_DIR}/helpers"

# 1. Check system dependencies
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

# 2. Build or copy deskctl binary
if [ -f "${SOURCE_DIR}/deskctl" ]; then
    echo "Installing deskctl binary to ${PREFIX}/deskctl..."
    cp "${SOURCE_DIR}/deskctl" "${PREFIX}/deskctl"
    chmod +x "${PREFIX}/deskctl"
else
    echo "Compiling deskctl from source..."
    cd "${SOURCE_DIR}"
    go build -o "${PREFIX}/deskctl" .
    chmod +x "${PREFIX}/deskctl"
fi

# 3. Copy helper scripts
if [ -d "${SOURCE_DIR}/helper" ]; then
    cp -r "${SOURCE_DIR}/helper/"* "${DESKCTL_DIR}/helpers/" 2>/dev/null || true
fi

# 4. Verify installation
echo "Verifying deskctl installation..."
"${PREFIX}/deskctl" version || "${PREFIX}/deskctl" screens

echo ""
echo "=== deskctl installation complete! ==="
echo "Binary path: ${PREFIX}/deskctl"
echo "Make sure ${PREFIX} is in your PATH."
