# Iron Signal Systems Design Standard

## Purpose

This document defines how Iron Signal Systems approaches software and system design.

The goal is not to require a particular architecture, diagram format, design document, programming language, framework, or deployment model.

The goal is to ensure that important engineering decisions are understood before complexity becomes difficult to remove.

A design should make it possible to explain:

- what the system does;
- why the system exists;
- what it depends on;
- what depends on it;
- what authority it has;
- what it trusts;
- what trusts it;
- what data enters and leaves it;
- what it changes;
- what happens when it fails;
- how it recovers;
- how its important behavior will be tested and verified.

A design that cannot be explained clearly is not ready to become complicated code.

---

## Design for the Actual Requirement

Design the system that is needed.

Do not build speculative frameworks, plugin systems, abstraction layers, generalized engines, compatibility layers, or extension mechanisms solely because they may be useful someday.

Future flexibility has a cost.

That cost may include:

- additional code;
- additional dependencies;
- additional security boundaries;
- additional testing;
- additional failure modes;
- additional operational complexity;
- additional maintenance;
- additional assumptions.

A design should accept that cost only when there is a demonstrated requirement for it.

Do not solve hypothetical future requirements at the expense of making the current system harder to understand, test, secure, operate, recover, or maintain.

---

## Start With the System Boundary

Before implementing a substantial component, identify its boundary.

At minimum, understand:

- what responsibility belongs inside the component;
- what responsibility belongs outside it;
- what inputs it accepts;
- what outputs it produces;
- what state it owns;
- what external systems it communicates with;
- what privileges it requires;
- what failures it can cause outside its own boundary.

A boundary should exist for an engineering reason.

Do not create components merely to make an architecture diagram appear clean.

Do not combine responsibilities merely to reduce file or process count if doing so creates an unclear trust or failure boundary.

---

## Make Data Flow Understandable

Important data flows should be explainable from source to destination.

For each significant flow, understand:

- where the data originates;
- whether the source is trusted;
- how the data is validated;
- how it is transformed;
- where it is stored;
- where it is transmitted;
- who can access it;
- how long it is retained;
- what happens if it is incomplete, malformed, stale, duplicated, or unavailable.

Avoid unnecessary transformations and serialization boundaries.

Every transformation can alter, lose, reinterpret, or obscure information.

Preserve original facts where those facts may be needed later for validation, troubleshooting, security analysis, or interpretation.

---

## Separate Observation From Interpretation

Where practical, systems that collect information should distinguish between what was observed and what was inferred.

An observed value should not silently become an interpreted conclusion.

For example:

```text
Observed:
The operating system returned status X.

Interpretation:
Status X probably means condition Y.
```

Those are different pieces of information.

Maintaining that distinction improves:

- troubleshooting;
- validation;
- future interpretation changes;
- compatibility work;
- user trust.

If interpretation changes later, the original observation should remain usable where practical.

---

## Separate Observation From Action

Systems that observe an environment should not automatically gain authority to modify that environment unless modification is part of their defined responsibility.

Collection, analysis, recommendation, and action are different capabilities.

Where practical, separate:

- collecting facts;
- interpreting facts;
- deciding what should happen;
- authorizing an action;
- performing the action.

This is especially important where actions affect:

- identity;
- security;
- configuration;
- networks;
- storage;
- backups;
- production services;
- recovery;
- destructive operations.

A component should not gain write authority merely because read authority was necessary.

---

## Authority Must Be Deliberate

Every privileged capability should have a reason to exist.

For each process, service, agent, or component, understand:

- what account or identity it runs as;
- what filesystem access it requires;
- what operating-system privileges it requires;
- what network access it requires;
- what administrative APIs it can use;
- what credentials or keys it can access;
- what actions those privileges make possible.

Use the minimum authority necessary for the component's defined responsibility.

Where practical, isolate privileged operations from ordinary operations.

Avoid designs where one compromised component automatically exposes unrelated administrative authority.

---

## Trust Must Be Explicit

Do not leave important trust relationships implicit.

Identify:

- trusted identities;
- trusted systems;
- trusted data sources;
- trusted configuration;
- trusted certificates or keys;
- trusted administrative channels;
- trusted update sources.

Then ask what happens when that trust is wrong.

A trusted source may become:

- compromised;
- unavailable;
- stale;
- misconfigured;
- replaced;
- malicious.

Designs should account for realistic trust failure.

---

## Treat External Input as Untrusted

Input crossing a system boundary must be validated according to the risk it presents.

This includes data from:

- users;
- files;
- network services;
- APIs;
- operating systems;
- databases;
- command output;
- configuration;
- logs;
- devices;
- dependencies;
- other ISS components.

"Internal" does not automatically mean safe.

