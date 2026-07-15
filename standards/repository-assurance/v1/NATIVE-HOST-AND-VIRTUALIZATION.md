# Native Host, VM, and Container Policy

## Native-first principle

Iron Signal Systems does not use containers as a universal substitute for
understanding dependencies, operating-system behavior, service management,
identity, networking, storage, or deployment.

Projects must be testable through declared native hosts, disposable virtual
machines, specialized labs, or a combination appropriate to their accepted
deployment model.

## Permitted container use

Containers may be used for:

- developer convenience;
- isolated third-party tools;
- disposable data services;
- test fixtures;
- a product whose accepted deployment is container-native.

## Prohibited container dependence

A container must not:

- conceal undeclared host requirements;
- become the only way to run ordinary source tests without an accepted reason;
- falsely represent systemd, kernel, AD, filesystem ACL, device, or host security
  behavior that is not actually exercised;
- carry untracked project inputs;
- replace backup, restore, failover, or deployment testing of the real target
  environment.

## Virtual machines

Disposable VMs are preferred for canonical and specialized campaigns requiring
real operating-system behavior, systemd, Windows domains, kernel features,
privileged networking, recovery, or destructive testing.

## Cross-system expectation

Every approved development system must be able to run portable validation and
identify unavailable specialized capabilities without reporting a false test
failure.
