import argparse
import json
import os
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

DEFAULT_IDA_PATH = Path(r"D:\CTF\tool\SURBIFByb2Zlc3Npb25hbCA5LjE\ida.exe")
DEFAULT_BOOTSTRAP_PATH = Path(r"D:\AI\Local\Workflows\windows\ida-mcp-bootstrap.py")
DEFAULT_PLUGIN_PATH = Path(r"D:\CTF\tool\SURBIFByb2Zlc3Npb25hbCA5LjE\plugins\mcp-plugin.py")
DEFAULT_LOG_PATH = Path(r"D:\AI\Local\tmp\ida-mcp-bootstrap.log")
DEFAULT_IDA_LOG_PATH = Path(r"D:\AI\Local\tmp\ida-mcp-ida.log")
DEFAULT_HOST = "127.0.0.1"
DEFAULT_PORT = 13337


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Launch IDA on a target file and wait until the MCP bridge is ready.")
    parser.add_argument("target", help="Binary or file to open in IDA")
    parser.add_argument("--ida-path", default=str(DEFAULT_IDA_PATH))
    parser.add_argument("--bootstrap-path", default=str(DEFAULT_BOOTSTRAP_PATH))
    parser.add_argument("--plugin-path", default=str(DEFAULT_PLUGIN_PATH))
    parser.add_argument("--bootstrap-log", default=str(DEFAULT_LOG_PATH))
    parser.add_argument("--ida-log", default=str(DEFAULT_IDA_LOG_PATH))
    parser.add_argument("--host", default=DEFAULT_HOST)
    parser.add_argument("--port", type=int, default=DEFAULT_PORT)
    parser.add_argument("--timeout", type=float, default=90.0)
    parser.add_argument("--analysis", action="store_true", help="Pass -A to reduce interactive prompts during startup")
    parser.add_argument("--allow-existing-bridge", action="store_true", help="Return success immediately if the bridge is already listening")
    parser.add_argument("--replace-existing", action="store_true", help="If another IDA MCP bridge is already active, close existing ida/idat processes and replace it")
    parser.add_argument("--close-only", action="store_true", help="Close active ida/idat processes and exit without opening a new target")
    parser.add_argument("--json", action="store_true", help="Emit machine-readable JSON only")
    return parser.parse_args()


def port_open(host: str, port: int) -> bool:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(0.5)
    try:
        sock.connect((host, port))
        return True
    except OSError:
        return False
    finally:
        sock.close()


