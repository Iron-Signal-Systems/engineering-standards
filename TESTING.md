# Iron Signal Systems Testing and Validation Standard

## Purpose

This document defines how Iron Signal Systems tests and validates software.

Testing and validation are required parts of ISS engineering.

They are not release paperwork.

They are how ISS determines whether the system actually behaves the way its design and documentation claim.

The objective is not to accumulate test counts, coverage percentages, reports, or passing checkboxes.

The objective is to understand the system well enough to trust its behavior and to make important claims verifiable by engineers, operators, reviewers, and users.

---

## Testing Is Part of Engineering

Testing begins with design and implementation.

It is not something added after development is considered finished.

When a capability introduces meaningful behavior, the engineer should understand how that behavior can be tested and validated.

If an important design claim cannot reasonably be tested, that limitation should be understood before the design is accepted.

A capability is not complete merely because:

- it compiles;
- it starts;
- it produces expected output once;
- the normal path works;
- a unit test passes.

Completion requires testing appropriate to the behavior and risk of the capability.

---

## Validation Is Mandatory

ISS projects must validate the behavior they claim.

The exact validation required depends on the system.

A parser and a recovery platform will not require the same testing.

A Windows service and a web interface will not require the same testing.

The standard does not require identical test suites.

It requires meaningful validation appropriate to the project.

Where applicable, validation should address:

- expected behavior;
- boundary conditions;
- malformed input;
- hostile input;
- failure conditions;
- privilege boundaries;
- trust boundaries;
- concurrency;
- recovery;
- platform behavior;
- compatibility;
- resource use;
- operational impact;
- security controls.

---

## Tests Belong With the Project

Tests and validation procedures must live with the project wherever practical.

They should be versioned alongside the implementation they validate.

This includes applicable:

- unit tests;
- integration tests;
- regression tests;
- platform tests;
- security tests;
- validation scripts;
- operational test procedures;
- test fixtures;
- reproducible test data.

A project should not depend on an undocumented internal vendor process to establish that it works.

If ISS uses a procedure to validate an important customer-visible claim, that procedure should be visible and reproducible by others where practical.

---

## Users Should Be Able to Verify Important Claims

ISS should not require users to blindly trust claims that can reasonably be demonstrated.

Where practical, users and operators should be able to run project-provided tests or validation procedures to confirm important behavior in their own environment.

Examples may include verifying:

- required permissions;
- read-only behavior;
- service identity;
- security boundaries;
- filesystem access;
- network communication;
- configuration;
- recovery behavior;
- compatibility;
- platform prerequisites;
- operational state.

The validation should explain what it tests and what a successful result actually establishes.

A passing validation command should not imply more than it proves.

---

## Tests Must Be Understandable

A test should make its purpose reasonably clear.

An engineer reviewing a test should be able to determine:

- what behavior is being tested;
- what conditions are being created;
- what result is expected;
- why that result matters.

Avoid tests whose only meaningful explanation is that they make a coverage tool happy.

Complex tests may require comments or supporting documentation.

Do not make simple behavior difficult to understand by building unnecessary test frameworks around it.

---

## Test the Claim Being Made

Use the type of testing appropriate to the claim.

If the claim concerns pure program logic, a unit test may be sufficient.

If the claim concerns interaction between components, use integration testing.

If the claim concerns actual operating-system behavior, test on the actual operating system.

If the claim concerns filesystem semantics, test against the relevant filesystem.

If the claim concerns privileges, verify the real privilege boundary.

If the claim concerns recovery, perform recovery.

If the claim concerns compatibility with a platform version, test that platform version.

The testing method must match the engineering claim.

---

## Unit Testing

Unit tests are appropriate for isolated program logic.

They are useful for validating behavior such as:

- parsing rules;
- transformations;
- calculations;
- state transitions;
- selection logic;
- validation rules;
- boundary conditions.

Unit tests should normally be:

- deterministic;
- repeatable;
- reasonably fast;
- isolated from unnecessary external systems.

Unit tests are valuable.

They do not establish behavior outside the boundary they actually test.

---

## Integration Testing

Integration testing validates that components interact correctly.

This may include interaction between:

