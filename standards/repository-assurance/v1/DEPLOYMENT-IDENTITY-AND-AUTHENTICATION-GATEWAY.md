# Deployment Identity and Authentication Gateway

Projects with authenticated services must freeze the complete production trust
path rather than validating an isolated API alone.

## Required decisions

- authenticating gateway identity and ownership;
- external identity-provider and directory integration;
- local Unix-domain-socket or network mTLS boundary;
- node and service identity;
- key and certificate generation, staging, activation, overlap, rotation,
  revocation, compromise, destruction, and audit;
- trusted time and clock-failure behavior;
- gateway-to-service availability and degraded behavior;
- audit and correlation records;
- deployment authorization.

## Message handoff

A normalized authentication handoff contains authentication and request-binding
information only. It must not become a caller-supplied authorization result.

Typical fields include:

- version and key identifier;
- gateway and intended audience;
- method, route, and body digest;
- request and correlation identifiers;
- immutable subject and source-directory identifiers;
- authentication mechanism and time;
- issuance, expiration, nonce, and session binding;
- signature.

## Transport

Co-located components may use a permission-controlled Unix-domain socket with
peer validation and signed messages.

Network-separated components require mutually authenticated encrypted transport
plus bounded signed message verification.

## Active Directory

AD establishes identity and account state. Application policy, purpose, scope,
resource, approval, and decision state remain governed by the application.
