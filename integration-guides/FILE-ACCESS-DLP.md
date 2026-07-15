# File-Access-DLP Integration

Canonical local macOS path currently used:

```text
~/Dev/projects/File-Access-DLP
```

Use the canonical repository URL configured by that project.

## Profile

```text
dotnet-python-powershell-windows-ad
```

## Required focus

- locked .NET restore, build, test, analyzers, and publish checks;
- Python syntax, tests, deterministic generation, and locked dependencies;
- PowerShell parser, PSScriptAnalyzer, Pester, `-WhatIf`, marker, OU, and
  teardown-scope validation;
- five deterministic forest scenarios;
- exact expected-count manifests;
- per-file SHA-256 manifests;
- byte-identical regeneration of source fixtures;
- runtime observation manifests for generated AD SIDs and GUIDs;
- ACL and effective-access campaigns;
- broken trust and unresolved principal campaigns;
- staging/ingestion server boundary validation;
- no scanner or endpoint-agent direct PostgreSQL access.

The Windows AD lab is specialized acceptance infrastructure and must not process
public pull-request code.