- packages;
- processes;
- services;
- databases;
- filesystems;
- APIs;
- authentication systems;
- network services;
- devices;
- other ISS components.

Mocks may assist development, but meaningful integration boundaries should eventually be tested using the actual participating components where practical.

A simulated integration is not the same as a validated real integration.

---

## Platform Testing

Platform-dependent behavior must be tested on the actual platform where practical.

Examples include:

- Windows APIs;
- Windows services;
- NTFS;
- Windows security;
- Linux system calls;
- Linux permissions;
- FreeBSD facilities;
- platform service managers;
- native event systems;
- filesystem-specific behavior;
- kernel interfaces.

A mock cannot establish native operating-system behavior.

A Linux implementation passing tests does not establish Windows compatibility.

A Windows Server 2022 test does not automatically establish Windows Server 2025 compatibility.

Platform claims must match the platforms actually tested.

---

## Operational Testing

Some behavior can only be meaningfully understood under realistic operational conditions.

Operational testing may examine:

- sustained runtime;
- realistic dataset sizes;
- large environments;
- service restarts;
- host restarts;
- network interruption;
- unavailable dependencies;
- storage pressure;
- high event rates;
- concurrent operations;
- actual deployment identities;
- realistic permissions.

Not every feature requires a full operational lab.

When the real environment materially changes the behavior, unit and integration testing alone are not enough.

---

## Security Testing

Security-sensitive behavior must be deliberately tested.

Applicable testing may include:

- authentication;
- authorization;
- privilege boundaries;
- access control;
- credential handling;
- certificate validation;
- signature verification;
- malformed input;
- hostile input;
- parser boundaries;
- filesystem protection;
- network exposure;
- update validation;
- administrative actions;
- recovery authority.

A security control is not established merely because the code appears correct.

Where practical, test both allowed and denied behavior.

---

## Test the Negative Case

Testing only what should succeed is not enough.

Where meaningful, test what must fail.

Examples include:

- invalid credentials are rejected;
- unauthorized identities are denied;
- malformed data is rejected;
- invalid signatures fail verification;
- insufficient privilege prevents the action;
- unsupported platforms are reported correctly;
- invalid configuration does not silently become valid configuration;
- read-only identities cannot perform write operations;
- destructive actions cannot occur through observational paths.

Negative tests are particularly important for security and trust boundaries.

---

## Test Failure Behavior

Systems must be tested when things go wrong.

Applicable failure testing may include:

- process termination;
- service restart;
- host restart;
- network interruption;
- unavailable storage;
- unavailable dependencies;
- timeouts;
- corrupt state;
- malformed state;
- incomplete input;
- failed writes;
- expired credentials;
- partial completion.

The purpose is not to invent every theoretical failure.

Test failures that are credible and meaningful to the system's responsibilities.

---

## Recovery Must Be Tested

Recovery claims require recovery testing.

Do not claim that a system can recover merely because a recovery path exists in the source code.

Where applicable, test:

- restart recovery;
- checkpoint recovery;
- interrupted operations;
- damaged or missing state;
- dependency restoration;
- credential replacement;
- rollback;
- restore procedures;
- rebuilding derived data.

For systems whose primary purpose includes recovery, recovery testing is a central part of validation.

---

## Regression Tests Stay

When a defect is discovered and can reasonably be reproduced, the correction should include a regression test.

The test should remain with the project.

Do not remove a regression test simply because the original defect has been fixed.

The test exists to make sure the defect remains fixed.

This is especially important for:

- platform compatibility problems;
- security defects;
- parser failures;
- malformed input;
- race conditions;
- recovery failures;
- dependency changes;
- operating-system updates.

A past failure becomes useful engineering knowledge when the project retains a reproducible test for it.

---

## Compatibility Fixes Should Preserve the Reproducer

When an external change breaks an ISS project, preserve a test or validation procedure that demonstrates the condition where practical.

This may involve:

- operating-system updates;
- API changes;
- dependency updates;
- protocol changes;
- device firmware;
- service behavior.

The project should then be able to demonstrate that the corrected implementation handles the condition properly.

Users should not have to rely only on a release note saying that compatibility was improved.

---

## Do Not Test Only the Happy Path

Normal successful operation is necessary but insufficient.