Malformed or unexpected input should not be allowed to create uncontrolled behavior.

Parsers deserve particular attention because they convert untrusted bytes or text into trusted internal structures.

---

## Native Platform Behavior Matters

Platform-specific behavior should be part of the design when it materially affects the system.

Do not design against an imagined generic operating system and then force each platform to imitate it.

A Windows component may appropriately depend on:

- Windows services;
- native Windows APIs;
- Windows security identities;
- NTFS;
- the Windows Event Log;
- Windows-specific authentication.

A Linux component may appropriately depend on Linux-native mechanisms.

A FreeBSD component may appropriately depend on FreeBSD-native mechanisms.

Platform-native implementations may differ internally while still satisfying the same higher-level ISS engineering requirement.

The architecture should reflect the platform it actually lives on.

---

## Dependencies Are Architecture

A dependency is not merely a coding convenience.

Every dependency can introduce:

- security exposure;
- update requirements;
- compatibility constraints;
- licensing considerations;
- supply-chain risk;
- additional failure modes;
- operational requirements.

Dependencies should therefore be considered during design.

Prefer:

1. an existing platform facility when it correctly solves the problem;
2. the language standard library when appropriate;
3. a well-understood external dependency when it provides substantial value;
4. custom implementation when the requirement is sufficiently small, specific, or security-sensitive to justify it.

Do not automatically choose custom code.

Do not automatically choose a dependency.

Make the engineering decision appropriate to the requirement.

---

## Failure Is Part of the Design

For every meaningful component, consider failure before declaring the design complete.

Ask:

- What happens if the process crashes?
- What happens if the host restarts?
- What happens if storage becomes unavailable?
- What happens if the network disappears?
- What happens if the dependency is slow?
- What happens if credentials expire?
- What happens if state is corrupted?
- What happens if an operation stops halfway through?
- What happens if input is malformed?
- What happens if output cannot be written?
- What happens if an operator makes a mistake?
- What happens if the same operation runs twice?

Failure behavior should be deliberate.

Do not rely on undefined or accidental recovery behavior.

---

## Fail Safely

When a failure could affect authority, security, integrity, or destructive behavior, prefer failure that preserves the safer state.

Examples include:

- inability to authenticate;
- inability to verify authorization;
- invalid signatures;
- corrupt security configuration;
- unverifiable update packages;
- ambiguous destructive targets.

Failing safely does not mean every error must stop the entire system.

The failure behavior should match the risk of the operation.

Systems should preserve useful partial functionality when that can be done truthfully and safely.

---

## Partial Results Must Remain Partial

A partial result must not silently become a complete result.

If a system can only retrieve part of the required information, that limitation should remain visible.

Where appropriate, distinguish between:

```text
complete
partial
unavailable
unsupported
invalid
failed
not applicable
```

Do not invent values solely to satisfy a schema or make output appear complete.

Unknown information should remain visibly unknown.

---

## Destructive Operations Require Stronger Boundaries

Operations that delete, overwrite, modify, revoke, disable, restore, reconfigure, or otherwise materially alter systems require additional care.

Where practical, destructive or high-impact actions should be:

- explicit;
- separately authorized;
- clearly identified;
- narrow in scope;
- validated immediately before execution;
- logged;
- recoverable where feasible.

A read-only operation should not quietly become a write operation.

A diagnostic command should not quietly become a remediation command.

---

## Recovery Is Architecture

Recovery must be considered while the system is being designed.

Do not wait until after implementation to ask how the system will recover.

Consider recovery from:

- component failure;
- server failure;
- corrupted state;
- lost credentials;
- broken trust;
- failed upgrades;
- interrupted transactions;
- incomplete writes;
- operator mistakes;
- damaged configuration.

Recovery mechanisms should not depend unnecessarily on the same component or trust relationship that has failed.

Where practical, recovery paths should be testable independently.

---

## State Must Have an Owner

Persistent state should have clear ownership.

For each state store, understand:

- who creates it;
- who reads it;
- who modifies it;
- what protects it;
- whether it can be rebuilt;
- whether corruption can be detected;
- whether partial writes are possible;
- whether concurrent access is possible;
- how old or stale state is handled.

Do not create persistent state merely because storing something is convenient.

State creates long-term compatibility and recovery responsibilities.

---

## Idempotence Should Be Considered

Operations that may be retried should be designed with repeated execution in mind.

Ask:

- Can this operation safely run twice?
- Can the system detect that it already completed?
- Can an interrupted operation resume?
- Can retry create duplicate data or destructive side effects?

Not every operation can be perfectly idempotent.

The important requirement is to understand repeated execution instead of discovering its behavior accidentally during failure recovery.

---

## Concurrency Must Be Justified

Concurrency can improve performance and responsiveness, but it also introduces complexity.

