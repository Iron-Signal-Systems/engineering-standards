[CmdletBinding()]
param(
    [string]$Profile = "portable"
)

$ErrorActionPreference = "Stop"
$repoRoot = (& git rev-parse --show-toplevel 2>$null)
if (-not $repoRoot) {
    throw "Not in a Git work tree."
}

$python = Get-Command python3 -ErrorAction SilentlyContinue
if (-not $python) {
    $python = Get-Command python -ErrorAction SilentlyContinue
}
if (-not $python) {
    throw "Python 3 is required for the baseline environment doctor."
}

& $python.Source "$repoRoot/tools/isras/doctor.py" `
    --repo-root "$repoRoot" `
    --profile $Profile

if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
