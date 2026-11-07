#!/usr/bin/env python3
"""
AT-SPI Accessibility helper for deskctl.
Provides fast JSON introspection of accessible desktop applications,
trees, coordinate finding, text extraction/manipulation, and element actions.
"""

import sys
import io
import os
import json
import socket
import signal
import argparse
import pyatspi

INTERACTIVE_ROLES = {
    "push button", "check box", "radio button", "combo box", "text", "password text",
    "menu item", "check menu item", "radio menu item", "tab", "tree table", "table cell",
    "list item", "link", "slider", "spin button", "toggle button", "entry", "page tab"
}

def is_interactive(node, role, name, states):
    if role.lower() in INTERACTIVE_ROLES:
        return True
    if name and name.strip() and role.lower() not in {"filler", "separator", "panel", "unknown"}:
        return True
    try:
        act = node.queryAction()
        if act and act.nActions > 0:
            return True
    except Exception:
        pass
    return False

def clean_states(acc):
    try:
        return [s.value_nick for s in acc.getState().getStates()]
    except Exception:
        return []

def get_bbox(acc):
    try:
        comp = acc.queryComponent()
        if comp:
            ext = comp.getExtents(pyatspi.DESKTOP_COORDS)
            if ext.x < -10000 or ext.y < -10000 or ext.width <= 0 or ext.height <= 0:
                return None
            return {
                "x": int(ext.x),
                "y": int(ext.y),
                "w": int(ext.width),
                "h": int(ext.height)
            }
    except Exception:
        pass
    return None

def find_target_app_or_frames(reg, pid=None, title_substr=None):
    desktop = reg.getDesktop(0)
    targets = []
    
    for i in range(desktop.childCount):
        try:
            app = desktop.getChildAtIndex(i)
            if not app:
                continue
            
            app_pid = None
            for attr in ['get_process_id', 'process_id']:
                if hasattr(app, attr):
                    try:
                        app_pid = getattr(app, attr)()
                        break
                    except Exception:
                        pass
            
            if pid is not None and app_pid == pid:
                targets.append(app)
                continue
            
            if title_substr:
                q = title_substr.lower()
                if q in (app.name or "").lower():
                    targets.append(app)
                    continue
                for j in range(app.childCount):
                    child = app.getChildAtIndex(j)
                    if child and q in (child.name or "").lower():
                        targets.append(child)
        except Exception:
            continue
            
    if not targets and pid is None and not title_substr:
        for i in range(desktop.childCount):
            try:
                targets.append(desktop.getChildAtIndex(i))
            except Exception:
                pass
    return targets

def collect_tree(node, depth, max_depth, max_nodes, collected, role_filter=None, name_filter=None, interactive_only=False):
    if depth > max_depth or len(collected) >= max_nodes or not node:
        return
        
    try:
        role = node.getRoleName() or "unknown"
        name = node.name or ""
        bbox = get_bbox(node)
        states = clean_states(node)
        
        matches = True
        if role_filter and role_filter.lower() not in role.lower():
            matches = False
        if name_filter and name_filter.lower() not in name.lower():
            matches = False
        if interactive_only and not is_interactive(node, role, name, states):
            matches = False
            
        if matches:
            center = None
            if bbox:
                center = {
                    "x": bbox["x"] + bbox["w"] // 2,
                    "y": bbox["y"] + bbox["h"] // 2
                }
            collected.append({
                "depth": depth,
                "role": role,
                "name": name,
                "bbox": bbox,
                "center": center,
                "states": states
            })
            
        child_count = node.childCount
        for i in range(min(child_count, 100)):
            if len(collected) >= max_nodes:
                break
            child = node.getChildAtIndex(i)
            collect_tree(child, depth + 1, max_depth, max_nodes, collected, role_filter, name_filter, interactive_only)
    except Exception:
        pass

