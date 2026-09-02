# Iron Signal Systems Security and Hardening Standard

## Purpose

This document defines how Iron Signal Systems approaches software security and hardening.

Security is a required part of ISS engineering.

It is not a final review performed after the architecture and implementation are already complete.

Security decisions begin with design and continue through implementation, testing, deployment, operation, maintenance, updates, and recovery.

The objective is not to satisfy a security checklist.

The objective is to understand what the system can do, what systems, data, services, and resources it can access, what authority it possesses, what it trusts, what can influence it, how it can fail, and whether those behaviors and boundaries can be independently verified.

---

## Security Is Architecture

Security boundaries should be visible in the design.

Important security decisions include:

- identity;
- authentication;
- authorization;
- privilege;
- filesystem access;
- network exposure;
- service accounts;
- credentials;
- cryptographic keys;
- certificates;
- trust relationships;
- data protection;
- command execution;
- update mechanisms;
- recovery authority;
- logging;
- persistent state.

Do not hide important security behavior behind generic abstractions merely to simplify architecture.

An engineer should be able to determine where authority enters the system and where it can be exercised.

---

## Least Authority

Every component should operate with the minimum authority necessary for its defined responsibility.

Do not grant administrative authority merely because doing so makes implementation easier.

For each privileged component, understand:

- what identity it uses;
- what permissions it requires;
- what resources it can access;
- what it can modify;
- what network destinations it can reach;
- what credentials it can obtain;
- what administrative APIs it can invoke.

Where practical, separate privileged operations from ordinary operations.

A read-only responsibility should not quietly require write authority.

---

## Separate Authority by Responsibility

Different responsibilities should not automatically share the same authority.

Where practical, separate:

- collection;
- analysis;
- administration;
- remediation;
- recovery;
- update;
- replication;
- credential management.

A component compromised in one role should not automatically inherit every other administrative capability in the system.

Security boundaries should reflect real responsibilities.

---

## Authentication and Authorization Are Different

Authentication establishes identity.

Authorization determines what that identity may do.

Do not treat successful authentication as unlimited authority.

Authorization decisions should be deliberate and appropriate to the requested operation.

Higher-impact actions should require stronger authorization where appropriate.

---

## Trust Must Be Explicit

Important trust relationships must be understandable.

Identify what the system trusts, including applicable:

- users;
- administrators;
- hosts;
- services;
- certificates;
- keys;
- configuration;
- repositories;
- update sources;
- network peers;
- identity providers;
- operating-system facilities.

Then consider what happens if that trusted entity becomes:

- compromised;
- unavailable;
- stale;
- misconfigured;
- replaced;
- malicious.

Trust should exist because the architecture requires it, not simply because two systems are inside the same network.

---

## Internal Does Not Mean Trusted

Do not automatically trust input because it originated:

- on the local machine;
- inside the organization;
- on a management network;
- from another ISS component;
- from an authenticated user.

Trust depends on the actual security boundary.

Internal systems can fail or become compromised.

Validate accordingly.

---

## Validate Untrusted Input

Input that crosses a trust boundary must be validated.

This includes applicable:

- network input;
- files;
- configuration;
- command output;
- API responses;
- serialized state;
- database values;
- device responses;
- operating-system records;
- user input.

Validation should consider:

- size;
- format;
- range;
- encoding;
- structure;
- unexpected values;
- malformed input;
- hostile input.

Do not assume external input will remain well formed simply because the normal producer behaves correctly.

---

## Parsers Are Security Boundaries

Parsers transform untrusted input into trusted internal structures.

Treat them accordingly.

Where applicable, parsers should:

- enforce input limits;
- reject malformed structures safely;
- avoid uncontrolled recursion;
- avoid uncontrolled memory growth;
- preserve uncertainty;
- avoid executing interpreted input;
- distinguish incomplete from complete input.

Parser failures must not create unintended authority or unsafe state.

---

## Fail Closed Where Authority Depends on Verification

When a security decision cannot be established, do not silently grant authority.

Examples include:

- authentication cannot be completed;
- authorization cannot be determined;
- a certificate cannot be validated;
- a signature is invalid;
- trusted configuration is corrupt;
- an update cannot be verified;
- a destructive target is ambiguous.

Failing closed does not mean every system error must terminate the entire application.

The failure behavior should preserve the safer state appropriate to the operation.

---

## Do Not Silently Downgrade Security

When a security mechanism fails, do not automatically replace it with a weaker mechanism.

