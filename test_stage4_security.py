#!/usr/bin/env python3
"""
Stage 4 Security and Scope Engine Verification Suite
Tests:
1. Protected zones (impossible to opt out): ~/.ssh, .env, ~/.aws, keyrings
2. Shell command protected zone inspection
3. Violation logging in ~/.deskctl/violations.log
4. Session scope boundaries (folders, apps, actions, domains)
5. Privacy mode in screenshots
6. Config & Budget limits in ~/.deskctl/config.toml
7. Emergency Panic command
"""

import os
import sys
import json
import time
import subprocess

DESKCTL = "/home/jimmy/.local/bin/deskctl"

def run_cmd(args):
    cmd = [DESKCTL] + args
    res = subprocess.run(cmd, capture_output=True, text=True)
    try:
        return json.loads(res.stdout)
    except Exception as e:
        print(f"Error parsing JSON from {cmd}:")
        print("STDOUT:", res.stdout)
        print("STDERR:", res.stderr)
        return {"ok": False, "error": f"JSON parse error: {e}", "raw": res.stdout}

def test_stage4():
    results = {}
    print("==================================================")
    print("Starting Stage 4 Security & Scope Verification Suite")
    print("==================================================")

    # 1. Protected Zones Enforcement
    print("\n--- [1/6] Protected Zones Enforcement ---")
    # SSH
    ssh_test = run_cmd(["file", "read", "~/.ssh/id_rsa"])
    assert not ssh_test.get("ok") and ssh_test.get("violation")
    print(f"Protected zone ~/.ssh blocked: {ssh_test.get('error')}")

    # .env
    env_test = run_cmd(["file", "read", "/tmp/.env.production"])
    assert not env_test.get("ok") and env_test.get("violation")
    print(f"Protected zone .env blocked: {env_test.get('error')}")

    # AWS
    aws_test = run_cmd(["file", "write", "~/.aws/credentials", "bad_data"])
    assert not aws_test.get("ok") and aws_test.get("violation")
    print(f"Protected zone ~/.aws blocked: {aws_test.get('error')}")

    # Shell protected zone inspection
    sh_test = run_cmd(["run", "cat ~/.ssh/config"])
    assert not sh_test.get("ok") and sh_test.get("violation")
    print(f"Shell command protected zone inspection blocked: {sh_test.get('error')}")
    results["protected_zones"] = True

    # 2. Violation Logging
    print("\n--- [2/6] Security Violation Logging ---")
    vlog_path = os.path.expanduser("~/.deskctl/violations.log")
    assert os.path.exists(vlog_path)
    with open(vlog_path, "r") as f:
        vlines = [json.loads(l) for l in f if l.strip()]
    assert len(vlines) > 0
    recent_types = [v.get("type") for v in vlines[-4:]]
    print(f"Logged violations verified: {recent_types}")
    results["violation_logging"] = True

    # 3. Session Scope Enforcement
    print("\n--- [3/6] Session Scope Boundaries ---")
    # Start session with declared scope: allowed folders [/tmp], allowed actions [screenshot, file]
    s_start = run_cmd(["session", "start", "--task", "Stage 4 Scope Test", "--folders", "/tmp", "--actions", "screenshot,file"])
    assert s_start.get("ok")
    print(f"Session started with scope: folders={s_start['session']['scope']['folders']}, actions={s_start['session']['scope']['actions']}")

    # Allowed in-scope action
    in_scope = run_cmd(["file", "write", "/tmp/deskctl_scope_ok.txt", "in-scope content"])
    assert in_scope.get("ok")
    print("In-scope file write to /tmp succeeded")

    # Out-of-scope folder access
    out_folder = run_cmd(["file", "read", "/etc/hosts"])
    assert not out_folder.get("ok") and out_folder.get("violation")
    print(f"Out-of-scope folder blocked: {out_folder.get('error')}")

    # Out-of-scope action
    out_action = run_cmd(["volume", "get"])
    assert not out_action.get("ok") and out_action.get("violation")
    print(f"Out-of-scope action blocked: {out_action.get('error')}")

    run_cmd(["session", "end"])
    results["session_scope"] = True

    # 4. Privacy Mode
    print("\n--- [4/6] Privacy Mode ---")
    shot_priv = run_cmd(["screenshot", "--privacy", "--max-width", "640"])
    assert shot_priv.get("ok")
    assert shot_priv.get("target_type") == "window"
    print(f"Privacy screenshot ok: target_type={shot_priv.get('target_type')}, path={shot_priv.get('path')}")
    results["privacy_mode"] = True

    # 5. Configuration & Budget
    print("\n--- [5/6] Configuration & Budget ---")
    cfg_get = run_cmd(["config", "show"])
    assert cfg_get.get("ok")
    orig_max = cfg_get["config"]["max_actions_per_session"]
    print(f"Config show ok: max_actions_per_session={orig_max}")

    cfg_set = run_cmd(["config", "set", "max_actions_per_session", "150"])
    assert cfg_set.get("ok")
    assert cfg_set["config"]["max_actions_per_session"] == 150
    print("Config set ok: max_actions_per_session set to 150")

    # Restore config
    run_cmd(["config", "set", "max_actions_per_session", str(orig_max)])
    results["config_budget"] = True

    # 6. Emergency Panic Command
    print("\n--- [6/6] Emergency Panic Command ---")
    panic_res = run_cmd(["panic"])
    assert panic_res.get("ok") and panic_res.get("panic")
    stop_path = os.path.expanduser("~/.deskctl/STOP")
    assert os.path.exists(stop_path)
    print(f"Emergency panic verified: STOP file created at {stop_path}")

    # Verify deskctl actions are blocked while STOP is active
    blocked = run_cmd(["mouse-position"])
    assert not blocked.get("ok") and "Kill switch active" in blocked.get("error", "")
    print("Actions blocked by killswitch verified")

    # Reset
    reset_res = run_cmd(["reset"])
    assert reset_res.get("ok")
    assert not os.path.exists(stop_path)
    print("deskctl reset cleared killswitch successfully")
    results["emergency_panic"] = True

    print("\n==================================================")
    print("STAGE 4 VERIFICATION SUMMARY:")
    all_ok = True
    for k, v in results.items():
        status = "PASSED" if v else "FAILED"
        print(f"  [✓] {k:<28}: {status}")
        if not v:
            all_ok = False
    print("==================================================")
    return all_ok

if __name__ == "__main__":
    success = test_stage4()
    sys.exit(0 if success else 1)