def cmd_tree(args):
    try:
        reg = pyatspi.Registry
    except Exception as e:
        print(json.dumps({"ok": False, "error": f"Failed to connect to AT-SPI registry: {e}"}))
        return
        
    role_filter = None
    if args.filter:
        parts = args.filter.split("=", 1)
        if len(parts) == 2 and parts[0].strip().lower() == "role":
            role_filter = parts[1].strip()
        else:
            role_filter = args.filter.strip()

    targets = find_target_app_or_frames(reg, pid=args.pid, title_substr=args.title)
    if not targets and (args.pid or args.title):
        print(json.dumps({
            "ok": False,
            "error": f"No AT-SPI application or frame found matching pid={args.pid} title={args.title}"
        }))
        return

    collected = []
    for root in targets:
        if len(collected) >= args.max_nodes:
            break
        collect_tree(root, depth=0, max_depth=args.max_depth, max_nodes=args.max_nodes,
                     collected=collected, role_filter=role_filter,
                     interactive_only=getattr(args, 'interactive_only', False))
        
    print(json.dumps({
        "ok": True,
        "nodes": collected,
        "total": len(collected),
        "truncated": len(collected) >= args.max_nodes
    }))

def search_nodes(node, query, role_exact, matches, max_matches=50):
    if len(matches) >= max_matches or not node:
        return
    try:
        role = node.getRoleName() or ""
        name = node.name or ""
        bbox = get_bbox(node)
        
        is_match = False
        q_lower = query.lower()
        if q_lower in name.lower() or q_lower == role.lower():
            is_match = True
            
        if role_exact and role_exact.lower() != role.lower():
            is_match = False
            
        if is_match and bbox:
            center = {
                "x": bbox["x"] + bbox["w"] // 2,
                "y": bbox["y"] + bbox["h"] // 2
            }
            matches.append({
                "name": name,
                "role": role,
                "bbox": bbox,
                "center": center,
                "states": clean_states(node)
            })
            
        for i in range(min(node.childCount, 100)):
            if len(matches) >= max_matches:
                break
            child = node.getChildAtIndex(i)
            search_nodes(child, query, role_exact, matches, max_matches)
    except Exception:
        pass

def cmd_find(args):
    try:
        reg = pyatspi.Registry
    except Exception as e:
        print(json.dumps({"ok": False, "error": f"Failed to connect to AT-SPI registry: {e}"}))
        return
        
    targets = find_target_app_or_frames(reg, pid=args.pid, title_substr=args.title)
    matches = []
    for root in targets:
        search_nodes(root, args.query, args.role, matches, max_matches=args.limit)
        if len(matches) >= args.limit:
            break
            
    print(json.dumps({
        "ok": True,
        "query": args.query,
        "role_filter": args.role,
        "matches": matches,
        "count": len(matches)
    }))

def find_first_node(node, query, role_exact):
    if not node:
        return None
    try:
        role = node.getRoleName() or ""
        name = node.name or ""
        is_match = False
        if not query or query.lower() in name.lower():
            is_match = True
        if role_exact and role_exact.lower() != role.lower():
            is_match = False
        if is_match:
            return node
        for i in range(node.childCount):
            res = find_first_node(node.getChildAtIndex(i), query, role_exact)
            if res:
                return res
    except Exception:
        pass
    return None

def cmd_get_text(args):
    try:
        reg = pyatspi.Registry
    except Exception as e:
        print(json.dumps({"ok": False, "error": f"Failed to connect to AT-SPI: {e}"}))
        return
    targets = find_target_app_or_frames(reg, pid=args.pid, title_substr=args.title)
    for root in targets:
        node = find_first_node(root, args.query, args.role)
        if node:
            try:
                txt_iface = node.queryText()
                if txt_iface:
                    char_count = txt_iface.characterCount
                    text = txt_iface.getText(0, char_count)
                    print(json.dumps({
                        "ok": True,
                        "text": text,
                        "length": len(text),
                        "role": node.getRoleName(),
                        "name": node.name
                    }))
                    return
            except Exception as e:
                pass
    print(json.dumps({"ok": False, "error": f"Element matching query='{args.query}' role='{args.role}' with Text interface not found"}))

