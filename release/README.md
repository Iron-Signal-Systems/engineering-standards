# Release Artifact Source Lists

`framework-files.txt` and `contract-files.txt` are the authoritative, sorted,
unique source-file inventories for the corresponding deterministic release
archives.

The artifact producer reads these files from the exact signed release commit
through Git. It does not discover files by walking the working tree. Every listed
path must resolve to a tracked regular blob at the release commit.

Changing either list changes the produced archive bytes and therefore requires a
new release commit, tag, manifests, provenance, and project pin.
