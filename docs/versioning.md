# Checklist versioning and re-assessment

Policy is not static. As an organization's risk posture changes, the checklist
changes with it: requirements get added, severities get promoted, thresholds get
tightened. GovGate treats checklists as versioned documents and keeps the full
audit trail of how each tool's risk changed across those versions.

## Versions

Every checklist carries a `version`. When a tool is assessed, the register
entry records the checklist name and the exact version used.

## Publishing a new version

When a new version of a checklist is published, every entry that was assessed
under an older version of the *same* checklist is flagged `stale_assessment`.
The flag is a signal, not a re-score: the original assessment is preserved
verbatim until a reviewer chooses to re-assess.

```
flagged = PublishChecklist(ctx, newChecklist)   // returns count flagged stale
```

`MarkStale` is a single indexed UPDATE: `checklist_name = $1 AND
checklist_version <> $2`.

## Re-assessment

`POST /v1/register/{id}/reassess` re-runs the *current* version of the entry's
checklist against the original submission, preserving the original human/LLM
judgments (reconstructed from the prior assessment's non-auto outcomes). It:

1. records the new assessment as the entry's live assessment,
2. appends it to the entry's assessment history,
3. clears the stale flag, and
4. returns a diff against the prior assessment.

```json
{
  "entry": { "...": "updated entry" },
  "diff": {
    "from_version": "1.0.0",
    "to_version": "2.0.0",
    "from_band": "low",
    "to_band": "critical",
    "band_rose": true,
    "new_criticals": ["mp.no_customer_training"]
  }
}
```

## Audit trail

`GET /v1/register/{id}/history` returns every assessment a tool has received,
oldest first. Each record carries the checklist version it was produced under,
so the sequence is a verifiable record of how a tool's risk moved as policy
evolved. History rows are written in the same transaction as the entry update,
so an entry and its history can never diverge.

## Worked scenario (from the test suite)

1. A tool that trains on customer data is submitted under checklist `govern`
   v1.0.0, which has no training requirement. It scores **low**.
2. `govern` v2.0.0 is published, adding a `critical` requirement that customer
   data be excluded from training. The entry is flagged stale.
3. The entry is re-assessed. The new critical requirement fails, the
   critical-severity gate engages, and the band rises **low to critical**.
4. Both assessments remain in history, and the diff records the rise and the new
   critical (`mp.no_customer_training`).