Examples include:

- disabling certificate verification;
- falling back from authenticated communication to unauthenticated communication;
- bypassing authorization;
- reducing encryption requirements;
- changing from protected storage to plaintext;
- substituting administrative credentials when a limited identity fails.

Any security fallback must be explicitly designed, justified, visible, and tested.

---

## Secrets Must Be Protected

Secrets include applicable:

- passwords;
- API keys;
- private keys;
- recovery credentials;
- bearer tokens;
- session tokens;
- encryption keys;
- signing keys.

Secrets should not be:

- committed to source control;
- embedded in binaries unnecessarily;
- printed in normal logs;
- exposed in command output;
- stored in plaintext when stronger platform facilities are practical.

Use appropriate operating-system or platform secret protection where possible.

---

## Credentials Should Be Scoped

Credentials should provide only the authority required for their role.

Avoid shared high-privilege credentials when a purpose-specific identity can be used.

Prefer managed identities and platform-native service identities where appropriate.

Credential lifetime, rotation, revocation, and recovery should be considered as part of the design.

---

## Protect Cryptographic Keys

Cryptographic keys deserve stronger protection than ordinary configuration.

Consider:

- storage;
- access control;
- generation;
- rotation;
- backup;
- recovery;
- revocation;
- destruction.

Private keys should not be exposed to components that do not require them.

Where practical, use platform or hardware-backed protection appropriate to the threat model.

---

## Do Not Invent Cryptography

ISS projects should use established cryptographic algorithms, protocols, libraries, and constructions.

Do not design custom:

- encryption algorithms;
- password hashing schemes;
- signature systems;
- key exchange protocols;
- authentication protocols;

when established solutions exist.

Cryptography should use maintained implementations appropriate to the platform and requirement.

---

## Encryption Must Have a Purpose

Encryption should protect a defined confidentiality or integrity requirement.

Understand:

- what is encrypted;
- where encryption begins and ends;
- where keys reside;
- who can decrypt;
- what metadata remains visible;
- how recovery works.

Encrypting data while leaving the relevant key accessible to every compromised component may not meaningfully improve the security boundary.

---

## Protect Data in Transit

Sensitive or privileged communication should use authenticated and encrypted transport appropriate to the requirement.

Do not rely solely on network location for confidentiality or identity.

Where certificates or equivalent trust mechanisms are used, validation must not be silently bypassed.

---

## Protect Sensitive Data at Rest

Sensitive persistent data should receive protection appropriate to its value and threat model.

This may include:

- operating-system access control;
- encryption;
- restricted service identities;
- hardware-backed protection;
- integrity validation.

Do not collect or retain sensitive information merely because it might become useful someday.

Data that is never stored cannot later be stolen from storage.

---

## Minimize Network Exposure

A service should listen only where necessary.

Consider:

- required interfaces;
- required ports;
- expected peers;
- authentication;
- authorization;
- firewall boundaries;
- whether remote access is needed at all.

Do not create network services merely for implementation convenience.

A local mechanism may be safer when remote access is unnecessary.

---

## Outbound Network Access Matters Too

Security design must consider what the application can reach, not only what can reach the application.

Unrestricted outbound connectivity can increase the effect of compromise.

Where practical, understand and document required external communication.

Unexpected network communication should be treated as a meaningful behavior change.

---

## Command Execution Is a Security Boundary

Code that executes external commands or processes requires deliberate review.

Consider:

- command construction;
- argument handling;
- shell invocation;
- inherited environment;
- working directory;
- executable resolution;
- privileges;
- user-controlled input;
- output handling.

Avoid invoking a shell when direct process execution is sufficient.

Never concatenate untrusted input into command strings without appropriate protection.

---

## Filesystem Boundaries Matter

Filesystem access should be deliberate.

Understand:

- what directories are read;
- what directories are written;
- ownership;
- access-control lists or permissions;
- symbolic links or reparse points;
- temporary files;
- file replacement behavior;
- path traversal;
- race conditions.

A component should not unexpectedly cross filesystem boundaries merely because a path exists.

---

## Temporary Files Need Security Too

Temporary data may contain sensitive information or influence privileged behavior.

Use appropriate:

- locations;
- permissions;
- naming;
- lifecycle;
- cleanup.

Do not assume temporary means harmless.

---

## Persistent State Must Be Protected

Persistent state may influence future privileged behavior.

Protect state according to its authority.

Where relevant, consider:

- tampering;
- corruption;
- rollback;
- stale state;
- partial writes;
- unauthorized modification.

A component should not blindly trust persistent state merely because it created the file previously.

---

## Updates Are Privileged Operations

An update changes trusted executable code.

Update mechanisms must therefore be treated as high-authority paths.

Where applicable, updates should consider:

- source authenticity;
- package integrity;
- signatures;
- version identity;
- rollback;
- compatibility;
- configuration migration;
- state migration;
- privilege changes.

An attacker should not gain easier privileged execution through the updater than through the application itself.

---

## Dependencies Are Part of the Attack Surface

Third-party dependencies introduce code that ISS must trust.

Before adding significant dependencies, consider:

- maintenance status;
- security history;
- licensing;
- update practices;
- transitive dependencies;
- runtime behavior;
- network behavior.

Dependencies must be updated when security or compatibility requires it.

Do not keep vulnerable dependencies merely to avoid development work.

---

## Vulnerability Scanning Is Useful, Not Sufficient

Use appropriate tooling to identify known vulnerable components where practical.

A clean vulnerability scan does not prove that the software is secure.

Scanning cannot replace:

- design review;
- privilege analysis;
- threat analysis;
- security testing;
- platform validation.

Likewise, a scanner finding must be understood rather than blindly accepted or ignored.

---

## Static Analysis Is Useful

Use language-appropriate and platform-appropriate static analysis where it provides meaningful value.

Static analysis can identify classes of defects.

It cannot establish complete system security.

Do not add dozens of overlapping tools merely to produce more reports.

Use tools that help engineers find real problems.

---

## Security Hardening Is Mandatory

Before operational use, ISS projects must deliberately review how the system can be hardened.

Applicable hardening may include:

- removing unnecessary privileges;
- removing unnecessary services;
- restricting network exposure;
- tightening filesystem permissions;
- protecting configuration;
- securing service identities;
- disabling unnecessary features;
- reducing writable paths;
- limiting outbound communication;
- protecting logs and state;
- validating update mechanisms.

Hardening should reflect the actual system.

There is no universal ISS hardening checklist that replaces understanding the project.

---

## System-Level Hardening

Application security should consider whether the underlying operating system or host can be hardened to reduce risk.

Depending on the system, useful hardening may include:

- reducing installed or enabled services;
- restricting network exposure;
- limiting administrative access;
- tightening filesystem or device permissions;
- restricting outbound communication;
- applying operating-system security controls;
- isolating service identities;
- restricting executable or writable locations;
- enabling appropriate auditing and logging;
- using platform security features appropriate to the threat model.

Host-level hardening must be appropriate to the environment.

ISS software may operate on dedicated systems where substantial host hardening is practical.

It may also operate on shared systems where other applications, administrators, operational requirements, or vendor dependencies prevent ISS from controlling the complete host configuration.

The project must not assume that recommended host hardening is present unless it has been verified.

When host-level hardening is optional or cannot reasonably be required, application security must not depend on that hardening being present.

Where additional host hardening materially reduces risk but cannot be required, document it as a recommended defense rather than pretending it is an enforced security boundary.

Security controls should distinguish between:

- protections enforced by the ISS application;
- protections enforced by the operating system;
- protections expected from deployment configuration;
- optional hardening that further reduces risk.

---

## Defaults Should Be Secure

Default behavior should not unnecessarily weaken security.

Avoid defaults that:

- grant administrative authority;
- expose remote services;
- disable verification;
- enable destructive actions;
- permit unrestricted access.

Where usability and security trade off, make the decision deliberate and visible.

---

## Configuration Cannot Quietly Disable Security

Security-relevant configuration must be understandable.

If a setting weakens an important security control, that effect should be obvious.

Do not hide security downgrades behind innocuous option names.

Invalid security configuration should fail visibly.

---

## Logging Must Not Leak Secrets

Security and operational logging must avoid unnecessary disclosure.

Do not log:

- passwords;
- private keys;
- bearer tokens;
- recovery credentials;
- sensitive authentication material.

Where identifiers or values must be logged for troubleshooting, consider redaction or safe representation.

---

## Security-Relevant Actions Should Be Visible

Where appropriate, record meaningful security events such as:

- authentication failures;
- authorization failures;
- privileged actions;
- configuration security changes;
- update verification failures;
- recovery authorization;
- destructive administrative actions.

Logging should help an operator understand what occurred.

