# Documentation and Traceability

## Same-change-set rule

Documentation updates are part of the same change set as the architecture,
implementation, validation, phase boundary, sequencing, environment, or
acceptance change they describe.

## Required states

A repository distinguishes:

- proposed;
- implementation candidate;
- accepted checkpoint;
- formally accepted boundary;
- deprecated;
- superseded;
- withdrawn.

A document must not claim acceptance before the exact acceptance record exists.

## Traceability

Material controls and capabilities should link:

- requirement identifier;
- architecture or decision record;
- implementation path;
- positive, negative, hostile, and concurrency tests;
- validator;
- accepted commit and tag;
- known non-claims.

## Generated documentation

Repositories that generate diagrams, reports, inventories, or documents retain:

- source inputs;
- generator source;
- schemas;
- normalization rules;
- expected counts or structure;
- per-file hashes;
- reproducibility classification.

Byte-identical regeneration is required only where the format permits it.
Unstable timestamps, archive ordering, GUIDs, or tool metadata must be
normalized or explicitly classified as observed output.