Where relevant, tests should include:

- empty input;
- minimum input;
- maximum expected input;
- unusual valid input;
- malformed input;
- duplicate input;
- stale input;
- interrupted operations;
- missing dependencies;
- permission failures;
- unexpected ordering;
- repeated execution.

The relevant cases depend on the component.

Testing should be driven by behavior and risk, not by a universal checklist.

---

## Boundary Testing Matters

Important limits should be tested around their boundaries.

If a project defines limits for:

- file size;
- line size;
- queue depth;
- worker count;
- retry count;
- timeouts;
- request size;
- record count;
- path length;
- memory use;

tests should verify behavior around those limits where practical.

A limit that has never been exercised is largely an assumption.

---

## Concurrency Requires Concurrency Testing

Concurrent code should be tested for the problems concurrency creates.

Applicable testing may include:

- races;
- ordering;
- duplicate work;
- deadlocks;
- cancellation;
- shutdown;
- resource exhaustion;
- partial completion;
- simultaneous access to state.

Use language or platform race detection tools when appropriate.

Passing ordinary functional tests does not establish that concurrent code is race-free.

---

## Performance Must Be Measured When It Matters

Performance claims should be based on measurement.

Where performance is operationally important, measure relevant behavior such as:

- CPU;
- memory;
- disk I/O;
- network use;
- latency;
- throughput;
- queue growth;
- storage growth;
- database impact.

Use realistic workloads where practical.

A microbenchmark does not necessarily establish system-level performance.

A benchmark should answer a meaningful engineering question.

---

## Operational Impact Must Be Validated

ISS infrastructure software must consider its effect on the systems around it.

Testing should verify, where applicable, that a capability does not create unreasonable impact on:

- production services;
- file servers;
- databases;
- domain controllers;
- network devices;
- storage;
- backup systems;
- authentication systems;
- other workloads sharing the host.

Correct output does not excuse destructive resource consumption.

---

## Test With Realistic Scale When Scale Matters

Behavior at ten records may not represent behavior at ten million records.

When project scale materially affects the implementation, include tests appropriate to realistic expected environments.

This may include:

- large files;
- large directories;
- many objects;
- long-running collection;
- sustained event rates;
- large configuration sets;
- large databases;
- many network devices.

Do not invent huge scale testing when scale does not materially affect the component.

---

## Tests Should Be Deterministic Where Practical

A test should produce the same conclusion when the tested conditions have not changed.

Avoid unnecessary dependence on:

- wall-clock timing;
- external public services;
- uncontrolled network conditions;
- random data;
- shared mutable environments.

Some platform and operational tests inherently involve variable conditions.

Where nondeterminism is unavoidable, design the validation so the expected behavior remains clear.

---

## Time-Based Testing Requires Care

Tests involving time should avoid fragile assumptions.

Where practical:

- control the clock;
- use reasonable timeouts;
- avoid arbitrary sleeps;
- verify state rather than guessing that enough time has passed.

Operational tests may sometimes require real waiting.

That should be deliberate rather than the default testing technique.

---

## Test Data Must Be Appropriate

Test data should represent the conditions being tested.

Use:

- synthetic data;
- generated fixtures;
- controlled lab data;
- safely sanitized real-world examples where appropriate.

Do not place customer secrets, credentials, sensitive production data, or unnecessary personally identifiable information into project test fixtures.

---

## Test Fixtures Are Part of the Test

Fixtures should be understandable and versioned when they are required to reproduce behavior.

If a parser depends on a specific malformed input that once caused a defect, preserve that input safely where licensing and sensitivity allow.

Do not silently regenerate important fixtures in ways that make the original failure difficult to reproduce.

---

## Do Not Hide Test Failures

A failed applicable test is a failed test.

Do not:

- convert failure into warning merely to make a build pass;
- silently skip failing tests;
- remove tests because they expose an uncomfortable problem;
- ignore a failure without understanding why.

A test may legitimately become obsolete.

If so, remove or change it because the requirement changed, not because the test became inconvenient.

---

## Skipped Tests Must Have a Reason

Tests may require specific:

- operating systems;
- privileges;
- devices;
- services;
- credentials;
- lab environments.

When a test cannot run in the current environment, the skip should be understandable.

