import builtins
import importlib.util
import os
import socket
import sys
import time
from pathlib import Path

DEFAULT_PLUGIN_PATH = r"D:\CTF\tool\SURBIFByb2Zlc3Npb25hbCA5LjE\plugins\mcp-plugin.py"
DEFAULT_HOST = "127.0.0.1"
DEFAULT_PORT = 13337
DEFAULT_LOG_PATH = Path(r"D:\AI\Local\tmp\ida-mcp-bootstrap.log")


def _log(message: str):
    try:
        log_path = Path(os.environ.get("SHEATHED_EDGE_IDA_MCP_LOG", str(DEFAULT_LOG_PATH)))
        log_path.parent.mkdir(parents=True, exist_ok=True)
        with log_path.open("a", encoding="utf-8") as handle:
            handle.write(f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] {message}\n")
    except Exception:
        pass
    print(message)


def _load_module(module_path: Path):
    spec = importlib.util.spec_from_file_location("ida_mcp_plugin_autoload", module_path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"Failed to create import spec for {module_path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def _port_open(host: str, port: int) -> bool:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(0.5)
    try:
        sock.connect((host, port))
        return True
    except OSError:
        return False
    finally:
        sock.close()


def main():
    host = os.environ.get("SHEATHED_EDGE_IDA_MCP_HOST", DEFAULT_HOST)
    port = int(os.environ.get("SHEATHED_EDGE_IDA_MCP_PORT", str(DEFAULT_PORT)))
    plugin_path = Path(os.environ.get("SHEATHED_EDGE_IDA_MCP_PLUGIN_PATH", DEFAULT_PLUGIN_PATH))

    _log(f"[IDA-MCP-AUTOSTART] bootstrap begin host={host} port={port} plugin={plugin_path}")

    if _port_open(host, port):
        _log(f"[IDA-MCP-AUTOSTART] bridge already listening on {host}:{port}, skip duplicate start")
        return

    if not plugin_path.exists():
        raise FileNotFoundError(f"IDA MCP plugin not found: {plugin_path}")

    module = _load_module(plugin_path)
    module.Server.HOST = host
    module.Server.PORT = port
    server = module.Server()
    builtins.__dict__["__sheathed_edge_ida_mcp_server"] = server
    builtins.__dict__["__sheathed_edge_ida_mcp_module"] = module
    server.start()

    deadline = time.time() + float(os.environ.get("SHEATHED_EDGE_IDA_MCP_BOOTSTRAP_TIMEOUT", "20"))
    while time.time() < deadline:
        if _port_open(host, port):
            _log(f"[IDA-MCP-AUTOSTART] bridge ready on http://{host}:{port}/mcp")
            return
        time.sleep(0.5)

    raise TimeoutError(f"IDA MCP bridge did not become ready on {host}:{port}")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        _log(f"[IDA-MCP-AUTOSTART] bootstrap failed: {exc!r}")
        raise
