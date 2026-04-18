import ast
import json
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

from mcp.server import FastMCP

ADAPTER_NAME = "ida-local-adapter"
DEFAULT_BRIDGE_URL = "http://127.0.0.1:13337/mcp"
DEFAULT_PLUGIN_PATH = Path(r"D:\CTF\tool\SURBIFByb2Zlc3Npb25hbCA5LjE\plugins\mcp-plugin.py")
DEFAULT_LAUNCHER_PATH = Path(r"D:\AI\Local\Workflows\windows\ida-open-and-wait.py")
DEFAULT_PYTHON = Path(sys.executable)

server = FastMCP(
    name=ADAPTER_NAME,
    instructions=(
        "Use ida_open_file before reverse-engineering a new binary. "
        "If another target is still open, the adapter should replace the old IDA session automatically. "
        "Then inspect ida_bridge_status or ida_list_rpc_methods, and finally call ida_rpc_call "
        "with real method names and concise JSON params."
    ),
)


def _bridge_request(method: str, params: Any) -> dict[str, Any]:
    payload = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": method,
        "params": params,
    }
    request = urllib.request.Request(
        DEFAULT_BRIDGE_URL,
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(request, timeout=20.0) as response:
            body = response.read().decode("utf-8", errors="replace")
            return json.loads(body)
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        try:
            return json.loads(body)
        except json.JSONDecodeError:
            return {"http_error": exc.code, "body": body}
    except Exception as exc:
        return {"transport_error": str(exc)}


def _parse_plugin_methods(plugin_path: Path = DEFAULT_PLUGIN_PATH) -> list[dict[str, str]]:
    module = ast.parse(plugin_path.read_text(encoding="utf-8"))
    methods: list[dict[str, str]] = []
    for node in module.body:
        if isinstance(node, ast.FunctionDef):
            decorator_names = []
            for deco in node.decorator_list:
                if isinstance(deco, ast.Name):
                    decorator_names.append(deco.id)
                elif isinstance(deco, ast.Call) and isinstance(deco.func, ast.Name):
                    decorator_names.append(deco.func.id)
            if "jsonrpc" not in decorator_names:
                continue
            doc = ast.get_docstring(node) or ""
            methods.append({"name": node.name, "doc": doc.strip()})
    return methods


def _run_launcher(args: list[str], timeout_seconds: int) -> dict[str, Any]:
    completed = subprocess.run(
        [str(DEFAULT_PYTHON), str(DEFAULT_LAUNCHER_PATH), *args],
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        timeout=timeout_seconds + 20,
    )
    stdout = completed.stdout.strip()
    stderr = completed.stderr.strip()
    if stdout:
        try:
            payload: dict[str, Any] = json.loads(stdout)
        except json.JSONDecodeError:
            payload = {"ok": False, "error": "launcher returned non-json stdout", "stdout": stdout}
    else:
        payload = {"ok": False, "error": "launcher returned empty stdout"}
    payload["returncode"] = completed.returncode
    if stderr:
        payload["stderr"] = stderr
    return payload


@server.tool(description="Open a target file in IDA 9.1 and wait until the local bridge at 127.0.0.1:13337/mcp is ready. By default it replaces any old IDA session that still owns the bridge port.")
def ida_open_file(target: str, analysis: bool = True, timeout_seconds: int = 90, replace_existing: bool = True) -> dict[str, Any]:
    launcher = DEFAULT_LAUNCHER_PATH
    if not launcher.exists():
        return {"ok": False, "error": f"launcher not found: {launcher}"}

    args = [target, "--timeout", str(timeout_seconds), "--json"]
    if analysis:
        args.append("--analysis")
    if replace_existing:
        args.append("--replace-existing")

    return _run_launcher(args, timeout_seconds)


@server.tool(description="Close any currently running ida.exe / idat.exe processes so the next ida_open_file call can start from a clean state.")
def ida_close_active_session(timeout_seconds: int = 20) -> dict[str, Any]:
    launcher = DEFAULT_LAUNCHER_PATH
    if not launcher.exists():
        return {"ok": False, "error": f"launcher not found: {launcher}"}
    return _run_launcher([r"C:\Windows\System32\notepad.exe", "--close-only", "--json"], timeout_seconds)


@server.tool(description="Check whether the local IDA bridge is reachable and return a lightweight probe result.")
def ida_bridge_status() -> dict[str, Any]:
    response = _bridge_request("__bridge_probe__", [])
    ok = "transport_error" not in response
    return {
        "ok": ok,
        "bridge_url": DEFAULT_BRIDGE_URL,
        "response": response,
    }


@server.tool(description="Get metadata for the currently opened IDA target.")
def ida_get_metadata() -> dict[str, Any]:
    response = _bridge_request("get_metadata", [])
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "response": response}


@server.tool(description="List functions from the currently opened IDA database. offset starts at 0.")
def ida_list_functions(count: int = 80, offset: int = 0) -> dict[str, Any]:
    response = _bridge_request("list_functions", {"offset": offset, "count": count})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "count": count, "offset": offset, "response": response}


@server.tool(description="List strings from the currently opened IDA database. offset starts at 0.")
def ida_list_strings(count: int = 120, offset: int = 0) -> dict[str, Any]:
    response = _bridge_request("list_strings", {"offset": offset, "count": count})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "count": count, "offset": offset, "response": response}


@server.tool(description="Get a function by name from the current IDA database.")
def ida_get_function_by_name(name: str) -> dict[str, Any]:
    response = _bridge_request("get_function_by_name", {"name": name})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "name": name, "response": response}


