use std::{
    fs::OpenOptions,
    io::Write,
    path::Path,
    time::{SystemTime, UNIX_EPOCH},
};

use serde_json::json;

use crate::models::{AppState, ExecuteRequest, ExecuteResponse};

pub fn append_tool_log(state: &AppState, req: &ExecuteRequest, resp: &ExecuteResponse, duration_ms: u128) {
    let log_path = Path::new(&state.config.logs.tool_log_dir)
        .join(format!("{}.jsonl", chrono_like_date()));
    let entry = json!({
        "time": now_ms(),
        "session_id": req.session_id,
        "mode": req.mode,
        "tool": req.tool,
        "arguments": req.arguments,
        "duration_ms": duration_ms,
        "ok": resp.ok,
        "summary": resp.summary,
        "error": resp.error
    });

    if let Ok(mut file) = OpenOptions::new().create(true).append(true).open(log_path) {
        let _ = writeln!(file, "{}", entry);
    }
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
