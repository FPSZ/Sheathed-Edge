use std::{
    fs::OpenOptions,
    io::Write,
    path::Path,
    time::{SystemTime, UNIX_EPOCH},
};

use serde_json::Value;
use serde_json::json;

use crate::models::{AppState, ExecuteRequest, ExecuteResponse};

pub fn append_tool_log(
    state: &AppState,
    req: &ExecuteRequest,
    arguments: &Value,
    resp: &ExecuteResponse,
    duration_ms: u128,
) {
    let log_path =
        Path::new(&state.config.logs.tool_log_dir).join(format!("{}.jsonl", chrono_like_date()));
    let entry = json!({
        "time": now_ms(),
        "session_id": req.session_id,
        "mode": req.mode,
        "user_email": user_email_for_log(req, arguments),
        "tool": req.tool,
        "target_id": result_string(resp, "target_id"),
        "transport": result_string(resp, "transport"),
        "host_id": result_string(resp, "host_id"),
        "remote_shell": result_string(resp, "remote_shell"),
        "connection_reused": result_bool(resp, "connection_reused"),
        "failure_phase": result_string(resp, "failure_phase"),
        "queue_wait_ms": result_u64(resp, "queue_wait_ms"),
        "arguments": arguments,
        "duration_ms": duration_ms,
        "ok": resp.ok,
        "summary": resp.summary,
        "error": resp.error
    });

    if let Ok(mut file) = OpenOptions::new().create(true).append(true).open(log_path) {
        let _ = writeln!(file, "{}", entry);
    }
}

fn user_email_for_log(req: &ExecuteRequest, arguments: &Value) -> String {
    if !req.user_email.trim().is_empty() {
        return req.user_email.trim().to_string();
    }
    arguments
        .get("user_email")
        .and_then(Value::as_str)
        .unwrap_or_default()
        .trim()
        .to_string()
}

fn result_string(resp: &ExecuteResponse, key: &str) -> String {
    resp.result
        .get(key)
        .and_then(Value::as_str)
        .unwrap_or_default()
        .to_string()
}

fn result_bool(resp: &ExecuteResponse, key: &str) -> bool {
    resp.result
        .get(key)
        .and_then(Value::as_bool)
        .unwrap_or(false)
}

fn result_u64(resp: &ExecuteResponse, key: &str) -> u64 {
    resp.result.get(key).and_then(Value::as_u64).unwrap_or(0)
}

pub fn now_ms() -> u128 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis()
}

fn chrono_like_date() -> String {
    let secs = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs();
    let days = secs / 86_400;
    let z = days as i64 + 719468;
    let era = if z >= 0 { z } else { z - 146096 } / 146097;
    let doe = z - era * 146097;
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let m = mp + if mp < 10 { 3 } else { -9 };
    let year = y + if m <= 2 { 1 } else { 0 };
    format!("{year:04}-{m:02}-{d:02}")
}