def cmd_set_text(args):
    try:
        reg = pyatspi.Registry
    except Exception as e:
        print(json.dumps({"ok": False, "error": f"Failed to connect to AT-SPI: {e}"}))
        return
    targets = find_target_app_or_frames(reg, pid=args.pid, title_substr=args.title)
    for root in targets:
        node = find_first_node(root, args.query, args.role)
        if node:
            try:
                ed_iface = node.queryEditableText()
                if ed_iface:
                    ed_iface.setTextContents(args.text)
                    print(json.dumps({
                        "ok": True,
                        "action": "set-text",
                        "text": args.text,
                        "role": node.getRoleName(),
                        "name": node.name
                    }))
                    return
            except Exception as e:
                pass
    print(json.dumps({"ok": False, "error": f"Element matching query='{args.query}' role='{args.role}' with EditableText interface not found"}))

def cmd_actions(args):
    try:
        reg = pyatspi.Registry
    except Exception as e:
        print(json.dumps({"ok": False, "error": f"Failed to connect to AT-SPI: {e}"}))
        return
    targets = find_target_app_or_frames(reg, pid=args.pid, title_substr=args.title)
    for root in targets:
        node = find_first_node(root, args.query, args.role)
        if node:
            try:
                act_iface = node.queryAction()
                if act_iface:
                    n = act_iface.nActions
                    actions_list = [act_iface.getName(i) for i in range(n)]
                    if args.do_action:
                        idx = -1
                        for i, name in enumerate(actions_list):
                            if name.lower() == args.do_action.lower():
                                idx = i
                                break
                        if idx >= 0:
                            success = act_iface.doAction(idx)
                            print(json.dumps({
                                "ok": True,
                                "action_performed": args.do_action,
                                "success": success,
                                "available_actions": actions_list
                            }))
                            return
                        else:
                            print(json.dumps({
                                "ok": False,
                                "error": f"Action '{args.do_action}' not found in {actions_list}"
                            }))
                            return
                    else:
                        print(json.dumps({
                            "ok": True,
                            "actions": actions_list,
                            "count": len(actions_list),
                            "role": node.getRoleName(),
                            "name": node.name
                        }))
                        return
            except Exception as e:
                pass
    print(json.dumps({"ok": False, "error": f"Element matching query='{args.query}' role='{args.role}' with Action interface not found"}))

def cmd_auth_check(args):
    reg = pyatspi.Registry
    desktop = reg.getDesktop(0)
    prompts = []

    def scan(node, app_name, depth=0):
        if not node or depth > 8 or len(prompts) >= 5:
            return
        try:
            role = node.getRole()
            role_name = node.getRoleName() or ""
            states = clean_states(node)
            name = node.name or ""
            
            is_pw = False
            if role == pyatspi.ROLE_PASSWORD_TEXT or "password text" in role_name.lower():
                is_pw = True
            elif "protected" in states and ("text" in role_name.lower() or "entry" in role_name.lower()):
                is_pw = True
                
            if is_pw and ("showing" in states or "visible" in states):
                bbox = get_bbox(node)
                prompts.append({
                    "app": app_name,
                    "role": role_name,
                    "name": name,
                    "states": states,
                    "bbox": bbox
                })
                return
                
            for i in range(node.childCount):
                scan(node.getChildAtIndex(i), app_name, depth + 1)
        except Exception:
            pass

    for i in range(desktop.childCount):
        try:
            app = desktop.getChildAtIndex(i)
            app_name = app.name or "unknown"
            scan(app, app_name, depth=0)
        except Exception:
            pass

    print(json.dumps({"ok": True, "prompts": prompts, "count": len(prompts)}))

SOCK_PATH = "/tmp/deskctl-atspi.sock"

def execute_request(args_list, parser):
    old_stdout = sys.stdout
    sys.stdout = buffer = io.StringIO()
    try:
        args = parser.parse_args(args_list)
        if args.cmd == "tree":
            cmd_tree(args)
        elif args.cmd == "find":
            cmd_find(args)
        elif args.cmd == "get-text":
            cmd_get_text(args)
        elif args.cmd == "set-text":
            cmd_set_text(args)
        elif args.cmd == "actions":
            cmd_actions(args)
        elif args.cmd == "auth-check":
            cmd_auth_check(args)
        out = buffer.getvalue().strip()
        if not out:
            return json.dumps({"ok": False, "error": "No output produced"})
        return out
    except SystemExit:
        return json.dumps({"ok": False, "error": "Invalid argument syntax"})
    except Exception as e:
        return json.dumps({"ok": False, "error": str(e)})
    finally:
        sys.stdout = old_stdout