def probe_bridge(host: str, port: int) -> tuple[bool, str]:
    url = f"http://{host}:{port}/mcp"
    payload = json.dumps({
        "jsonrpc": "2.0",
        "id": 1,
        "method": "__bridge_probe__",
        "params": []
    }).encode("utf-8")
    request = urllib.request.Request(url, data=payload, headers={"Content-Type": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(request, timeout=2.0) as response:
            body = response.read().decode("utf-8", errors="replace")
            return True, body
    except urllib.error.HTTPError as exc:
        return True, exc.read().decode("utf-8", errors="replace")
    except Exception as exc:
        return False, str(exc)


def emit(payload: dict, json_only: bool):
    if json_only:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
        return

    print("===== ida-open-and-wait =====")
    print(json.dumps(payload, ensure_ascii=False, indent=2))


def list_ida_processes() -> list[dict[str, str]]:
    completed = subprocess.run(
        ["tasklist", "/FO", "CSV", "/NH"],
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        timeout=15,
    )
    rows = []
    for line in completed.stdout.splitlines():
        line = line.strip()
        if not line:
            continue
        if not (line.startswith('"ida.exe"') or line.startswith('"idat.exe"')):
            continue
        parts = [item.strip('"') for item in line.split('","')]
        if len(parts) < 2:
            continue
        rows.append({"image": parts[0], "pid": parts[1]})
    return rows


def kill_ida_processes() -> dict:
    before = list_ida_processes()
    killed = []
    errors = []
    seen = set()
    for proc in before:
        pid = proc.get("pid", "").strip()
        if not pid or pid in seen:
            continue
        seen.add(pid)
        completed = subprocess.run(
            ["taskkill", "/PID", pid, "/F"],
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            timeout=20,
        )
        if completed.returncode == 0:
            killed.append(proc)
        else:
            errors.append({
                "pid": pid,
                "image": proc.get("image", ""),
                "stderr": completed.stderr.strip(),
                "stdout": completed.stdout.strip(),
            })
    return {
        "found": before,
        "killed": killed,
        "errors": errors,
    }


def wait_port_closed(host: str, port: int, timeout_seconds: float) -> bool:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        if not port_open(host, port):
            return True
        time.sleep(0.5)
    return not port_open(host, port)


if __name__ == "__main__":
    args = parse_args()
    target = Path(args.target).expanduser().resolve()
    ida_path = Path(args.ida_path).expanduser().resolve()
    bootstrap_path = Path(args.bootstrap_path).expanduser().resolve()
    plugin_path = Path(args.plugin_path).expanduser().resolve()
    bootstrap_log = Path(args.bootstrap_log).expanduser().resolve()
    ida_log = Path(args.ida_log).expanduser().resolve()

    result = {
        "ok": False,
        "target": str(target),
        "ida_path": str(ida_path),
        "bootstrap_path": str(bootstrap_path),
        "plugin_path": str(plugin_path),
        "bootstrap_log": str(bootstrap_log),
        "ida_log": str(ida_log),
        "bridge_url": f"http://{args.host}:{args.port}/mcp",
        "ida_launched": False,
        "bridge_ready": False,
        "existing_bridge": False,
        "replaced_existing": False,
        "closed_existing": None,
        "command": None,
        "pid": None,
        "note": ""
    }

    if args.close_only:
        close_result = kill_ida_processes()
        result["closed_existing"] = close_result
        port_closed = wait_port_closed(args.host, args.port, 12.0)
        result["bridge_ready"] = port_open(args.host, args.port)
        result["ok"] = port_closed
        result["replaced_existing"] = bool(close_result["killed"])
        result["note"] = "closed active IDA processes" if port_closed else "IDA processes were closed but bridge port is still busy"
        emit(result, args.json)
        sys.exit(0 if port_closed else 4)

    for path_obj, label in ((target, "target"), (ida_path, "ida_path"), (bootstrap_path, "bootstrap_path"), (plugin_path, "plugin_path")):
        if not path_obj.exists():
            result["note"] = f"missing {label}: {path_obj}"
            emit(result, args.json)
            sys.exit(1)

    bootstrap_log.parent.mkdir(parents=True, exist_ok=True)
    ida_log.parent.mkdir(parents=True, exist_ok=True)

    if port_open(args.host, args.port):
        ready, detail = probe_bridge(args.host, args.port)
        result["bridge_ready"] = ready
        result["existing_bridge"] = True
        if args.allow_existing_bridge and ready:
            result["ok"] = True
            result["note"] = "bridge already running; launcher skipped opening another IDA instance"
            result["probe"] = detail
            emit(result, args.json)
            sys.exit(0)
        if args.replace_existing:
            close_result = kill_ida_processes()
            result["closed_existing"] = close_result
            result["replaced_existing"] = bool(close_result["killed"])
            if not wait_port_closed(args.host, args.port, 12.0):
                result["note"] = "existing bridge was detected but the port stayed busy after closing ida/idat processes"
                result["probe"] = detail
                emit(result, args.json)
                sys.exit(5)
        else:
            result["note"] = "bridge already running on the target port; rerun with --replace-existing to close old IDA and open the new target"
            result["probe"] = detail
            emit(result, args.json)
            sys.exit(2)

    env = os.environ.copy()
    env["SHEATHED_EDGE_IDA_MCP_HOST"] = args.host
    env["SHEATHED_EDGE_IDA_MCP_PORT"] = str(args.port)
    env["SHEATHED_EDGE_IDA_MCP_PLUGIN_PATH"] = str(plugin_path)
    env["SHEATHED_EDGE_IDA_MCP_LOG"] = str(bootstrap_log)
    env.setdefault("SHEATHED_EDGE_IDA_MCP_BOOTSTRAP_TIMEOUT", "20")

    if bootstrap_log.exists():
        bootstrap_log.unlink()
    if ida_log.exists():
        ida_log.unlink()

    command_parts = [f'"{ida_path}"']
    if args.analysis:
        command_parts.append("-A")
    command_parts.append(f'-L"{ida_log}"')
    command_parts.append(f'-S"{bootstrap_path}"')
    command_parts.append(f'"{target}"')
    command_line = " ".join(command_parts)
    result["command"] = command_line

    creationflags = 0
    if os.name == "nt":
        creationflags = getattr(subprocess, "CREATE_NEW_PROCESS_GROUP", 0) | getattr(subprocess, "DETACHED_PROCESS", 0)

    process = subprocess.Popen(command_line, env=env, creationflags=creationflags)
    result["ida_launched"] = True
    result["pid"] = process.pid

    deadline = time.time() + args.timeout
    last_probe = "bridge not responding yet"
    while time.time() < deadline:
        ready, detail = probe_bridge(args.host, args.port)
        last_probe = detail
        if ready:
            result["ok"] = True
            result["bridge_ready"] = True
            result["probe"] = detail
            if result["replaced_existing"]:
                result["note"] = "existing IDA session was closed; new IDA target opened and MCP bridge is ready"
            else:
                result["note"] = "IDA launched and MCP bridge is ready"
            emit(result, args.json)
            sys.exit(0)
        time.sleep(1.0)

    result["note"] = f"IDA launched but MCP bridge did not become ready within {args.timeout} seconds"
    result["probe"] = last_probe
    emit(result, args.json)
    sys.exit(3)
