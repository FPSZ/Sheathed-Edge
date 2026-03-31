from __future__ import annotations

import argparse
from pathlib import Path
from typing import Any


BASE_TEMPLATE = {
    "base_model": "Qwen/Qwen2.5-7B-Instruct",
    "chat_template": "tokenizer_default",
    "adapter": "qlora",
    "load_in_4bit": True,
    "strict": False,
    "sequence_len": 8192,
    "sample_packing": False,
    "pad_to_sequence_len": True,
    "train_on_inputs": False,
    "group_by_length": False,
    "micro_batch_size": 1,
    "gradient_accumulation_steps": 8,
    "num_epochs": 1,
    "max_steps": 20,
    "learning_rate": 0.00002,
    "optimizer": "adamw_bnb_8bit",
    "lr_scheduler": "cosine",
    "gradient_checkpointing": True,
    "lora_r": 32,
    "lora_alpha": 64,
    "lora_dropout": 0.05,
    "lora_target_linear": True,
    "save_steps": 25,
    "logging_steps": 1,
}

PROFILE_OVERRIDES = {
    "reverse-dry-run": {
        "max_steps": 2,
        "save_steps": 2,
        "logging_steps": 1,
        "warmup_steps": 0,
    },
    "reverse-small-sft": {
        "max_steps": 40,
        "save_steps": 20,
        "logging_steps": 2,
        "warmup_steps": 4,
    },
    "reverse-formal-sft": {
        "max_steps": 200,
        "num_epochs": 2,
        "save_steps": 50,
        "logging_steps": 5,
        "warmup_steps": 10,
    },
}


def dump_yaml(value: Any, indent: int = 0) -> str:
    prefix = " " * indent
    if isinstance(value, dict):
        lines: list[str] = []
        for key, item in value.items():
            if isinstance(item, (dict, list)):
                lines.append(f"{prefix}{key}:")
                lines.append(dump_yaml(item, indent + 2))
            else:
                lines.append(f"{prefix}{key}: {format_scalar(item)}")
        return "\n".join(lines)
    if isinstance(value, list):
        lines = []
        for item in value:
            if isinstance(item, (dict, list)):
                lines.append(f"{prefix}-")
                lines.append(dump_yaml(item, indent + 2))
            else:
                lines.append(f"{prefix}- {format_scalar(item)}")
        return "\n".join(lines)
    return f"{prefix}{format_scalar(value)}"


def format_scalar(value: Any) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, (int, float)):
        return str(value)
    text = str(value)
    escaped = text.replace("\\", "\\\\").replace('"', '\\"')
    return f'"{escaped}"'


def main() -> int:
    parser = argparse.ArgumentParser(description="Render a runnable Axolotl config from profile defaults.")
    parser.add_argument("--profile", choices=sorted(PROFILE_OVERRIDES), required=True)
    parser.add_argument("--train-file", type=Path, required=True)
    parser.add_argument("--val-file", type=Path, default=None)
    parser.add_argument("--base-model", default=BASE_TEMPLATE["base_model"])
    parser.add_argument("--output-dir", type=Path, required=True)
    parser.add_argument("--save-path", type=Path, required=True)
    parser.add_argument("--prepared-path", type=Path, default=None)
    args = parser.parse_args()

    config = dict(BASE_TEMPLATE)
    config.update(PROFILE_OVERRIDES[args.profile])
    config["base_model"] = args.base_model
    config["output_dir"] = str(args.output_dir).replace("\\", "/")
    config["dataset_prepared_path"] = str(
        (args.prepared_path or (args.output_dir / "prepared_dataset"))
    ).replace("\\", "/")
    config["datasets"] = [
        {
            "path": str(args.train_file).replace("\\", "/"),
            "type": "chat_template",
            "field_messages": "messages",
        }
    ]
    if args.val_file:
        config["test_datasets"] = [
            {
                "path": str(args.val_file).replace("\\", "/"),
                "type": "chat_template",
                "field_messages": "messages",
            }
        ]

    save_path = args.save_path
    if not save_path.is_absolute():
        save_path = Path.cwd() / save_path
    save_path.parent.mkdir(parents=True, exist_ok=True)
    save_path.write_text(dump_yaml(config) + "\n", encoding="utf-8", newline="\n")
    print(save_path)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