def run_socket_server(parser):
    if os.path.exists(SOCK_PATH):
        try:
            os.unlink(SOCK_PATH)
        except OSError:
            pass

    server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    server.bind(SOCK_PATH)
    server.listen(16)

    def cleanup(signum, frame):
        try:
            if os.path.exists(SOCK_PATH):
                os.unlink(SOCK_PATH)
        except Exception:
            pass
        sys.exit(0)

    signal.signal(signal.SIGTERM, cleanup)
    signal.signal(signal.SIGINT, cleanup)

    try:
        _ = pyatspi.Registry.getDesktop(0).name
    except Exception:
        pass

    while True:
        try:
            conn, _ = server.accept()
            with conn:
                data = b""
                while not data.endswith(b"\n"):
                    chunk = conn.recv(4096)
                    if not chunk:
                        break
                    data += chunk
                if not data:
                    continue
                req = json.loads(data.decode("utf-8"))
                cmd_args = req.get("args", [])
                resp_json = execute_request(cmd_args, parser)
                conn.sendall(resp_json.encode("utf-8") + b"\n")
        except Exception:
            pass

def build_parser():
    parser = argparse.ArgumentParser(description="AT-SPI helper for deskctl")
    parser.add_argument("--daemon", action="store_true", help="Run as persistent socket server daemon")
    sub = parser.add_subparsers(dest="cmd")
    
    p_tree = sub.add_parser("tree")
    p_tree.add_argument("--pid", type=int, default=None)
    p_tree.add_argument("--title", type=str, default=None)
    p_tree.add_argument("--filter", type=str, default=None)
    p_tree.add_argument("--max-depth", type=int, default=5)
    p_tree.add_argument("--max-nodes", type=int, default=150)
    p_tree.add_argument("--interactive-only", action="store_true", default=False)
    
    p_find = sub.add_parser("find")
    p_find.add_argument("query", type=str)
    p_find.add_argument("--pid", type=int, default=None)
    p_find.add_argument("--title", type=str, default=None)
    p_find.add_argument("--role", type=str, default=None)
    p_find.add_argument("--limit", type=int, default=20)

    p_get = sub.add_parser("get-text")
    p_get.add_argument("--query", type=str, default="")
    p_get.add_argument("--pid", type=int, default=None)
    p_get.add_argument("--title", type=str, default=None)
    p_get.add_argument("--role", type=str, default=None)

    p_set = sub.add_parser("set-text")
    p_set.add_argument("text", type=str)
    p_set.add_argument("--query", type=str, default="")
    p_set.add_argument("--pid", type=int, default=None)
    p_set.add_argument("--title", type=str, default=None)
    p_set.add_argument("--role", type=str, default=None)

    p_act = sub.add_parser("actions")
    p_act.add_argument("--query", type=str, default="")
    p_act.add_argument("--pid", type=int, default=None)
    p_act.add_argument("--title", type=str, default=None)
    p_act.add_argument("--role", type=str, default=None)
    p_act.add_argument("--do-action", type=str, default=None)
    
    p_auth = sub.add_parser("auth-check")
    return parser

def main():
    parser = build_parser()
    if "--daemon" in sys.argv:
        run_socket_server(parser)
        return

    args = parser.parse_args()
    if not args.cmd:
        parser.print_help()
        sys.exit(1)

    if args.cmd == "tree":
        cmd_tree(args)
    elif args.cmd == "find":
        cmd_find(args)
    elif args.cmd == "get-text":
        cmd_get_text(args)
    elif args.cmd == "set-text":
        cmd_set_text(args)
    elif args.cmd == "actions":
        cmd_actions(args)
    elif args.cmd == "auth-check":
        cmd_auth_check(args)

if __name__ == "__main__":
    main()
