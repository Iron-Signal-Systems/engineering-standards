[CmdletBinding()]
param(
    [string]$Profile = "portable",
    [string]$FingerprintOutput
)
$ErrorActionPreference = "Stop"
$repoRoot = (& git rev-parse --show-toplevel 2>$null)
if (-not $repoRoot) { throw "Not in a Git work tree." }
$pythonPath = $env:ISRAS_PYTHON
if (-not $pythonPath) {
    $python = Get-Command python3 -ErrorAction SilentlyContinue
    if (-not $python) { $python = Get-Command python -ErrorAction SilentlyContinue }
    if (-not $python) { throw "Python 3 is required for the baseline environment doctor." }
    $pythonPath = $python.Source
}
$argsList = @(
    "$repoRoot/tools/isras/doctor.py",
    "--repo-root", "$repoRoot",
    "--profile", $Profile
)
if ($FingerprintOutput) { $argsList += @("--fingerprint-output", $FingerprintOutput) }
& $pythonPath @argsList
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
