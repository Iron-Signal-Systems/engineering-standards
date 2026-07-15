# Validation Environment Model

Repositories define portable, canonical, specialized, release, and recovery
environment profiles as applicable.

Profiles declare:

- permitted operating systems and architectures;
- required commands and command-version patterns;
- pinned Python validation modules;
- required environment-variable names without secret values;
- capabilities and explicit non-capabilities;
- whether containers are required by an accepted product boundary.

The environment doctor emits a human-readable result and can write a
machine-readable fingerprint. Command presence without version verification is
not sufficient for canonical or formal acceptance.

Containers remain optional unless the accepted product deployment is itself
container-native. Native hosts and disposable virtual machines are first-class
validation environments.
