# Iron Signal Systems Engineering Standard

## Purpose

This document defines the implementation practices expected in Iron Signal Systems software.

The goal is not to force every project into the same code structure, architecture, programming style, library set, or framework.

The goal is to produce code that is:

- understandable;
- correct;
- testable;
- secure;
- maintainable;
- appropriate to its target platform;
- operationally responsible.

Implementation decisions should make the system easier to reason about, not merely easier to extend.

---

## Prefer Clear Code

Code should communicate its purpose without requiring unnecessary mental reconstruction.

Prefer:

- explicit control flow;
- descriptive names;
- clear ownership;
- small focused functions;
- visible error handling;
- obvious state transitions;
- direct use of platform capabilities where appropriate.

Avoid cleverness that saves a few lines while making behavior harder to understand.

The shortest implementation is not always the simplest implementation.

The simplest implementation is the one whose behavior can be understood and verified with the least unnecessary complexity.

---

## Implement the Requirement That Exists

Do not make code more generic than the current requirement.

Do not introduce:

- plugin systems;
- generalized factories;
- extension registries;
- compatibility frameworks;
- universal adapters;
- shared abstraction layers;
- configuration switches for hypothetical future behavior;

unless a demonstrated requirement justifies them.

Future requirements should be addressed when they become real enough to design correctly.

Do not create complexity today merely to avoid the possibility of changing code later.

Changing understandable code is often cheaper and safer than maintaining speculative abstractions.

---

## Preserve Existing Sound Architecture

Do not refactor working code merely to make it resemble another ISS project, an example in the standards repository, or a preferred architectural pattern.

A change should improve something concrete.

Examples include:

- correctness;
- security;
- testability;
- maintainability;
- performance;
- platform integration;
- recovery;
- operational visibility.

"Consistency" alone is not sufficient justification for restructuring sound project-specific code.

---

## Keep Responsibilities Clear

Functions, types, packages, modules, processes, and services should have understandable responsibilities.

Avoid components that accumulate unrelated behavior.

A component should not become the default location for functionality merely because it already exists.

When responsibilities differ materially in:

- authority;
- lifecycle;
- failure behavior;
- data ownership;
- platform behavior;
- security boundary;

consider separating them.

Do not split code solely to satisfy arbitrary size limits.

Separation should improve understanding or behavior.

---

## Functions Should Be Focused

A function should perform a coherent operation.

Large functions are not automatically wrong.

Small functions are not automatically good.

Split a function when separation improves:

- understanding;
- testing;
- reuse;
- error handling;
- responsibility boundaries.

Do not create one-line helper functions that merely hide straightforward operations without improving meaning.

---

## Types Should Represent Real Concepts

Create types when they clarify a meaningful domain or engineering concept.

Do not create types merely to wrap existing values without adding meaning, validation, safety, or behavior.

A type should make incorrect use harder or correct use clearer.

Avoid large generic result objects that accumulate unrelated fields simply to standardize output across unrelated operations.

Use structures appropriate to the actual data being represented.

---

## Interfaces Must Earn Their Place

Do not create an interface merely because a concrete type exists.

Interfaces are useful when they provide a real benefit, such as:

- multiple meaningful implementations;
- a clear architectural boundary;
- improved testability;
- isolation of an external system;
- separation of platform-specific behavior.

A one-implementation interface with no demonstrated boundary benefit may simply add indirection.

Do not design interfaces around hypothetical future implementations.

Introduce them when the requirement becomes real.

---

## Abstraction Must Reduce Complexity

An abstraction is successful only when it makes the system easier to understand or maintain.

Do not hide important behavior behind abstraction solely to make code appear cleaner.

The following should remain visible when they materially affect operation:

- privilege;
- platform behavior;
- network communication;
- filesystem access;
- destructive operations;
- persistence;
- security boundaries;
- retries;
- timeouts;
- failure modes;
- resource use.

If an engineer needs to understand the underlying mechanism to operate or secure the software, do not obscure it unnecessarily.

---

## Use Native Platform Facilities Where Appropriate

Platform-specific software should use the appropriate mechanisms provided by the platform when practical.

Examples may include:

### Windows

- Windows APIs;
- Windows services;
- Windows security identities;
- service control mechanisms;
- NTFS facilities;
- Event Log;
- native authentication and authorization;
- Windows-specific system calls.

### Linux

- Linux system calls;
- native service management;
- filesystem facilities;
- identity and permission mechanisms;
- kernel interfaces appropriate to the requirement.

### FreeBSD

- FreeBSD-native APIs;
- service mechanisms;
- filesystem and security facilities;
- platform-specific kernel interfaces.

Do not replace appropriate platform-native behavior with a generic third-party abstraction solely to make implementations look identical.

Different platforms may require different internal implementations.

That is acceptable.

---

## Standard Library Before Dependency

