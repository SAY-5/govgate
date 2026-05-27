"""LLM provider abstraction for requirement extraction.

The real system would call a hosted LLM to read a free-text tool description and
return structured facts. To keep tests and CI hermetic, ``FakeProvider`` returns
scripted extractions keyed by the input text, and ``KeywordProvider`` applies a
deterministic rule set. No network access is required by either.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Protocol


@dataclass
class Extraction:
    """Structured facts pulled from a free-text description.

    Fields mirror the Go ``scoring.StructuredFields`` shape. ``None`` means the
    description did not state the fact, which is distinct from a stated ``False``.
    """

    data_region: str | None = None
    approved_regions: list[str] = field(default_factory=list)
    processes_pii: bool | None = None
    pii_masked_before_send: bool | None = None
    model_family: str | None = None
    retention_days: int | None = None
    encryption_in_transit: bool | None = None
    subprocessors: list[str] = field(default_factory=list)
    unvetted_subprocessors: list[str] = field(default_factory=list)
    trains_on_customer_data: bool | None = None

    def to_structured_fields(self) -> dict[str, object]:
        """Render as the JSON shape the Go scoring engine consumes."""
        return {
            "data_region": self.data_region or "",
            "approved_regions": self.approved_regions,
            "processes_pii": self.processes_pii,
            "pii_masked_before_send": self.pii_masked_before_send,
            "model_family": self.model_family or "",
            "retention_days": self.retention_days,
            "encryption_in_transit": self.encryption_in_transit,
            "subprocessors": self.subprocessors,
            "unvetted_subprocessors": self.unvetted_subprocessors,
            "trains_on_customer_data": self.trains_on_customer_data,
        }


class Provider(Protocol):
    """A source of structured extractions for a free-text description."""

    def extract(self, description: str) -> Extraction:  # pragma: no cover - protocol
        ...


class FakeProvider:
    """Returns scripted extractions, matched by exact text or substring.

    Used in tests so the extraction pipeline can be exercised without an LLM.
    Register scripts with :meth:`script`; the first substring match wins, and an
    exact match takes priority over substring matches.
    """

    def __init__(self) -> None:
        self._exact: dict[str, Extraction] = {}
        self._substr: list[tuple[str, Extraction]] = []

    def script(self, text: str, extraction: Extraction, *, exact: bool = False) -> None:
        """Register an extraction to return for the given text."""
        if exact:
            self._exact[text] = extraction
        else:
            self._substr.append((text, extraction))

    def extract(self, description: str) -> Extraction:
        if description in self._exact:
            return self._exact[description]
        for needle, extraction in self._substr:
            if needle in description:
                return extraction
        return Extraction()
