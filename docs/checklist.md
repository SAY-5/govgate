# Checklist format

A checklist is a versioned YAML policy document. It groups weighted,
severity-tagged requirements into categories.

## Top level

| Field | Type | Notes |
|-------|------|-------|
| `version` | string | Required. Semantic version of the policy. |
| `name` | string | Required. Unique within a checklist directory. |
| `description` | string | Optional. |
| `categories` | list | Required, non-empty. |

## Category

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | Required, unique. e.g. `pii_handling`. |
| `weight` | number | Required, `> 0`. Relative contribution to the overall band. |
| `requirements` | list | Required, non-empty. |

## Requirement

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | Required, unique across the whole checklist. |
| `question` | string | Human-readable prompt. |
| `weight` | number | Required, `> 0`. Relative contribution within its category. |
| `severity` | enum | `low`, `medium`, `high`, `critical`. |
| `auto_check` | string | Optional. Name of an auto-check evaluated from submission fields. |

## Auto-checks

A requirement with an `auto_check` key is evaluated mechanically from the
submission's structured fields (see [extraction.md](extraction.md) for how those
fields are produced). Requirements without an `auto_check` need a human or
LLM judgment supplied at assessment time. The recognized auto-check keys are
listed in [scoring.md](scoring.md).

## Validation

`checklist.Load` enforces: non-empty `version`/`name`, at least one category,
positive weights, recognized severities, and globally unique category and
requirement identifiers. Invalid documents fail closed.

## Shipped checklists

- `checklists/default.yaml` — baseline intake policy.
- `checklists/strict.yaml` — hardened policy for high-sensitivity environments;
  promotes several requirements to `critical` and adds a no-customer-training
  gate.
