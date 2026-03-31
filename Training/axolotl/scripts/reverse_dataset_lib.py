from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[3]
DATASETS_ROOT = REPO_ROOT / "Datasets" / "reverse"
EVAL_ROOT = REPO_ROOT / "Eval" / "reverse"

CASE_SECTIONS = ("success_cases", "failure_cases", "review_queue")


@dataclass
class ReverseCaseBundle:
    section: str
    directory: Path
    case: dict[str, Any]
    review: dict[str, Any] | None
    skill_delta: dict[str, Any] | None


def load_json(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def discover_case_bundles(root: Path = DATASETS_ROOT) -> list[ReverseCaseBundle]:
    bundles: list[ReverseCaseBundle] = []
    for section in CASE_SECTIONS:
        section_dir = root / section
        if not section_dir.exists():
            continue
        for candidate in sorted(section_dir.iterdir()):
            if not candidate.is_dir():
                continue
            case_path = candidate / "case.json"
            if not case_path.exists():
                continue
            review_path = candidate / "review.json"
            skill_delta_path = candidate / "skill-delta.json"
            bundles.append(
                ReverseCaseBundle(
                    section=section,
                    directory=candidate,
                    case=load_json(case_path),
                    review=load_json(review_path) if review_path.exists() else None,
                    skill_delta=load_json(skill_delta_path) if skill_delta_path.exists() else None,
                )
            )
    return bundles


def discover_eval_records(root: Path = EVAL_ROOT) -> dict[str, list[dict[str, Any]]]:
    records: dict[str, list[dict[str, Any]]] = {"dev": [], "eval": []}
    for split in ("dev", "eval"):
        split_dir = root / split
        if not split_dir.exists():
            continue
        for path in sorted(split_dir.glob("*.json")):
            records[split].append(load_json(path))
    return records


def required_keys(payload: dict[str, Any], keys: list[str], prefix: str) -> list[str]:
    missing: list[str] = []
    for key in keys:
        if key not in payload:
            missing.append(f"{prefix}{key}")
    return missing


def validate_case_bundle(bundle: ReverseCaseBundle) -> list[str]:
    errors: list[str] = []
    case = bundle.case

    errors.extend(
        required_keys(
            case,
            [
                "case_id",
                "task_meta",
                "input_files",
                "student_trace",
                "teacher_trace",
                "error_tags",
                "gold_outcome",
                "skill_delta",
                "sft_ready",
                "audit",
            ],
            "case.",
        )
    )
    if bundle.review is None:
        errors.append("review.json missing")
    if bundle.skill_delta is None:
        errors.append("skill-delta.json missing")

    task_meta = case.get("task_meta", {})
    audit = case.get("audit", {})
    student_trace = case.get("student_trace", {})
    teacher_trace = case.get("teacher_trace", {})

    if task_meta.get("category") != "reverse":
        errors.append("case.task_meta.category must be reverse")
    if not isinstance(case.get("input_files"), list) or not case["input_files"]:
        errors.append("case.input_files must contain at least one item")
    if not isinstance(student_trace.get("tool_events", []), list):
        errors.append("case.student_trace.tool_events must be a list")
    if not isinstance(teacher_trace.get("tool_events", []), list):
        errors.append("case.teacher_trace.tool_events must be a list")
    if not isinstance(case.get("error_tags", []), list):
        errors.append("case.error_tags must be a list")
    if not isinstance(case.get("skill_delta", []), list):
        errors.append("case.skill_delta must be a list")
    if not isinstance(audit.get("can_use_for_train"), bool):
        errors.append("case.audit.can_use_for_train must be a boolean")
    if not isinstance(audit.get("can_use_for_eval"), bool):
        errors.append("case.audit.can_use_for_eval must be a boolean")

    if bundle.review is not None:
        review = bundle.review
        errors.extend(required_keys(review, ["case_id", "student_vs_teacher", "scored_dimensions", "next_actions"], "review."))
        if review.get("case_id") != case.get("case_id"):
            errors.append("review.case_id does not match case.case_id")

    if bundle.skill_delta is not None:
        skill_delta = bundle.skill_delta
        errors.extend(
            required_keys(
                skill_delta,
                ["skill_id", "trigger_case_id", "problem", "rule_update", "checklist_update", "requires_eval_refresh"],
                "skill-delta.",
            )
        )
        if skill_delta.get("trigger_case_id") != case.get("case_id"):
            errors.append("skill-delta.trigger_case_id does not match case.case_id")

    return errors


def validate_eval_record(record: dict[str, Any], split: str, case_index: dict[str, ReverseCaseBundle]) -> list[str]:
    errors: list[str] = []
    errors.extend(required_keys(record, ["case_id", "budget", "result", "notes"], f"{split}."))
    case_id = str(record.get("case_id", "")).strip()
    bundle = case_index.get(case_id)
    if split == "eval" and bundle is not None:
        can_use_for_eval = bool(bundle.case.get("audit", {}).get("can_use_for_eval"))
        if not can_use_for_eval:
            errors.append(f"{split}.case_id={case_id} is not allowed for eval")
    return errors


def is_train_eligible(bundle: ReverseCaseBundle, allow_reviewed: bool) -> bool:
    audit = bundle.case.get("audit", {})
    if not bool(audit.get("can_use_for_train")):
        return False
    if bundle.review is None:
        return False
    return bool(bundle.case.get("sft_ready")) or allow_reviewed


def visibility(case: dict[str, Any]) -> str:
    return str(case.get("task_meta", {}).get("visibility", "")).strip().lower()


def is_dev_eligible(bundle: ReverseCaseBundle, allow_reviewed: bool) -> bool:
    if not is_train_eligible(bundle, allow_reviewed):
        return False
    return visibility(bundle.case) in {"train-dev-only", "dev-only"}


def normalize_text(value: str) -> str:
    return value.replace("\r\n", "\n").strip()


def build_training_row(bundle: ReverseCaseBundle, source_split: str) -> dict[str, Any]:
    case = bundle.case
    review = bundle.review or {}
    case_id = str(case.get("case_id", "")).strip()
    difficulty = str(case.get("task_meta", {}).get("difficulty", "")).strip() or "unknown"
    error_tags = case.get("error_tags", [])
    skill_context = [
        str(item.get("skill_id", "")).strip()
        for item in case.get("skill_delta", [])
        if isinstance(item, dict) and str(item.get("skill_id", "")).strip()
    ]
    teacher_verified = bool(case.get("teacher_trace", {}).get("tool_events"))
    review_status = "reviewed" if bundle.review is not None else "unreviewed"

    user_prompt = normalize_text(
        f"""
        Reverse case review task.

        case_id: {case_id}
        difficulty: {difficulty}
        visibility: {case.get("task_meta", {}).get("visibility", "")}
        source_split: {source_split}

        Input files:
        {format_input_files(case.get("input_files", []))}

        Student trace summary:
        {case.get("student_trace", {}).get("summary", "")}

        Student tool events:
        {format_tool_events(case.get("student_trace", {}).get("tool_events", []))}

        Teacher trace summary:
        {case.get("teacher_trace", {}).get("summary", "")}

        Review findings:
        {format_review_findings(review)}

        Gold outcome:
        {format_gold_outcome(case.get("gold_outcome", {}))}

        Produce the corrected reverse-solving process. Keep it grounded in the available evidence, show the preferred tool sequence, and do not invent a solved flag when the case remains unsolved.
        """
    )

    assistant_response = normalize_text(
        f"""
        # Reverse Process Correction

        case_id: {case_id}
        review_status: {review_status}
        error_tags: {", ".join(error_tags) if error_tags else "none"}
        skill_context: {", ".join(skill_context) if skill_context else "none"}

        ## Corrected Workflow
        {format_teacher_workflow(case.get("teacher_trace", {}).get("tool_events", []))}

        ## What To Avoid
        {format_review_anti_patterns(review)}

        ## Evidence Discipline
        - Do not promote string fragments into a flag without verification.
        - Record exact extracted paths, binary classification hints, and the next verification step.
        - If still unsolved, report the blocker clearly and stop at the highest-signal next action.

        ## Expected Final State
        {format_gold_outcome(case.get("gold_outcome", {}))}
        """
    )

    return {
        "messages": [
            {
                "role": "system",
                "content": "You are a reverse-engineering teacher producing a corrected, evidence-driven solving process for a local offline CTF training case.",
            },
            {"role": "user", "content": user_prompt},
            {"role": "assistant", "content": assistant_response},
        ],
        "metadata": {
            "case_id": case_id,
            "difficulty": difficulty,
            "tags": error_tags,
            "skill_context": skill_context,
            "source_split": source_split,
            "review_status": review_status,
            "sft_ready": bool(case.get("sft_ready")),
            "error_tags": error_tags,
            "teacher_verified": teacher_verified,
        },
    }


def format_input_files(items: list[dict[str, Any]]) -> str:
    lines = []
    for item in items:
        lines.append(
            f"- role={item.get('role', '')}; path={item.get('path', '')}; sha256={item.get('sha256', '')}"
        )
    return "\n".join(lines) if lines else "- none"


def format_tool_events(items: list[dict[str, Any]]) -> str:
    lines = []
    for index, item in enumerate(items, start=1):
        lines.append(
            "\n".join(
                [
                    f"{index}. phase={item.get('phase', '')}",
                    f"   goal={item.get('goal', '')}",
                    f"   chosen_tool={item.get('chosen_tool', '')}",
                    f"   why_this_tool={item.get('why_this_tool', '')}",
                    f"   expected_evidence={item.get('expected_evidence', '')}",
                    f"   observed_result={item.get('observed_result', '')}",
                    f"   fallback_if_fail={item.get('fallback_if_fail', '')}",
                ]
            )
        )
    return "\n".join(lines) if lines else "No tool events."


def format_teacher_workflow(items: list[dict[str, Any]]) -> str:
    if not items:
        return "- No teacher workflow recorded."
    lines = []
    for index, item in enumerate(items, start=1):
        lines.append(
            "\n".join(
                [
                    f"{index}. phase={item.get('phase', '')}",
                    f"   goal={item.get('goal', '')}",
                    f"   chosen_tool={item.get('chosen_tool', '')}",
                    f"   why_this_tool={item.get('why_this_tool', '')}",
                    f"   expected_evidence={item.get('expected_evidence', '')}",
                    f"   fallback_if_fail={item.get('fallback_if_fail', '')}",
                ]
            )
        )
    return "\n".join(lines)


def format_review_findings(review: dict[str, Any]) -> str:
    student_vs_teacher = review.get("student_vs_teacher", {})
    return "\n".join(
        [
            f"missed_steps={student_vs_teacher.get('missed_steps', [])}",
            f"bad_tool_choices={student_vs_teacher.get('bad_tool_choices', [])}",
            f"bad_recovery_points={student_vs_teacher.get('bad_recovery_points', [])}",
            f"evidence_gaps={student_vs_teacher.get('evidence_gaps', [])}",
        ]
    )


def format_review_anti_patterns(review: dict[str, Any]) -> str:
    student_vs_teacher = review.get("student_vs_teacher", {})
    bullets = []
    for item in student_vs_teacher.get("bad_tool_choices", []):
        bullets.append(f"- {item}")
    for item in student_vs_teacher.get("bad_recovery_points", []):
        bullets.append(f"- {item}")
    for item in student_vs_teacher.get("evidence_gaps", []):
        bullets.append(f"- {item}")
    return "\n".join(bullets) if bullets else "- none"


def format_gold_outcome(outcome: dict[str, Any]) -> str:
    status = outcome.get("status", "unknown")
    evidence = outcome.get("evidence", [])
    lines = [f"status={status}"]
    for item in evidence:
        lines.append(f"- {item}")
    return "\n".join(lines)


def dump_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8", newline="\n") as handle:
        json.dump(payload, handle, ensure_ascii=False, indent=2)
        handle.write("\n")


def dump_jsonl(path: Path, rows: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8", newline="\n") as handle:
        for row in rows:
            handle.write(json.dumps(row, ensure_ascii=False))
            handle.write("\n")