Do not generate enormous security logs that hide the important event in noise.

---

## Logs Must Match Their Value

If logs are used to investigate security-relevant behavior, protect them accordingly.

Consider:

- write access;
- deletion;
- retention;
- remote forwarding;
- tamper resistance;
- storage growth.

Do not claim logs provide strong audit assurance if an ordinary compromised process can freely rewrite them.

---

## Recovery Security Matters

Recovery often requires elevated authority.

Treat recovery mechanisms as security-sensitive architecture.

Consider:

- who may initiate recovery;
- how identity is established;
- what authority recovery grants;
- whether ordinary compromised credentials can trigger it;
- how temporary authority expires;
- how recovery actions are logged.

Recovery must not become an undocumented bypass around normal security controls.

---

## Emergency Access Must Be Controlled

Some systems may require emergency or break-glass access.

Where used, emergency access should be:

- deliberate;
- narrowly scoped;
- strongly authenticated;
- logged;
- temporary where practical;
- independently reviewable.

Emergency mechanisms should not become the normal administrative path.

---

## Destructive Operations Need Strong Authorization

Operations that:

- delete;
- overwrite;
- revoke;
- restore;
- disable;
- reconfigure;
- modify security;
- change identity;

require security treatment appropriate to their impact.

Where practical:

- verify the target;
- verify authorization;
- make the action explicit;
- record the action;
- provide recovery or rollback when feasible.

---

## Protect Administrative Interfaces

Administrative interfaces have greater impact than ordinary user interfaces.

Where applicable, protect them through:

- strong authentication;
- explicit authorization;
- restricted network exposure;
- session protection;
- audit logging;
- appropriate rate limits.

Do not expose administrative capabilities through interfaces designed for ordinary read-only operations.

---

## Multi-Factor Authentication

Where the threat model and operational environment justify stronger authentication, ISS systems should support or require multi-factor authentication.

MFA should protect meaningful authority rather than merely adding another prompt to low-risk operations.

Recovery from lost MFA credentials must also be deliberately designed.

---

## Security Testing Is Required

Important security boundaries must be tested.

Applicable tests may verify:

- unauthorized access is denied;
- read-only identities cannot write;
- malformed input is rejected;
- invalid certificates fail;
- invalid signatures fail;
- privilege boundaries hold;
- secrets are not exposed;
- destructive actions require proper authorization;
- security failures do not silently downgrade.

Security controls should be demonstrated, not merely described.

---

## Test Both Allowed and Denied Behavior

A security test should not only show that an authorized action succeeds.

Where practical, also verify that an unauthorized action fails.

For example:

```text
Authorized collector identity:
    Read succeeds.

Unauthorized identity:
    Read fails.

Collector identity:
    Write fails.
```

That establishes much more about the actual security boundary than a successful access test alone.

---

## Security Validation Should Be User-Verifiable

Where practical, ISS should provide procedures allowing administrators to verify important security claims themselves.

Examples may include:

- filesystem permissions;
- service identities;
- privileges;
- listening ports;
- certificate identity;
- process authority;
- configured trust;
- read/write boundaries.

A customer should not need to accept a vendor statement that a control exists when the control can safely be inspected.

---

## Security Validation Must Be Safe

Security validation procedures must clearly identify when they:

- require privilege;
- change configuration;
- create accounts;
- modify permissions;
- alter policy;
- generate network traffic;
- perform destructive actions.

Prefer read-only validation where it can prove the required condition.

---

## Threat Analysis Should Match the System

ISS projects should consider realistic threats appropriate to their role.

Do not create enormous theoretical threat models for trivial components.

Do not ignore obvious threats merely because a formal threat-model document has not been created.

Ask practical questions:

```text
What can an unauthenticated user do?

What can an ordinary authenticated user do?

What can the service identity do?

What happens if the service is compromised?

What happens if configuration is modified?

What happens if a trusted peer is compromised?

What happens if an administrator makes a mistake?

What happens if a credential is stolen?

What happens if an attacker controls the input?
```

The project should understand the answers.

---

## Root and Administrator Are Powerful

Do not design security claims that assume an operating-system superuser or equivalent administrator cannot control the operating system.

Root, SYSTEM, kernel-level authority, or equivalent administrative control may be able to:

- read or modify application memory;
- replace executable code;
- modify configuration;
- alter filesystem permissions;
- inspect or change application state;
- interfere with processes;
- modify operating-system security controls;
- observe credentials or keys available to the host;
- alter logging or auditing available to that host.

