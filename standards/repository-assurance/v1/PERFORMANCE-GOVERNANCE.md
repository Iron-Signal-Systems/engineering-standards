# Performance Governance

## Progression

```text
OBSERVED
  ↓
BASELINE_ESTABLISHED
  ↓
PROVISIONAL_BUDGET
  ↓
GOVERNED_BUDGET
  ↓
ACCEPTED_CHANGE
```

Do not invent thresholds before representative runs exist.

## Workload profiles

Projects define representative small, medium, degraded, recovery, backlog,
burst, reconnect, failover, and destination-outage profiles as applicable.

## Metrics

Examples:

- API and operation p50, p95, and p99 latency;
- connection and worker saturation;
- lock and queue wait;
- queue age and throughput;
- retry growth;
- CPU and memory;
- disk and database I/O;
- WAL or journal growth;
- storage growth;
- reconnect and live-update delay;
- recovery time.

Correctness and performance remain separate outcomes.
