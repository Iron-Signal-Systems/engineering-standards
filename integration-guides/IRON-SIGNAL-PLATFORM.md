# Iron Signal Platform Integration

Canonical repository:

```text
Iron-Signal-Systems/iron-signal-platform
```

Canonical Arch path:

```text
/src/iron-signal-platform
```

## Sequence

1. Complete and accept Phase 6 Step 9 before opening the repository-assurance
   implementation boundary.
2. Create a purpose-named branch from `dev`.
3. Preview adoption:

   ```bash
   cd /path/to/engineering-standards
   python3 tools/isras/adopt.py \
     --target /src/iron-signal-platform \
     --repository Iron-Signal-Systems/iron-signal-platform \
     --canonical-origin git@github.com:Iron-Signal-Systems/iron-signal-platform.git \
     --development-branch dev \
     --release-branch main \
     --profile go-postgresql-systemd \
     --dry-run
   ```

4. Do not overwrite existing phase gates or documentation. Merge the baseline
   governance, environment, and entrypoint model into the existing structure.
5. Preserve the exact historical isolated-clone method.
6. Configure portable validation separately from PostgreSQL 18 and systemd
   canonical validation.
7. Run fresh-clone validation from all three development systems.
8. Add GitHub workflows in observation mode.
9. Accept the repository-assurance boundary before requiring checks.
10. Begin the thin CAD vertical slice only after that boundary is stable.

## Platform-specific specialized profiles

- canonical Arch and pinned Go toolchain;
- PostgreSQL 18;
- systemd deployment;
- Windows AD and authentication gateway;
- backup, PITR, failover, and trusted recovery;
- multi-instance replay;
- delivery terminal-state campaign;
- performance and backlog recovery.
