# Requirement extraction

Some checklist auto-checks read structured fields (data region, retention,
whether PII is processed, and so on). A submitter can fill these in by hand, but
they can also be extracted from the free-text description. The Python reporter
owns extraction.

## Provider protocol

`Provider.extract(description) -> Extraction` is the single seam. Two
implementations ship:

- **`KeywordExtractor`** — a deterministic, transparent rule set. It is the
  default and requires no network access.
- **`FakeProvider`** — returns scripted extractions, used in tests so the
  extraction pipeline (and the report pipeline built on it) stays hermetic in
  CI. A real hosted-LLM provider would slot in behind the same protocol.

## Extraction shape

`Extraction` mirrors the Go `scoring.StructuredFields` shape. Optional booleans
are `None` when the description does not state the fact, which is distinct from a
stated `False`. `to_structured_fields()` renders the JSON the Go scoring engine
consumes.

| Field | Type | Source heuristic |
|-------|------|------------------|
| `data_region` | str? | first `aa-word-9` token |
| `processes_pii` | bool? | mentions of PII / personal data / anonymization |
| `pii_masked_before_send` | bool? | mask / unmasked language |
| `model_family` | str? | known family tokens (llama-style, gpt-style, ...) |
| `retention_days` | int? | "retained for N days" / "retention ... N days" |
| `encryption_in_transit` | bool? | encrypt + transit language |
| `trains_on_customer_data` | bool? | training-on-customer-data language |
| `subprocessors` / `unvetted_subprocessors` | list | subprocessor / unknown-third-party mentions |

## Why a Fake instead of mocking

Scripted extractions keep the report golden tests stable and free of any LLM
dependency. The keyword extractor provides a second, independently verifiable
path so the protocol is exercised by real logic as well as by scripts.