Concurrent designs must consider:

- shared state;
- ordering;
- locking;
- races;
- resource exhaustion;
- cancellation;
- shutdown;
- duplicate work;
- partial completion.

Do not introduce concurrency solely because the language makes it easy.

Use it where the workload or operational requirement justifies it.

Test the behavior that concurrency introduces.

---

## Performance Is a System Property

Performance should be considered in context.

Do not optimize only the local function while ignoring the system around it.

A design may affect:

- CPU;
- memory;
- storage;
- filesystem cache;
- database load;
- network throughput;
- network latency;
- device resources;
- service responsiveness;
- other applications sharing the host.

Measure important behavior where practical.

Do not rely only on assumptions about what should be fast or cheap.

---

## Resource Use Must Be Bounded Where Necessary

Any input-driven or long-running operation should consider resource limits.

Examples include:

- file size;
- line length;
- queue depth;
- worker count;
- memory allocation;
- recursion depth;
- network connections;
- request duration;
- retry count;
- retained history.

An attacker, damaged system, or simply unusually large environment should not be able to create unlimited resource consumption by accident.

Limits should be chosen based on actual engineering requirements rather than arbitrary values.

---

## Do Not Hide Important Behavior Behind Abstraction

Abstraction is useful when it simplifies a real repeated concept.

It is harmful when it hides behavior engineers need to understand.

Do not abstract away:

- privilege;
- security boundaries;
- platform differences;
- destructive behavior;
- failure states;
- network communication;
- storage behavior;
- performance-sensitive operations.

If understanding the underlying mechanism is necessary to operate or secure the system, the design should keep that mechanism visible.

---

## Observability Must Support Diagnosis

A design should consider how operators will understand system behavior after deployment.

Useful operational visibility may include:

- meaningful logs;
- status reporting;
- counters;
- health state;
- diagnostics;
- test commands;
- validation commands.

Do not generate telemetry merely because telemetry is fashionable.

Every operational signal should answer a useful question.

Sensitive information must not be exposed merely to improve diagnostics.

---

## Updates Are Privileged Changes

Software updates alter trusted code.

Update design should therefore consider:

- source authenticity;
- package integrity;
- version identity;
- compatibility;
- rollback;
- configuration migration;
- state migration;
- service interruption;
- privilege changes;
- security impact.

An update mechanism should not become an easier path to privileged execution than the application itself.

---

## Compatibility Must Be Demonstrated

Compatibility is an engineering claim.

Do not assume compatibility solely because:

- an API appears unchanged;
- a dependency uses semantic versioning;
- an operating system is from the same family;
- a compiler accepts the code;
- a previous release worked.

Where compatibility matters, validate the behavior that matters.

Compatibility documentation should distinguish between environments that are:

- tested;
- validated;
- expected to work but unverified;
- unsupported.

---

## Design for Verification

Important design claims should be testable wherever practical.

If the design claims:

- a process cannot modify a protected directory;
- a service operates with limited privilege;
- a parser rejects malformed input;
- a recovery path works without a primary dependency;
- an update preserves compatibility;
- a component does not traverse a boundary;

there should be a practical way to verify that behavior.

The verification method should live with the project where practical.

Designing something that cannot reasonably be tested should trigger additional scrutiny.

---

## Design Documentation Should Be Useful

Document important design decisions when doing so helps future engineers, operators, reviewers, or users understand the system.

Documentation may be appropriate when explaining:

- architecture;
- trust boundaries;
- data flow;
- privilege;
- unusual platform behavior;
- security decisions;
- recovery;
- significant tradeoffs;
- rejected alternatives.

Do not create design documents merely because a process says one must exist.

A short explanation that clearly answers the important question is better than a large document nobody uses.

---

## Questions Before Implementation

Before substantial implementation begins, an engineer should be able to answer the following where applicable:

```text
What problem are we solving?

Why does this component exist?

What responsibility does it own?

What inputs does it accept?

What outputs does it produce?

What state does it own?

What operating-system or platform facilities does it depend on?

What privileges does it require?

What does it trust?

What trusts it?

What external systems does it communicate with?

What does it read?

What does it write?

What can it modify?

What happens if its input is malformed?

What happens if a dependency disappears?

What happens if the process crashes?

What happens if the host restarts?

What happens if an operation stops halfway through?

What happens if the operation runs twice?

How does the component recover?

How will its important behavior be tested?

How can a user or operator verify its important claims?
```

Not every project needs a formal written answer to every question.

The engineer designing the system should nevertheless understand the answers.

---

## Final Principle

Good architecture is not the architecture with the most layers, interfaces, services, diagrams, or abstractions.

Good architecture makes the system easier to understand, operate, test, secure, recover, and change.

Every layer should earn its place.
