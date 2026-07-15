# Repository Assurance Adoption

This repository adopts the Iron Signal Repository Assurance Standard.

## Governing rule

A change is complete only when its exact pushed commit can be reconstructed,
validated, and evidenced from the canonical repository using declared
environments and committed project-owned assets.

## Native-first boundary

Portable validation runs directly on approved hosts. Canonical and specialized
validation may use native hosts or disposable VMs. Containers are optional and
are not the sole validation path unless the accepted deployment model requires
them.

## Adoption status

- Standard commit: `UNPINNED-BOOTSTRAP`
- Adoption level: `RECORDED`
- Required checks: observation mode
- Independent human review: not claimed
- Production readiness: not claimed
