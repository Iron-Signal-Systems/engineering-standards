# Failure Logging and Remediation

## 1. Failure logs

Every failed check shall create a local file ending in `.log` under:

```text
.local/validation/logs/
```

Failure logs are local support artifacts and are ignored by Git by default.
They shall include:

- repository and working directory;
- branch and exact current commit when available;
- validation mode;
- check name;
- start and failure times;
- command and exit code when applicable;
- expected and observed outcomes;
- censored standard output and standard error;
- exact safe next actions.

Logs shall be created with owner-only permissions where supported.

## 2. Censoring

Censoring protects terminal output and logs. It does not repair source and does
not turn a failure into a pass.

Potential secrets shall appear as `[REDACTED]`. Paths, rule names, finding IDs,
line numbers, and non-sensitive context may remain visible.

## 3. Terminal actions

Each actionable failure shall display only commands relevant to that failure.
Every displayed command shall be labeled as one of:

- `READ ONLY`
- `CREATES LOCAL PLAN`
- `CREATES EXCEPTION PROPOSAL`
- `MODIFIES WORKING TREE`
- `MODIFIES TRACKED ALLOWLIST`
- `NETWORK ACCESS — INSTALLS PINNED TOOL`

Commands shall be shell-safe, omit detected values, and identify when human
judgment is required.

## 4. Secret response

If a real credential may have been committed or pushed:

1. rotate or revoke it;
2. remove it from source;
3. determine which commits, branches, tags, releases, logs, or artifacts contain
   it;
4. clean history when appropriate;
5. record the incident without recording the secret;
6. rerun complete validation.

Redaction alone does not invalidate an already exposed credential.
