"""Thin CLI for the GovGate reporter.

Reads a register entry or assessment JSON (from a file or stdin) and writes a
report. Two subcommands:

    govgate-report render <entry.json> [--format json|md]
    govgate-report extract <description.txt>   # structured fields as JSON
"""

from __future__ import annotations

import argparse
import json
import sys
from collections.abc import Sequence

from govgate_reporter.extract import KeywordExtractor
from govgate_reporter.report import build_report


def main(argv: Sequence[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="govgate-report")
    sub = parser.add_subparsers(dest="cmd", required=True)

    p_render = sub.add_parser("render", help="render an assessment report")
    p_render.add_argument("path", nargs="?", default="-", help="entry JSON file, or - for stdin")
    p_render.add_argument("--format", choices=["json", "md"], default="md")

    p_extract = sub.add_parser("extract", help="extract structured fields from a description")
    p_extract.add_argument("path", nargs="?", default="-", help="text file, or - for stdin")

    args = parser.parse_args(argv)

    if args.cmd == "render":
        entry = json.loads(_read(args.path))
        report = build_report(entry)
        out = report.to_json() if args.format == "json" else report.to_markdown()
        print(out)
        return 0

    if args.cmd == "extract":
        fields = KeywordExtractor().extract(_read(args.path)).to_structured_fields()
        print(json.dumps(fields, indent=2, sort_keys=True))
        return 0

    parser.error("unknown command")
    return 2


def _read(path: str) -> str:
    if path == "-":
        return sys.stdin.read()
    with open(path, encoding="utf-8") as fh:
        return fh.read()


if __name__ == "__main__":
    raise SystemExit(main())