Before adding a dependency, determine whether the language standard library or operating system already provides the required capability.

Prefer fewer dependencies when they provide equivalent clarity and correctness.

A dependency may be justified when it materially improves:

- correctness;
- security;
- interoperability;
- maintainability;
- implementation complexity.

Do not reject a high-quality dependency merely to avoid all external code.

Do not add a dependency merely to avoid writing a small amount of straightforward code.

Every dependency adds an ongoing maintenance decision.

---

## Dependencies Must Be Understood

Before introducing a significant dependency, understand:

- what capability it provides;
- what parts of it are actually used;
- its maintenance status;
- its licensing;
- its security implications;
- its update behavior;
- its transitive dependencies where relevant;
- whether it introduces runtime services or network access.

Do not add large frameworks when a small portion of their functionality is required.

---

## Validate at Trust Boundaries

Validation should occur when data crosses a meaningful trust boundary.

Examples include:

- user input;
- network input;
- files;
- configuration;
- database records;
- external API responses;
- operating-system output;
- device output;
- serialized state;
- command output.

Internal values do not need to be repeatedly revalidated merely because defensive programming sounds desirable.

Validate where trust changes or where malformed data can cause meaningful harm.

---

## Preserve Source Facts

Do not unnecessarily destroy or overwrite raw information while transforming it.

If interpretation, normalization, or conversion occurs, preserve enough original information where practical to:

- troubleshoot;
- validate;
- reproduce;
- reinterpret;
- explain discrepancies.

A parser should not silently change uncertain input into a confident conclusion.

---

## Do Not Invent Missing Data

Missing or unknown information should remain visibly missing or unknown.

Do not manufacture default values solely to:

- satisfy a structure;
- avoid an empty field;
- make output look complete;
- prevent downstream code from dealing with uncertainty.

Use explicit states appropriate to the project, such as:

```text
not_known
no_record
not_applicable
unsupported
partial
```

where those concepts are meaningful.

Do not use one universal vocabulary merely because another ISS project uses it.

---

## Errors Must Be Meaningful

Errors should provide enough information to understand what failed.

Do not swallow errors without a specific and justified reason.

Do not convert meaningful failures into success.

Do not return vague errors when useful context is available.

Useful error information may include:

- the operation that failed;
- the relevant target;
- the underlying error;
- whether retry is appropriate;
- whether the result is partial.

Avoid leaking secrets or sensitive data through error messages.

---

## Error Handling Must Preserve Context

When propagating an error, retain enough context to identify the failed operation.

Do not repeatedly wrap errors with meaningless text.

Good context explains what the current layer was attempting.

Bad context merely restates that an error occurred.

---

## Fail Deliberately

Failure behavior should be intentional.

Depending on the operation, the correct response may be:

- stop;
- retry;
- return a partial result;
- skip an item;
- mark the operation unsupported;
- continue safely;
- require operator intervention.

Do not choose failure behavior merely to keep the process running.

Do not terminate an entire system for a recoverable local error unless the architecture requires it.

---

## Do Not Hide Security Downgrades

A security control must not silently disappear when it fails.

Examples include:

- certificate validation;
- authentication;
- authorization;
- signature checking;
- encryption;
- privilege checks;
- protected storage;
- trusted update verification.

If a secure mechanism is unavailable, do not silently fall back to a weaker mechanism unless that fallback is explicitly designed, documented, and appropriate to the security model.

---

## Resource Ownership Must Be Clear

Code that opens or allocates resources must clearly establish who is responsible for releasing them.

Examples include:

- files;
- handles;
- sockets;
- database connections;
- locks;
- buffers;
- goroutines or threads;
- temporary files;
- operating-system objects.

Resource lifetime should be understandable from the code.

Do not rely on distant cleanup mechanisms when local cleanup is practical.

---

## Bound Untrusted Work

Operations influenced by external or untrusted input should consider limits.

Examples include:

- maximum input size;
- maximum line length;
- queue size;
- worker count;
- recursive depth;
- retry count;
- connection count;
- timeout;
- retained history;
- memory growth.

Limits should reflect realistic engineering constraints.

Do not choose arbitrary limits without understanding their operational effect.

---

## Concurrency Must Solve a Real Problem

Do not introduce concurrency simply because the language makes it convenient.

Concurrency should address a real need, such as:

- independent I/O;
- latency;
- throughput;
- responsiveness;
- parallelizable work.

Concurrent code must consider:

- ordering;
- cancellation;
- shutdown;
- shared state;
- races;
- resource exhaustion;
- duplicate work;
- error propagation.

Test concurrency-specific behavior.

---

## Avoid Unnecessary Data Movement

Do not repeatedly copy, serialize, deserialize, encode, decode, or transform data without an engineering reason.

Unnecessary data movement can increase:

- memory use;
- CPU use;
- latency;
- complexity;
- opportunities for data loss or reinterpretation.

This does not mean optimizing every allocation.

Avoid obvious waste first.

Measure before making more invasive optimizations.

---

## Measure Before Optimizing

Performance changes should be based on observed behavior where practical.

Do not complicate code for speculative performance gains.

When performance matters:

1. measure;
2. identify the real bottleneck;
3. change the relevant code;
4. measure again;
5. retain a benchmark or regression test when useful.

An optimization that cannot be shown to improve the meaningful workload deserves scrutiny.

---

## Comments Explain Why

Comments should explain information that is not obvious from the code.

Useful comments may describe:

- why a particular approach was chosen;
- platform behavior;
- security implications;
- unusual edge cases;
- compatibility requirements;
- protocol details;
- external constraints.

Avoid comments that merely restate syntax.

Keep comments accurate as code changes.

Incorrect comments are worse than missing comments.

---

## Explain Unusual Low-Level Code

Low-level or platform-native code may require additional explanation.

Where appropriate, document:

- the native API being used;
- why particular flags are required;
- security implications;
- lifetime rules;
- platform assumptions;
- unusual behavior;
- compatibility considerations.

Do not replace correct low-level implementation with a generic abstraction merely because the low-level code requires explanation.

Explain it.

---

## External Examples Must Be Identified

If code or an implementation approach is materially derived from external sample code, documentation, or another project, identify the source where appropriate.

Do not present copied or adapted code as original implementation.

Ensure licensing permits the use.

Prefer official platform or language documentation when available.

---

## Configuration Should Be Necessary

Do not create a configuration option for every internal decision.

Configuration creates:

- additional states;
- additional testing;
- additional documentation;
- additional support burden;
- additional failure modes.

Expose configuration when operators have a legitimate reason to control the behavior.

Keep internal implementation decisions internal when there is no operational need to expose them.

---

## Defaults Must Be Safe and Useful

When configuration is optional, defaults should be reasonable for the intended deployment.

A default should not:

- unnecessarily weaken security;
- create uncontrolled resource use;
- enable destructive behavior;
- expose services unexpectedly.

Avoid defaults that technically work but require every real deployment to immediately override them.

---

## Configuration Errors Must Be Visible

Invalid configuration should not silently become another configuration.

If an option is invalid, unsupported, malformed, or contradictory, report it clearly.

Do not silently substitute a default when doing so could hide an administrator mistake.

---

## Logging Must Serve a Purpose

Logs should help answer useful operational questions.

Log information such as:

- meaningful state changes;
- failures;
- important security actions;
- recovery events;
- significant configuration changes;
- unexpected conditions.

Avoid excessive repetitive logging that obscures important events or creates unnecessary storage and I/O.

Never log secrets merely for troubleshooting convenience.

---

## Structured Output Should Reflect Meaning

Use structured output when it provides useful machine-readable information.

Do not force every operation into a universal schema.

A structure should reflect the actual meaning of the data.

Do not add fields merely because another command or project has them.

Avoid deeply nested output without a clear reason.

Human-readable and machine-readable output may serve different purposes.

Design each intentionally.

---

## APIs and Contracts Must Be Explicit

When an interface becomes a real external contract, define it clearly.

This includes:

- network APIs;
- file formats;
- database schemas;
- persistent state;
- command-line interfaces;
- public packages;
- integration protocols.

Changes to established contracts should be deliberate.

Do not accidentally create public contracts from internal implementation details.

---

## Compatibility Has a Cost

Once external systems or users depend on a behavior, changing it may become expensive.

Be deliberate about what is exposed publicly.

Do not promise compatibility that the project is not prepared to maintain.

Internal formats should remain internal where external compatibility is unnecessary.

---

## Security-Sensitive Code Deserves Extra Scrutiny

Code involving the following should receive deliberate security review:

- authentication;
- authorization;
- privilege;
- credentials;
- cryptography;
- parsing untrusted input;
- update mechanisms;
- command execution;
- filesystem modification;
- recovery authority;
- administrative APIs;
- network listeners.

The implementation should make the security boundary understandable.

Do not bury security-critical decisions inside generic helpers where they become difficult to inspect.

---

## Destructive Code Must Look Destructive

Code that can:

- delete;
- overwrite;
- disable;
- revoke;
- modify security;
- alter system configuration;
- restore data;
- execute administrative actions;

should be visibly distinct from read-only or observational code.

Avoid generic functions that can silently perform either read or write behavior depending on hidden configuration.

Make high-impact behavior obvious to reviewers.

---

## Tests Are Part of the Implementation

Testing is not separate from engineering.

When implementation introduces meaningful behavior, its tests should normally be developed with it.

A bug fix should include a regression test where practical.

A security fix should include a test that demonstrates the protected condition where practical.