Do not report a skipped platform test as though the platform was validated.

---

## Automated Testing Does Not Replace Human Review

Automation is valuable for repeatability.

It does not understand every design assumption, operational consequence, security boundary, or unexpected interaction.

Important changes still require engineering review.

Likewise, human review does not replace executable testing.

Use both where appropriate.

---

## Coverage Is Information, Not the Goal

Code coverage can identify untested code.

It must not become the objective of the test suite.

ISS does not require an arbitrary organization-wide coverage percentage.

A project with 95% coverage can still fail to test its most important security boundary.

A project with lower coverage may thoroughly test every meaningful operational behavior.

Use coverage to ask better questions.

Do not use it as a substitute for understanding what was tested.

---

## Test Count Is Not a Quality Metric

More tests do not automatically mean better testing.

One meaningful test of an actual security boundary may be more valuable than hundreds of trivial tests.

Do not split tests merely to increase their count.

Do not write redundant tests solely to create impressive statistics.

Test meaningful behavior.

---

## Validation Results Must Be Truthful

When reporting validation, state what actually occurred.

Where useful, identify:

- what was tested;
- the platform;
- the platform version;
- relevant filesystem or runtime;
- important configuration;
- whether the test passed or failed;
- meaningful limitations.

Do not describe a test as independent validation when it was performed by the developer.

Do not describe simulated testing as platform validation.

Do not describe partial testing as complete testing.

---

## Platform and Version Information Matters

Where behavior depends on the environment, retain enough information to understand the validation context.

Examples may include:

```text
Operating system: Windows Server 2022
Build: 20348.x
Filesystem: NTFS
Architecture: amd64
Application version: x.y.z
```

The exact information required depends on the project.

Do not collect environment information merely to produce a large report.

Capture what is needed to understand and reproduce the result.

---

## User Validation Should Be Safe

Tests intended for users and operators should clearly identify their operational effect.

A validation procedure should state when it is:

- read-only;
- state-changing;
- networked;
- privileged;
- potentially disruptive;
- destructive.

Do not disguise modification as validation.

If a validation procedure changes the system, that should be clear before it runs.

---

## Read-Only Validation Is Preferred Where Possible

When a claim can be verified without modifying the environment, prefer a read-only validation method.

Some behavior cannot be proven without controlled modification.

When change is necessary:

- keep it narrow;
- explain it;
- make cleanup clear;
- avoid unnecessary production impact.

---

## Destructive Testing Requires Controlled Environments

Destructive tests should not be casually run against production systems.

Testing involving:

- deletion;
- restore;
- overwrite;
- credential destruction;
- security-policy modification;
- service disruption;
- filesystem corruption;
- disaster recovery;

should use controlled environments unless the test itself is an explicitly approved operational exercise.

The project should make this distinction clear.

---

## Tests Must Not Require Unnecessary Vendor Access

A customer should not need to grant ISS broad administrative access merely so ISS can determine whether its own software is working.

Where practical, diagnostic and validation procedures should allow the customer's own administrators to gather and interpret the necessary results.

When vendor assistance is needed, the project should make clear what information is required and why.

---

## Validation Output Should Be Understandable

A validation tool should not produce only:

```text
PASS
```

when understanding the checked condition matters.

Where appropriate, show enough information for the operator to determine what was validated.

Likewise, a failure should identify the condition that failed and provide useful diagnostic information without exposing secrets.

The amount of output should match the complexity of the validation.

---

## Validation Must Not Become a Framework Requirement

ISS projects may use:

- native language tests;
- scripts;
- command-line validation tools;
- lab procedures;
- integration environments;
- project-specific utilities.

There is no requirement for every ISS project to adopt one universal validator.

The engineering standard defines what must be understood and demonstrated.

Each project chooses the simplest appropriate mechanism for doing so.

---

## CI Is Useful but Not Authority

Continuous integration can automatically run:

- unit tests;
- static analysis;
- builds;
- integration tests;
- platform tests;
- security checks.

CI is useful for repeatability.

A green CI result does not automatically establish that the software is operationally safe, secure, compatible, or production-ready.

CI reports the checks it actually ran.

Nothing more.

---

## Local Testing Must Remain Possible

