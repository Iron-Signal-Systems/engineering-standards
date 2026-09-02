# Iron Signal Systems Engineering Standards

The Iron Signal Systems Engineering Standards define how Iron Signal Systems software is designed, engineered, implemented, tested, validated, secured, hardened, maintained, and operated.

These standards apply to all ISS software projects.

They exist to help produce systems that engineers, administrators, operators, reviewers, and customers can understand and trust.

The objective is not to satisfy a checklist.

The objective is to understand the system well enough to trust its behavior, and to make that behavior clear enough that users can verify the results for themselves.

---

## Purpose

ISS builds software intended to operate in real environments where reliability, security, maintainability, and operational impact matter.

A system is not well engineered simply because it compiles, passes a test suite, or performs its primary function under ideal conditions.

ISS engineering considers:

- what the system does;
- why it exists;
- what it reads and writes;
- what authority it requires;
- what it trusts;
- what trusts it;
- how it behaves when something goes wrong;
- how it affects the systems around it;
- how it can be tested and validated;
- how users can verify its behavior;
- how it can be recovered, maintained, and changed safely.

The standard exists to make those questions part of normal engineering rather than something considered only after implementation.

---

## One Standard

This is the common engineering standard for Iron Signal Systems projects.

Projects may differ greatly in purpose, architecture, programming language, operating system, deployment model, security boundary, performance requirements, and operational environment.

The standard therefore defines engineering principles, required behaviors, and expected outcomes without forcing unrelated projects into the same architecture or implementation.

A project must meet the engineering intent and explicitly stated requirements of this standard using an implementation appropriate to that project.

Consistency does not require identical code.

---

## Understandable Before Clever

ISS software should be understandable by the engineers and administrators responsible for operating and maintaining it.

Prefer straightforward designs over unnecessary abstraction.

Do not introduce frameworks, generalized engines, compatibility layers, plugin systems, shared interfaces, or other architectural machinery solely because they may become useful later.

A new abstraction should solve a demonstrated problem.

Do not solve hypothetical future requirements at the expense of making the current system harder to understand, test, secure, operate, or maintain.

Build what is needed.

---

## Platform-Native Engineering

Software should be engineered for the platform on which it operates.

Windows software should use appropriate Windows mechanisms.

Linux software should use appropriate Linux mechanisms.

FreeBSD software should use appropriate FreeBSD mechanisms.

The existence of multiple supported platforms does not justify reducing every implementation to the lowest common denominator.

Where operating-system behavior, security, filesystems, services, networking, identity, storage, recovery, or other platform facilities materially affect the design, the implementation should use the appropriate native facilities where practical.

Portability is valuable when required.

Artificial portability that weakens the implementation or hides important platform behavior is not.

---

## Testing, Validation, and Hardening Are Required

Testing, validation, and security hardening are mandatory parts of ISS engineering.

A capability is not considered complete merely because it builds, runs, or produces the expected result under normal conditions.

Projects must test and validate the behavior that matters to their design.

Applicable testing should include areas such as:

- expected behavior;
- boundary conditions;
- failure paths;
- malformed input;
- hostile input;
- trust boundaries;
- privilege boundaries;
- recovery behavior;
- regression conditions;
- platform-specific behavior;
- operational impact.

Security must be considered during design and implementation and deliberately reviewed and hardened before a capability is considered ready for operational use.

The amount and type of testing will differ between projects.

The requirement for meaningful testing does not.

---

## Tests Belong With the Project

Tests and validation procedures must live with the project wherever practical.

They should be:

- versioned with the code;
- visible to reviewers and users;
- understandable;
- repeatable where practical;
- retained when they protect against a meaningful regression.

A user or operator should be able, where practical, to execute the same validation procedures and independently verify the behavior or result being claimed.

ISS should not require blind trust in a vendor statement when the behavior can reasonably be demonstrated.

---

## Claims Must Match Validation

Testing establishes only what was actually tested.

Unit testing does not establish real operating-system behavior.

A simulated integration does not establish production compatibility.

Testing on one operating-system version does not automatically establish compatibility with another.

Testing one deployment model does not establish all deployment models.

Projects must clearly distinguish what has been implemented, tested, validated, supported, partially validated, remains unverified, or is unsupported.

The terminology used to represent those conditions may be appropriate to the individual project. These distinctions do not define a mandatory ISS status schema.

ISS must not claim a level of compatibility, security, reliability, or validation that has not been demonstrated.

---

## Failure Must Be Visible

ISS software must not hide meaningful failure.

Silent fallback, silent data loss, silent privilege changes, silent security downgrades, or misleading success states are unacceptable.

Where applicable, systems should distinguish between conditions such as:

- complete;
- partial;
- unavailable;
- unsupported;
- invalid;
- failed;
- not applicable.

These examples describe meaningful distinctions and do not require a universal status vocabulary across ISS projects.

Users and operators should be able to determine when the system did not do what was expected.

---

## Security Is Part of the Architecture

Security is not a final-stage scan performed after the system has already been designed.

Security decisions include:

- authentication;
- authorization;
- identity;
- privilege;
- trust boundaries;
- credential handling;
- secrets;
- filesystem permissions;
- service accounts;
- network exposure;
- cryptography;
- update mechanisms;
- logging;
- state protection;
- recovery;
- parser behavior;
- failure handling.

