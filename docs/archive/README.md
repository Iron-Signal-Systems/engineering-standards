# Previous ISRAS Work Archive

The restart installer preserves the complete pre-restart repository before it
replaces the active tree.

The installer creates:

- remote/local branch `archive/isras-v1-v3-development-2026-07-16`;
- signed annotated tag `pre-practical-baseline-archive-2026-07-16`;
- a local `engineering-standards-isras-v1-v3-archive-20260716.bundle`;
- SHA-256 and SHA-512 digest files for the bundle;
- an archive manifest identifying the exact source commit and repository.

The archive retains accepted v1 and v2 work, the v3 development candidate, its
schemas, templates, evidence models, Python tooling, historical validators,
release records, and all Git history available at the archive point.

This material is not discarded. It remains a future design source for stronger
team, production, regulated, historical-verification, and independently
reviewed ISRAS profiles.

The active solo-developer baseline does not automatically inherit every archived
control. Controls move forward deliberately when they solve a current risk and
can be performed truthfully.
