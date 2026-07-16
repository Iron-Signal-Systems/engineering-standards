# Deterministic Validation-Tool Bootstrap

## Developer bootstrap

The developer bootstrap creates an isolated virtual environment, does not
implicitly upgrade pip, installs declared top-level versions, and records the
Python, pip, platform, requirements, and installed-distribution state. It may
use a package index and resolver and therefore is development evidence only.

## Release bootstrap

Release bootstrap starts from an **absent** destination virtual environment and
an accepted environment-specific offline wheelhouse. Reusing or overlaying an
existing environment is prohibited because undeclared distributions and
configuration can survive an otherwise successful install.

The release path:

- verifies the base Python implementation, version, ABI, architecture, and
  executable SHA-512 identity before creating the environment;
- verifies the wheelhouse using only the Python standard library;
- requires the wheelhouse file set to match `SHA512SUMS` exactly;
- verifies each wheel against its recorded SHA-512 and retained upstream archive
  hashes and source URL;
- force-installs the accepted pip wheel, rather than accepting an already
  installed pip distribution;
- invokes Python with isolated mode and pip with `--isolated`, `--no-index`,
  `--no-cache-dir`, `--only-binary=:all:`, and `--require-hashes`;
- disables user-site packages, `PYTHONPATH`, `PYTHONHOME`, and external pip
  configuration;
- removes bootstrap-only distributions after installation;
- proves that the final installed distribution set exactly equals the lock; and
- records the requirements, lock, wheelhouse manifest, Python executable, pip
  distribution tree, platform, and installed-distribution identities.

`tools/environment/prepare_tool_wheelhouse.py` creates a candidate wheelhouse in
a connected controlled environment. Pip resolution reports are transient inputs
and are intentionally excluded from the accepted wheelhouse file set. The
preparer verifies upstream archive hashes before materializing wheels and writes
sanitized provenance into `bootstrap-lock.json`.

Completion does not make a wheelhouse trusted. It must be reviewed,
vulnerability-checked, malware-scanned under the approved process, validated,
and accepted by exact digest. One wheelhouse is required for each accepted OS,
architecture, Python implementation, version, ABI, and executable boundary.

## Digest transition

SHA-512 is primary for new v3 wheelhouse and evidence relationships. SHA-256 is
retained for immutable v1/v2 identities, Git and SSH ecosystem identities,
external formats requiring it, and explicit transition records.

Digest length alone does not create trust. Source, signer, environment,
authority, and accepted decision boundaries remain required.

## Restricted-network operation

An approved wheelhouse may be staged through governed artifact storage or
approved removable media. Release runners require no package-index access.
Network denial should also be enforced independently by the runner environment.
