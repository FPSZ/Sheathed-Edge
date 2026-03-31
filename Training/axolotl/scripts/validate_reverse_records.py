from __future__ import annotations

import argparse
import sys

from reverse_dataset_lib import discover_case_bundles, discover_eval_records, validate_case_bundle, validate_eval_record


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate reverse training and eval records.")
    args = parser.parse_args()
    _ = args

    bundles = discover_case_bundles()
    eval_records = discover_eval_records()

    errors: list[str] = []
    seen_case_ids: dict[str, str] = {}
    case_index = {}

    for bundle in bundles:
        case_id = str(bundle.case.get("case_id", "")).strip()
        if not case_id:
            errors.append(f"{bundle.directory}: empty case_id")
            continue
        if case_id in seen_case_ids:
            errors.append(f"duplicate case_id: {case_id} ({seen_case_ids[case_id]} and {bundle.directory})")
        else:
            seen_case_ids[case_id] = str(bundle.directory)
        case_index[case_id] = bundle

        for error in validate_case_bundle(bundle):
            errors.append(f"{bundle.directory.name}: {error}")

    for split, records in eval_records.items():
        seen_eval_ids: set[str] = set()
        for record in records:
            case_id = str(record.get("case_id", "")).strip()
            if case_id in seen_eval_ids:
                errors.append(f"{split}: duplicate case_id {case_id}")
            else:
                seen_eval_ids.add(case_id)
            for error in validate_eval_record(record, split, case_index):
                errors.append(error)

    print(f"case bundles: {len(bundles)}")
    print(f"dev eval records: {len(eval_records['dev'])}")
    print(f"held-out eval records: {len(eval_records['eval'])}")

    if errors:
        print("validation failed:")
        for item in errors:
            print(f"- {item}")
        return 1

    print("validation ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