ISS systems should operate with the minimum authority necessary for their purpose.

Privileged behavior should be narrow, understandable, and testable.

Security must not be silently weakened merely to preserve compatibility or make an implementation easier.

---

## Operational Impact Matters

Software that produces correct output while damaging the environment around it is not well engineered.

ISS projects must consider their operational effect on the systems they use.

Depending on the project, this may include:

- CPU;
- memory;
- disk usage;
- storage growth;
- filesystem I/O;
- network traffic;
- connection counts;
- locks;
- latency;
- service responsiveness;
- database impact;
- privilege;
- failure amplification.

Performance requirements should be based on actual system behavior and measurement rather than assumption.

---

## Systems Must Be Diagnosable

ISS systems should expose enough information for administrators and operators to understand what they are doing and diagnose problems.

Operational visibility should serve a real engineering purpose.

ISS does not require unnecessary telemetry systems, excessive metrics, or logging merely to create the appearance of observability.

The goal is useful visibility.

An administrator should not require proprietary vendor access merely to understand ordinary system behavior.

---

## Change Transparency

Meaningful changes must be understandable.

When a platform update, dependency change, protocol change, API change, or ISS defect causes a compatibility problem, the project should document the cause when it is known.

When ISS corrects the problem, users should be able to understand:

- what changed;
- what failed;
- why it failed;
- what ISS changed;
- whether security or privilege changed;
- how the correction was tested;
- what environments were validated;
- what remains unverified.

Regression tests should remain with the project whenever practical.

An update should not require blind trust.

A user should be able to understand what changed, determine whether it affects their environment, and verify the corrected behavior where practical.

---

## Recovery Is Part of Design

Failure and recovery behavior are part of system architecture.

Projects should consider how they recover from:

- process failure;
- system restart;
- corrupted or incomplete state;
- interrupted operations;
- unavailable dependencies;
- credential failure;
- platform changes;
- operator mistakes;
- partial completion.

Recovery behavior should be deliberate rather than accidental.

A system should not depend on undocumented assumptions about what will happen after failure.

---

## Dependencies Must Earn Their Place

Dependencies introduce functionality, but they also introduce security, maintenance, compatibility, supply-chain, and operational costs.

Before adding a dependency, consider:

- what problem it solves;
- whether the problem is substantial enough to justify it;
- whether the language standard library already provides the capability;
- whether the operating system already provides the capability;
- what additional maintenance responsibility it introduces.

Avoid dependencies added only for convenience when the required behavior is simple and can be implemented clearly without them.

---

## Reference Code and Examples

Code contained in this repository is example and reference material unless a requirement is explicitly identified as mandatory.

The existence of example code does not require an adopting project to use the same:

- types;
- interfaces;
- functions;
- architecture;
- libraries;
- APIs;
- configuration;
- directory layout;
- data structures;
- implementation technique.

Language-specific and platform-specific examples remain examples.

A language-specific section may define ISS guidance or explicitly stated requirements that apply when that language is used.

A platform-specific section may do the same for that platform.

Mandatory requirements must be stated explicitly.

They must not be inferred merely because an example exists or because this repository discusses a particular language or operating system.

Projects must satisfy the engineering requirement using an implementation appropriate to the project.

The standards repository must not become an application framework that ISS projects are forced to inherit.

---

## Engineering Records, Not Checkbox Compliance

ISS projects should retain useful technical information when it helps engineers and users understand the system.

This may include:

- design decisions;
- test procedures;
- validation results;
- compatibility information;
- security decisions;
- known limitations;
- regression information;
- operational procedures.

Documentation should exist because it is useful.

ISS does not create documentation, records, reports, or process steps solely to prove that a checklist was completed.

Engineering discipline must produce better systems, not more paperwork.

---

## Trust Should Be Inspectable

ISS software should make its important behavior inspectable wherever practical.

Users should be able to understand:

- what the system is designed to do;
- what it actually does;
- what permissions it requires;
- what data it accesses;
- what it changes;
- what it sends or receives;
- what happens when it fails;
- what has been tested;
- how it was validated;
- what remains uncertain;
- how important results can be independently verified.

Trust should come from understandable design, visible behavior, meaningful testing, deliberate security, and reproducible validation.

Not from a vendor asking the customer to simply believe that the system works.

---

## Supporting Standards

The detailed ISS engineering requirements are maintained in:

- [`DESIGN.md`](DESIGN.md) — system design, architecture, boundaries, data flow, authority, failure, and recovery;
- [`ENGINEERING.md`](ENGINEERING.md) — implementation practices and day-to-day software engineering;
- [`TESTING.md`](TESTING.md) — testing, validation, regression testing, platform testing, and user-verifiable validation;
- [`SECURITY.md`](SECURITY.md) — security engineering, trust, privilege, hardening, and security validation;
- [`AGENTS.md`](AGENTS.md) — requirements for AI agents and automated contributors working on ISS projects.

Additional language-specific or platform-specific guidance should be added only when there is enough real engineering need to justify it.

The standards should remain understandable, practical, and directly connected to building better systems.

---

## License

This repository and its example and reference code are licensed under the [BSD 3-Clause License](LICENSE) unless otherwise stated.

Adoption of the Iron Signal Systems Engineering Standards does not require an adopting project to use the BSD 3-Clause License.

Each project retains its own licensing decisions.
