# Conditional approval workflow

Approval is rarely unconditional. A tool is often approved *provided* something
is true by some date: "approved provided PII is masked; re-review in 90 days."
GovGate models these as first-class conditions with due dates, surfaces overdue
ones, and transitions a tool to fully approved once every condition is closed.

## Approving with conditions

`POST /v1/register/{id}/approve-with-conditions` sets the entry to `approved`
and attaches one or more conditions. Each condition has a description and a
`due_in_days`, from which a due date is computed against the service clock.

```json
{
  "reviewer_notes": "ok pending PII masking",
  "conditions": [
    { "description": "Mask PII before send; re-review in 90 days.", "due_in_days": 90 }
  ]
}
```

Returns the updated entry and the created conditions (each with an id and a
computed `due_at`).

## Overdue conditions

`GET /v1/register/overdue` returns every entry that has at least one
unsatisfied condition whose due date has passed, paired with just those overdue
conditions:

```json
{
  "overdue": [
    {
      "entry": { "id": "...", "status": "approved" },
      "overdue_conditions": [
        { "id": "...", "description": "Mask PII...", "due_at": "...", "satisfied": false }
      ]
    }
  ]
}
```

"Overdue" is evaluated against the service clock, which is injectable in tests
so the lifecycle can be verified without sleeping.

## Satisfying a condition

`POST /v1/register/{id}/conditions/{cid}/satisfy` records evidence and closes a
condition:

```json
{ "evidence": "PII masking deployed; ticket OPS-42" }
```

When the last open condition on an entry is satisfied, the entry transitions
from `approved` to `fully-approved`. Until then, satisfying some-but-not-all
conditions leaves the entry `approved`.

## Worked scenario (from the test suite)

1. A tool is submitted and approved with one condition due in 90 days.
2. At day 0 it is not overdue.
3. The clock advances to day 91. `GET /v1/register/overdue` now lists the entry
   and its single overdue condition.
4. The condition is satisfied with evidence. It clears, the entry is no longer
   overdue, and it transitions to `fully-approved`.

A second test confirms that with two conditions, satisfying only one keeps the
entry `approved`; only when both are satisfied does it become `fully-approved`.
