pub mod reserved;
pub mod search;
pub mod terminal;

use crate::models::{AppState, ExecuteResponse, ToolDef};
use serde_json::Value;

pub async fn dispatch_tool(
    state: &AppState,
    def: &ToolDef,
    arguments: &Value,
) -> std::result::Result<ExecuteResponse, (String, String)> {
    match def.entry.name.as_str() {
        "filesystem/search" => search::run_rg_search(&def.entry, arguments, false).await,
        "retrieval" => search::run_rg_search(&def.entry, arguments, true).await,
        "terminal" => terminal::run_terminal(state, &def.entry, arguments).await,
        _ => reserved::not_implemented(state, def, arguments).await,
    }
}
