# Assessment: Globex Vision API

- **Vendor:** Globex
- **Checklist:** default v1.0.0
- **Overall band:** critical
- **Overall score:** 0.28

## Checklist outcomes

| Category | Band | Score | Requirement | Severity | Outcome |
|----------|------|-------|-------------|----------|---------|
| data_residency | critical | 0.33 | dr.region_known | medium | pass |
|  |  |  | dr.approved_region | high | fail |
| pii_handling | critical | 0.17 | pii.declared | medium | pass |
|  |  |  | pii.no_unmasked | critical | fail |
|  |  |  | pii.dpa_signed | high | fail |
| model_provenance | critical | 0.00 | mp.family_known | medium | fail |
|  |  |  | mp.training_data | high | fail |
| retention | critical | 0.33 | ret.policy_declared | medium | pass |
|  |  |  | ret.bounded | high | fail |
| human_oversight | critical | 0.00 | ho.human_in_loop | high | fail |
|  |  |  | ho.appeal_path | medium | fail |
| security | high | 0.60 | sec.encryption_transit | high | pass |
|  |  |  | sec.subprocessors_known | medium | pass |
|  |  |  | sec.no_unvetted_subproc | high | fail |
| vendor_stability | high | 0.50 | vs.vendor_named | low | pass |
|  |  |  | vs.support_terms | low | unknown |

## Critical failures

- `pii.no_unmasked`

## Recommended conditions for approval

- Move processing to an approved data-residency region.
- Add human review of consequential outputs.
- Mask or remove PII before any data leaves the network.
- Reduce the retention window or document a justification.
