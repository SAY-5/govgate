"""Deterministic keyword extractor for tool descriptions.

``KeywordExtractor`` implements the :class:`~govgate_reporter.provider.Provider`
protocol with a transparent rule set. It is the default extractor when no hosted
LLM is configured, and it doubles as a baseline the FakeProvider scripts can be
compared against in tests.
"""

from __future__ import annotations

import re

from govgate_reporter.provider import Extraction

# Region tokens of the form aa-word-9 (e.g. eu-west-1, us-east-1).
_REGION_RE = re.compile(r"\b([a-z]{2}-[a-z]+-\d)\b")
_RETENTION_RE = re.compile(r"retain(?:ed|s|ing)?[^.\d]*(\d+)\s*day", re.IGNORECASE)
_RETENTION_ALT_RE = re.compile(r"retention[^.\d]*(\d+)\s*day", re.IGNORECASE)

_MODEL_FAMILIES = ("llama-style", "gpt-style", "claude-style", "mistral-style")


class KeywordExtractor:
    """Extracts structured fields from free text using keyword heuristics."""

    def __init__(self, approved_regions: list[str] | None = None) -> None:
        self.approved_regions = approved_regions or ["eu-west-1", "eu-central-1"]

    def extract(self, description: str) -> Extraction:
        text = description.lower()
        ex = Extraction(approved_regions=list(self.approved_regions))

        ex.data_region = self._region(text)
        ex.processes_pii = self._pii(text)
        ex.pii_masked_before_send = self._masked(text)
        ex.model_family = self._model_family(text)
        ex.retention_days = self._retention(text)
        ex.encryption_in_transit = self._encryption(text)
        ex.trains_on_customer_data = self._training(text)
        ex.subprocessors, ex.unvetted_subprocessors = self._subprocessors(text)
        return ex

    @staticmethod
    def _region(text: str) -> str | None:
        m = _REGION_RE.search(text)
        return m.group(1) if m else None

    @staticmethod
    def _pii(text: str) -> bool | None:
        if "no pii" in text or "not process pii" in text or "pii is not" in text:
            return False
        if "anonymiz" in text and "pii" not in text:
            return False
        if "pii" in text or "personal data" in text:
            return True
        return None

    @staticmethod
    def _masked(text: str) -> bool | None:
        if "mask" in text:
            return "not mask" not in text and "unmask" not in text
        if "unmasked" in text:
            return False
        return None

    @staticmethod
    def _model_family(text: str) -> str | None:
        for fam in _MODEL_FAMILIES:
            if fam in text:
                return fam
        if "undisclosed" in text or "not disclosed" in text:
            return None
        return None

    @staticmethod
    def _retention(text: str) -> int | None:
        m = _RETENTION_RE.search(text) or _RETENTION_ALT_RE.search(text)
        return int(m.group(1)) if m else None

    @staticmethod
    def _encryption(text: str) -> bool | None:
        if "not encrypt" in text or "no encryption" in text:
            return False
        if "encrypt" in text and "transit" in text:
            return True
        if "encrypted in transit" in text:
            return True
        return None

    @staticmethod
    def _training(text: str) -> bool | None:
        negatives = ("does not train", "not train on customer", "no training on customer")
        if any(n in text for n in negatives):
            return False
        if "trains on customer" in text or "train on customer data" in text:
            return True
        return None

    @staticmethod
    def _subprocessors(text: str) -> tuple[list[str], list[str]]:
        subs: list[str] = []
        unvetted: list[str] = []
        if "unknown" in text and ("cdn" in text or "subprocessor" in text or "third part" in text):
            unvetted.append("unknown-third-party")
            subs.append("unknown-third-party")
        if "subprocessor" in text or "hosting provider" in text or "cloud" in text:
            subs.append("declared-subprocessor")
        # de-duplicate while preserving order
        seen: set[str] = set()
        deduped: list[str] = []
        for s in subs:
            if s not in seen:
                seen.add(s)
                deduped.append(s)
        return deduped, unvetted
