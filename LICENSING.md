# Licensing Decision

## Current decision

Repository-authored materials in source revisions containing the root
`LICENSE` file are licensed under the:

**BSD 3-Clause "New" or "Revised" License**

SPDX identifier: `BSD-3-Clause`

Copyright (c) 2026, Iron Signal Systems.

The complete controlling license text is retained in [`LICENSE`](LICENSE).

## Effective source boundary

This decision becomes effective at the first exact repository commit containing:

- the root `LICENSE` file;
- this revised `LICENSING.md`;
- synchronized validation and source-manifest updates.

The exact commit containing those files is the auditable licensing boundary.
This wording avoids attempting to place a commit's own future object identifier
inside itself.

## Covered materials

Unless a file or directory contains a clearly identified different license,
BSD-3-Clause applies to repository-authored:

- normative documentation;
- engineering and acceptance documentation;
- schemas;
- templates;
- validation and workflow tooling;
- integration guides;
- examples and fixtures.

Third-party materials remain subject to their respective licenses and notices.

## Historical release boundary

The signed `isras-v2.0.0` release and remote `main` source at commit
`781246e69f8a9a382c25040f94b62dfe3b25ba89` predate this licensing change.

This decision does not modify, replace, retag, or rewrite:

- the immutable `isras-v2.0.0` tag;
- its annotated tag object;
- its accepted source commit;
- its historical acceptance evidence;
- the licensing record contained in that signed source tree.

BSD-3-Clause applies to source revisions that carry the new root `LICENSE`
file and to future releases that include it.

## Compatibility impact

BSD-3-Clause permits source and binary redistribution, with or without
modification, provided its copyright notice, conditions, and disclaimer are
preserved.

The license does not authorize use of the Iron Signal Systems name or
contributor names to endorse or promote derived products without specific prior
written permission.

The license provides the materials without warranty and limits liability as
stated in `LICENSE`.

## Contribution licensing

Contributions accepted after this decision must be provided under terms
compatible with BSD-3-Clause. Contributors must have authority to submit their
work and must identify any third-party material or incompatible licensing
obligation.
