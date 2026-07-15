# Integration Delivery Lifecycle

Integration delivery must not retry forever without governed disposition.

## Recommended states

```text
PENDING
CLAIMED
RETRY_WAIT
DELIVERED
QUARANTINED
FAILED_TERMINAL
CANCELED
```

## Retry budget

Each contract defines:

- maximum attempts;
- maximum elapsed retry age;
- retry classification;
- bounded delay;
- claim lease;
- operator review conditions.

Either attempt or elapsed-age exhaustion may terminate automatic delivery.

## Quarantine

Quarantine stops automatic delivery while retaining immutable failure history.
Requeue requires governed authority, justification, and a new attempt generation
rather than rewriting prior history.

## Closed reason codes

Use stable reason codes for timeout, unavailability, rejection, invalid payload,
revoked contract, credential rejection, disabled destination, exhausted budget,
and operator cancellation.

## Monitoring

Observe and alert on:

- oldest pending and retry age;
- pending, quarantined, and terminal counts;
- retry rate;
- lease recovery;
- repeated destination rejection;
- contract-specific concentration.

The model remains operation-specific and must not silently become a generic job
framework.
