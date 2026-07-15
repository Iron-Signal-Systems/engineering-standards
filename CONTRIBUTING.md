# Contributing

## Development branch

Create a purpose-named work branch from `dev`. Submit a pull
request back to `dev`.

## Required change set

A material change includes all applicable:

- requirement and architecture updates;
- implementation;
- tests and hostile cases;
- fixtures and expected outcomes;
- validation changes;
- environment changes;
- synchronized documentation;
- acceptance and non-claim updates.

Documentation is not follow-up cleanup.

## Validation

Before opening a pull request:

```bash
./tools/validation/validate_portable.sh
```

On Windows:

```powershell
.\tools\validation\Validate-Portable.ps1
```

Before formal acceptance, run fresh-clone, canonical, specialized, and
historical predecessor validation as applicable.

## Secrets

Never commit passwords, tokens, private keys, full credential-bearing
connection strings, production data, or unrestricted logs.
