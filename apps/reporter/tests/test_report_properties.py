import random

from govgate_reporter.report import _CONDITION_HINTS, build_report

REQ_IDS = list(_CONDITION_HINTS) + ["pii.declared", "vs.vendor_named", "mp.family_known"]
SEVERITIES = ["low", "medium", "high", "critical"]


def _random_assessment(rng: random.Random) -> dict[str, object]:
    """Build an assessment with a random pass/fail assignment over known ids."""
    results = []
    crit = []
    for rid in REQ_IDS:
        outcome = rng.choice(["pass", "fail", "unknown"])
        severity = rng.choice(SEVERITIES)
        results.append(
            {"id": rid, "category": "c", "question": "q", "severity": severity, "outcome": outcome}
        )
        if outcome != "pass" and severity == "critical":
            crit.append(rid)
    return {
        "checklist_name": "default",
        "checklist_version": "1.0.0",
        "overall_band": "high" if crit else "low",
        "overall_score": 0.5,
        "categories": [{"id": "c", "band": "high", "score": 0.5, "results": results}],
        "critical_failures": sorted(crit),
    }


def test_conditions_only_for_failed_requirements() -> None:
    rng = random.Random(7)
    for _ in range(300):
        assessment = _random_assessment(rng)
        passing = {
            r["id"]
            for cat in assessment["categories"]
            for r in cat["results"]
            if r["outcome"] == "pass"
        }
        report = build_report(assessment)
        # Every recommended condition must trace to a failed requirement.
        for rid, hint in _CONDITION_HINTS.items():
            if hint in report.recommended_conditions:
                assert rid not in passing, f"condition for passing requirement {rid}"


def test_conditions_are_unique() -> None:
    rng = random.Random(11)
    for _ in range(100):
        report = build_report(_random_assessment(rng))
        conds = report.recommended_conditions
        assert len(conds) == len(set(conds))
