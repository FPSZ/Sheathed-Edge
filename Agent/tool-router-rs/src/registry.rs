use std::{collections::HashMap, fs, sync::Arc};

use anyhow::{Context, Result};

use crate::{
    config::normalize_runtime_path,
    models::{Registry, ToolDef},
};

pub fn load_registry(path: &str) -> Result<Registry> {
    let data = fs::read_to_string(path).with_context(|| format!("read registry {path}"))?;
    let mut registry: Registry = serde_json::from_str(&data).context("parse registry")?;
    for tool in &mut registry.tools {
        tool.workdir = normalize_runtime_path(&tool.workdir);
        tool.allowed_paths = tool
            .allowed_paths
            .iter()
            .map(|p| normalize_runtime_path(p))
            .collect();
    }
    Ok(registry)
}

pub fn build_tool_map(registry: Registry) -> Result<HashMap<String, ToolDef>> {
    let mut map = HashMap::new();
    for entry in registry.tools {
        let validator = jsonschema::validator_for(&entry.parameter_schema)
            .context("compile parameter schema")?;
        map.insert(
            entry.name.clone(),
            ToolDef {
                entry,
                validator: Arc::new(validator),
            },
        );
    }
    Ok(map)
}
