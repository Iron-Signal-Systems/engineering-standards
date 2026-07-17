# Platform Support

## Evidence terminology

Platform-support statements shall identify the evidence actually maintained.
The following terms are not interchangeable:

- **Native CI-validated** means the validation job runs directly on a fresh
  virtual machine for the named operating-system release.
- **Distribution-userland CI-validated** means the validation job runs inside an
  official distribution OCI image on a GitHub-hosted Linux runner.
- **Native developer-validated** means a developer has run the accepted
  validation workflow on a native host, but that result is not continuous
  hosted CI evidence.
- **Declared target** means the platform is within intended scope but does not
  yet have equivalent automated evidence.

A support claim shall not silently imply that every declared platform has the
same validation depth.

## Default profile

Unless a project declares another scope, Iron Signal Systems validation and
project tooling are focused on the following platforms.

### Primary development platform

- Arch Linux is the primary native development platform.
- Arch Linux receives continuous distribution-userland CI validation using the
  official `archlinux:base-devel` OCI image.
- Native Arch Linux development validation remains distinct from the container
  evidence and shall not be represented as native hosted CI.

### Supported server platforms

- Ubuntu Server 22.04 LTS receives native CI validation on the GitHub-hosted
  `ubuntu-22.04` virtual-machine runner.
- Ubuntu Server 24.04 LTS receives native CI validation on the GitHub-hosted
  `ubuntu-24.04` virtual-machine runner.
- Fedora Server 43 receives continuous distribution-userland CI validation
  using the official `fedora:43` OCI image.
- Fedora Server 44 receives continuous distribution-userland CI validation
  using the official `fedora:44` OCI image.

Ubuntu 26.04 GitHub-hosted runners remain public-preview evidence and are not
part of the required Solo Developer Baseline matrix until a deliberate,
documented support change promotes that release line.

A project may explicitly add Windows Server, Windows workstations, macOS,
appliances, embedded systems, specialized public-safety workstations, other
Linux distributions, or additional operating-system release lines.

## Active automated evidence

The active workflow evidence is:

| Platform | Validation environment | Required evidence |
| --- | --- | --- |
| Ubuntu Server 24.04 LTS | Native GitHub-hosted virtual machine | Every applicable push, pull request, and weekly schedule |
| Ubuntu Server 22.04 LTS | Native GitHub-hosted virtual machine | Every applicable push, pull request, and weekly schedule |
| Arch Linux | Official Arch Linux OCI userland | Every applicable push, pull request, and weekly schedule |
| Fedora Server 43 | Official Fedora OCI userland | Every applicable push, pull request, and weekly schedule |
| Fedora Server 44 | Official Fedora OCI userland | Every applicable push, pull request, and weekly schedule |

The Arch image intentionally follows the rolling official `base-devel` tag and
performs a complete package update before validation. Fedora images follow the
named supported release lines and receive current updates before validation.

## Container-userland evidence boundary

Distribution-userland CI validates behavior involving the declared
userland, including:

- package-manager and package availability;
- shell and core utility behavior;
- Git behavior;
- declared Go toolchain installation;
- source formatting, static analysis, tests, builds, and module integrity;
- known-vulnerability validation;
- repository-owned sensitive-value validation;
- filesystem layout and dynamically linked userland compatibility used by the
  validation workflow.

It does not establish native evidence for:

- the distribution kernel or boot process;
- systemd service behavior;
- SELinux enforcement;
- host firewall behavior;
- native package installation and upgrade behavior outside the CI bootstrap;
- storage, hardware, driver, or device integration;
- production deployment, recovery, or operational readiness.

Those properties require project-specific native-host, integration,
deployment, recovery, and operational validation.

## Lifecycle maintenance

Platform release lines and workflow images shall be changed together with this
document and the changelog. A distribution reaching end of upstream support
shall not remain described as supported merely because an old image tag still
exists.

A new Ubuntu LTS, Fedora release, or material Arch image-policy change begins as
an explicitly identified preview or declared target until its required evidence
is accepted. Promotion into the required matrix is a deliberate change, not an
implicit consequence of an upstream release.

## Native-first rule

Native hosts, virtual machines, and specialized labs are valid declared
environments. Containers are optional and shall not hide undeclared runtime or
host dependencies. A container is not required unless the accepted deployment
model is itself container-native.

The platform CI containers exist to provide bounded compatibility evidence;
they do not redefine the accepted deployment model as container-native.

## Validation behavior

The validator records the detected operating system and architecture. An
undeclared platform is reported clearly and shall not be silently represented as
supported.
