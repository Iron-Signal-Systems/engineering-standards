# Threat Modeling and Abuse Cases

## Required scope

Threat modeling covers:

- assets and protected outcomes;
- actors and service identities;
- trust boundaries;
- data flows;
- entry points;
- authority and privilege;
- dependencies;
- failure and degraded modes;
- abuse cases;
- detection and response;
- residual risk.

## Change triggers

Review the threat model when a change adds or alters:

- a listener, route, protocol, or external integration;
- authentication, authorization, approval, or session behavior;
- a database role, migration, privilege, or controlled operation;
- a credential, key, certificate, or secret lifecycle;
- a worker, queue, retry, or scheduler;
- a deployment identity or trust boundary;
- a specialized lab or self-hosted runner;
- protected data or evidence retention;
- release or update authority.

## Validation linkage

Each material threat or abuse case identifies:

- preventive control;
- detective control;
- hostile test;
- accepted evidence;
- residual risk and non-claim.