A platform-specific behavior claim should include platform validation where practical.

Do not remove difficult tests merely because they reveal a design problem.

Fix the problem.

---

## Testability Should Influence Design

Code should be testable without distorting the production architecture.

Appropriate techniques may include:

- dependency injection;
- small interfaces;
- test doubles;
- temporary files or repositories;
- controlled fixtures;
- native integration environments.

Do not introduce large abstraction frameworks solely to make testing easier.

Use the smallest boundary that provides meaningful test control.

---

## Do Not Test the Mock Instead of the System

Mocks and fakes are useful for testing logic.

They do not establish that the actual platform, service, API, filesystem, database, or device behaves as assumed.

Where real platform behavior matters, validate against the real platform.

Use the correct test level for the claim being made.

---

## Source Layout Should Reflect the Project

Repository structure should make the project understandable.

Do not force all ISS projects into identical directory layouts.

Organize source according to:

- language conventions;
- platform requirements;
- component boundaries;
- testing needs;
- project size.

A structure should serve the project.

The project should not serve the structure.

---

## Language Style Should Be Language Appropriate

Each language has conventions that improve readability and maintainability.

ISS code should generally follow the established conventions of the language unless there is a clear reason not to.

Language-specific ISS guidance may define additional preferences.

Those preferences apply only to projects using that language.

They do not establish rules for unrelated languages.

---

## Platform-Specific Code Should Be Explicit

When behavior genuinely differs between operating systems or environments, make the distinction visible.

Do not hide major platform differences behind complicated conditional logic where separate implementations would be clearer.

Use language-supported platform separation mechanisms where appropriate.

The goal is not identical source.

The goal is correct behavior on each supported platform.

---

## Keep Build and Development Steps Understandable

A developer should be able to determine how to:

- build the project;
- run tests;
- perform important validation;
- reproduce common development tasks.

Avoid build systems whose complexity substantially exceeds the software being built.

Generated steps should not hide important behavior.

---

## Generated Code Must Have a Reason

Generated code can be appropriate where it provides clear value.

If code is generated:

- identify the generator;
- identify the source;
- make regeneration understandable;
- avoid manual edits to generated output;
- ensure generated artifacts can be reproduced where practical.

Do not generate code merely to avoid writing straightforward source.

---

## Dead Code Should Not Accumulate

Remove code that no longer serves the project unless there is a specific compatibility or transition reason to retain it.

Do not keep abandoned implementations "just in case."

Version control already preserves history.

Commented-out code should not become long-term storage.

---

## TODOs Must Be Actionable

TODO comments should identify real unfinished work.

Prefer enough context to understand:

- what remains;
- why it remains;
- whether it affects correctness or safety.

Do not use TODOs as permanent placeholders for vague future improvements.

---

## Fix Root Causes Where Practical

Do not repeatedly patch symptoms when the underlying defect is understood and can reasonably be corrected.

Workarounds may be necessary for:

- platform bugs;
- external compatibility;
- legacy systems;
- operational constraints.

When a workaround is required, document why it exists and retain a test for the condition where practical.

---

## Review the Change, Not Just the Diff

Before considering implementation complete, understand the effect of the change on:

- behavior;
- security;
- permissions;
- data;
- performance;
- recovery;
- compatibility;
- tests;
- documentation.

A small code diff can create a large operational change.

A large refactor may create no intended behavioral change.

Review based on impact.

---

## Keep Changes Focused

Prefer changes that solve one understandable problem.

Do not combine unrelated cleanup, architectural redesign, feature development, and security changes into one change unless there is a strong engineering reason.

Focused changes are easier to:

- review;
- test;
- understand;
- revert;
- diagnose.

Do not split tightly coupled changes merely to satisfy arbitrary size limits.

---

## Be Truthful About Incomplete Work

Incomplete functionality should not be presented as complete.

Temporary limitations, unimplemented paths, unsupported cases, and partial validation should remain visible.

Do not hide unfinished behavior behind optimistic naming or status.

---

## Reference Code Remains Reference Code

Code in the Iron Signal Systems Engineering Standards repository is example and reference material unless a requirement is explicitly identified as mandatory.

Agents and developers must not:

- copy example architecture into projects by default;
- create shared frameworks from standards examples;
- infer mandatory types or interfaces from sample code;
- refactor project code merely to resemble an example;
- generalize one language's implementation into another language;
- generalize one platform's implementation into another platform.

A mandatory requirement must be stated explicitly.

Examples explain possible implementations.

They do not silently define them.

---

## Final Principle

Good implementation is not the code with the most abstractions, the fewest lines, the newest dependencies, or the most elaborate framework.

Good implementation makes the intended behavior clear, keeps authority and failure visible, fits the platform it runs on, can be tested meaningfully, and remains understandable when someone has to diagnose it years later.

Code should earn trust by making its behavior understandable.
