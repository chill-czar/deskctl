#!/usr/bin/env python3
"""
Kill switch daemon for deskctl.
Monitors X11 for global hotkey Ctrl+Alt+Escape.
When detected, writes ~/.deskctl/STOP and terminates running deskctl actions.
"""

import os
import sys
import time
import signal
import subprocess
from pathlib import Path
from Xlib import X, display, XK
from Xlib.ext import record
from Xlib.protocol import rq

DESKCTL_DIR = Path.home() / ".deskctl"
STOP_FILE = DESKCTL_DIR / "STOP"
LOG_FILE = DESKCTL_DIR / "actions.log"
PID_FILE = DESKCTL_DIR / "killswitch.pid"

def log_action(msg):
    DESKCTL_DIR.mkdir(parents=True, exist_ok=True)
    ts = time.strftime("%Y-%m-%dT%H:%M:%S%z")
    with open(LOG_FILE, "a") as f:
        f.write(f"[{ts}] [KILLSWITCH] {msg}\n")

def trigger_killswitch():
    try:
        DESKCTL_DIR.mkdir(parents=True, exist_ok=True)
        with open(STOP_FILE, "w") as f:
            f.write(f"TRIGGERED at {time.strftime('%Y-%m-%dT%H:%M:%S%z')} via Ctrl+Alt+Escape\n")
        log_action("Ctrl+Alt+Escape detected! Created ~/.deskctl/STOP and terminating deskctl actions.")
        
        # Signal running deskctl action commands (excluding daemon)
        my_pid = os.getpid()
        # Find deskctl processes
        res = subprocess.run(["pgrep", "-f", "deskctl"], capture_output=True, text=True)
        if res.returncode == 0:
            for pid_str in res.stdout.strip().split():
                try:
                    p = int(pid_str)
                    if p != my_pid:
                        os.kill(p, signal.SIGTERM)
                except Exception:
                    pass
    except Exception as e:
        sys.stderr.write(f"Error triggering killswitch: {e}\n")

def main():
    DESKCTL_DIR.mkdir(parents=True, exist_ok=True)
    with open(PID_FILE, "w") as f:
        f.write(str(os.getpid()))

    record_d = display.Display()
    control_d = display.Display()

    keysym = XK.string_to_keysym("Escape")
    escape_keycode = record_d.keysym_to_keycode(keysym)

    ctx = record_d.record_create_context(
        0,
        [record.AllClients],
        [{
            "core_requests": (0, 0),
            "core_replies": (0, 0),
            "ext_requests": (0, 0, 0, 0),
            "ext_replies": (0, 0, 0, 0),
            "delivered_events": (0, 0),
            "device_events": (X.KeyPress, X.KeyRelease),
            "errors": (0, 0),
            "client_started": False,
            "client_died": False,
        }]
    )

    def record_cb(reply):
        data = reply.data
        while len(data):
            event, data = rq.EventField(None).parse_binary_value(data, record_d.display, None, None)
            if event.type == X.KeyPress:
                # event.detail is keycode, event.state has modifier masks
                # ControlMask = 4, Mod1Mask = 8 (Alt)
                is_ctrl = bool(event.state & X.ControlMask)
                is_alt = bool(event.state & X.Mod1Mask)
                if event.detail == escape_keycode and is_ctrl and is_alt:
                    trigger_killswitch()

    def handle_sigterm(signum, frame):
        try:
            if PID_FILE.exists():
                PID_FILE.unlink()
            control_d.record_disable_context(ctx)
            control_d.record_free_context(ctx)
            control_d.sync()
        except Exception:
            pass
        sys.exit(0)

    signal.signal(signal.SIGTERM, handle_sigterm)
    signal.signal(signal.SIGINT, handle_sigterm)

    log_action("Kill switch daemon started listening for Ctrl+Alt+Escape")
    record_d.record_enable_context(ctx, record_cb)

if __name__ == "__main__":
    main()
