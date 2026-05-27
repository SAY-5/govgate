# Scoring algorithm

Scoring is deterministic: a given submission and checklist always produce the
same assessment.

## Requirement outcomes

Each requirement resolves to one of:

- `pass` — satisfied.
- `fail` — not satisfied.
- `unknown` — insufficient information. Treated as a failure for scoring, but
  surfaced separately so reviewers can request more information.

A requirement is resolved by its `auto_check` if it has one, otherwise by a
human/LLM judgment supplied at assessment time. A missing judgment is `unknown`.

## Category score

For each category, the score is the weighted fraction of satisfied requirements:

```
category_score = sum(weight of passing reqs) / sum(weight of all reqs)
```

This is always in `[0, 1]`.

## Overall score

The overall score is the category-weight-weighted mean of category scores:

```
overall_score = sum(category_weight * category_score) / sum(category_weight)
```

Also always in `[0, 1]`.

## Bands

A satisfaction score maps to a risk band (higher satisfaction, lower risk):

| Score | Band |
|-------|------|
| `>= 0.90` | low |
| `>= 0.70` | medium |
| `>= 0.40` | high |
| `< 0.40` | critical |

## Critical-severity gate

The band derived from the numeric score is only half the story. Any failed
requirement with `critical` severity caps the overall band at `high` or worse,
regardless of how high the weighted score is. This is the load-bearing rule:
one unmasked-PII failure or one unencrypted-transit failure cannot be averaged
away by strong scores elsewhere.

```
overall_band = max(band(overall_score), critical_gate_band)
```

where `critical_gate_band` is `high` if any critical requirement failed, else
`low`.

The decision table below summarizes the gate for a two-requirement checklist
(one critical, one noncritical):

| critical | noncritical | overall band |
|----------|-------------|--------------|
| pass | pass | low |
| pass | fail | depends on score (>= medium) |
| fail | pass | >= high |
| fail | fail | >= high |

## Auto-checks

Recognized `auto_check` keys:

| Key | Passes when |
|-----|-------------|
| `data_region_present` | a data region is declared |
| `data_region_approved` | the declared region is in the approved set |
| `pii_flag_declared` | the submission states whether PII is processed |
| `pii_not_unmasked` | PII is not processed, or is masked before send |
| `model_family_known` | the model family is disclosed |
| `retention_declared` | a retention period is declared |
| `retention_bounded` | retention is 90 days or fewer |
| `encryption_in_transit` | data is encrypted in transit |
| `subprocessors_listed` | at least one subprocessor is enumerated |
| `no_customer_training` | customer data is excluded from training |
| `vendor_named` | the submission names a vendor |
