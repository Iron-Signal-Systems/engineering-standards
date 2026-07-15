[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)]
    [string]$Checkpoint
)

$ErrorActionPreference = "Stop"
$repoRoot = (& git rev-parse --show-toplevel 2>$null)
if (-not $repoRoot) { throw "Not in a Git work tree." }

$python = Get-Command python3 -ErrorAction SilentlyContinue
if (-not $python) { $python = Get-Command python -ErrorAction SilentlyContinue }
if (-not $python) { throw "Python 3 is required." }

& $python.Source "$repoRoot/tools/isras/validate_checkpoint.py" `
    --repo-root "$repoRoot" `
    --checkpoint $Checkpoint

if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
