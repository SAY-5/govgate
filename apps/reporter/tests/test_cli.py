import json
from pathlib import Path

from govgate_reporter.cli import main

REPO_ROOT = Path(__file__).resolve().parents[3]
ACME_ASSESSMENT = REPO_ROOT / "examples" / "assessments" / "acme_chat.json"


def test_render_markdown(capsys, monkeypatch) -> None:  # type: ignore[no-untyped-def]
    monkeypatch.setattr("sys.stdin", ACME_ASSESSMENT.open())
    rc = main(["render", "-", "--format", "md"])
    out = capsys.readouterr().out
    assert rc == 0
    assert "# Assessment: Acme Chat Assistant" in out


def test_render_json_from_path(capsys) -> None:  # type: ignore[no-untyped-def]
    rc = main(["render", str(ACME_ASSESSMENT), "--format", "json"])
    out = capsys.readouterr().out
    assert rc == 0
    payload = json.loads(out)
    assert payload["overall_band"] == "low"


def test_extract_subcommand(capsys, monkeypatch) -> None:  # type: ignore[no-untyped-def]
    import io

    monkeypatch.setattr(
        "sys.stdin",
        io.StringIO("Hosted in eu-west-1, encrypted in transit, retained for 30 days."),
    )
    rc = main(["extract", "-"])
    out = capsys.readouterr().out
    assert rc == 0
    fields = json.loads(out)
    assert fields["data_region"] == "eu-west-1"
    assert fields["retention_days"] == 30
