from __future__ import annotations

import argparse
from pathlib import Path

from reverse_dataset_lib import (
    build_training_row,
    discover_case_bundles,
    dump_json,
    dump_jsonl,
    is_dev_eligible,
    is_train_eligible,
)


def main() -> int:
    parser = argparse.ArgumentParser(description="Build Axolotl-ready reverse SFT datasets.")
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path("Training/axolotl/datasets/reverse_pilot"),
        help="Directory to write train/dev jsonl files into.",
    )
    parser.add_argument(
        "--allow-reviewed",
        action="store_true",
        help="Allow reviewed cases with sft_ready=false for scaffold validation.",
    )
    args = parser.parse_args()

    output_dir = args.output_dir
    if not output_dir.is_absolute():
        output_dir = Path.cwd() / output_dir

    bundles = discover_case_bundles()
    train_rows = []
    dev_rows = []
    skipped = []

    for bundle in bundles:
        case_id = str(bundle.case.get("case_id", "")).strip()
        train_ok = is_train_eligible(bundle, args.allow_reviewed)
        dev_ok = is_dev_eligible(bundle, args.allow_reviewed)
        if train_ok:
            train_rows.append(build_training_row(bundle, "train"))
        if dev_ok:
            dev_rows.append(build_training_row(bundle, "dev"))
        if not train_ok and not dev_ok:
            skipped.append(
                {
                    "case_id": case_id,
                    "reason": "not eligible for train/dev export under current flags",
                }
            )

    dump_jsonl(output_dir / "train.jsonl", train_rows)
    dump_jsonl(output_dir / "dev.jsonl", dev_rows)
    dump_json(
        output_dir / "manifest.json",
        {
            "train_count": len(train_rows),
            "dev_count": len(dev_rows),
            "allow_reviewed": args.allow_reviewed,
            "skipped": skipped,
        },
    )

    print(f"train rows: {len(train_rows)}")
    print(f"dev rows: {len(dev_rows)}")
    print(f"manifest: {output_dir / 'manifest.json'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
