# Replay and Idempotency

## Single-process boundary

A process-local bounded replay store may be accepted only when deployment is
explicitly restricted to one process instance with no load balancing, restart
ambiguity, or cross-node retry.

Those restrictions are deployment requirements, not implied assumptions.

## Multi-instance boundary

Before active/passive or active/active deployment, select and accept one or more:

- per-instance keys and instance-bound requests;
- gateway-enforced single delivery;
- shared bounded replay state;
- durable single-use authentication assertions;
- server-issued challenge or monotonic state;
- durable idempotent request records.

## Failure window

The design must address a crash or timeout:

- before protected-state transition;
- after authentication consumption;
- after database commit but before response;
- during failover;
- during gateway retry.

Preferred designs consume the single-use assertion in the same durable
transaction as the protected operation or create a durable idempotent request
record that returns the recorded outcome on retry.

## Required campaigns

- concurrent replay to two nodes;
- restart during replay window;
- gateway timeout and ambiguous retry;
- old/new key overlap;
- revoked key;
- database unavailability and failover;
- capacity exhaustion and cleanup;
- nonce flooding;
- restored replay state.
