# Worked examples and the end-to-end flow

The repository ships three example submissions that exercise the full pipeline
and span the risk-band space. Their assessments and rendered reports are
committed as golden fixtures.

| Example | Overall band | Why |
|---------|--------------|-----|
| `acme_chat` | low | approved region, masked PII, bounded retention, no customer-data training |
| `globex_vision` | critical | unmasked PII (critical gate), out-of-region, 365-day retention, trains on customer data |
| `initech_summarizer` | low | anonymized inputs, approved region, short retention, human review |

## Files

- `examples/submissions/*.json` — the inputs (submission plus reviewer judgments).
- `examples/assessments/*.json` — the scored register entries (golden).
- `examples/reports/*.md` and `*.report.json` — the rendered reports (golden).

## Reproducing the flow offline

Score a submission with the Go engine (no Postgres needed):

```bash
cd apps/register
go run ./cmd/govgate assess \
  --checklist ../../checklists/default.yaml \
  --input ../../examples/submissions/globex_vision.json > entry.json
```

Render the report with the Python reporter:

```bash
cd apps/reporter
poetry run govgate-report render ../../entry.json --format md
poetry run govgate-report render ../../entry.json --format json
```

Extract structured fields from free text:

```bash
echo "Hosted in eu-west-1, encrypted in transit, retained for 30 days." \
  | poetry run govgate-report extract -
```

## Reproducing over the wire

`docker-compose up --build` starts Postgres and the register service. Then:

```bash
curl -X POST localhost:8080/v1/submissions \
  -d @examples/submissions/acme_chat.json
curl 'localhost:8080/v1/register?band=critical'
```

The integration test `TestEndToEndExamples` performs exactly this flow against a
throwaway Postgres and asserts each example lands in its expected band.