Application-level protections may still make compromise more difficult, limit accidental access, reduce the authority of ordinary processes, or provide valuable defense in depth.

Appropriate system-level hardening may further reduce the likelihood or impact of compromise.

However, hardening must not be represented as stronger isolation than it actually provides.

On a dedicated system, ISS may be able to recommend or require substantial operating-system hardening.

On a shared system, some hardening may be impractical because the host must also support other applications, services, administrators, or operational requirements.

The security design must state those limitations truthfully.

Where protection from the host administrator, root, SYSTEM, kernel compromise, or equivalent authority is actually required, the security boundary must be placed outside that authority domain where practical.

That may require mechanisms such as:

- a separate host;
- an isolated service;
- hardware-backed protection;
- an independent trust root;
- another independently administered system.

Logical separation inside a single fully controlled host should not be described as protection from the authority that controls that host.

---

## Separation Must Be Real

A security boundary implemented entirely within the same fully compromised authority domain may not provide meaningful isolation.

If the threat model requires protection from the host administrator, consider architecture outside that administrator's control, such as:

- separate hosts;
- hardware-backed facilities;
- independent trust roots;
- isolated services.

Do not describe logical separation as strong isolation when the same authority can rewrite both sides.

---

## Security Claims Must Be Precise

Avoid claims such as:

- secure;
- tamper-proof;
- impossible to bypass;
- zero trust;
- military-grade;
- unhackable.

Describe the actual property instead.

For example:

```text
The service identity has read access to this directory and does not have write access under the tested ACL configuration.
```

That is useful and verifiable.

---

## Unsupported Security States Must Be Visible

If a required security control cannot be established, the system should report that condition.

Do not silently treat an unsupported security configuration as fully protected.

Where appropriate, distinguish between:

- protected;
- partially protected;
- unverified;
- unsupported;
- failed.

Use project-appropriate terminology.

Do not force one universal security-state schema across ISS software.

---

## Security Fixes Require Regression Testing

When a security defect can reasonably be reproduced, retain a regression test for the corrected condition.

The test should demonstrate that the unsafe behavior no longer occurs.

Where appropriate, also verify that legitimate behavior continues to work.

Security history should improve the project's future resilience rather than disappearing into old release notes.

---

## Platform Security Must Be Tested on the Platform

Do not assume platform security behavior.

Where ISS depends on native:

- access-control lists;
- Unix permissions;
- service identities;
- capabilities;
- security tokens;
- filesystem protections;
- kernel behavior;
- authentication facilities;

validate the actual behavior on the supported platform.

Mocks do not prove platform security boundaries.

---

## Security and Compatibility Must Be Balanced Deliberately

A platform update must not automatically cause ISS to weaken security simply to remain compatible.

When compatibility and security conflict:

1. understand what changed;
2. identify the affected security property;
3. determine whether a safe correction exists;
4. test the correction;
5. document any remaining limitation.

Do not quietly trade security for compatibility.

---

## Security Documentation Should Be Useful

Document security information that helps engineers and operators understand the system.

Useful information may include:

- required privileges;
- service identities;
- trust boundaries;
- network communication;
- credential handling;
- key storage;
- hardening guidance;
- recovery authority;
- known limitations.

Do not create security documentation solely to satisfy a process requirement.

---

## Before Operational Use

Before a meaningful capability is considered ready for operational use, engineers should be able to answer where applicable:

```text
What identity does it run as?

What privileges does it require?

What can it read?

What can it modify?

What network access does it require?

What trusts it?

What does it trust?

What credentials or keys does it hold?

How are those protected?

What happens if the component is compromised?

What happens if authentication fails?

What happens if authorization cannot be determined?

What security boundaries were tested?

What hardening was performed?

What security behavior can the administrator verify?

What remains unverified?
```

Not every project requires a formal document answering every question.

The project must nevertheless understand the answers.

---

## Final Principle

Security should not depend on customers trusting that ISS probably implemented the right controls.

Important security properties should be understandable, testable, and inspectable wherever practical.

ISS should be able to say:

> This is the authority the system requires.  
> This is why it requires it.  
> This is what it can and cannot do.  
> This is what it can access.  
> This is what protects the boundary.  
> This is how we tested it.  
> This is what remains outside that boundary.  
> And this is how you can verify the important parts yourself.

Security earns trust through understandable design, limited authority, deliberate hardening, meaningful testing, and visible behavior.