@server.tool(description="Get a function by address from the current IDA database. Example address: 0x401000")
def ida_get_function_by_address(address: str) -> dict[str, Any]:
    response = _bridge_request("get_function_by_address", {"address": address})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "address": address, "response": response}


@server.tool(description="Decompile the function at the given address. Example address: 0x401000")
def ida_decompile_function(address: str) -> dict[str, Any]:
    response = _bridge_request("decompile_function", {"address": address})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "address": address, "response": response}


@server.tool(description="Disassemble the function at the given address. Example address: 0x401000")
def ida_disassemble_function(address: str) -> dict[str, Any]:
    response = _bridge_request("disassemble_function", {"address": address})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "address": address, "response": response}


@server.tool(description="Get xrefs to the given address. Example address: 0x401000")
def ida_get_xrefs_to(address: str) -> dict[str, Any]:
    response = _bridge_request("get_xrefs_to", {"address": address})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "address": address, "response": response}


@server.tool(description="Get all callers of the given address. Example address: 0x401000")
def ida_get_callers(address: str) -> dict[str, Any]:
    response = _bridge_request("get_callers", {"address": address})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "address": address, "response": response}


@server.tool(description="Get all callees of the function at the given address. Example function_address: 0x401000")
def ida_get_callees(function_address: str) -> dict[str, Any]:
    response = _bridge_request("get_callees", {"function_address": function_address})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "function_address": function_address, "response": response}


@server.tool(description="Read a named global variable from the current IDA database. Prefer this over raw byte reads when a table or constant already has a symbol name.")
def ida_get_global_variable_value_by_name(variable_name: str) -> dict[str, Any]:
    response = _bridge_request("get_global_variable_value_by_name", {"variable_name": variable_name})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "variable_name": variable_name, "response": response}


@server.tool(description="Read a global/static variable by address from the current IDA database. Prefer this over raw byte reads when the data object is already defined in IDA.")
def ida_get_global_variable_value_at_address(address: str) -> dict[str, Any]:
    response = _bridge_request("get_global_variable_value_at_address", {"address": address})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "address": address, "response": response}


@server.tool(description="Read a string literal at the given address from the current IDA database.")
def ida_read_string(address: str) -> dict[str, Any]:
    response = _bridge_request("data_read_string", {"address": address})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "address": address, "response": response}


@server.tool(description="Read raw bytes at the given memory address from the current IDA database. Use this only when a higher-level global/table read is not available.")
def ida_read_memory_bytes(memory_address: str, size: int) -> dict[str, Any]:
    response = _bridge_request("read_memory_bytes", {"memory_address": memory_address, "size": size})
    ok = "result" in response and "error" not in response and "transport_error" not in response
    return {"ok": ok, "memory_address": memory_address, "size": size, "response": response}


def _read_scalar_array(method: str, field_name: str, base_address: str, count: int, step: int) -> dict[str, Any]:
    try:
        base_int = int(str(base_address), 0)
    except ValueError:
        return {"ok": False, "error": f"invalid base_address: {base_address}"}
    if count < 1:
        return {"ok": False, "error": "count must be >= 1"}
    if step < 1:
        return {"ok": False, "error": "step must be >= 1"}

    values: list[dict[str, Any]] = []
    for index in range(count):
        current = hex(base_int + index * step)
        response = _bridge_request(method, {"address": current})
        ok = "result" in response and "error" not in response and "transport_error" not in response
        values.append({
            "index": index,
            "address": current,
            "ok": ok,
            "response": response,
        })
        if not ok:
            return {
                "ok": False,
                field_name: base_address,
                "count": count,
                "step": step,
                "values": values,
            }

    return {
        "ok": True,
        field_name: base_address,
        "count": count,
        "step": step,
        "values": values,
    }


@server.tool(description="Read a DWORD array from the current IDA database. Useful for index tables, permutation arrays, and small static transform tables.")
def ida_read_dword_array(base_address: str, count: int, step: int = 4) -> dict[str, Any]:
    return _read_scalar_array("data_read_dword", "base_address", base_address, count, step)


@server.tool(description="Read a BYTE array from the current IDA database. Useful for encoded flag blobs, xor tables, sboxes, and small static byte buffers.")
def ida_read_byte_array(base_address: str, count: int, step: int = 1) -> dict[str, Any]:
    return _read_scalar_array("data_read_byte", "base_address", base_address, count, step)


@server.tool(description="List the JSON-RPC methods exported by the local IDA plugin so the model can choose the right reverse-analysis call.")
def ida_list_rpc_methods() -> dict[str, Any]:
    methods = _parse_plugin_methods()
    return {"ok": True, "count": len(methods), "methods": methods}


@server.tool(description="Call a raw JSON-RPC method exposed by the local IDA bridge. Use params_json as a JSON array or object string, for example [] or {\"address\":\"0x401000\"}.")
def ida_rpc_call(method: str, params_json: str = "[]") -> dict[str, Any]:
    try:
        params = json.loads(params_json)
    except json.JSONDecodeError as exc:
        return {"ok": False, "error": f"invalid params_json: {exc}", "params_json": params_json}

    response = _bridge_request(method, params)
    ok = "error" not in response and "transport_error" not in response and "http_error" not in response
    return {"ok": ok, "method": method, "params": params, "response": response}


if __name__ == "__main__":
    server.run(transport="stdio")
