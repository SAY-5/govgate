# Benchmarks and regression gate

GovGate ships microbenchmarks for the scoring engine and the register store, and
a throughput regression gate wired into CI.

## What is measured

| Benchmark | Tag | Measures |
|-----------|-----|----------|
| `BenchmarkScore` | unit | scoring time per submission |
| `BenchmarkInsert` | integration | register write throughput (assessments/sec) |
| `BenchmarkListAtScale` | integration | filtered+paginated query latency P50/P95/P99 |

Run them:

```bash
make bench                                   # all benchmarks
go test -bench=BenchmarkScore ./internal/scoring/
go test -tags=integration -bench=. ./internal/store/   # needs Docker
```

## The regression gate

`make bench-regress` (or `go run ./cmd/govgate benchregress`) times scoring
throughput and compares it against the committed baseline in
`apps/register/bench/baseline.json`. It fails if throughput regresses by more
than the threshold (default 30%):

```bash
go run ./cmd/govgate benchregress --threshold 0.30 \
  --checklist ../../checklists/default.yaml --baseline bench/baseline.json
```

Update the baseline deliberately after an intended performance change:

```bash
go run ./cmd/govgate benchregress --update --baseline bench/baseline.json
```

The committed baseline is set conservatively so the gate is stable across the
range of hardware CI runs on; it is a floor, not a target.

## Reference numbers

Measured on an Apple-silicon laptop (indicative only):

- Scoring: ~360 ns per submission, 7 allocs/op (roughly 600k assessments/sec).
- Register insert: ~350 us/op against a containerized Postgres.
- Register list at 5k rows: P50 ~1.3 ms, P95 ~1.7 ms, P99 ~2.3 ms.

CI runs a smoke pass of every benchmark (`-benchtime=500x`) plus the regression
gate, so a performance cliff fails the build rather than landing silently.
