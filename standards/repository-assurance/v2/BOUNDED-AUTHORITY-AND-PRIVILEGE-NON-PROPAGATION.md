# Bounded Authority and Privilege Non-Propagation

## 1. Architectural invariant

No authentication event, identity, token, session, process, service, worker, API
handler, database connection, migration runner, background task, scheduler,
administrator account, delegated operation, or accumulated privilege set shall
create an **unrestricted execution context** (God Access / God Mode).

Every component shall operate only with the minimum authority explicitly
required for its current operation. Authority shall not automatically propagate
from a caller, user, service, process, session, queue message, upstream token,
administrator, or database principal.

## 2. Required boundary behavior

At every applicable process, service, API, queue, worker, database, module,
administrative, deployment, lifecycle, and trust boundary:

- authentication establishes identity and credential validity;
- authorization is evaluated independently;
- authorization is deny-by-default;
- requested authority is constrained to the current operation;
- delegated authority is constrained to the delegated action and lifetime;
- effective authority is attributable and auditable;
- failure, retry, replay, recovery, and failover do not increase authority.

## 3. Identity separation

Repositories shall design applicable systems so that:

- ordinary user identities are separate from administrative identities;
- service identities are separate from user identities;
- distinct services and workers use independently scoped identities where their
  trust or operation differs;
- CI, release, deployment, migration, runtime, and break-glass identities are
  not casually reused across boundaries;
- shared credentials across unrelated trust boundaries are prohibited.

## 4. Database authority

Application services shall not routinely connect as database owners,
superusers, migration owners, or unrestricted administrative roles.

Database connections shall use operation-appropriate roles. Read, write,
maintenance, migration, replication, reporting, and administrative operations
shall be separated where their authority differs. Connection pooling shall not
cause authority from one operation or tenant to become available to another.

Migration runners shall use explicit, bounded migration authority and shall not
transfer that authority to normal application runtime.

## 5. Workers, queues, and background execution

A queue message or scheduled task is a request for an operation, not a transfer
of the caller's full authority. Workers shall:

- authenticate the message source or accepted ingress boundary;
- validate message integrity and scope;
- re-authorize the requested operation;
- use a worker-specific identity;
- prevent replay and duplicate execution from increasing authority;
- reject forged, stale, malformed, or over-scoped work;
- retain acting principal, delegated scope, decision, and result evidence.

## 6. Role accumulation

Role unions, nested groups, delegated grants, cached claims, inherited database
roles, and combined administrative capabilities shall be evaluated for
accumulated authority. No combination shall silently produce an unrestricted
execution context.

Explicit conflict rules, separation-of-duties constraints, permission ceilings,
or equivalent technical controls shall prevent prohibited accumulation.

## 7. Elevation and break-glass

Privilege elevation and break-glass use shall be:

- explicit and separately authenticated where applicable;
- limited to a defined operation or incident;
- time-bounded;
- attributable to an individual or controlled service identity;
- logged with requested and effective authority;
- monitored;
- revocable;
- reviewed after use.

Break-glass capability shall not become routine administration and shall not
create an unmonitored unrestricted execution context.

## 8. Revocation and session invalidation

Applicable systems shall enforce credential revocation, role removal, session
invalidation, delegated-authority expiry, and break-glass termination. Stale
sessions, cached claims, retries, and queued work shall not retain authority
beyond the accepted revocation model.

## 9. Audit minimum

Audit evidence shall identify, as applicable:

- originating principal;
- acting service or worker identity;
- effective authority and delegated scope;
- requested action and protected resource;
- authorization decision and deciding boundary;
- elevation or break-glass context;
- result, failure, retry, and rollback state;
- correlation identifiers and time.

## 10. Required records

Each new or materially changed authority boundary shall have an authority
boundary record conforming to
`schemas/authority-boundary-record-v1.schema.json`. The record shall identify
minimum authority, prohibited propagation, hostile tests, maturity, evidence,
and acceptance state.
