# GovGate

AI tool intake and risk assessment automation.

GovGate is a pipeline for governing the adoption of AI tools inside an
organization. A team submits a tool (name, vendor, data-handling description,
intended use). GovGate evaluates the submission against a configurable
requirements checklist (data residency, PII handling, model provenance,
retention, human oversight, security, vendor stability), scores risk per
category and overall, generates a structured assessment report (JSON and
Markdown), and maintains a queryable register of every reviewed tool with its
status (pending / approved / rejected / needs-info).

## Why

Most governance tooling checks operating-system or infrastructure posture.
GovGate sits a layer up: it governs the *intake* of AI tools themselves, the
risk that a given vendor and data flow introduces, and the audit trail of how
that risk changes as policy evolves.

## Architecture

```
govgate/
├── apps/
│   ├── register/                 # Go: REST API + checklist engine + Postgres
│   │   ├── cmd/govgate/main.go
│   │   ├── internal/{checklist,scoring,register,store,api}/
│   │   └── *_test.go
│   └── reporter/                 # Python: report generation + LLM extraction
│       ├── src/govgate_reporter/{extract,report,provider}.py
│       └── tests/
├── checklists/{default.yaml, strict.yaml}
├── examples/submissions/*.json
├── docs/
├── docker-compose.yml
└── .github/workflows/ci.yml
```

The split is deliberate:

- **Go** powers the high-throughput register API and the deterministic
  checklist scoring engine, backed by Postgres.
- **Python** powers report generation and the LLM-assisted requirement
  extraction (a `FakeProvider` is used in tests so CI stays hermetic).

## Components

| Component | Language | Responsibility |
|-----------|----------|----------------|
| Checklist engine | Go | Load versioned YAML checklists, run auto-checks |
| Scoring engine | Go | Weighted per-category scores, critical-severity gate |
| Register | Go | Postgres-backed store of every assessment, REST API |
| Extractor | Python | Pull structured fields from free-text descriptions |
| Reporter | Python | Render JSON + Markdown assessment reports |

## Risk model

- Each requirement has a `weight` and a `severity` (low / medium / high / critical).
- A category score is the weighted fraction of satisfied requirements in that category.
- The overall band is the weighted aggregate, gated by severity: a single
  failed `critical` requirement caps the overall band at `high` or worse.
- Bands: `low`, `medium`, `high`, `critical`.

See [docs/scoring.md](docs/scoring.md) for the exact algorithm.

## Quick start

```bash
# Run the full stack locally
docker-compose up --build

# Submit a tool and fetch its assessment
make run-go
curl -X POST localhost:8080/v1/submissions -d @examples/submissions/acme_chat.json
curl localhost:8080/v1/register
```

## Development

```bash
make test          # all tests
make test-go       # Go unit + integration (testcontainers Postgres)
make test-py       # Python unit tests
make lint          # go vet + ruff
make typecheck     # go vet + mypy strict
make bench-regress # scoring + register throughput regression gate
```

## Documentation

- [Checklist format](docs/checklist.md)
- [Scoring algorithm](docs/scoring.md)
- [Register API](docs/register.md)
- [Requirement extraction](docs/extraction.md)
- [Checklist versioning and re-assessment](docs/versioning.md)
- [Conditional approval workflow](docs/conditional-approval.md)

## License

MIT