Where practical, developers should be able to run important tests without depending entirely on a remote CI platform.

Project tests should not become inaccessible merely because a hosted service changes or disappears.

Platform or large-scale testing may legitimately require dedicated infrastructure.

The ordinary development test path should remain understandable.

---

## Tests Must Evolve With the System

When behavior changes deliberately, its tests must change with it.

Do not preserve outdated expectations merely because they once existed.

When a contract changes:

- update the implementation;
- update the relevant tests;
- update user validation where necessary;
- update documentation describing the behavior.

Tests preserve intended behavior.

They should not freeze accidental behavior forever.

---

## Testing Security Fixes

A security correction should include a reproducible test for the protected condition where practical.

The test should demonstrate that:

- the vulnerable or unsafe behavior is no longer allowed;
- intended legitimate behavior still works where applicable.

Do not publish sensitive exploitation details when doing so would create unnecessary risk.

The retained test should still protect against regression.

---

## Testing Bug Fixes

When correcting a defect:

1. reproduce the defect where practical;
2. create or identify a test that demonstrates it;
3. correct the implementation;
4. confirm the test now passes;
5. run related tests to detect unintended effects;
6. retain the regression test.

This creates a permanent record of the behavior the project intends to preserve without requiring a separate bureaucracy.

---

## Testing Platform Updates

When an operating-system or platform update changes behavior:

1. reproduce the changed condition where practical;
2. identify what behavior changed;
3. determine whether the cause is upstream, ISS, or an interaction between them;
4. create or retain a validation procedure for the condition;
5. correct ISS behavior if necessary;
6. validate the correction on the affected platform;
7. clearly identify what remains untested.

Do not reduce the result to "compatibility improved" when useful technical information is known.

---

## Testing Dependencies

Dependencies should be tested at the boundary where ISS relies on them.

Do not attempt to retest the entire dependency.

Test the assumptions ISS makes about it.

When a dependency update affects behavior, retain a regression test where practical.

---

## Testing Native Platform Assumptions

If ISS depends on an operating-system behavior that is important to correctness or security, validate that behavior directly.

Examples may include:

- file sharing semantics;
- access-control behavior;
- filesystem journal behavior;
- service identity;
- process privileges;
- event delivery;
- locking;
- reparse or symbolic-link behavior;
- system-call semantics.

Do not rely solely on documentation when a practical validation can establish the assumption in the target environment.

Documentation tells us what should happen.

Testing tells us what happened.

---

## Testing Must Be Maintainable

Tests are production engineering assets.

Avoid test suites that are so complicated or fragile that engineers stop trusting them.

Remove unnecessary test infrastructure.

Keep helpers understandable.

Avoid hidden global setup that makes individual tests difficult to interpret.

A test suite that nobody can maintain eventually stops protecting the system.

---

## Testing Must Not Distort Production Design

Do not weaken production architecture solely to make a test easier to write.

Do not add public interfaces, configuration switches, runtime hooks, or insecure bypasses merely for testing convenience.

Create the smallest appropriate test boundary.

Test-specific code must not introduce unnecessary production risk.

---

## Before Declaring a Capability Complete

For a meaningful capability, an engineer should be able to answer where applicable:

```text
What behavior was tested?

What was not tested?

What failure paths were tested?

What security boundaries were tested?

What platform behavior was validated?

What environment was used?

What regression tests protect known failures?

What happens when dependencies disappear?

What happens after restart or interruption?

What operational impact was measured?

Can another engineer reproduce the result?

Can an administrator or user verify the important claims?

Are any tests skipped or unavailable in this environment?

Does the validation support the claims being made?
```

Not every project needs a formal document answering every question.

The project must nevertheless understand the answers.

---

## Final Principle

Testing is not proof that software has no defects.

Validation is not a certificate that a system cannot fail.

They are disciplined ways of replacing assumptions with observed behavior.

ISS testing should make it possible to say:

> This is what we claim the system does.  
> This is how we tested it.  
> This is where we tested it.  
> This is what happened.  
> This is what we have not yet verified.  
> And this is how you can check the important parts yourself.

That is the standard.

Trust should be something a user can inspect and verify, not something a vendor asks them to accept.
