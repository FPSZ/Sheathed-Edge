use std::{collections::HashMap, fs};

use anyhow::{Context, Result, anyhow};

use crate::models::{Config, LlamaServerConfig, ModelProfile, ModelProfilesFile};

pub fn load_config(path: &str) -> Result<Config> {
    let data = fs::read_to_string(path).with_context(|| format!("read config: {path}"))?;
    let cfg: Config =
        serde_json::from_str(&data).with_context(|| format!("parse config: {path}"))?;
    Ok(cfg)
}

pub fn load_llama_config(path: &str) -> Result<LlamaServerConfig> {
    let data = fs::read_to_string(path).with_context(|| format!("read llama config: {path}"))?;
    let cfg: LlamaServerConfig =
        serde_json::from_str(&data).with_context(|| format!("parse llama config: {path}"))?;
    Ok(cfg)
}

pub fn load_profiles(path: &str) -> Result<HashMap<String, ModelProfile>> {
    let parsed = load_profiles_file(path)?;
    let mut out = HashMap::new();
    for profile in parsed {
        out.insert(profile.id.clone(), profile);
    }
    Ok(out)
}

pub fn load_profiles_file(path: &str) -> Result<Vec<ModelProfile>> {
    let data = fs::read_to_string(path).with_context(|| format!("read model profiles: {path}"))?;
    let parsed: ModelProfilesFile =
        serde_json::from_str(&data).with_context(|| format!("parse model profiles: {path}"))?;
    Ok(parsed.profiles)
}

pub fn save_profiles_file(path: &str, profiles: &[ModelProfile]) -> Result<()> {
    let payload = serde_json::to_string_pretty(&ModelProfilesFile {
        profiles: profiles.to_vec(),
    })
    .context("serialize model profiles")?;
    fs::write(path, format!("{payload}\n")).with_context(|| format!("write model profiles: {path}"))
}

pub fn validate_default_profile(
    profiles: &HashMap<String, ModelProfile>,
    default_profile_id: &str,
) -> Result<()> {
    let profile = profiles
        .get(default_profile_id)
        .ok_or_else(|| anyhow!("default profile not found: {default_profile_id}"))?;
    if !profile.enabled {
        return Err(anyhow!("default profile is disabled: {default_profile_id}"));
    }
    Ok(())
}
