import json
from pathlib import Path

import pytest
from govgate_reporter.report import build_report

# examples/ lives at the repo root, three levels up from this test file.
REPO_ROOT = Path(__file__).resolve().parents[3]
ASSESSMENTS = REPO_ROOT / "examples" / "assessments"
REPORTS = REPO_ROOT / "examples" / "reports"

CASES = ["acme_chat", "globex_vision", "initech_summarizer"]


def _load(name: str) -> dict[str, object]:
    return json.loads((ASSESSMENTS / name).read_text())


@pytest.mark.parametrize("name", CASES)
def test_markdown_matches_golden(name: str) -> None:
    entry = _load(f"{name}.json")
    rendered = build_report(entry).to_markdown()
    golden = (REPORTS / f"{name}.md").read_text()
    assert rendered == golden


@pytest.mark.parametrize("name", CASES)
def test_json_matches_golden(name: str) -> None:
    entry = _load(f"{name}.json")
    rendered = build_report(entry).to_json()
    golden = (REPORTS / f"{name}.report.json").read_text()
    # to_json() does not emit a trailing newline; the golden file may.
    assert rendered.strip() == golden.strip()


def test_critical_example_surfaces_failures_and_conditions() -> None:
    report = build_report(_load("globex_vision.json"))
    assert report.overall_band == "critical"
    assert "pii.no_unmasked" in report.critical_failures
    assert report.recommended_conditions  # non-empty
    assert any("PII" in c for c in report.recommended_conditions)


def test_clean_example_has_no_conditions() -> None:
    report = build_report(_load("acme_chat.json"))
    assert report.overall_band == "low"
    assert report.critical_failures == []
    assert report.recommended_conditions == []


def test_build_report_accepts_bare_assessment() -> None:
    entry = _load("acme_chat.json")
    bare = entry["assessment"]
    report = build_report(bare)
    assert report.overall_band == "low"
    assert report.tool_name == "(unnamed)"  # no submission wrapper
