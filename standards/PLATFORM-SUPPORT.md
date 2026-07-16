# Platform Support

## Default profile

Unless a project declares another scope, Iron Signal Systems validation and
project tooling are focused on:

### Primary development platform

- Arch Linux

### Supported server platforms

- Ubuntu Server LTS releases that remain within upstream support;
- Fedora Server releases that remain supported upstream.

A project may explicitly add Windows Server, Windows workstations, macOS,
appliances, embedded systems, specialized public-safety workstations, or other
Linux distributions.

## Native-first rule

Native hosts, virtual machines, and specialized labs are valid declared
environments. Containers are optional and shall not hide undeclared runtime or
host dependencies. A container is not required unless the accepted deployment
model is itself container-native.

## Validation behavior

The validator records the detected operating system and architecture. An
undeclared platform is reported clearly and shall not be silently represented as
supported.
