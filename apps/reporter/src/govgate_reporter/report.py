"""Assessment report rendering.

Given an assessment (the JSON shape the Go scoring engine emits, optionally
wrapped in a register entry), produce a structured report as JSON and as
Markdown. Rendering is deterministic so reports can be golden-tested.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from typing import Any

# Recommended approval conditions, keyed by failed-requirement id. When a
# requirement in this map is among the criticals or category failures, the
# corresponding condition is surfaced in the report.
_CONDITION_HINTS: dict[str, str] = {
    "pii.no_unmasked": "Mask or remove PII before any data leaves the network.",
    "sec.encryption_transit": "Enable encryption in transit for all data flows.",
    "dr.approved_region": "Move processing to an approved data-residency region.",
    "mp.no_customer_training": "Contractually exclude customer data from model training.",
    "ho.human_in_loop": "Add human review of consequential outputs.",
    "ret.bounded": "Reduce the retention window or document a justification.",
}


@dataclass
class Report:
    """A rendered assessment report."""

    tool_name: str
    vendor: str
    checklist_name: str
    checklist_version: str
    overall_band: str
    overall_score: float
    categories: list[dict[str, Any]] = field(default_factory=list)
    critical_failures: list[str] = field(default_factory=list)
    recommended_conditions: list[str] = field(default_factory=list)

    def to_dict(self) -> dict[str, Any]:
        return {
            "tool_name": self.tool_name,
            "vendor": self.vendor,
            "checklist_name": self.checklist_name,
            "checklist_version": self.checklist_version,
            "overall_band": self.overall_band,
            "overall_score": round(self.overall_score, 4),
            "categories": self.categories,
            "critical_failures": self.critical_failures,
            "recommended_conditions": self.recommended_conditions,
        }

    def to_json(self) -> str:
        return json.dumps(self.to_dict(), indent=2, sort_keys=True)

    def to_markdown(self) -> str:
        lines: list[str] = []
        lines.append(f"# Assessment: {self.tool_name}")
        lines.append("")
        lines.append(f"- **Vendor:** {self.vendor}")
        lines.append(f"- **Checklist:** {self.checklist_name} v{self.checklist_version}")
        lines.append(f"- **Overall band:** {self.overall_band}")
        lines.append(f"- **Overall score:** {self.overall_score:.2f}")
        lines.append("")

        lines.append("## Checklist outcomes")
        lines.append("")
        lines.append("| Category | Band | Score | Requirement | Severity | Outcome |")
        lines.append("|----------|------|-------|-------------|----------|---------|")
        for cat in self.categories:
            results = cat.get("results", [])
            if not results:
                lines.append(
                    f"| {cat['id']} | {cat['band']} | {cat['score']:.2f} | - | - | - |"
                )
                continue
            for i, r in enumerate(results):
                cat_id = cat["id"] if i == 0 else ""
                band = cat["band"] if i == 0 else ""
                score = f"{cat['score']:.2f}" if i == 0 else ""
                lines.append(
                    f"| {cat_id} | {band} | {score} | {r['id']} "
                    f"| {r['severity']} | {r['outcome']} |"
                )
        lines.append("")

        lines.append("## Critical failures")
        lines.append("")
        if self.critical_failures:
            for c in self.critical_failures:
                lines.append(f"- `{c}`")
        else:
            lines.append("None.")
        lines.append("")

        lines.append("## Recommended conditions for approval")
        lines.append("")
        if self.recommended_conditions:
            for cond in self.recommended_conditions:
                lines.append(f"- {cond}")
        else:
            lines.append("No conditions required.")
        lines.append("")
        return "\n".join(lines)


def build_report(entry: dict[str, Any]) -> Report:
    """Build a report from a register entry or a bare assessment payload.

    Accepts either a full register entry (with ``submission`` and ``assessment``
    keys) or a bare assessment object.
    """
    if "assessment" in entry:
        submission = entry.get("submission", {})
        assessment = entry["assessment"]
    else:
        submission = {}
        assessment = entry

    categories = assessment.get("categories") or []
    conditions = _recommend_conditions(assessment, categories)

    return Report(
        tool_name=submission.get("name", "(unnamed)"),
        vendor=submission.get("vendor", "(unknown)"),
        checklist_name=assessment.get("checklist_name", ""),
        checklist_version=assessment.get("checklist_version", ""),
        overall_band=assessment.get("overall_band", ""),
        overall_score=float(assessment.get("overall_score", 0.0)),
        categories=categories,
        critical_failures=list(assessment.get("critical_failures") or []),
        recommended_conditions=conditions,
    )


def _recommend_conditions(
    assessment: dict[str, Any], categories: list[dict[str, Any]]
) -> list[str]:
    """Collect approval conditions for every failed requirement that maps to a hint."""
    failed_ids: set[str] = set(assessment.get("critical_failures") or [])
    for cat in categories:
        for r in cat.get("results", []):
            if r.get("outcome") != "pass":
                failed_ids.add(r["id"])
    conditions = [_CONDITION_HINTS[rid] for rid in sorted(failed_ids) if rid in _CONDITION_HINTS]
    return conditions
