# Hostile Authority Validation

## 1. Applicability

Every new or changed security, authority, trust, lifecycle, or operational
boundary shall be evaluated for hostile-condition testing. The evaluation shall
be recorded even when a specific test class is not applicable.

## 2. Minimum hostile-condition classes

Applicable campaigns shall consider:

- unauthenticated access;
- forged identity, token, signature, or claims;
- horizontal privilege escalation;
- vertical privilege escalation;
- role-union and privilege-accumulation escalation;
- confused-deputy behavior;
- cross-service privilege propagation;
- database-owner or superuser misuse;
- migration-authority leakage into runtime;
- worker, queue, scheduler, and background-task authority expansion;
- replay, duplicate execution, and stale message use;
- malformed, ambiguous, and oversized input;
- race conditions and time-of-check/time-of-use behavior;
- cancellation, timeout, and abandoned work;
- partial failure, retry, failover, rollback, and recovery;
- resource exhaustion and admission-control failure;
- credential, token, or secret exposure;
- stale-session and revoked-authority use;
- break-glass misuse and failure to terminate elevation.

## 3. Required assertions

The campaign shall demonstrate, as applicable, that:

- authorization remains deny-by-default;
- a caller's authority does not become the callee's unrestricted authority;
- failed, duplicated, replayed, or recovered operations do not gain authority;
- role combinations do not create an unrestricted execution context;
- database and migration roles remain bounded;
- revocation and session invalidation take effect within the accepted model;
- audit evidence records attempted and successful abuse paths.

## 4. Evidence separation

Correctness PASS or FAIL is separate from resource observation, performance
budget, security-finding disposition, and operational readiness. Resource
exhaustion tests may generate both correctness and resource evidence, but one
result shall not conceal or relabel the other.

## 5. Acceptance

A hostile test is not accepted merely because a test file exists or ran once.
The exact test, fixture, implementation, environment, identity boundary,
expected result, observed result, and candidate commit shall be retained.

An authority boundary cannot reach `VALIDATED` maturity without applicable
hostile-condition evidence. It cannot reach `ACCEPTED` maturity until the exact
validated boundary receives formal acceptance.
